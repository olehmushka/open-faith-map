// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M11.2's own stated goal — "the log has no blind spot from day one" — against a real
// Postgres instance: every one of core.application.Service's mutating super-admin methods
// (GrantUnitRole, RevokeRoleAssignment, GrantInstanceAdmin, RevokeInstanceAdmin, DeactivateAccount,
// ReactivateAccount, and M11.3's RevokeSession) writes exactly one identity_audit_log row with the
// expected actor/action/target/before/after shape. See internal/authz/authz_integration_test.go's
// own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestSuperAdminAuditTrailIntegration -v
package core_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

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
	identitydomain "github.com/olehmushka/open-faith-map/internal/identity/domain"
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
	var assignmentID, instanceAdminGrantID, invitedPersonID string
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
		if invitedPersonID != "" {
			// identity_invites references both the invited account and person (ON DELETE RESTRICT
			// on both), so it must go first.
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_invites WHERE person_id = $1`, invitedPersonID); err != nil {
				t.Errorf("cleanup: delete invite: %v", err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_accounts WHERE person_id = $1`, invitedPersonID); err != nil {
				t.Errorf("cleanup: delete invited account: %v", err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, invitedPersonID); err != nil {
				t.Errorf("cleanup: delete invited person: %v", err)
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
	var targetSessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_sessions (account_id, issuer)
		VALUES ($1, 'urn:test:m11-3-audit-issuer') RETURNING id`, targetAccountID).Scan(&targetSessionID); err != nil {
		t.Fatalf("insert target session: %v", err)
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
	if _, err := coreApp.InvitePerson(ctx, "m11-6-no-subject@example.test", "No Subject"); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("InvitePerson with no subject = %v, want ErrPermissionDenied", err)
	}

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

	// --- RevokeSession (M11.3).
	if err := coreApp.RevokeSession(actorCtx, targetID, targetSessionID); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	row = mustAuditRow(ctx, t, pool, "REVOKE_SESSION", targetSessionID)
	if row.actorPersonID != actorID || row.targetKind != "SESSION" {
		t.Errorf("REVOKE_SESSION audit row = %+v, want actor=%s target_kind=SESSION", row, actorID)
	}
	var revokedAt *string
	if err := pool.QueryRow(ctx, `SELECT revoked_at::text FROM openfaithmap.identity_sessions WHERE id = $1`, targetSessionID).Scan(&revokedAt); err != nil {
		t.Fatalf("read back session revoked_at: %v", err)
	}
	if revokedAt == nil {
		t.Error("session revoked_at is still NULL after RevokeSession")
	}

	// --- InvitePerson (M11.6).
	invite, err := coreApp.InvitePerson(actorCtx, "m11-6-invitee@example.test", "M11.6 Invitee")
	if err != nil {
		t.Fatalf("InvitePerson: %v", err)
	}
	invitedPersonID = invite.PersonID
	if invite.Token == "" {
		t.Error("InvitePerson returned an empty token")
	}
	row = mustAuditRow(ctx, t, pool, "CREATE_INVITE", invite.PersonID)
	if row.actorPersonID != actorID || row.targetKind != "PERSON" {
		t.Errorf("CREATE_INVITE audit row = %+v, want actor=%s target_kind=PERSON", row, actorID)
	}
	assertJSONField(t, row.after, "email", "m11-6-invitee@example.test")
	assertJSONField(t, row.after, "displayName", "M11.6 Invitee")

	// --- ListAuditLog itself: every action above must be visible, filterable by actor.
	entries, err := coreApp.ListAuditLog(actorCtx, coreapplication.AuditLogFilter{ActorPersonID: actorID}, 100, nil)
	if err != nil {
		t.Fatalf("ListAuditLog: %v", err)
	}
	if len(entries) != 8 {
		t.Errorf("ListAuditLog(actor=%s) returned %d entries, want 8 (one per mutation above)", actorID, len(entries))
	}
}

// TestLastActiveIntegration proves M11.4's revoked-inclusive last-active signal against a real
// Postgres instance, over both of its read paths (GetAccountStatus for the person detail page,
// SearchPersons for the people list) — plain reads, no audit-log mutation, so unlike
// TestSuperAdminAuditTrailIntegration this test's cleanup never touches identity_audit_log's
// append-only trigger.
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestLastActiveIntegration -v
func TestLastActiveIntegration(t *testing.T) {
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

	identitySvc := identityapplication.NewService(identityadapters.NewStore(pool))
	coreApp := coreapplication.NewService(nil, nil, nil, identitySvc, nil, nil, nil)

	var noAccountID, withAccountID, accountID, sessionID string
	t.Cleanup(func() {
		bg := context.Background()
		if sessionID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_sessions WHERE id = $1`, sessionID); err != nil {
				t.Errorf("cleanup: delete session: %v", err)
			}
		}
		if accountID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_accounts WHERE id = $1`, accountID); err != nil {
				t.Errorf("cleanup: delete account: %v", err)
			}
		}
		for _, id := range []string{noAccountID, withAccountID} {
			if id == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
	})

	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.4 NoAccount Test', 'M11.4', 'NoAccount') RETURNING id`).Scan(&noAccountID); err != nil {
		t.Fatalf("insert no-account person: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.4 WithAccount Test', 'M11.4', 'WithAccount') RETURNING id`).Scan(&withAccountID); err != nil {
		t.Fatalf("insert with-account person: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_accounts (person_id, email)
		VALUES ($1, 'm11-4-target@example.test') RETURNING id`, withAccountID).Scan(&accountID); err != nil {
		t.Fatalf("insert account: %v", err)
	}

	findByID := func(persons []identitydomain.Person, id string) (identitydomain.Person, bool) {
		for _, p := range persons {
			if p.ID == id {
				return p, true
			}
		}
		return identitydomain.Person{}, false
	}

	// --- No account at all: status "none", LastActiveAt nil on both read paths.
	status, err := coreApp.GetAccountStatus(ctx, noAccountID)
	if err != nil {
		t.Fatalf("GetAccountStatus(no account): %v", err)
	}
	if status.Status != coreapplication.AccountStatusNone || status.LastActiveAt != nil {
		t.Errorf("GetAccountStatus(no account) = %+v, want status=none lastActiveAt=nil", status)
	}

	// --- Has an account, but no session yet: active, LastActiveAt still nil.
	status, err = coreApp.GetAccountStatus(ctx, withAccountID)
	if err != nil {
		t.Fatalf("GetAccountStatus(no session): %v", err)
	}
	if status.Status != "active" || status.LastActiveAt != nil {
		t.Errorf("GetAccountStatus(no session) = %+v, want status=active lastActiveAt=nil", status)
	}
	persons, err := coreApp.SearchPersons(ctx, "M11.4 WithAccount Test", 10)
	if err != nil {
		t.Fatalf("SearchPersons(no session): %v", err)
	}
	p, ok := findByID(persons, withAccountID)
	if !ok {
		t.Fatalf("SearchPersons(no session) = %+v, want it to include %s", persons, withAccountID)
	}
	if p.LastActiveAt != nil {
		t.Errorf("SearchPersons(no session) person.LastActiveAt = %v, want nil", p.LastActiveAt)
	}

	// --- A session exists: both read paths must reflect its last_seen_at.
	var wantLastActive time.Time
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_sessions (account_id, issuer, last_seen_at)
		VALUES ($1, 'urn:test:m11-4-issuer', now() - interval '1 hour')
		RETURNING id, last_seen_at`, accountID).Scan(&sessionID, &wantLastActive); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	status, err = coreApp.GetAccountStatus(ctx, withAccountID)
	if err != nil {
		t.Fatalf("GetAccountStatus(active session): %v", err)
	}
	if status.LastActiveAt == nil || !status.LastActiveAt.Equal(wantLastActive) {
		t.Errorf("GetAccountStatus(active session).LastActiveAt = %v, want %v", status.LastActiveAt, wantLastActive)
	}
	persons, err = coreApp.SearchPersons(ctx, "M11.4 WithAccount Test", 10)
	if err != nil {
		t.Fatalf("SearchPersons(active session): %v", err)
	}
	p, ok = findByID(persons, withAccountID)
	if !ok || p.LastActiveAt == nil || !p.LastActiveAt.Equal(wantLastActive) {
		t.Errorf("SearchPersons(active session) person = %+v, want LastActiveAt=%v", p, wantLastActive)
	}

	// --- Revoke the session: last-active must still reflect the same historical timestamp
	// (M11.4's revoked-inclusive decision — revoking a session doesn't erase that it happened).
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.identity_sessions SET revoked_at = now() WHERE id = $1`, sessionID); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	status, err = coreApp.GetAccountStatus(ctx, withAccountID)
	if err != nil {
		t.Fatalf("GetAccountStatus(revoked session): %v", err)
	}
	if status.LastActiveAt == nil || !status.LastActiveAt.Equal(wantLastActive) {
		t.Errorf("GetAccountStatus(revoked session).LastActiveAt = %v, want unchanged %v (revoked-inclusive)", status.LastActiveAt, wantLastActive)
	}
}

// TestBulkGrantUnitRoleIntegration proves M11.7's own explicit "in a transaction" requirement against
// a real Postgres instance: a batch either fully applies (every person gets the assignment, one audit
// row each) or fully rolls back (a bad id anywhere in the batch leaves zero new assignments and zero
// new audit rows for every id in that batch, not just the bad one) — plus the specific regression this
// milestone's own store method exists to avoid: an in-batch idempotent conflict (a duplicate/
// pre-existing grant) must not abort the surrounding transaction.
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestBulkGrantUnitRoleIntegration -v
func TestBulkGrantUnitRoleIntegration(t *testing.T) {
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
	authzSvc := authz.NewService(authzdomain.NewPDP(noopClosure{}), authzadapters.NewStore(pool))
	auditLogSvc := auditlogapplication.NewService(auditlogadapters.NewStore(pool))
	coreApp := coreapplication.NewService(nil, nil, nil, nil, nil, authzSvc, auditLogSvc)

	var actorID string
	var unit directorydomain.Unit
	var personIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, actorID); err != nil {
			t.Errorf("cleanup: delete audit rows: %v", err)
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
		if unit.ID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE target_unit_id = $1`, unit.ID); err != nil {
				t.Errorf("cleanup: delete assignments: %v", err)
			}
		}
		for _, id := range append([]string{actorID}, personIDs...) {
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

	unit, err = dir.CreateUnit(ctx, directorydomain.Unit{Name: "M11.7 bulk grant test unit"})
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.7 Bulk Actor', 'M11.7', 'Actor') RETURNING id`).Scan(&actorID); err != nil {
		t.Fatalf("insert actor person: %v", err)
	}

	insertPerson := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
			VALUES ($1, 'M11.7', 'Bulk') RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		personIDs = append(personIDs, id)
		return id
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

	countAssignments := func() int {
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM openfaithmap.authz_role_assignments
			WHERE target_unit_id = $1 AND revoked_at IS NULL`, unit.ID).Scan(&n); err != nil {
			t.Fatalf("count assignments: %v", err)
		}
		return n
	}
	countAuditRows := func() int {
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM openfaithmap.identity_audit_log
			WHERE action = 'BULK_GRANT_UNIT_ROLE' AND actor_person_id = $1`, actorID).Scan(&n); err != nil {
			t.Fatalf("count audit rows: %v", err)
		}
		return n
	}

	// --- requireSubject gate: no subject in context must fail loud, nothing written.
	p0 := insertPerson("M11.7 NoSubject")
	if err := coreApp.BulkGrantUnitRole(ctx, []string{p0}, roleID, unit.ID); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("BulkGrantUnitRole with no subject = %v, want ErrPermissionDenied", err)
	}
	if n := countAssignments(); n != 0 {
		t.Errorf("after no-subject call: %d assignments, want 0", n)
	}

	// --- Happy path: 3 persons -> 3 new active assignments, 3 audit rows.
	p1, p2, p3 := insertPerson("M11.7 Happy One"), insertPerson("M11.7 Happy Two"), insertPerson("M11.7 Happy Three")
	if err := coreApp.BulkGrantUnitRole(actorCtx, []string{p1, p2, p3}, roleID, unit.ID); err != nil {
		t.Fatalf("BulkGrantUnitRole happy path: %v", err)
	}
	if n := countAssignments(); n != 3 {
		t.Errorf("after happy-path batch: %d active assignments, want 3", n)
	}
	if n := countAuditRows(); n != 3 {
		t.Errorf("after happy-path batch: %d BULK_GRANT_UNIT_ROLE audit rows, want 3", n)
	}
	assignments, err := authzSvc.ListRoleAssignmentsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsByUnit: %v", err)
	}
	byPerson := map[string]string{}
	for _, a := range assignments {
		byPerson[a.PersonID] = a.ID
	}
	row := mustAuditRow(ctx, t, pool, "BULK_GRANT_UNIT_ROLE", byPerson[p1])
	if row.actorPersonID != actorID || row.targetKind != "ROLE_ASSIGNMENT" || row.before != nil {
		t.Errorf("BULK_GRANT_UNIT_ROLE audit row = %+v, want actor=%s target_kind=ROLE_ASSIGNMENT before=nil", row, actorID)
	}
	assertJSONField(t, row.after, "personId", p1)
	assertJSONField(t, row.after, "roleId", roleID)
	assertJSONField(t, row.after, "unitId", unit.ID)

	// --- Real rollback proof: a batch with one nonexistent person id must roll back entirely, not
	// partially apply — this is the ticket's explicit "in a transaction" requirement.
	p4, p5 := insertPerson("M11.7 Rollback Four"), insertPerson("M11.7 Rollback Five")
	const nonexistentPersonID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	before := countAssignments()
	if err := coreApp.BulkGrantUnitRole(actorCtx, []string{p4, p5, nonexistentPersonID}, roleID, unit.ID); err == nil {
		t.Fatal("BulkGrantUnitRole with a nonexistent person id = nil error, want a real error (FK violation)")
	}
	if n := countAssignments(); n != before {
		t.Errorf("after rolled-back batch: %d active assignments, want unchanged %d (no partial apply)", n, before)
	}
	for _, p := range []string{p4, p5} {
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM openfaithmap.authz_role_assignments
			WHERE target_unit_id = $1 AND subject_person_id = $2`, unit.ID, p).Scan(&n); err != nil {
			t.Fatalf("count assignments for %s: %v", p, err)
		}
		if n != 0 {
			t.Errorf("person %s has %d assignment rows after a rolled-back batch, want 0", p, n)
		}
	}
	if n := countAuditRows(); n != 3 {
		t.Errorf("after rolled-back batch: %d total BULK_GRANT_UNIT_ROLE audit rows, want still 3 (unchanged from the happy path)", n)
	}

	// --- In-batch idempotent-conflict proof: a pre-existing active grant inside a batch must not
	// abort the transaction (the regression this milestone's ON CONFLICT DO UPDATE design exists to
	// prevent — see internal/authz/adapters/store.go's BulkInsertRoleAssignments doc comment).
	if err := coreApp.GrantUnitRole(actorCtx, p4, roleID, unit.ID); err != nil {
		t.Fatalf("pre-grant for idempotent-conflict case: %v", err)
	}
	auditBefore := countAuditRows()
	if err := coreApp.BulkGrantUnitRole(actorCtx, []string{p4, p5}, roleID, unit.ID); err != nil {
		t.Fatalf("BulkGrantUnitRole with an in-batch pre-existing grant: %v", err)
	}
	for _, p := range []string{p4, p5} {
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM openfaithmap.authz_role_assignments
			WHERE target_unit_id = $1 AND subject_person_id = $2 AND revoked_at IS NULL`, unit.ID, p).Scan(&n); err != nil {
			t.Fatalf("count active assignments for %s: %v", p, err)
		}
		if n != 1 {
			t.Errorf("person %s has %d active assignment rows after idempotent-conflict batch, want exactly 1", p, n)
		}
	}
	// GrantUnitRole's own single-grant call above already wrote one BULK_GRANT_UNIT_ROLE-unrelated
	// GRANT_UNIT_ROLE row; the batch call adds one row per person in it (p4's re-touch included).
	if n := countAuditRows(); n != auditBefore+2 {
		t.Errorf("after idempotent-conflict batch: %d BULK_GRANT_UNIT_ROLE audit rows, want %d (+2)", n, auditBefore+2)
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
