// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M10.6's vouching cutover — requireCongregationStanding denied for a caller with no grant on
// their claimed guarantor unit and allowed for a real congregation-admin grant, and requireModerate
// denied/allowed the same way for ListVouches/RevokeGuarantor — against a real Postgres instance.
// Same invocation shape as internal/registration/registration_integration_test.go:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/vouching/... -run TestVouchingIntegration -v
//
// The live HTTP surface is NOT exercised here — see registration_integration_test.go's own header
// comment for why. This test drives application.Service directly, with authz.NewContext standing in
// for what the identity middleware will inject once it's live.
package vouching_test

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
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	vouchingadapters "github.com/olehmushka/open-faith-map/internal/vouching/adapters"
	"github.com/olehmushka/open-faith-map/internal/vouching/application"
	vouchingdomain "github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

// noopModerationReporter satisfies application.ModerationReporter without touching
// internal/moderation — the real moderationVouchReporter wiring is composition-root-only
// (cmd/openfaithmap-api/deps.go), proven when vouching/moderation are both live in the full HTTP
// verification pass at the end of M10.6, not here.
type noopModerationReporter struct{ calls int }

func (r *noopModerationReporter) ReportGuarantorRevoked(ctx context.Context, event application.GuarantorRevokedEvent) error {
	r.calls++
	return nil
}

func TestVouchingIntegration(t *testing.T) {
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
	authzStore := authzadapters.NewStore(pool)
	authzSvc := authz.NewService(pdp, authzStore)
	vouchingStore := vouchingadapters.NewRepository(pool)
	reporter := &noopModerationReporter{}
	vouchSvc := application.NewService(vouchingStore, reporter, authzSvc, application.Config{
		RootUnitID: seed.RootUnitID,
	})

	// vouching_edges is genuinely append-only (migrations/0008_vouching.sql's own reject_mutation
	// trigger) but carries no FK to persons/units — leaving created vouch rows behind is correct,
	// not a leak; only the mutable guarantor-status overlay and directory/authz/identity rows below
	// need real cleanup.
	var personIDs, unitIDs, assignmentIDs, guarantorIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range guarantorIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.vouching_guarantor_status WHERE guarantor_person_rid = $1`, id); err != nil {
				t.Errorf("cleanup: delete guarantor status for %s: %v", id, err)
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
	guarantorID := insertPerson("M10.6 Vouching Test Guarantor")
	claimantID := insertPerson("M10.6 Vouching Test Claimant")
	moderatorID := insertPerson("M10.6 Vouching Test Moderator")
	nonModeratorID := insertPerson("M10.6 Vouching Test Non-Moderator")

	guarantorUnit, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M10.6 Vouching Test Guarantor Congregation"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(guarantor): %v", err)
	}
	unitIDs = append(unitIDs, guarantorUnit.ID)
	claimUnit, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M10.6 Vouching Test Claim Congregation"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(claim): %v", err)
	}
	unitIDs = append(unitIDs, claimUnit.ID)
	guarantorIDs = append(guarantorIDs, guarantorID)

	// --- CreateVouch (requireCongregationStanding) is denied for a caller with no grant on their
	// claimed guarantor unit.
	guarantorCtx := authz.NewContext(ctx, authz.Subject{PersonID: guarantorID})
	in := vouchingdomain.CreateVouchInput{
		ClaimantPersonRID: claimantID, CongregationUnitID: claimUnit.ID, GuarantorCongregationUnitID: guarantorUnit.ID,
	}
	if _, err := vouchSvc.CreateVouch(guarantorCtx, guarantorID, in); !errors.Is(err, vouchingdomain.ErrForbidden) {
		t.Errorf("CreateVouch with no standing error = %v, want ErrForbidden", err)
	}

	// --- Grant guarantorID congregation-admin on their own unit — real standing.
	var standingAssignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		guarantorID, seed.CongregationAdminRoleID, guarantorUnit.ID,
	).Scan(&standingAssignmentID); err != nil {
		t.Fatalf("grant congregation-admin to guarantor: %v", err)
	}
	assignmentIDs = append(assignmentIDs, standingAssignmentID)

	vouch, err := vouchSvc.CreateVouch(guarantorCtx, guarantorID, in)
	if err != nil {
		t.Fatalf("CreateVouch with real standing: %v", err)
	}
	if vouch.GuarantorPersonRID != guarantorID {
		t.Errorf("CreateVouch result = %+v, want GuarantorPersonRID %s", vouch, guarantorID)
	}

	// --- ListVouches/RevokeGuarantor (requireModerate) are denied for a non-moderator, allowed for
	// a real platform-moderator grant.
	nonModCtx := authz.NewContext(ctx, authz.Subject{PersonID: nonModeratorID})
	if _, err := vouchSvc.ListVouches(nonModCtx, nil, nil, 10); !errors.Is(err, vouchingdomain.ErrForbidden) {
		t.Errorf("ListVouches by non-moderator error = %v, want ErrForbidden", err)
	}
	if _, err := vouchSvc.RevokeGuarantor(nonModCtx, nonModeratorID, guarantorID, "test"); !errors.Is(err, vouchingdomain.ErrForbidden) {
		t.Errorf("RevokeGuarantor by non-moderator error = %v, want ErrForbidden", err)
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
	if _, err := vouchSvc.ListVouches(modCtx, nil, nil, 10); err != nil {
		t.Errorf("ListVouches by real moderator: %v", err)
	}
	status, err := vouchSvc.RevokeGuarantor(modCtx, moderatorID, guarantorID, "test revoke")
	if err != nil {
		t.Fatalf("RevokeGuarantor by real moderator: %v", err)
	}
	if status.Status != vouchingdomain.StatusRevoked {
		t.Errorf("RevokeGuarantor result status = %s, want revoked", status.Status)
	}
	if reporter.calls != 1 {
		t.Errorf("moderation fan-out called %d times, want exactly 1 (one prior vouch)", reporter.calls)
	}

	// --- CanVouch's own rule: a revoked guarantor may not vouch again, even with real standing.
	if _, err := vouchSvc.CreateVouch(guarantorCtx, guarantorID, in); !errors.Is(err, vouchingdomain.ErrGuarantorRevoked) {
		t.Errorf("CreateVouch by revoked guarantor error = %v, want ErrGuarantorRevoked", err)
	}
}
