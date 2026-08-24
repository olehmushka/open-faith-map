// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M11.5's self-service profile page backend against a real Postgres instance: UpdateMyProfile
// is audit-logged and gated the same way every other mutation this arc adds is (no subject in
// context -> ErrPermissionDenied, zero audit rows, zero write), and — the concrete IDOR-safety proof
// the milestone exists to make — ListMyRoleAssignments never leaks another person's role assignments,
// even though the endpoint takes no personId argument at all. See core_super_admin_integration_test.go's
// own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestSelfServiceProfileIntegration -v
package core_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
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
	identitydomain "github.com/olehmushka/open-faith-map/internal/identity/domain"
)

func TestSelfServiceProfileIntegration(t *testing.T) {
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
	coreApp := coreapplication.NewService(nil, nil, nil, identitySvc, nil, authzSvc, auditLogSvc, pool, "")

	var personAID, personBID string
	var unit directorydomain.Unit
	var assignmentAID, assignmentBID string
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, personAID); err != nil {
			t.Errorf("cleanup: delete audit rows: %v", err)
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
		for _, id := range []string{assignmentAID, assignmentBID} {
			if id == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete assignment %s: %v", id, err)
			}
		}
		for _, id := range []string{personAID, personBID} {
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

	unit, err = dir.CreateUnit(ctx, directorydomain.Unit{Name: "M11.5 self-service test unit"})
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.5 Person A', 'M11.5', 'A') RETURNING id`).Scan(&personAID); err != nil {
		t.Fatalf("insert person A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.5 Person B', 'M11.5', 'B') RETURNING id`).Scan(&personBID); err != nil {
		t.Fatalf("insert person B: %v", err)
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

	// ---------------------------------------------------------------- UpdateMyProfile

	if _, err := coreApp.UpdateMyProfile(ctx, "Should Not Land"); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("UpdateMyProfile with no subject = %v, want ErrPermissionDenied", err)
	}
	assertNoAuditRow(ctx, t, pool, "UPDATE_PROFILE", personAID)

	actorCtxA := authz.NewContext(ctx, authz.Subject{PersonID: personAID})
	updated, err := coreApp.UpdateMyProfile(actorCtxA, "M11.5 Person A Updated")
	if err != nil {
		t.Fatalf("UpdateMyProfile(A): %v", err)
	}
	if updated.DisplayName != "M11.5 Person A Updated" {
		t.Errorf("UpdateMyProfile(A).DisplayName = %q, want %q", updated.DisplayName, "M11.5 Person A Updated")
	}
	row := mustAuditRow(ctx, t, pool, "UPDATE_PROFILE", personAID)
	if row.actorPersonID != personAID || row.targetKind != "PERSON" {
		t.Errorf("UPDATE_PROFILE audit row = %+v, want actor=%s target_kind=PERSON", row, personAID)
	}
	assertJSONField(t, row.before, "displayName", "M11.5 Person A")
	assertJSONField(t, row.after, "displayName", "M11.5 Person A Updated")

	// B's row must be untouched by A's update.
	untouchedB, err := coreApp.GetPerson(ctx, personBID)
	if err != nil {
		t.Fatalf("GetPerson(B): %v", err)
	}
	if untouchedB.DisplayName != "M11.5 Person B" {
		t.Errorf("GetPerson(B).DisplayName = %q, want unchanged %q", untouchedB.DisplayName, "M11.5 Person B")
	}

	// ---------------------------------------------------------------- ListMyRoleAssignments

	if _, err := coreApp.ListMyRoleAssignments(ctx); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("ListMyRoleAssignments with no subject = %v, want ErrPermissionDenied", err)
	}

	assignmentAID, err = authzSvc.GrantUnitRole(ctx, personAID, roleID, unit.ID, authzdomain.ScopeUnit, "", personAID)
	if err != nil {
		t.Fatalf("GrantUnitRole(A): %v", err)
	}
	assignmentBID, err = authzSvc.GrantUnitRole(ctx, personBID, roleID, unit.ID, authzdomain.ScopeUnit, "", personAID)
	if err != nil {
		t.Fatalf("GrantUnitRole(B): %v", err)
	}

	// The concrete IDOR-safety proof: A's subject sees only A's own assignment, never B's, even
	// though ListMyRoleAssignments takes no personId argument at all.
	mine, err := coreApp.ListMyRoleAssignments(actorCtxA)
	if err != nil {
		t.Fatalf("ListMyRoleAssignments(A): %v", err)
	}
	if len(mine) != 1 || mine[0].ID != assignmentAID {
		t.Fatalf("ListMyRoleAssignments(A) = %+v, want exactly [%q]", mine, assignmentAID)
	}
	for _, a := range mine {
		if a.PersonID == personBID {
			t.Fatalf("ListMyRoleAssignments(A) leaked B's assignment: %+v", a)
		}
	}

	actorCtxB := authz.NewContext(ctx, authz.Subject{PersonID: personBID})
	theirs, err := coreApp.ListMyRoleAssignments(actorCtxB)
	if err != nil {
		t.Fatalf("ListMyRoleAssignments(B): %v", err)
	}
	if len(theirs) != 1 || theirs[0].ID != assignmentBID {
		t.Fatalf("ListMyRoleAssignments(B) = %+v, want exactly [%q]", theirs, assignmentBID)
	}
}

