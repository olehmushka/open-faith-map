// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M11.2's own stated goal — "the log has no blind spot from day one" — against a real
// Postgres instance: every one of core.application.Service's six mutating super-admin methods
// (GrantUnitRole, RevokeRoleAssignment, GrantInstanceAdmin, RevokeInstanceAdmin, DeactivateAccount,
// ReactivateAccount) writes exactly one identity_audit_log row with the expected actor/action/
// target/before/after shape. See internal/authz/authz_integration_test.go's own header comment for
// the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestSuperAdminAuditTrailIntegration -v
package core_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	auditlogadapters "github.com/olehmushka/open-faith-map/internal/auditlog/adapters"
	auditlogapplication "github.com/olehmushka/open-faith-map/internal/auditlog/application"
	"github.com/olehmushka/open-faith-map/internal/authz"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	coreapplication "github.com/olehmushka/open-faith-map/internal/core/application"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	identityadapters "github.com/olehmushka/open-faith-map/internal/identity/adapters"
	identityapplication "github.com/olehmushka/open-faith-map/internal/identity/application"
)

// noopClosure satisfies authzdomain.ClosurePort without touching the DB — none of the six mutating
// methods under test ever calls PDP.Decide (same reasoning authz_integration_test.go's own
// noopClosure gives).
type noopClosure struct{}

