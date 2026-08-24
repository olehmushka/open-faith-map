// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M10.7's new store-backed methods (ListRoles, ListRoleAssignmentsByUnit,
// RevokeRoleAssignment, ListInstanceAdmins, GrantInstanceAdmin, RevokeInstanceAdmin,
// RequireInstanceAdmin) against a real Postgres instance — see
// internal/directory/directory_integration_test.go's own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/authz/... -run TestAuthzAdminSurfaceIntegration -v
package authz_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/authz/adapters"
	"github.com/olehmushka/open-faith-map/internal/authz/domain"
	directoryapp "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
)

func TestAuthzAdminSurfaceIntegration(t *testing.T) {
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
	// A noop ClosurePort is fine here: everything under test (ListRoles, GrantUnitRole,
	// ListRoleAssignmentsByUnit, RevokeRoleAssignment, the instance-admin plane) never calls
	// PDP.Decide, so the closure port is never actually invoked.
	pdp := domain.NewPDP(noopClosure{})
	svc := authz.NewService(pdp, adapters.NewStore(pool))

	var personID string
	var unit directorydomain.Unit
	var assignmentIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range assignmentIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete assignment %s: %v", id, err)
			}
		}
		if personID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_instance_admins WHERE person_id = $1`, personID); err != nil {
				t.Errorf("cleanup: delete instance-admin grant: %v", err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, personID); err != nil {
				t.Errorf("cleanup: delete person: %v", err)
			}
		}
		if unit.ID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, unit.ID); err != nil {
				t.Errorf("cleanup: delete unit: %v", err)
			}
		}
	})

	unit, err = dir.CreateUnit(ctx, directorydomain.Unit{Name: "M10.7 authz admin surface test unit"})
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M10.7 Authz Test Person', 'M10.7', 'Test') RETURNING id`).Scan(&personID); err != nil {
		t.Fatalf("insert test person: %v", err)
	}

	// --- ListRoles: the three seeded base roles are present.
	roles, err := svc.ListRoles(ctx)
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	var registrationOperatorRoleID string
	for _, r := range roles {
		if r.Code == "registration-operator" {
			registrationOperatorRoleID = r.ID
		}
	}
	if registrationOperatorRoleID == "" {
		t.Fatalf("ListRoles = %+v, want it to include the seeded registration-operator role", roles)
	}

	// --- GrantUnitRole (existing, M10.6) + ListRoleAssignmentsByUnit (new) + RevokeRoleAssignment (new).
	if _, err := svc.GrantUnitRole(ctx, personID, registrationOperatorRoleID, unit.ID, domain.ScopeUnit, "", ""); err != nil {
		t.Fatalf("GrantUnitRole: %v", err)
	}
	assignments, err := svc.ListRoleAssignmentsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsByUnit: %v", err)
	}
	if len(assignments) != 1 || assignments[0].PersonID != personID || assignments[0].RoleCode != "registration-operator" {
		t.Fatalf("ListRoleAssignmentsByUnit = %+v, want one assignment for person=%s role=registration-operator", assignments, personID)
	}
	assignmentIDs = append(assignmentIDs, assignments[0].ID)

	if _, err := svc.RevokeRoleAssignment(ctx, assignments[0].ID, ""); err != nil {
		t.Fatalf("RevokeRoleAssignment: %v", err)
	}
	afterRevoke, err := svc.ListRoleAssignmentsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsByUnit (after revoke): %v", err)
	}
	if len(afterRevoke) != 0 {
		t.Errorf("ListRoleAssignmentsByUnit after revoke = %+v, want none", afterRevoke)
	}
	if _, err := svc.RevokeRoleAssignment(ctx, assignments[0].ID, ""); !errors.Is(err, domain.ErrAssignmentNotFound) {
		t.Errorf("RevokeRoleAssignment (already revoked) error = %v, want ErrAssignmentNotFound", err)
	}

	// --- GrantInstanceAdmin + ListInstanceAdmins + RequireInstanceAdmin + RevokeInstanceAdmin.
	if _, err := svc.GrantInstanceAdmin(ctx, personID, ""); err != nil {
		t.Fatalf("GrantInstanceAdmin: %v", err)
	}
	admins, err := svc.ListInstanceAdmins(ctx)
	if err != nil {
		t.Fatalf("ListInstanceAdmins: %v", err)
	}
	var sawPerson bool
	for _, a := range admins {
		if a.PersonID == personID {
			sawPerson = true
		}
	}
	if !sawPerson {
		t.Errorf("ListInstanceAdmins = %+v, want it to include %s", admins, personID)
	}

	adminCtx := authz.NewContext(ctx, authz.Subject{PersonID: personID})
	if err := svc.RequireInstanceAdmin(adminCtx); err != nil {
		t.Errorf("RequireInstanceAdmin for a real instance admin = %v, want nil", err)
	}

	if _, err := svc.RevokeInstanceAdmin(ctx, personID, ""); err != nil {
		t.Fatalf("RevokeInstanceAdmin: %v", err)
	}
	if err := svc.RequireInstanceAdmin(adminCtx); !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("RequireInstanceAdmin after revoke = %v, want ErrPermissionDenied", err)
	}
	if _, err := svc.RevokeInstanceAdmin(ctx, personID, ""); !errors.Is(err, domain.ErrInstanceAdminGrantNotFound) {
		t.Errorf("RevokeInstanceAdmin (already revoked) error = %v, want ErrInstanceAdminGrantNotFound", err)
	}
}

type noopClosure struct{}

func (noopClosure) IsAncestorOrSelf(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (noopClosure) IsAuthorityBearing(context.Context, string) (bool, error) { return false, nil }