// TestSelfServiceApiKeysIntegration proves M11.9's self-service API-key CRUD against a real Postgres
// instance: the same no-subject-denies-with-zero-audit-rows and IDOR-safety shape every other M11.x
// self-service mutation already proves, plus the one piece of genuinely new logic this milestone
// introduces — the allowlist ∩ live-grants intersection — exercised here against REAL
// ActiveGrantsForSubject SQL (internal/authz's own unit tests already cover the same logic against a
// fake store; this is the end-to-end proof the two actually compose).
func TestSelfServiceApiKeysIntegration(t *testing.T) {
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
	coreApp := coreapplication.NewService(nil, nil, nil, identitySvc, nil, authzSvc, auditLogSvc, pool, "")

	var personAID, personBID string
	var unit directorydomain.Unit
	var assignmentID string
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, personAID); err != nil {
			t.Errorf("cleanup: delete audit rows: %v", err)
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
		if personAID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_api_keys WHERE person_id = $1`, personAID); err != nil {
				t.Errorf("cleanup: delete api keys: %v", err)
			}
		}
		if assignmentID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, assignmentID); err != nil {
				t.Errorf("cleanup: delete assignment: %v", err)
			}
		}
		for _, id := range []string{personAID, personBID} {
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

	unit, err = dir.CreateUnit(ctx, directorydomain.Unit{Name: "M11.9 self-service api-key test unit"})
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.9 Person A', 'M11.9', 'A') RETURNING id`).Scan(&personAID); err != nil {
		t.Fatalf("insert person A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.9 Person B', 'M11.9', 'B') RETURNING id`).Scan(&personBID); err != nil {
		t.Fatalf("insert person B: %v", err)
	}

	// ---------------------------------------------------------------- no-subject denial

	if _, err := coreApp.CreateApiKey(ctx, "should not land", []string{"person.read"}); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("CreateApiKey with no subject = %v, want ErrPermissionDenied", err)
	}
	var preAuditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, personAID).Scan(&preAuditCount); err != nil {
		t.Fatalf("count pre-existing audit rows: %v", err)
	}
	if preAuditCount != 0 {
		t.Fatalf("audit rows for a fresh person before any successful call = %d, want 0", preAuditCount)
	}

	// ---------------------------------------------------------------- CreateApiKey + audit shape

	actorCtxA := authz.NewContext(ctx, authz.Subject{PersonID: personAID})
	result, err := coreApp.CreateApiKey(actorCtxA, "M11.9 test key", []string{"person.read"})
	if err != nil {
		t.Fatalf("CreateApiKey(A): %v", err)
	}
	if !strings.HasPrefix(result.Token, identitydomain.APIKeyTokenPrefix) {
		t.Errorf("CreateApiKey token = %q, want a %q-prefixed value", result.Token, identitydomain.APIKeyTokenPrefix)
	}
	if len(result.PermissionCodes) != 1 || result.PermissionCodes[0] != "person.read" {
		t.Errorf("CreateApiKey.PermissionCodes = %v, want [person.read]", result.PermissionCodes)
	}

	row := mustAuditRow(ctx, t, pool, "CREATE_API_KEY", result.ID)
	if row.actorPersonID != personAID || row.targetKind != "API_KEY" {
		t.Errorf("CREATE_API_KEY audit row = %+v, want actor=%s target_kind=API_KEY", row, personAID)
	}
	assertJSONKeyAbsent(t, row.after, "token")
	assertJSONKeyAbsent(t, row.after, "tokenHash")

	// ---------------------------------------------------------------- IDOR proof

	keysB, err := coreApp.ListMyApiKeys(authz.NewContext(ctx, authz.Subject{PersonID: personBID}))
	if err != nil {
		t.Fatalf("ListMyApiKeys(B): %v", err)
	}
	for _, k := range keysB {
		if k.ID == result.ID {
			t.Fatalf("ListMyApiKeys(B) leaked A's key: %+v", k)
		}
	}
	if err := coreApp.RevokeMyApiKey(authz.NewContext(ctx, authz.Subject{PersonID: personBID}), result.ID); !errorsIs(err, identitydomain.ErrAPIKeyNotFound) {
		t.Errorf("RevokeMyApiKey(B, A's key id) = %v, want ErrAPIKeyNotFound", err)
	}
	keysA, err := coreApp.ListMyApiKeys(actorCtxA)
	if err != nil {
		t.Fatalf("ListMyApiKeys(A): %v", err)
	}
	if len(keysA) != 1 || keysA[0].ID != result.ID || keysA[0].RevokedAt != nil {
		t.Fatalf("ListMyApiKeys(A) after B's failed revoke attempt = %+v, want A's key still active", keysA)
	}

	// ---------------------------------------------------------------- RevokeMyApiKey (self)

	if err := coreApp.RevokeMyApiKey(actorCtxA, result.ID); err != nil {
		t.Fatalf("RevokeMyApiKey(A): %v", err)
	}
	revokeRow := mustAuditRow(ctx, t, pool, "REVOKE_API_KEY", result.ID)
	if revokeRow.actorPersonID != personAID {
		t.Errorf("REVOKE_API_KEY audit row actor = %q, want %q", revokeRow.actorPersonID, personAID)
	}

	// ------------------------------------------------- allowlist ∩ live-grants intersection (real DB)

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
	// registration-operator grants unit.read and person.update, but NOT person.read
	// (migrations/0015_core_seed.sql) — deliberately picked so the allowlist below can be broader in
	// one direction (includes person.read, which A doesn't hold) and narrower in the other (excludes
	// person.update, which A does hold), proving the intersection cuts both ways.
	assignmentID, err = authzSvc.GrantUnitRole(ctx, personAID, roleID, unit.ID, authzdomain.ScopeUnit, "", personAID)
	if err != nil {
		t.Fatalf("GrantUnitRole(A): %v", err)
	}

	apiKeyCtx := authz.NewContext(ctx, authz.Subject{
		PersonID: personAID, APIKeyPermissionCodes: []string{"unit.read", "person.read"},
	})
	if err := authzSvc.Require(apiKeyCtx, authzdomain.PermUnitRead, unit.ID); err != nil {
		t.Errorf("Require(unit.read) for an API key in-allowlist and in-live-grants = %v, want nil", err)
	}
	if err := authzSvc.Require(apiKeyCtx, authzdomain.PermPersonRead, unit.ID); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("Require(person.read) for an API key in-allowlist but NOT in live grants = %v, want ErrPermissionDenied", err)
	}
	if err := authzSvc.Require(apiKeyCtx, authzdomain.PermPersonUpdate, unit.ID); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("Require(person.update) for an API key in live grants but NOT in-allowlist = %v, want ErrPermissionDenied", err)
	}
}

// assertJSONKeyAbsent fails if raw's top-level JSON object contains key — used to prove an audit
// payload never carries a secret-adjacent field (M11.9: the raw token/hash must never reach the audit
// log), where the payload's other values aren't all strings (e.g. permissionCodes is a list), so
// assertJSONField's map[string]string shape doesn't apply.
func assertJSONKeyAbsent(t *testing.T, raw []byte, key string) {
	t.Helper()
	if raw == nil {
		return
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal audit payload: %v", err)
	}
	if _, ok := m[key]; ok {
		t.Errorf("audit payload unexpectedly contains key %q: %v", key, m)
	}
}