func (noopClosure) IsAncestorOrSelf(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (noopClosure) IsAuthorityBearing(context.Context, string) (bool, error) { return false, nil }

func TestSuperAdminAuditTrailIntegration(t *testing.T) {
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

	dir := directoryapplication.NewService(pool)
	identitySvc := identityapplication.NewService(identityadapters.NewStore(pool))
	authzSvc := authz.NewService(authzdomain.NewPDP(noopClosure{}), authzadapters.NewStore(pool))
	auditLogSvc := auditlogapplication.NewService(auditlogadapters.NewStore(pool))
	// directory/religion/membership/refdata are nil: none of the six methods under test touch them
	// (they're core.application.Service's read-only surfaces, wired by other tests).
	coreApp := coreapplication.NewService(nil, nil, nil, identitySvc, nil, authzSvc, auditLogSvc)

	var actorID, targetID, targetAccountID string
	var unit directorydomain.Unit
	var assignmentID, instanceAdminGrantID string
	t.Cleanup(func() {
		bg := context.Background()
		// identity_audit_log is append-only (reject_mutation trigger) — disable it just for this
		// test's own cleanup, same discipline the migration's own manual verification used.
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, actorID); err != nil {
			t.Errorf("cleanup: delete audit rows: %v", err)
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
		if assignmentID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, assignmentID); err != nil {
				t.Errorf("cleanup: delete assignment: %v", err)
			}
		}
		if instanceAdminGrantID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_instance_admins WHERE id = $1`, instanceAdminGrantID); err != nil {
				t.Errorf("cleanup: delete instance-admin grant: %v", err)
			}
		}
		if targetAccountID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_accounts WHERE id = $1`, targetAccountID); err != nil {
				t.Errorf("cleanup: delete target account: %v", err)
			}
		}
		for _, id := range []string{actorID, targetID} {
			if id == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
		if unit.ID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, unit.ID); err != nil {
				t.Errorf("cleanup: delete unit: %v", err)
			}
		}
	})

	unit, err = dir.CreateUnit(ctx, directorydomain.Unit{Name: "M11.2 audit trail test unit"})
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.2 Audit Actor', 'M11.2', 'Actor') RETURNING id`).Scan(&actorID); err != nil {
		t.Fatalf("insert actor person: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.2 Audit Target', 'M11.2', 'Target') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("insert target person: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_accounts (person_id, email)
		VALUES ($1, 'm11-2-target@example.test') RETURNING id`, targetID).Scan(&targetAccountID); err != nil {
		t.Fatalf("insert target account: %v", err)
	}

	roles, err := authzSvc.ListRoles(ctx)
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	var roleID string
	for _, r := range roles {
		if r.Code == "registration-operator" {
			roleID = r.ID
		}
	}
	if roleID == "" {
		t.Fatalf("ListRoles = %+v, want it to include the seeded registration-operator role", roles)
	}

	actorCtx := authz.NewContext(ctx, authz.Subject{PersonID: actorID})

	// --- requireSubject gate: no subject in context must fail loud, with no audit row and no
	// side effect (M11.2's fail-loud-on-missing-subject contract).
	if _, err := coreApp.GetAccountStatus(ctx, targetID); err != nil {
		t.Fatalf("GetAccountStatus setup read: %v", err)
	}
	if _, err := coreApp.DeactivateAccount(ctx, targetID); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("DeactivateAccount with no subject = %v, want ErrPermissionDenied", err)
	}
	assertNoAuditRow(ctx, t, pool, "DEACTIVATE_ACCOUNT", targetID)

	// --- GrantUnitRole.
	if err := coreApp.GrantUnitRole(actorCtx, targetID, roleID, unit.ID); err != nil {
		t.Fatalf("GrantUnitRole: %v", err)
	}
	assignments, err := authzSvc.ListRoleAssignmentsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsByUnit: %v", err)
	}
	if len(assignments) != 1 {
		t.Fatalf("ListRoleAssignmentsByUnit = %+v, want exactly one", assignments)
	}
	assignmentID = assignments[0].ID
	row := mustAuditRow(ctx, t, pool, "GRANT_UNIT_ROLE", assignmentID)
	if row.actorPersonID != actorID || row.targetKind != "ROLE_ASSIGNMENT" || row.before != nil {
		t.Errorf("GRANT_UNIT_ROLE audit row = %+v, want actor=%s target_kind=ROLE_ASSIGNMENT before=nil", row, actorID)
	}
	assertJSONField(t, row.after, "personId", targetID)
	assertJSONField(t, row.after, "roleId", roleID)
	assertJSONField(t, row.after, "unitId", unit.ID)

	// --- RevokeRoleAssignment.
	if err := coreApp.RevokeRoleAssignment(actorCtx, assignmentID); err != nil {
		t.Fatalf("RevokeRoleAssignment: %v", err)
	}
	row = mustAuditRow(ctx, t, pool, "REVOKE_ROLE_ASSIGNMENT", assignmentID)
	if row.actorPersonID != actorID || row.targetKind != "ROLE_ASSIGNMENT" || row.after != nil {
		t.Errorf("REVOKE_ROLE_ASSIGNMENT audit row = %+v, want actor=%s target_kind=ROLE_ASSIGNMENT after=nil", row, actorID)
	}
	assertJSONField(t, row.before, "personId", targetID)
	assertJSONField(t, row.before, "roleId", roleID)

	// --- GrantInstanceAdmin.
	grant, err := coreApp.GrantInstanceAdmin(actorCtx, targetID)
	if err != nil {
		t.Fatalf("GrantInstanceAdmin: %v", err)
	}
	instanceAdminGrantID = grant.ID
	row = mustAuditRow(ctx, t, pool, "GRANT_INSTANCE_ADMIN", grant.ID)
	if row.actorPersonID != actorID || row.targetKind != "INSTANCE_ADMIN" || row.before != nil {
		t.Errorf("GRANT_INSTANCE_ADMIN audit row = %+v, want actor=%s target_kind=INSTANCE_ADMIN before=nil", row, actorID)
	}
	assertJSONField(t, row.after, "personId", targetID)

	// --- RevokeInstanceAdmin.
	if err := coreApp.RevokeInstanceAdmin(actorCtx, targetID); err != nil {
		t.Fatalf("RevokeInstanceAdmin: %v", err)
	}
	row = mustAuditRow(ctx, t, pool, "REVOKE_INSTANCE_ADMIN", grant.ID)
	if row.actorPersonID != actorID || row.targetKind != "INSTANCE_ADMIN" || row.after != nil {
		t.Errorf("REVOKE_INSTANCE_ADMIN audit row = %+v, want actor=%s target_kind=INSTANCE_ADMIN after=nil", row, actorID)
	}
	assertJSONField(t, row.before, "personId", targetID)

	// --- DeactivateAccount.
	if _, err := coreApp.DeactivateAccount(actorCtx, targetID); err != nil {
		t.Fatalf("DeactivateAccount: %v", err)
	}
	row = mustAuditRow(ctx, t, pool, "DEACTIVATE_ACCOUNT", targetID)
	if row.actorPersonID != actorID || row.targetKind != "ACCOUNT" {
		t.Errorf("DEACTIVATE_ACCOUNT audit row = %+v, want actor=%s target_kind=ACCOUNT", row, actorID)
	}
	assertJSONField(t, row.before, "status", "active")
	assertJSONField(t, row.after, "status", "disabled")

	// --- ReactivateAccount.
	if _, err := coreApp.ReactivateAccount(actorCtx, targetID); err != nil {
		t.Fatalf("ReactivateAccount: %v", err)
	}
	row = mustAuditRow(ctx, t, pool, "REACTIVATE_ACCOUNT", targetID)
	if row.actorPersonID != actorID || row.targetKind != "ACCOUNT" {
		t.Errorf("REACTIVATE_ACCOUNT audit row = %+v, want actor=%s target_kind=ACCOUNT", row, actorID)
	}
	assertJSONField(t, row.before, "status", "disabled")
	assertJSONField(t, row.after, "status", "active")

	// --- ListAuditLog itself: every action above must be visible, filterable by actor.
	entries, err := coreApp.ListAuditLog(actorCtx, coreapplication.AuditLogFilter{ActorPersonID: actorID}, 100, nil)
	if err != nil {
		t.Fatalf("ListAuditLog: %v", err)
	}
	if len(entries) != 6 {
		t.Errorf("ListAuditLog(actor=%s) returned %d entries, want 6 (one per mutation above)", actorID, len(entries))
	}
}

func errorsIs(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

type auditRow struct {
	actorPersonID string
	targetKind    string
	before, after []byte
}

func mustAuditRow(ctx context.Context, t *testing.T, pool *pgxpool.Pool, action, targetID string) auditRow {
	t.Helper()
	var r auditRow
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(actor_person_id::text, ''), target_kind, before, after
		FROM openfaithmap.identity_audit_log
		WHERE action = $1 AND target_id = $2
		ORDER BY created_at DESC LIMIT 1`, action, targetID,
	).Scan(&r.actorPersonID, &r.targetKind, &r.before, &r.after)
	if err != nil {
		t.Fatalf("query audit row for action=%s target=%s: %v", action, targetID, err)
	}
	return r
}

func assertNoAuditRow(ctx context.Context, t *testing.T, pool *pgxpool.Pool, action, targetID string) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM openfaithmap.identity_audit_log WHERE action = $1 AND target_id = $2`,
		action, targetID).Scan(&count); err != nil {
		t.Fatalf("count audit rows for action=%s target=%s: %v", action, targetID, err)
	}
	if count != 0 {
		t.Errorf("found %d audit rows for action=%s target=%s, want 0 (the mutation should never have run)", count, action, targetID)
	}
}

func assertJSONField(t *testing.T, raw []byte, key, want string) {
	t.Helper()
	if raw == nil {
		t.Errorf("json field %q: payload is nil, want it to contain %q", key, want)
		return
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal audit payload: %v", err)
	}
	if m[key] != want {
		t.Errorf("json field %q = %q, want %q", key, m[key], want)
	}
}
