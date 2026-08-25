// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M10.6's moderation cutover — requireModerate denied for a non-moderator and allowed for a
// real platform-moderator grant on root, requireCongregationAdmin's own target-scoped gate on
// FileAppeal, and CheckExclusion's SystemContext-wrapped religion read — against a real Postgres
// instance. Same invocation shape as internal/registration/registration_integration_test.go:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/moderation/... -run TestModerationIntegration -v
//
// The live HTTP surface is NOT exercised here — see registration_integration_test.go's own header
// comment for why. This test drives application.Service directly, with authz.NewContext standing in
// for what the identity middleware will inject once it's live.
package moderation_test

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
	moderationadapters "github.com/olehmushka/open-faith-map/internal/moderation/adapters"
	"github.com/olehmushka/open-faith-map/internal/moderation/application"
	moderationdomain "github.com/olehmushka/open-faith-map/internal/moderation/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

func TestModerationIntegration(t *testing.T) {
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
	modStore := moderationadapters.NewRepository(pool)
	modSvc := application.NewService(modStore, religionSvc, authzSvc, application.Config{
		RootUnitID: seed.RootUnitID,
	})

	// The filed report and taken action below are logged (t.Logf) for visibility but deliberately
	// never cleaned up: moderation_actions is genuinely append-only (migrations/0007_moderation.sql's
	// own reject_mutation trigger blocks UPDATE/DELETE unconditionally), and deleting a report it
	// references would cascade an UPDATE (ON DELETE SET NULL on actions.report_id) that trigger
	// blocks too — confirmed live, not assumed. Leaving these test rows behind is the correct
	// behaviour for an audit-log-shaped table, not a leak this cleanup should fight.
	var personIDs, unitIDs, assignmentIDs, appealIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range appealIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.moderation_appeals WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete appeal %s: %v", id, err)
			}
		}
		for _, id := range assignmentIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignment %s: %v", id, err)
			}
		}
		for _, id := range unitIDs {
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
	moderatorID := insertPerson("M10.6 Moderation Test Moderator")
	nonModeratorID := insertPerson("M10.6 Moderation Test Non-Moderator")
	congAdminID := insertPerson("M10.6 Moderation Test Congregation Admin")

	unit, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M10.6 Moderation Test Congregation"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge: %v", err)
	}
	unitIDs = append(unitIDs, unit.ID)

	// --- ListReports (requireModerate) is denied for a non-moderator, allowed for a real
	// platform-moderator grant on root.
	nonModCtx := authz.NewContext(ctx, authz.Subject{PersonID: nonModeratorID})
	if _, err := modSvc.ListReports(nonModCtx, nil, nil, 10, nil); !errors.Is(err, moderationdomain.ErrForbidden) {
		t.Errorf("ListReports by non-moderator error = %v, want ErrForbidden", err)
	}

	var modAssignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		moderatorID, seed.PlatformModeratorRoleID, seed.RootUnitID,
	).Scan(&modAssignmentID); err != nil {
		t.Fatalf("grant platform-moderator: %v", err)
	}
	assignmentIDs = append(assignmentIDs, modAssignmentID)

	modCtx := authz.NewContext(ctx, authz.Subject{PersonID: moderatorID})
	if _, err := modSvc.ListReports(modCtx, nil, nil, 10, nil); err != nil {
		t.Errorf("ListReports by real moderator: %v", err)
	}

	// --- FileReport is genuinely anonymous — no ctx subject at all.
	report, err := modSvc.FileReport(context.Background(), moderationdomain.FileReportInput{
		TargetKind: moderationdomain.TargetCongregation,
		TargetRef:  unit.ID,
		ReasonCode: moderationdomain.ReasonOther,
	})
	if err != nil {
		t.Fatalf("FileReport: %v", err)
	}
	t.Logf("filed report %s — deliberately not cleaned up (append-only table)", report.ID)

	// --- TakeActionOnReport is denied for a non-moderator, allowed for the real moderator.
	if _, err := modSvc.TakeActionOnReport(nonModCtx, nonModeratorID, report.ID, moderationdomain.ActionSuspend, "test"); !errors.Is(err, moderationdomain.ErrForbidden) {
		t.Errorf("TakeActionOnReport by non-moderator error = %v, want ErrForbidden", err)
	}
	action, err := modSvc.TakeActionOnReport(modCtx, moderatorID, report.ID, moderationdomain.ActionSuspend, "test")
	if err != nil {
		t.Fatalf("TakeActionOnReport by real moderator: %v", err)
	}
	t.Logf("took action %s — deliberately not cleaned up (append-only table)", action.ID)

	// --- FileAppeal gates on requireCongregationAdmin against the action's own target unit, not
	// requireModerate — denied for a caller with no grant on unit, allowed for a real
	// congregation-admin grant on it.
	nonAdminCtx := authz.NewContext(ctx, authz.Subject{PersonID: nonModeratorID})
	if _, err := modSvc.FileAppeal(nonAdminCtx, nonModeratorID, action.ID, "I disagree"); !errors.Is(err, moderationdomain.ErrForbidden) {
		t.Errorf("FileAppeal by non-admin error = %v, want ErrForbidden", err)
	}

	var congAssignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		congAdminID, seed.CongregationAdminRoleID, unit.ID,
	).Scan(&congAssignmentID); err != nil {
		t.Fatalf("grant congregation-admin: %v", err)
	}
	assignmentIDs = append(assignmentIDs, congAssignmentID)

	congAdminCtx := authz.NewContext(ctx, authz.Subject{PersonID: congAdminID})
	appeal, err := modSvc.FileAppeal(congAdminCtx, congAdminID, action.ID, "I disagree")
	if err != nil {
		t.Fatalf("FileAppeal by real congregation-admin: %v", err)
	}
	appealIDs = append(appealIDs, appeal.ID)

	// --- CheckExclusion is genuinely anonymous and runs its internal/religion read under
	// authz.SystemContext — proves the in-process GetTaxon call works with no ctx subject at all.
	var taxonID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_taxa WHERE code = 'russian_orthodox_church' AND deleted_at IS NULL`).Scan(&taxonID); err != nil {
		t.Fatalf("lookup russian_orthodox_church taxon: %v", err)
	}
	excluded, code, err := modSvc.CheckExclusion(context.Background(), taxonID)
	if err != nil {
		t.Fatalf("CheckExclusion: %v", err)
	}
	if !excluded || code != "russian_orthodox_church" {
		t.Errorf("CheckExclusion(russian_orthodox_church) = (%v, %s), want (true, russian_orthodox_church)", excluded, code)
	}
}
