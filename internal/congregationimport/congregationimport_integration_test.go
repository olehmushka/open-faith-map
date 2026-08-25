// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M10.6's congregationimport cutover against a real Postgres instance:
//   - requireOperator denied for a non-operator and allowed for a real operator (EditCandidate,
//     RejectCandidate).
//   - RunConnector's D-Exclusions check and matchCountry both work in-process under
//     authz.SystemContext — a russian_orthodox_church-aliased record ends REJECTED_EXCLUDED, a real
//     Christian record gets its country hint resolved.
//   - ApproveCandidate performs the real writes (CreateChildOrg/CreateLocation/CreateSite) under the
//     operator's own subject.
//   - RunJurisdictionSync's new requireOperator gate (the confirmed live gap this cutover fixes):
//     denied for a non-operator before the source lookup even runs, reaches ErrJurisdictionSourceNotFound
//     for a real operator instead.
//
// Invocation:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/congregationimport/... -run TestCongregationImportIntegration -v
//
// The live HTTP surface is NOT exercised here — see
// internal/registration/registration_integration_test.go's own header comment for why. This test
// drives application.Service directly, with authz.NewContext standing in for what the identity
// middleware will inject once it's live.
package congregationimport_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	congregationimportadapters "github.com/olehmushka/open-faith-map/internal/congregationimport/adapters"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/application"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	directoryadapters "github.com/olehmushka/open-faith-map/internal/directory/adapters"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	refdataapplication "github.com/olehmushka/open-faith-map/internal/refdata/application"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

// fakeConnector yields a fixed, in-memory batch of records — no network, no file. Mirrors the shape
// every real connector (uaedr/arrnc/osm) implements, narrowed to exactly what RunConnector's loop
// needs to prove the D-Exclusions/matchCountry path works in-process post-cutover.
type fakeConnector struct {
	records []domain.RawRecord
	byID    map[string]domain.NormalizedCandidate
	served  bool
}

func (c *fakeConnector) Code() string                    { return "m106-fake" }
func (c *fakeConnector) Citation() domain.SourceCitation { return domain.SourceCitation{} }
func (c *fakeConnector) Clone() domain.Connector {
	return &fakeConnector{records: c.records, byID: c.byID}
}
func (c *fakeConnector) Normalize(raw domain.RawRecord) (domain.NormalizedCandidate, error) {
	return c.byID[raw.SourceRecordID], nil
}
func (c *fakeConnector) Fetch(ctx context.Context, cursor *string) ([]domain.RawRecord, *string, error) {
	if c.served {
		return nil, nil, nil
	}
	c.served = true
	return c.records, nil, nil
}

