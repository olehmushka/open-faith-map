// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M10.6's registration cutover — Submit, the operator gate (denied for a non-operator,
// allowed for one), the full Approve orchestration (child org + site + position + filled + grant),
// and the resumable Reparent state machine — against a real Postgres instance. Same invocation shape
// as internal/directory/directory_integration_test.go's own header comment:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/registration/... -run TestRegistrationIntegration -v
//
// The live HTTP surface is NOT exercised here — server.WithMiddleware(authenticator.Handle) is
// deliberately not attached yet (M10.6 is mid-cutover; attaching it now would 401 every currently-
// anonymous public route across all six modules, not just this one — see this session's own
// continuation notes). This test drives application.Service directly, with authz.NewContext
// standing in for what the identity middleware will inject once it's live.
package registration_test

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
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	membershipapplication "github.com/olehmushka/open-faith-map/internal/membership/application"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	regadapters "github.com/olehmushka/open-faith-map/internal/registration/adapters"
	"github.com/olehmushka/open-faith-map/internal/registration/application"
	regdomain "github.com/olehmushka/open-faith-map/internal/registration/domain"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

func TestRegistrationIntegration(t *testing.T) {
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
	membershipSvc := membershipapplication.NewService(pool)
	regStore := regadapters.NewRepository(pool)
	regSvc := application.NewService(regStore, religionSvc, locationSvc, membershipSvc, directorySvc, authzSvc, application.Config{
		RootUnitID:              seed.RootUnitID,
		CongregationAdminRoleID: seed.CongregationAdminRoleID,
	})

	var personIDs, unitIDs, requestIDs, assignmentIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range requestIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.jurisdiction_reparenting_jobs WHERE registration_request_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete reparent jobs for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.registration_requests WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete request %s: %v", id, err)
			}
		}
		for _, id := range assignmentIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignment %s: %v", id, err)
			}
		}
		// ensureGrant (Approve) creates its own congregation-admin assignment this test never
		// captured an id for — sweep by unit/person instead of relying on a tracked list.
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE target_unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignments targeting unit %s: %v", id, err)
			}
		}
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE subject_person_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignments for person %s: %v", id, err)
			}
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.membership_memberships WHERE unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete memberships for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.membership_positions WHERE unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete positions for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_sites WHERE org_unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete sites for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_org_classifications WHERE unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete org classifications for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_org_profiles WHERE unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete org profile for %s: %v", id, err)
			}
			// M12.2: directory_unit_move_jobs (Reparent's now-generic backing store) FKs into
			// directory_units ON DELETE RESTRICT, unlike the old jurisdiction_reparenting_jobs' opaque
			// text columns — must clear these before the unit delete below.
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_move_jobs WHERE unit_id = $1 OR old_parent_unit_id = $1 OR new_parent_unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete move jobs for %s: %v", id, err)
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
	submitterID := insertPerson("M10.6 Test Submitter")
	nonOperatorID := insertPerson("M10.6 Test Non-Operator")
	operatorID := insertPerson("M10.6 Test Operator")

	var taxonID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_taxa WHERE code = 'christianity' AND deleted_at IS NULL`).Scan(&taxonID); err != nil {
		t.Fatalf("lookup christianity taxon: %v", err)
	}
	var countryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.refdata_countries WHERE code = 'UA'`).Scan(&countryID); err != nil {
		t.Fatalf("lookup UA country: %v", err)
	}

	// --- Submit.
	req, err := regSvc.Submit(ctx, submitterID, regdomain.SubmitInput{
		TaxonID: taxonID, CongregationName: "M10.6 Test Congregation", CountryID: countryID,
		Coordinate: regdomain.Coordinate{Latitude: 50.45, Longitude: 30.52},
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	requestIDs = append(requestIDs, req.ID)
	if req.Status != regdomain.StatusPending {
		t.Errorf("Submit status = %s, want PENDING", req.Status)
	}

	// --- Approve is denied for a non-operator (the real authorization gap this cutover had to close
	// explicitly — see requireOperator's own doc comment).
	nonOpCtx := authz.NewContext(ctx, authz.Subject{PersonID: nonOperatorID})
	if _, err := regSvc.Approve(nonOpCtx, nonOperatorID, req.ID, nil, nil); !errors.Is(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("Approve by non-operator error = %v, want ErrPermissionDenied", err)
	}

	// --- Grant operatorID the registration-operator role on root, scope unit (matches how an
	// instance admin would grant it for real).
	var assignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		operatorID, seed.RegistrationOperatorRoleID, seed.RootUnitID,
	).Scan(&assignmentID); err != nil {
		t.Fatalf("grant registration-operator to test operator: %v", err)
	}
	assignmentIDs = append(assignmentIDs, assignmentID)

	// --- Approve, as the real operator.
	opCtx := authz.NewContext(ctx, authz.Subject{PersonID: operatorID})
	approved, err := regSvc.Approve(opCtx, operatorID, req.ID, nil, nil)
	if err != nil {
		t.Fatalf("Approve by operator: %v", err)
	}
	if approved.Status != regdomain.StatusApproved || approved.CreatedUnitID == nil {
		t.Fatalf("Approve result = %+v, want APPROVED with a CreatedUnitID", approved)
	}
	unitID := *approved.CreatedUnitID
	unitIDs = append(unitIDs, unitID)

	if _, err := directorySvc.GetUnit(ctx, unitID); err != nil {
		t.Errorf("GetUnit(approved congregation): %v", err)
	}
	ancestors, err := directorySvc.Ancestors(ctx, unitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("Ancestors(congregation): %v", err)
	}
	if len(ancestors) != 1 || ancestors[0].ID != seed.RootUnitID {
		t.Errorf("Ancestors(congregation) = %+v, want [root]", ancestors)
	}

	sites, err := religionSvc.ListSitesByUnit(ctx, unitID)
	if err != nil {
		t.Fatalf("ListSitesByUnit: %v", err)
	}
	if len(sites) != 1 || !sites[0].IsPrimary {
		t.Errorf("ListSitesByUnit = %+v, want exactly one primary site", sites)
	}

	positions, err := membershipSvc.ListPositionsByUnit(ctx, unitID)
	if err != nil {
		t.Fatalf("ListPositionsByUnit: %v", err)
	}
	if len(positions) != 1 || positions[0].Code != "admin" {
		t.Errorf("ListPositionsByUnit = %+v, want exactly one 'admin' position", positions)
	}

	submitterCtx := authz.NewContext(ctx, authz.Subject{PersonID: submitterID})
	if err := authzSvc.Require(submitterCtx, authzdomain.PermReligionOrgManage, unitID); err != nil {
		t.Errorf("submitter's congregation-admin grant does not reach their own unit: %v", err)
	}

	// --- A resumed Approve (already APPROVED) is rejected as not-pending, not re-run.
	if _, err := regSvc.Approve(opCtx, operatorID, req.ID, nil, nil); !errors.Is(err, regdomain.ErrNotPending) {
		t.Errorf("re-Approve of an APPROVED request error = %v, want ErrNotPending", err)
	}

	// --- Reparent onto a fresh jurisdiction unit under root, as the operator; denied for a
	// non-operator first.
	newParent, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M10.6 test jurisdiction"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(jurisdiction): %v", err)
	}
	unitIDs = append(unitIDs, newParent.ID)

	if _, err := regSvc.Reparent(nonOpCtx, nonOperatorID, req.ID, newParent.ID); !errors.Is(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("Reparent by non-operator error = %v, want ErrPermissionDenied", err)
	}
	job, err := regSvc.Reparent(opCtx, operatorID, req.ID, newParent.ID)
	if err != nil {
		t.Fatalf("Reparent: %v", err)
	}
	if job.Status != regdomain.ReparentVerified {
		t.Fatalf("Reparent job status = %s, want VERIFIED", job.Status)
	}
	newAncestors, err := directorySvc.Ancestors(ctx, unitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("Ancestors(congregation) after reparent: %v", err)
	}
	if len(newAncestors) != 2 {
		t.Fatalf("Ancestors(congregation) after reparent = %+v, want [jurisdiction, root]", newAncestors)
	}
	var sawNewParent, sawOldParentStillPresent bool
	for _, a := range newAncestors {
		if a.ID == newParent.ID {
			sawNewParent = true
		}
		if a.ID == seed.RootUnitID {
			sawOldParentStillPresent = true // root stays an ancestor via the new jurisdiction unit
		}
	}
	if !sawNewParent || !sawOldParentStillPresent {
		t.Errorf("Ancestors(congregation) after reparent = %+v, want to include both the new jurisdiction and root", newAncestors)
	}
}
