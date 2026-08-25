// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M12.1's unit lifecycle CRUD (CreateUnit/UpdateUnit/SetUnitState/DeleteUnit) against a real
// Postgres instance: the unit.lifecycle gate actually denies an ungranted caller and actually allows
// a caller holding a real (non-instance-admin) role assignment for the permission, the root unit is
// hard-guarded against SetUnitState/DeleteUnit regardless of grant, DeleteUnit's three
// orphan-protection checks (children, active role assignments, an existing religion org profile)
// each block correctly and clear once the blocking condition is gone, and every mutation writes
// exactly one identity_audit_log row. See core_super_admin_integration_test.go's own header comment
// for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestUnitLifecycleIntegration -v
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
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

func TestUnitLifecycleIntegration(t *testing.T) {
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
	religionSvc := religionapplication.NewService(pool, dir)
	authzSvc := authz.NewService(authzdomain.NewPDP(noopClosure{}), authzadapters.NewRepository(pool), pool)
	auditLogSvc := auditlogapplication.NewService(auditlogadapters.NewRepository(pool))
	coreApp := coreapplication.NewService(dir, religionSvc, nil, nil, nil, authzSvc, auditLogSvc, pool, seed.RootUnitID)

	var adminPersonID, grantedPersonID, ungrantedPersonID string
	var parentID, childID, orphanTestUnitID string
	var grantedAssignmentID string
	var unitIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		for _, pid := range []string{adminPersonID, grantedPersonID, ungrantedPersonID} {
			if pid == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, pid); err != nil {
				t.Errorf("cleanup: delete audit rows for %s: %v", pid, err)
			}
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
		// Revoking a role assignment (below, mid-test) sets revoked_at — it never deletes the row —
		// so cleanup must sweep every assignment touching our test units/persons unconditionally,
		// regardless of revoked status, before those units/persons can themselves be deleted.
		for _, pid := range []string{adminPersonID, grantedPersonID, ungrantedPersonID} {
			if pid == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE subject_person_id = $1`, pid); err != nil {
				t.Errorf("cleanup: delete role assignments for person %s: %v", pid, err)
			}
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE target_unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignments on %s: %v", id, err)
			}
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_permissions WHERE role_id = $1 AND permission_code = 'unit.lifecycle'`, seed.RegistrationOperatorRoleID); err != nil {
			t.Errorf("cleanup: revoke test-local unit.lifecycle from registration-operator: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_instance_admins WHERE person_id = $1`, adminPersonID); err != nil {
			t.Errorf("cleanup: revoke instance admin: %v", err)
		}
		if orphanTestUnitID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_org_profiles WHERE unit_id = $1`, orphanTestUnitID); err != nil {
				t.Errorf("cleanup: delete org profile: %v", err)
			}
		}
		for _, pid := range []string{adminPersonID, grantedPersonID, ungrantedPersonID} {
			if pid == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, pid); err != nil {
				t.Errorf("cleanup: delete person %s: %v", pid, err)
			}
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_edges WHERE parent_id = $1 OR child_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete edges touching %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_closure WHERE ancestor_id = $1 OR descendant_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete closure rows touching %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete unit %s: %v", id, err)
			}
		}
	})

	// ---------------------------------------------------------------- setup: three persons.
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M12.1 Admin', 'M12.1', 'Admin') RETURNING id`).Scan(&adminPersonID); err != nil {
		t.Fatalf("insert admin person: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M12.1 Granted', 'M12.1', 'Granted') RETURNING id`).Scan(&grantedPersonID); err != nil {
		t.Fatalf("insert granted person: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M12.1 Ungranted', 'M12.1', 'Ungranted') RETURNING id`).Scan(&ungrantedPersonID); err != nil {
		t.Fatalf("insert ungranted person: %v", err)
	}
	if _, err := authzSvc.GrantInstanceAdmin(ctx, adminPersonID, adminPersonID); err != nil {
		t.Fatalf("GrantInstanceAdmin: %v", err)
	}
	adminCtx := authz.NewContext(ctx, authz.Subject{PersonID: adminPersonID})
	ungrantedCtx := authz.NewContext(ctx, authz.Subject{PersonID: ungrantedPersonID})

	// ---------------------------------------------------------------- no subject at all.
	if _, err := coreApp.CreateUnit(ctx, seed.RootUnitID, "m121-noctx", "No Context", nil); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("CreateUnit with no subject = %v, want ErrPermissionDenied", err)
	}

	// ---------------------------------------------------------------- an authenticated caller with
	// zero grants is denied too (proves unit.lifecycle is actually checked, not just "any subject").
	if _, err := coreApp.CreateUnit(ungrantedCtx, seed.RootUnitID, "m121-ungranted", "Ungranted", nil); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("CreateUnit(ungranted) = %v, want ErrPermissionDenied", err)
	}

	// ---------------------------------------------------------------- instance-admin path: create a
	// parent unit under root, then exercise update/state/audit against it.
	parent, err := coreApp.CreateUnit(adminCtx, seed.RootUnitID, "m121-parent", "M12.1 Parent", nil)
	if err != nil {
		t.Fatalf("CreateUnit(parent): %v", err)
	}
	parentID = parent.ID
	unitIDs = append(unitIDs, parentID)
	createRow := mustAuditRow(ctx, t, pool, "CREATE_UNIT", parentID)
	if createRow.actorPersonID != adminPersonID || createRow.targetKind != "UNIT" {
		t.Errorf("CREATE_UNIT audit row = %+v, want actor=%s target_kind=UNIT", createRow, adminPersonID)
	}
	assertJSONFieldAny(t, createRow.after, "name", "M12.1 Parent")

	level := int16(2)
	updated, err := coreApp.UpdateUnit(adminCtx, parentID, "M12.1 Parent Renamed", nil, &level)
	if err != nil {
		t.Fatalf("UpdateUnit(parent): %v", err)
	}
	if updated.Name != "M12.1 Parent Renamed" {
		t.Errorf("UpdateUnit(parent).Name = %q, want %q", updated.Name, "M12.1 Parent Renamed")
	}
	updateRow := mustAuditRow(ctx, t, pool, "UPDATE_UNIT", parentID)
	assertJSONFieldAny(t, updateRow.before, "name", "M12.1 Parent")
	assertJSONFieldAny(t, updateRow.after, "name", "M12.1 Parent Renamed")

	suspended, err := coreApp.SetUnitState(adminCtx, parentID, directorydomain.StateSuspended)
	if err != nil {
		t.Fatalf("SetUnitState(parent, suspended): %v", err)
	}
	if suspended.State != directorydomain.StateSuspended {
		t.Errorf("SetUnitState(parent).State = %q, want suspended", suspended.State)
	}
	stateRow := mustAuditRow(ctx, t, pool, "SET_UNIT_STATE", parentID)
	assertJSONField(t, stateRow.before, "state", "active")
	assertJSONField(t, stateRow.after, "state", "suspended")

	// ---------------------------------------------------------------- root-unit guard: refuses
	// SetUnitState/DeleteUnit regardless of the caller holding instance-admin.
	if _, err := coreApp.SetUnitState(adminCtx, seed.RootUnitID, directorydomain.StateSuspended); !errorsIs(err, coreapplication.ErrRootUnitProtected) {
		t.Errorf("SetUnitState(root) = %v, want ErrRootUnitProtected", err)
	}
	if _, err := coreApp.DeleteUnit(adminCtx, seed.RootUnitID); !errorsIs(err, coreapplication.ErrRootUnitProtected) {
		t.Errorf("DeleteUnit(root) = %v, want ErrRootUnitProtected", err)
	}

	// ---------------------------------------------------------------- a real (non-instance-admin)
	// unit.lifecycle grant, via a real role assignment, is sufficient on its own — proves the
	// permission code itself gates correctly, not merely "any instance admin can do this."
	// registration-operator holds no unit.lifecycle after M12.0's split (migrations/
	// 0015_core_seed.sql); grant it back onto that role for this test's
	// duration only, cleaned up above.
	if _, err := pool.Exec(ctx, `
		INSERT INTO openfaithmap.authz_role_permissions (role_id, permission_code)
		VALUES ($1, 'unit.lifecycle') ON CONFLICT DO NOTHING`, seed.RegistrationOperatorRoleID); err != nil {
		t.Fatalf("grant test-local unit.lifecycle to registration-operator: %v", err)
	}
	grantedAssignmentID, err = authzSvc.GrantUnitRole(ctx, grantedPersonID, seed.RegistrationOperatorRoleID, parentID, authzdomain.ScopeUnit, "", adminPersonID, nil)
	if err != nil {
		t.Fatalf("GrantUnitRole(grantedPerson, registration-operator, parent): %v", err)
	}
	if grantedAssignmentID == "" {
		t.Fatal("GrantUnitRole returned empty assignment id")
	}
	grantedCtx := authz.NewContext(ctx, authz.Subject{PersonID: grantedPersonID})
	child, err := coreApp.CreateUnit(grantedCtx, parentID, "m121-child", "M12.1 Child", nil)
	if err != nil {
		t.Fatalf("CreateUnit(child) via real unit.lifecycle grant: %v", err)
	}
	childID = child.ID
	unitIDs = append(unitIDs, childID)

	// ---------------------------------------------------------------- DeleteUnit orphan-protection:
	// children.
	if _, err := coreApp.DeleteUnit(adminCtx, parentID); !errorsIs(err, directorydomain.ErrUnitHasChildren) {
		t.Errorf("DeleteUnit(parent with a child) = %v, want ErrUnitHasChildren", err)
	}
	if _, err := coreApp.DeleteUnit(adminCtx, childID); err != nil {
		t.Fatalf("DeleteUnit(child): %v", err)
	}
	deleteRow := mustAuditRow(ctx, t, pool, "DELETE_UNIT", childID)
	assertJSONField(t, deleteRow.before, "name", "M12.1 Child")
	if deleteRow.after != nil {
		t.Errorf("DELETE_UNIT audit row.after = %s, want nil (a delete has no after state)", deleteRow.after)
	}

	// ---------------------------------------------------------------- DeleteUnit orphan-protection:
	// active role assignments — parent still has grantedAssignmentID targeting it.
	if _, err := coreApp.DeleteUnit(adminCtx, parentID); !errorsIs(err, coreapplication.ErrUnitHasActiveRoleAssignments) {
		t.Errorf("DeleteUnit(parent with an active role assignment) = %v, want ErrUnitHasActiveRoleAssignments", err)
	}
	if _, err := authzSvc.RevokeRoleAssignment(ctx, grantedAssignmentID, adminPersonID); err != nil {
		t.Fatalf("RevokeRoleAssignment: %v", err)
	}

	// ---------------------------------------------------------------- DeleteUnit orphan-protection:
	// an existing religion org profile — exercised on a fresh unit (a delete-blocked unit is never
	// itself deleted, so this needs its own unit + its own cleanup, handled by orphanTestUnitID above).
	orphanUnit, err := coreApp.CreateUnit(adminCtx, seed.RootUnitID, "m121-profile", "M12.1 Profile Test", nil)
	if err != nil {
		t.Fatalf("CreateUnit(orphan profile test unit): %v", err)
	}
	orphanTestUnitID = orphanUnit.ID
	unitIDs = append(unitIDs, orphanTestUnitID)
	if _, err := religionSvc.SetOrgProfile(ctx, orphanTestUnitID, nil, nil); err != nil {
		t.Fatalf("SetOrgProfile: %v", err)
	}
	if _, err := coreApp.DeleteUnit(adminCtx, orphanTestUnitID); !errorsIs(err, coreapplication.ErrUnitHasOrgProfile) {
		t.Errorf("DeleteUnit(unit with an org profile) = %v, want ErrUnitHasOrgProfile", err)
	}

	// ---------------------------------------------------------------- now-childless, grant-free
	// parent can finally be deleted.
	if _, err := coreApp.DeleteUnit(adminCtx, parentID); err != nil {
		t.Errorf("DeleteUnit(parent, now clear of every orphan-protection reason): %v", err)
	}
}

// assertJSONFieldAny is assertJSONField's counterpart for a payload that mixes string and non-string
// fields (M12.1's UpdateUnit/CreateUnit audit maps include an int level) — assertJSONField's blanket
// map[string]string unmarshal rejects those outright.
func assertJSONFieldAny(t *testing.T, raw []byte, key, want string) {
	t.Helper()
	if raw == nil {
		t.Errorf("json field %q: payload is nil, want it to contain %q", key, want)
		return
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal audit payload: %v", err)
	}
	got, _ := m[key].(string)
	if got != want {
		t.Errorf("json field %q = %q, want %q", key, got, want)
	}
}