func TestCongregationImportIntegration(t *testing.T) {
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
	closurePort := directoryadapters.NewStore(pool)
	pdp := authzdomain.NewPDP(closurePort)
	authzStore := authzadapters.NewRepository(pool)
	authzSvc := authz.NewService(pdp, authzStore, pool)
	religionSvc := religionapplication.NewService(pool, directorySvc)
	locationSvc := locationapplication.NewService(pool)
	refdataSvc := refdataapplication.NewService(pool)
	store := congregationimportadapters.NewStore(pool)

	christianTaxonHint := "M10.6 Fake Christian Church"
	rocTaxonHint := "M10.6 Fake Russian Orthodox Church Hall"
	connector := &fakeConnector{
		records: []domain.RawRecord{
			{SourceRecordID: "christian-1", RawPayload: json.RawMessage(`{}`), FetchedAt: time.Now()},
			{SourceRecordID: "roc-1", RawPayload: json.RawMessage(`{}`), FetchedAt: time.Now()},
		},
		byID: map[string]domain.NormalizedCandidate{
			"christian-1": {
				Name: christianTaxonHint, TaxonHint: &christianTaxonHint, CountryHint: strPtr("Ukraine"),
				Latitude: float64Ptr(50.45), Longitude: float64Ptr(30.52),
			},
			"roc-1": {
				Name: rocTaxonHint, TaxonHint: &rocTaxonHint,
			},
		},
	}
	congImportSvc := application.NewService(store, religionSvc, locationSvc, refdataSvc, authzSvc, application.Config{
		RootUnitID: seed.RootUnitID,
	}, []domain.Connector{connector}, nil, nil)

	var personIDs, unitIDs, locationIDs, siteIDs, assignmentIDs, candidateIDs, aliasIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range candidateIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.congregationimport_candidates WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete candidate %s: %v", id, err)
			}
		}
		for _, id := range aliasIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.congregationimport_taxon_aliases WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete taxon alias %s: %v", id, err)
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
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.congregationimport_congregation_status WHERE congregation_unit_rid = $1`, id); err != nil {
				t.Errorf("cleanup: delete congregation_status for %s: %v", id, err)
			}
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
	operatorID := insertPerson("M10.6 CongregationImport Test Operator")
	nonOperatorID := insertPerson("M10.6 CongregationImport Test Non-Operator")

	opCtx := authz.NewContext(ctx, authz.Subject{PersonID: operatorID})
	nonOpCtx := authz.NewContext(ctx, authz.Subject{PersonID: nonOperatorID})

	// --- requireOperator: EditCandidate/RejectCandidate denied for a non-operator against a garbage
	// id — proving the gate itself runs before any store lookup.
	if _, err := congImportSvc.EditCandidate(nonOpCtx, "00000000-0000-0000-0000-000000000000", domain.EditInput{}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("EditCandidate by non-operator error = %v, want ErrForbidden", err)
	}
	if _, err := congImportSvc.RejectCandidate(nonOpCtx, nonOperatorID, "00000000-0000-0000-0000-000000000000", "test"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("RejectCandidate by non-operator error = %v, want ErrForbidden", err)
	}

	// --- Grant operatorID the registration-operator role on root (the same role/permission
	// congregationimport's requireOperator reuses from registration's own IsOperator).
	var assignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		operatorID, seed.RegistrationOperatorRoleID, seed.RootUnitID,
	).Scan(&assignmentID); err != nil {
		t.Fatalf("grant registration-operator to test operator: %v", err)
	}
	assignmentIDs = append(assignmentIDs, assignmentID)

	if _, err := congImportSvc.EditCandidate(opCtx, "00000000-0000-0000-0000-000000000000", domain.EditInput{}); errors.Is(err, domain.ErrForbidden) {
		t.Errorf("EditCandidate by real operator error = %v, want past the operator gate (not ErrForbidden)", err)
	}

	// --- RunJurisdictionSync's new requireOperator gate — the confirmed live gap this cutover
	// fixes: denied for a non-operator before the source lookup ever runs; a real operator instead
	// reaches ErrJurisdictionSourceNotFound (no jurisdiction source registered in this test), proving
	// the gate — not the (irrelevant here) source lookup — is what distinguishes the two calls.
	if _, err := congImportSvc.RunJurisdictionSync(nonOpCtx, "does-not-exist"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("RunJurisdictionSync by non-operator error = %v, want ErrForbidden", err)
	}
	if _, err := congImportSvc.RunJurisdictionSync(opCtx, "does-not-exist"); !errors.Is(err, domain.ErrJurisdictionSourceNotFound) {
		t.Errorf("RunJurisdictionSync by real operator error = %v, want ErrJurisdictionSourceNotFound", err)
	}

	// --- Taxon aliases so matchTaxon resolves both fake records (real writes, as the operator).
	var christianityTaxonID, rocTaxonID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_taxa WHERE code = 'christianity' AND deleted_at IS NULL`).Scan(&christianityTaxonID); err != nil {
		t.Fatalf("lookup christianity taxon: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_taxa WHERE code = 'russian_orthodox_church' AND deleted_at IS NULL`).Scan(&rocTaxonID); err != nil {
		t.Fatalf("lookup russian_orthodox_church taxon: %v", err)
	}
	christianAlias, err := congImportSvc.CreateTaxonAlias(opCtx, operatorID, nil, christianTaxonHint, christianityTaxonID)
	if err != nil {
		t.Fatalf("CreateTaxonAlias(christian): %v", err)
	}
	aliasIDs = append(aliasIDs, christianAlias.ID)
	rocAlias, err := congImportSvc.CreateTaxonAlias(opCtx, operatorID, nil, rocTaxonHint, rocTaxonID)
	if err != nil {
		t.Fatalf("CreateTaxonAlias(roc): %v", err)
	}
	aliasIDs = append(aliasIDs, rocAlias.ID)

	// --- RunConnector: proves the D-Exclusions check and matchCountry both work in-process under
	// authz.SystemContext — a background pipeline call with no ctx subject at all.
	run, err := congImportSvc.RunConnector(context.Background(), "m106-fake", operatorID, nil)
	if err != nil {
		t.Fatalf("RunConnector: %v", err)
	}
	if run.Status != domain.RunStatusSucceeded {
		t.Fatalf("RunConnector status = %s, want SUCCEEDED", run.Status)
	}

	candidates, err := congImportSvc.ListCandidates(ctx, nil, strPtr("m106-fake"), 10, nil)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	var christianCand, rocCand *domain.Candidate
	for i := range candidates {
		candidateIDs = append(candidateIDs, candidates[i].ID)
		switch candidates[i].SourceRecordID {
		case "christian-1":
			christianCand = &candidates[i]
		case "roc-1":
			rocCand = &candidates[i]
		}
	}
	if rocCand == nil || rocCand.Status != domain.StatusRejectedExcluded {
		t.Fatalf("roc candidate = %+v, want status REJECTED_EXCLUDED (D-Exclusions check must run in-process)", rocCand)
	}
	if christianCand == nil {
		t.Fatalf("christian candidate not found in ListCandidates result")
	}
	if christianCand.CountryID == nil {
		t.Errorf("christian candidate CountryID = nil, want matchCountry to have resolved Ukraine via internal/refdata")
	} else {
		var countryCode string
		if err := pool.QueryRow(ctx, `SELECT code FROM openfaithmap.refdata_countries WHERE id = $1`, *christianCand.CountryID).Scan(&countryCode); err != nil {
			t.Fatalf("lookup matched country: %v", err)
		}
		if countryCode != "UA" {
			t.Errorf("christian candidate matched country code = %s, want UA", countryCode)
		}
	}
	if christianCand.TaxonID == nil || *christianCand.TaxonID != christianityTaxonID {
		t.Errorf("christian candidate TaxonID = %v, want %s (matchTaxon via the alias just created)", christianCand.TaxonID, christianityTaxonID)
	}

	// --- ApproveCandidate performs the real writes (CreateChildOrg/CreateLocation/CreateSite) under
	// the operator's own subject.
	if !isApprovableForTest(christianCand.Status) {
		t.Fatalf("christian candidate status = %s, not approvable — RunConnector's own processing must have left it reviewable", christianCand.Status)
	}
	approved, err := congImportSvc.ApproveCandidate(opCtx, operatorID, christianCand.ID, nil)
	if err != nil {
		t.Fatalf("ApproveCandidate: %v", err)
	}
	if approved.Status != domain.StatusProvisioned || approved.CreatedUnitID == nil {
		t.Fatalf("ApproveCandidate result = %+v, want PROVISIONED with a CreatedUnitID", approved)
	}
	unitID := *approved.CreatedUnitID
	unitIDs = append(unitIDs, unitID)

	if _, err := directorySvc.GetUnit(ctx, unitID); err != nil {
		t.Errorf("GetUnit(approved congregation): %v", err)
	}
	sites, err := religionSvc.ListSitesByUnit(ctx, unitID)
	if err != nil {
		t.Fatalf("ListSitesByUnit: %v", err)
	}
	if len(sites) != 1 || !sites[0].IsPrimary {
		t.Errorf("ListSitesByUnit = %+v, want exactly one primary site", sites)
	}
	locationIDs = append(locationIDs, sites[0].LocationID)
	siteIDs = append(siteIDs, sites[0].ID)
}

func isApprovableForTest(status domain.Status) bool {
	switch status {
	case domain.StatusStaged, domain.StatusNeedsTaxonReview, domain.StatusNeedsGeocode,
		domain.StatusPossibleDuplicate, domain.StatusApproved, domain.StatusProvisioning:
		return true
	default:
		return false
	}
}

func strPtr(s string) *string       { return &s }
func float64Ptr(f float64) *float64 { return &f }
