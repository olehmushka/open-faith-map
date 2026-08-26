// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves CreateChildOrg's directory-wrapping path and SearchSites' position-oracle fix against a real
// Postgres instance — see internal/directory/directory_integration_test.go's own header comment for
// why this class of correctness needs a live database, not a mocked store, and the exact invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/religion/... -run TestReligionIntegration -v
package religion_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	directoryapp "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	locationapp "github.com/olehmushka/open-faith-map/internal/location/application"
	locationdomain "github.com/olehmushka/open-faith-map/internal/location/domain"
	"github.com/olehmushka/open-faith-map/internal/religion/adapters"
	"github.com/olehmushka/open-faith-map/internal/religion/application"
	"github.com/olehmushka/open-faith-map/internal/religion/domain"
)

func TestReligionIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close) // registered first: LIFO runs DB cleanup (below) before the pool closes.

	dir := directoryapp.NewService(pool)
	rel := application.NewService(pool, dir)
	loc := locationapp.NewService(pool)

	var unitIDs, locationIDs, siteIDs, policyIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range siteIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_sites WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete site %s: %v", id, err)
			}
		}
		for _, id := range locationIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.location_locations WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete location %s: %v", id, err)
			}
		}
		for _, id := range policyIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_org_policies WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete policy %s: %v", id, err)
			}
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_org_classifications WHERE unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete org classifications for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_org_profiles WHERE unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete org profile for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_closure WHERE ancestor_id = $1 OR descendant_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete closure rows for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_edges WHERE parent_id = $1 OR child_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete edges for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete unit %s: %v", id, err)
			}
		}
	})

	// --- CreateChildOrg happy path: wraps directory.CreateUnitWithEdge atomically, then sets the
	// profile + primary classification.
	root, err := dir.CreateUnit(ctx, directorydomain.Unit{Name: "M10.5 test root"})
	if err != nil {
		t.Fatalf("CreateUnit(root): %v", err)
	}
	unitIDs = append(unitIDs, root.ID)

	var taxonID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_taxa WHERE code = 'christianity' AND deleted_at IS NULL`).Scan(&taxonID); err != nil {
		t.Fatalf("lookup christianity taxon: %v", err)
	}

	profile, err := rel.CreateChildOrg(ctx, root.ID, "m10-5-test-child", "M10.5 Test Child", nil, &taxonID)
	if err != nil {
		t.Fatalf("CreateChildOrg: %v", err)
	}
	unitIDs = append(unitIDs, profile.UnitID)
	if len(profile.Classifications) != 1 || !profile.Classifications[0].IsPrimary || profile.Classifications[0].TaxonID != taxonID {
		t.Errorf("CreateChildOrg profile.Classifications = %+v, want one primary classification on %s", profile.Classifications, taxonID)
	}
	if _, err := dir.GetUnit(ctx, profile.UnitID); err != nil {
		t.Errorf("GetUnit(child): %v (CreateUnitWithEdge should have created a real directory unit)", err)
	}
	ancestors, err := dir.Ancestors(ctx, profile.UnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("Ancestors(child): %v", err)
	}
	if len(ancestors) != 1 || ancestors[0].ID != root.ID {
		t.Errorf("Ancestors(child) = %+v, want [root]", ancestors)
	}

	// --- CreateChildOrg is blocked by an active excludes_child_creation policy on the parent.
	excludedParent, err := dir.CreateUnit(ctx, directorydomain.Unit{Name: "M10.5 test excluded parent"})
	if err != nil {
		t.Fatalf("CreateUnit(excludedParent): %v", err)
	}
	unitIDs = append(unitIDs, excludedParent.ID)

	var policyKindID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_policy_kinds WHERE code = $1`, domain.PolicyExcludesChildCreation).Scan(&policyKindID); err != nil {
		t.Fatalf("lookup excludes_child_creation policy kind: %v", err)
	}
	var policyID string
	if err := pool.QueryRow(ctx, `INSERT INTO openfaithmap.religion_org_policies (unit_id, policy_kind_id) VALUES ($1, $2) RETURNING id`, excludedParent.ID, policyKindID).Scan(&policyID); err != nil {
		t.Fatalf("insert excludes_child_creation policy: %v", err)
	}
	policyIDs = append(policyIDs, policyID)

	if _, err := rel.CreateChildOrg(ctx, excludedParent.ID, "m10-5-test-blocked", "Blocked", nil, nil); err != domain.ErrChildCreationExcluded {
		t.Errorf("CreateChildOrg under excluded parent error = %v, want ErrChildCreationExcluded", err)
	}

	// --- SearchSites' position-oracle fix: a `hidden` site must never appear in the result set, even
	// at a radius that would trivially include it; a sibling `exact` site at the same point must.
	var countryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.refdata_countries WHERE code = 'UA'`).Scan(&countryID); err != nil {
		t.Fatalf("lookup UA country: %v", err)
	}
	var churchSiteTypeID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_site_types WHERE code = 'church' LIMIT 1`).Scan(&churchSiteTypeID); err != nil {
		t.Fatalf("lookup church site type: %v", err)
	}

	lat, lng := 50.4501, 30.5234 // Kyiv — arbitrary, just needs to be a real WGS84 point.
	hiddenLoc, err := loc.CreateLocation(ctx, locationdomain.LocationInput{Latitude: lat, Longitude: lng, CountryID: countryID})
	if err != nil {
		t.Fatalf("CreateLocation(hidden): %v", err)
	}
	locationIDs = append(locationIDs, hiddenLoc.ID)
	hiddenSite, err := rel.CreateSite(ctx, siteInput(profile.UnitID, hiddenLoc.ID, churchSiteTypeID))
	if err != nil {
		t.Fatalf("CreateSite(hidden): %v", err)
	}
	siteIDs = append(siteIDs, hiddenSite.ID)
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.religion_sites SET public_precision = 'hidden' WHERE id = $1`, hiddenSite.ID); err != nil {
		t.Fatalf("mark site hidden: %v", err)
	}

	exactLoc, err := loc.CreateLocation(ctx, locationdomain.LocationInput{Latitude: lat, Longitude: lng, CountryID: countryID})
	if err != nil {
		t.Fatalf("CreateLocation(exact): %v", err)
	}
	locationIDs = append(locationIDs, exactLoc.ID)
	exactSite, err := rel.CreateSite(ctx, siteInput(excludedParent.ID, exactLoc.ID, churchSiteTypeID))
	if err != nil {
		t.Fatalf("CreateSite(exact): %v", err)
	}
	siteIDs = append(siteIDs, exactSite.ID)

	// M13.0's enrichment assertions below need excludedParent to carry a primary classification —
	// unlike profile.UnitID (classified via CreateChildOrg above), excludedParent never got one.
	if _, err := rel.AddOrgClassification(ctx, excludedParent.ID, taxonID, true); err != nil {
		t.Fatalf("AddOrgClassification(excludedParent): %v", err)
	}

	radius := 1000.0
	hits, err := rel.SearchSites(ctx, domain.DiscoveryQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites: %v", err)
	}
	var sawHidden, sawExact bool
	for _, h := range hits {
		if h.ID == hiddenSite.ID {
			sawHidden = true
		}
		if h.ID == exactSite.ID {
			sawExact = true
		}
	}
	if sawHidden {
		t.Errorf("SearchSites returned the hidden site — position-oracle fix regressed")
	}
	if !sawExact {
		t.Errorf("SearchSites did not return the exact-precision site at the same point")
	}

	// --- M10.6: SearchSites' new Language/DayOfWeek filter — a site with a matching schedule row
	// must be returned, a site with no schedule at all must not.
	var mainServiceTypeID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_service_types WHERE code = 'main' AND tradition_taxon_id IS NULL`).Scan(&mainServiceTypeID); err != nil {
		t.Fatalf("lookup main service type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO openfaithmap.religion_service_schedules (site_id, service_type_id, day_of_week, start_time, timezone, language)
		VALUES ($1, $2, 0, '10:00', 'Europe/Kyiv', 'uk')`, exactSite.ID, mainServiceTypeID); err != nil {
		t.Fatalf("insert schedule: %v", err)
	}

	uk := "uk"
	sunday := 0
	filtered, err := rel.SearchSites(ctx, domain.DiscoveryQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Language: &uk, DayOfWeek: &sunday, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites(Language+DayOfWeek): %v", err)
	}
	var sawScheduled bool
	for _, h := range filtered {
		if h.ID == exactSite.ID {
			sawScheduled = true
		}
	}
	if !sawScheduled {
		t.Errorf("SearchSites(Language=uk, DayOfWeek=0) did not return the site with a matching schedule")
	}

	es := "es" // no site has a Spanish-language schedule
	noMatch, err := rel.SearchSites(ctx, domain.DiscoveryQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Language: &es, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites(Language=es): %v", err)
	}
	if len(noMatch) != 0 {
		t.Errorf("SearchSites(Language=es) = %+v, want no matches", noMatch)
	}

	// --- M13.0: SearchSites now returns name, address (precision-coarsened), primary tradition
	// tag, and site attributes — previously the projection carried none of these.
	enriched, err := rel.SearchSites(ctx, domain.DiscoveryQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites (enrichment check): %v", err)
	}
	var gotExact *domain.DiscoverySite
	for i := range enriched {
		if enriched[i].ID == exactSite.ID {
			gotExact = &enriched[i]
		}
	}
	if gotExact == nil {
		t.Fatalf("SearchSites (enrichment check) did not return exactSite")
	}
	if gotExact.Name != "M10.5 test excluded parent" {
		t.Errorf("DiscoverySite.Name = %q, want %q", gotExact.Name, "M10.5 test excluded parent")
	}
	if gotExact.TraditionTaxonCode == nil || *gotExact.TraditionTaxonCode != "christianity" {
		t.Errorf("DiscoverySite.TraditionTaxonCode = %v, want christianity (via profile's primary classification)", gotExact.TraditionTaxonCode)
	}
	var sawUK bool
	for _, l := range gotExact.ServiceLanguages {
		if l == "uk" {
			sawUK = true
		}
	}
	if !sawUK {
		t.Errorf("DiscoverySite.ServiceLanguages = %v, want it to include the schedule's language uk", gotExact.ServiceLanguages)
	}

	// --- M13.0: address is populated at exact precision (full text), omitted for a hidden site.
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.location_locations SET locality = 'Kyiv', admin_area_1 = 'Kyiv City', street = 'Khreshchatyk St', house_number = '1' WHERE id = $1`, exactLoc.ID); err != nil {
		t.Fatalf("set exactLoc address: %v", err)
	}
	withAddress, err := rel.SearchSites(ctx, domain.DiscoveryQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites (address check): %v", err)
	}
	var gotAddress *domain.DiscoverySite
	for i := range withAddress {
		if withAddress[i].ID == exactSite.ID {
			gotAddress = &withAddress[i]
		}
	}
	if gotAddress == nil || gotAddress.Address == nil || *gotAddress.Address != "Khreshchatyk St 1, Kyiv, Kyiv City" {
		t.Errorf("DiscoverySite.Address = %v, want \"Khreshchatyk St 1, Kyiv, Kyiv City\"", gotAddress)
	}

	// --- M13.0: religion_sites.attributes round-trips through SearchSites unfiltered by precision.
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.religion_sites SET attributes = '{"onlineStream": true, "accessibility": {"stepFreeEntrance": true}}' WHERE id = $1`, exactSite.ID); err != nil {
		t.Fatalf("set exactSite attributes: %v", err)
	}
	withAttrs, err := rel.SearchSites(ctx, domain.DiscoveryQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites (attributes check): %v", err)
	}
	var gotAttrs *domain.DiscoverySite
	for i := range withAttrs {
		if withAttrs[i].ID == exactSite.ID {
			gotAttrs = &withAttrs[i]
		}
	}
	if gotAttrs == nil || !gotAttrs.Attributes.OnlineStream || !gotAttrs.Attributes.Accessibility.StepFreeEntrance {
		t.Errorf("DiscoverySite.Attributes = %+v, want OnlineStream=true, Accessibility.StepFreeEntrance=true", gotAttrs)
	}

	// --- M13.0: DiscoveryQuery.UnitID scopes SearchSites to exactly one unit's public sites,
	// preferring the primary site, for the detail page's single-record lookup.
	byUnit, err := rel.SearchSites(ctx, domain.DiscoveryQuery{UnitID: &excludedParent.ID, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites(UnitID): %v", err)
	}
	if len(byUnit) != 1 || byUnit[0].ID != exactSite.ID {
		t.Errorf("SearchSites(UnitID=%s) = %+v, want exactly [exactSite]", excludedParent.ID, byUnit)
	}
	otherUnitID := profile.UnitID
	byOtherUnit, err := rel.SearchSites(ctx, domain.DiscoveryQuery{UnitID: &otherUnitID, Limit: 50})
	if err != nil {
		t.Fatalf("SearchSites(UnitID=other): %v", err)
	}
	for _, h := range byOtherUnit {
		if h.ID == exactSite.ID {
			t.Errorf("SearchSites(UnitID=%s) unexpectedly returned exactSite (belongs to a different unit)", otherUnitID)
		}
	}

	// --- ListTaxa (M10.7): a code/name search finds the seeded "christianity" taxon.
	taxa, err := rel.ListTaxa(ctx, "christianity", 10)
	if err != nil {
		t.Fatalf("ListTaxa: %v", err)
	}
	var sawChristianity bool
	for _, tx := range taxa {
		if tx.ID == taxonID {
			sawChristianity = true
		}
	}
	if !sawChristianity {
		t.Errorf("ListTaxa(%q) = %+v, want it to include taxon %s", "christianity", taxa, taxonID)
	}
}

func siteInput(unitID, locationID, siteTypeID string) adapters.CreateSiteInput {
	return adapters.CreateSiteInput{OrgUnitID: unitID, LocationID: locationID, SiteTypeID: siteTypeID, IsPrimary: true}
}
