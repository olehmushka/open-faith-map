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
	"time"

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
	svc := authz.NewService(pdp, adapters.NewRepository(pool), pool)

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
	if _, err := svc.GrantUnitRole(ctx, personID, registrationOperatorRoleID, unit.ID, domain.ScopeUnit, "", "", nil); err != nil {
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

	// --- M12.3: GrantUnitRole with an expiry, ListRoleAssignmentsBy* round-trips it,
	// ClearRoleAssignmentExpiry clears it, and a real expired row is excluded from
	// ActiveGrantsForSubject (the PDP's own enforcement path, unchanged by this milestone but
	// re-checked here now that a real expiry can finally be written).
	futureExpiry := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	expiringAssignmentID, err := svc.GrantUnitRole(ctx, personID, registrationOperatorRoleID, unit.ID, domain.ScopeUnit, "", "", &futureExpiry)
	if err != nil {
		t.Fatalf("GrantUnitRole (with expiry): %v", err)
	}
	assignmentIDs = append(assignmentIDs, expiringAssignmentID)

	byUnit, err := svc.ListRoleAssignmentsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsByUnit (with expiry): %v", err)
	}
	if len(byUnit) != 1 || byUnit[0].ExpiresAt == nil || !byUnit[0].ExpiresAt.Equal(futureExpiry) {
		t.Fatalf("ListRoleAssignmentsByUnit = %+v, want one assignment with ExpiresAt=%s", byUnit, futureExpiry)
	}
	byPerson, err := svc.ListRoleAssignmentsByPerson(ctx, personID)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsByPerson (with expiry): %v", err)
	}
	if len(byPerson) != 1 || byPerson[0].ExpiresAt == nil || !byPerson[0].ExpiresAt.Equal(futureExpiry) {
		t.Fatalf("ListRoleAssignmentsByPerson = %+v, want one assignment with ExpiresAt=%s", byPerson, futureExpiry)
	}

	// A real expired row (set directly, bypassing the future-only validation GrantUnitRole's own
	// caller — internal/core/application.Service — enforces) must be denied by the PDP, proving
	// ActiveGrantsForSubject's existing expires_at filter still works now that a real expiry can
	// finally be written and isn't just always NULL.
	if err := svc.DecideFor(ctx, personID, domain.Permission("unit.read"), unit.ID); err != nil {
		t.Fatalf("DecideFor (unit.read, before expiry) = %v, want nil (allowed)", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.authz_role_assignments SET expires_at = now() - interval '1 hour' WHERE id = $1`, expiringAssignmentID); err != nil {
		t.Fatalf("directly expire assignment: %v", err)
	}
	if err := svc.DecideFor(ctx, personID, domain.Permission("unit.read"), unit.ID); !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("DecideFor (unit.read, after expiry) = %v, want ErrPermissionDenied", err)
	}

	if _, err := svc.ClearRoleAssignmentExpiry(ctx, expiringAssignmentID); err != nil {
		t.Fatalf("ClearRoleAssignmentExpiry: %v", err)
	}
	afterClear, err := svc.ListRoleAssignmentsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsByUnit (after clear): %v", err)
	}
	if len(afterClear) != 1 || afterClear[0].ExpiresAt != nil {
		t.Fatalf("ListRoleAssignmentsByUnit after clear = %+v, want ExpiresAt nil", afterClear)
	}
	if _, err := svc.ClearRoleAssignmentExpiry(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrAssignmentNotFound) {
		t.Errorf("ClearRoleAssignmentExpiry (unknown id) error = %v, want ErrAssignmentNotFound", err)
	}
}

type noopClosure struct{}

func (noopClosure) IsAncestorOrSelf(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (noopClosure) IsAuthorityBearing(context.Context, string) (bool, error) { return false, nil }
