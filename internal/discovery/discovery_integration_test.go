// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M10.6's discovery cutover — the anonymous Search live-fallback resolving a real site
// in-process via internal/religion.SearchSites (no cache row to hit yet), and RefreshRegion's
// requireOperator gate denied for a non-operator and allowed for a real operator grant — against a
// real Postgres instance. Same invocation shape as
// internal/registration/registration_integration_test.go:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/discovery/... -run TestDiscoveryIntegration -v
//
// The live HTTP surface is NOT exercised here — see registration_integration_test.go's own header
// comment for why. This test drives application.Service directly, with authz.NewContext standing in
// for what the identity middleware will inject once it's live.
package discovery_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	directoryadapters "github.com/olehmushka/open-faith-map/internal/directory/adapters"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/discovery/adapters"
	"github.com/olehmushka/open-faith-map/internal/discovery/application"
	discoverydomain "github.com/olehmushka/open-faith-map/internal/discovery/domain"
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	locationdomain "github.com/olehmushka/open-faith-map/internal/location/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	religionadapters "github.com/olehmushka/open-faith-map/internal/religion/adapters"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

func TestDiscoveryIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	directorySvc := directoryapplication.NewService(pool)
	closurePort := directoryadapters.NewRepository(pool)
	pdp := authzdomain.NewPDP(closurePort)
	authzStore := authzadapters.NewRepository(pool)
	authzSvc := authz.NewService(pdp, authzStore, pool)
	religionSvc := religionapplication.NewService(pool, directorySvc, authzSvc)
	locationSvc := locationapplication.NewService(pool)
	discoveryStore := adapters.NewRepository(pool)
	discoverySvc := application.NewService(discoveryStore, nil, religionSvc, authzSvc, application.Config{
		RootUnitID: seed.RootUnitID,
	})

	var personIDs, unitIDs, locationIDs, siteIDs, assignmentIDs, cacheIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range cacheIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.discovery_site_cache WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete cache row %s: %v", id, err)
			}
		}
		for _, id := range assignmentIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignment %s: %v", id, err)
			}
		}
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
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
	})

	insertPerson := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
			VALUES ($1, $1, 'Test') RETURNING id`, name,
		).Scan(&id); err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		personIDs = append(personIDs, id)
		return id
	}
	operatorID := insertPerson("M10.6 Discovery Test Operator")
	nonOperatorID := insertPerson("M10.6 Discovery Test Non-Operator")

	unit, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M10.6 Discovery Test Congregation"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge: %v", err)
	}
	unitIDs = append(unitIDs, unit.ID)

	var countryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.refdata_countries WHERE code = 'UA'`).Scan(&countryID); err != nil {
		t.Fatalf("lookup UA country: %v", err)
	}
	var churchSiteTypeID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_site_types WHERE code = 'church' LIMIT 1`).Scan(&churchSiteTypeID); err != nil {
		t.Fatalf("lookup church site type: %v", err)
	}
	lat, lng := 50.4501, 30.5234 // Kyiv
	loc, err := locationSvc.CreateLocation(ctx, locationdomain.LocationInput{Latitude: lat, Longitude: lng, CountryID: countryID})
	if err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	locationIDs = append(locationIDs, loc.ID)
	site, err := religionSvc.CreateSite(ctx, religionadapters.CreateSiteInput{OrgUnitID: unit.ID, LocationID: loc.ID, SiteTypeID: churchSiteTypeID, IsPrimary: true})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	siteIDs = append(siteIDs, site.ID)

	// --- Search's live fallback resolves the real site in-process — no go-oikumenea round-trip. A
	// Query filter forces BypassesCache() (domain.SearchQuery's own rule), so this exercises
	// refreshFromLive/religion.SearchSites for real rather than risking a stale cache row from an
	// earlier session's own manual verification serving a plain lat/lng/radius query instead.
	radius := 1000.0
	query := "M10.6 Discovery Test"
	results, err := discoverySvc.Search(context.Background(), discoverydomain.SearchQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Query: &query})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	var saw bool
	for _, r := range results {
		if r.CongregationUnitRID == unit.ID {
			saw = true
			cacheIDs = append(cacheIDs, r.ID)
			// M13.0: the cache row now carries the congregation's real name straight from the
			// live SearchSites hit, not just an opaque RID/coordinate.
			if r.Name != "M10.6 Discovery Test Congregation" {
				t.Errorf("CacheRow.Name = %q, want %q", r.Name, "M10.6 Discovery Test Congregation")
			}
		}
	}
	if !saw {
		t.Fatalf("Search(lat/lng/radius) = %+v, want to include unit %s", results, unit.ID)
	}

	// --- M13.1: Accessibility/OnlineOnly force the live path (BypassesCache) and filter via JSONB
	// containment against the real site's attributes.
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.religion_sites SET attributes = '{"onlineStream": true, "accessibility": {"hearingLoop": true}}' WHERE id = $1`, site.ID); err != nil {
		t.Fatalf("set site attributes: %v", err)
	}
	accessibility := "hearingLoop"
	byAccessibility, err := discoverySvc.Search(context.Background(), discoverydomain.SearchQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Accessibility: &accessibility})
	if err != nil {
		t.Fatalf("Search(Accessibility=hearingLoop): %v", err)
	}
	var sawAccessibility bool
	for _, r := range byAccessibility {
		if r.CongregationUnitRID == unit.ID {
			sawAccessibility = true
		}
	}
	if !sawAccessibility {
		t.Errorf("Search(Accessibility=hearingLoop) did not return unit %s", unit.ID)
	}

	onlineOnly := true
	byOnlineOnly, err := discoverySvc.Search(context.Background(), discoverydomain.SearchQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, OnlineOnly: &onlineOnly})
	if err != nil {
		t.Fatalf("Search(OnlineOnly=true): %v", err)
	}
	var sawOnlineOnly bool
	for _, r := range byOnlineOnly {
		if r.CongregationUnitRID == unit.ID {
			sawOnlineOnly = true
		}
	}
	if !sawOnlineOnly {
		t.Errorf("Search(OnlineOnly=true) did not return unit %s", unit.ID)
	}

	// --- M13.1: an unrecognized accessibility= value is a real client error (ErrInvalidFilter),
	// not a silent zero-result match — the same "fail loudly" fix applied to the tradition bug.
	bogus := "not-a-real-criterion"
	if _, err := discoverySvc.Search(context.Background(), discoverydomain.SearchQuery{Lat: &lat, Lng: &lng, RadiusM: &radius, Accessibility: &bogus}); !errors.Is(err, discoverydomain.ErrInvalidFilter) {
		t.Errorf("Search(Accessibility=not-a-real-criterion) error = %v, want ErrInvalidFilter", err)
	}

	// --- M13.1: Facets is always live and answers without error (religion_integration_test.go
	// covers the actual distinct-value/hidden-site-exclusion logic against religion.SearchFacets
	// directly; this only proves the discovery-module delegation wiring).
	if _, err := discoverySvc.Facets(ctx); err != nil {
		t.Fatalf("Facets: %v", err)
	}

	// --- M13.0: GetSiteByUnit answers the detail page's server-rendered fetch, always live, for
	// exactly the unit asked about.
	got, found, err := discoverySvc.GetSiteByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("GetSiteByUnit: %v", err)
	}
	if !found {
		t.Fatalf("GetSiteByUnit(%s) found = false, want true", unit.ID)
	}
	if got.Name != "M10.6 Discovery Test Congregation" {
		t.Errorf("GetSiteByUnit.Name = %q, want %q", got.Name, "M10.6 Discovery Test Congregation")
	}
	if got.ReligionSiteRID != site.ID {
		t.Errorf("GetSiteByUnit.ReligionSiteRID = %q, want %q", got.ReligionSiteRID, site.ID)
	}

	if _, found, err := discoverySvc.GetSiteByUnit(ctx, seed.RootUnitID); err != nil {
		t.Errorf("GetSiteByUnit(rootUnit, no site): %v", err)
	} else if found {
		t.Errorf("GetSiteByUnit(rootUnit, no site) found = true, want false")
	}

	// --- M13.0: a malformed unitId (org_unit_id is a real `uuid` column) must degrade to
	// found=false, not a raw Postgres type-coercion error surfacing as a 500.
	if _, found, err := discoverySvc.GetSiteByUnit(ctx, "not-a-uuid"); err != nil {
		t.Errorf("GetSiteByUnit(malformed id): %v, want a clean found=false, no error", err)
	} else if found {
		t.Errorf("GetSiteByUnit(malformed id) found = true, want false")
	}

	// --- RefreshRegion (requireOperator) is denied for a non-operator, allowed for a real operator.
	nonOpCtx := authz.NewContext(ctx, authz.Subject{PersonID: nonOperatorID})
	region := discoverydomain.RefreshRegion{MinLat: lat - 0.01, MinLng: lng - 0.01, MaxLat: lat + 0.01, MaxLng: lng + 0.01}
	if _, err := discoverySvc.RefreshRegion(nonOpCtx, region); !errors.Is(err, discoverydomain.ErrForbidden) {
		t.Errorf("RefreshRegion by non-operator error = %v, want ErrForbidden", err)
	}

	var assignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		operatorID, seed.RegistrationOperatorRoleID, seed.RootUnitID,
	).Scan(&assignmentID); err != nil {
		t.Fatalf("grant registration-operator to test operator: %v", err)
	}
	assignmentIDs = append(assignmentIDs, assignmentID)

	opCtx := authz.NewContext(ctx, authz.Subject{PersonID: operatorID})
	count, err := discoverySvc.RefreshRegion(opCtx, region)
	if err != nil {
		t.Fatalf("RefreshRegion by real operator: %v", err)
	}
	if count < 1 {
		t.Errorf("RefreshRegion by real operator refreshed %d sites, want at least 1", count)
	}
}
