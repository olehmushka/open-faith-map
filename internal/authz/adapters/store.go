// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is authz's Postgres store: one indexed join over authz_role_assignments per
// request (D-InProcessAuthz's amendment — no grant cache, grants are read fresh every call) plus the
// instance-admin plane's point lookups. Hand-written pgx, matching this repo's own convention for a
// small, single-purpose query surface (see internal/registration/adapters).
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olehmushka/open-faith-map/internal/authz/domain"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so a Store can be bound either to the pool
// for normal request-scoped Require calls or to a single pgx.Tx for the boot-time admin seed's
// atomic instance-admin grant (internal/identity/bootstrap).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Store struct {
	pool Querier
}

func NewStore(pool Querier) *Store {
	return &Store{pool: pool}
}

// IsActiveInstanceAdmin reports whether personID currently holds an active instance-admin grant.
func (s *Store) IsActiveInstanceAdmin(ctx context.Context, personID string) (bool, error) {
	var isAdmin bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM openfaithmap.authz_instance_admins
			WHERE person_id = $1 AND revoked_at IS NULL
		)`, personID).Scan(&isAdmin)
	return isAdmin, err
}

// HasActiveInstanceAdmin reports whether any active instance admin exists at all — the boot seed's
// idempotency gate.
func (s *Store) HasActiveInstanceAdmin(ctx context.Context) (bool, error) {
	var has bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM openfaithmap.authz_instance_admins WHERE revoked_at IS NULL
		)`).Scan(&has)
	return has, err
}

// InsertInstanceAdmin grants personID the instance-admin plane. grantedBy is empty for bootstrap
// (NULL — D-SeedBootstrap: the first admin has no grantor).
func (s *Store) InsertInstanceAdmin(ctx context.Context, personID, grantedBy string) (string, error) {
	var id string
	var grantedByArg any
	if grantedBy != "" {
		grantedByArg = grantedBy
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_instance_admins (person_id, granted_by)
		VALUES ($1, $2)
		RETURNING id`, personID, grantedByArg).Scan(&id)
	return id, err
}

// ListRoles returns the grantable role catalog — M10.7's super-admin role-grants screen's role
// picker.
func (s *Store) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, code, name, COALESCE(description, ''), is_base
		FROM openfaithmap.authz_roles
		WHERE deleted_at IS NULL
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Role
	for rows.Next() {
		var r domain.Role
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.Description, &r.IsBase); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListRoleAssignmentsByUnit lists unitID's active role assignments, with the subject's display name
// denormalized in — M10.7's super-admin role-grants screen.
func (s *Store) ListRoleAssignmentsByUnit(ctx context.Context, unitID string) ([]domain.RoleAssignment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.subject_person_id, p.display_name, a.role_id, r.code, a.target_unit_id, a.scope, a.granted_at
		FROM openfaithmap.authz_role_assignments a
		JOIN openfaithmap.authz_roles r ON r.id = a.role_id
		JOIN openfaithmap.identity_persons p ON p.id = a.subject_person_id
		WHERE a.target_unit_id = $1 AND a.revoked_at IS NULL
		ORDER BY a.granted_at DESC`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RoleAssignment
	for rows.Next() {
		var a domain.RoleAssignment
		var scope string
		if err := rows.Scan(&a.ID, &a.PersonID, &a.PersonName, &a.RoleID, &a.RoleCode, &a.TargetUnitID, &scope, &a.GrantedAt); err != nil {
			return nil, err
		}
		a.Scope = domain.Scope(scope)
		out = append(out, a)
	}
	return out, rows.Err()
}

// RevokeRoleAssignment soft-revokes assignmentID — sets revoked_at/revoked_by, only if it is
// currently active. Returns domain.ErrAssignmentNotFound if it was already revoked or never existed;
// unlike InsertRoleAssignment's insert-side idempotency, a repeat revoke has no natural "already
// done" success reading.
func (s *Store) RevokeRoleAssignment(ctx context.Context, assignmentID, revokedBy string) error {
	var revokedByArg any
	if revokedBy != "" {
		revokedByArg = revokedBy
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE openfaithmap.authz_role_assignments
		SET revoked_at = now(), revoked_by = $2
		WHERE id = $1 AND revoked_at IS NULL`, assignmentID, revokedByArg)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAssignmentNotFound
	}
	return nil
}

// ListInstanceAdmins returns every active instance-admin grant, with the subject's display name
// denormalized in — M10.7's super-admin people/instance-admins screen.
func (s *Store) ListInstanceAdmins(ctx context.Context) ([]domain.InstanceAdminGrant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.person_id, p.display_name, a.granted_at
		FROM openfaithmap.authz_instance_admins a
		JOIN openfaithmap.identity_persons p ON p.id = a.person_id
		WHERE a.revoked_at IS NULL
		ORDER BY a.granted_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.InstanceAdminGrant
	for rows.Next() {
		var g domain.InstanceAdminGrant
		if err := rows.Scan(&g.ID, &g.PersonID, &g.PersonName, &g.GrantedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// RevokeInstanceAdmin soft-revokes personID's active instance-admin grant, if any. Returns
// domain.ErrInstanceAdminGrantNotFound if personID holds no active grant.
func (s *Store) RevokeInstanceAdmin(ctx context.Context, personID, revokedBy string) error {
	var revokedByArg any
	if revokedBy != "" {
		revokedByArg = revokedBy
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE openfaithmap.authz_instance_admins
		SET revoked_at = now(), revoked_by = $2
		WHERE person_id = $1 AND revoked_at IS NULL`, personID, revokedByArg)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInstanceAdminGrantNotFound
	}
	return nil
}

// ActiveGrantsForSubject fetches every active, unexpired role assignment for personID with its
// role's full permission set, joined with the assignment's graph code when scope='subtree'. Grouped
// in Go (one ActiveGrant per assignment id) since a role carries many permissions.
func (s *Store) ActiveGrantsForSubject(ctx context.Context, personID string) ([]domain.ActiveGrant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.role_id, r.code, a.target_unit_id, a.scope, COALESCE(a.graph_id::text, ''),
		       COALESCE(g.code, ''), rp.permission_code
		FROM openfaithmap.authz_role_assignments a
		JOIN openfaithmap.authz_roles r ON r.id = a.role_id AND r.deleted_at IS NULL
		JOIN openfaithmap.authz_role_permissions rp ON rp.role_id = a.role_id
		LEFT JOIN openfaithmap.directory_graphs g ON g.id = a.graph_id
		WHERE a.subject_person_id = $1
		  AND a.revoked_at IS NULL
		  AND (a.expires_at IS NULL OR a.expires_at > now())
		ORDER BY a.id`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byAssignment := map[string]*domain.ActiveGrant{}
	var order []string
	for rows.Next() {
		var assignmentID, roleID, roleCode, targetUnitID, scope, graphID, graphCode, permCode string
		if err := rows.Scan(&assignmentID, &roleID, &roleCode, &targetUnitID, &scope, &graphID, &graphCode, &permCode); err != nil {
			return nil, err
		}
		g, ok := byAssignment[assignmentID]
		if !ok {
			g = &domain.ActiveGrant{
				AssignmentID: assignmentID,
				RoleID:       roleID,
				RoleCode:     roleCode,
				TargetUnitID: targetUnitID,
				Scope:        domain.Scope(scope),
				GraphID:      graphID,
				GraphCode:    graphCode,
				Perms:        map[domain.Permission]struct{}{},
			}
			byAssignment[assignmentID] = g
			order = append(order, assignmentID)
		}
		g.Perms[domain.Permission(permCode)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]domain.ActiveGrant, 0, len(order))
	for _, id := range order {
		out = append(out, *byAssignment[id])
	}
	return out, nil
}

// InsertRoleAssignment grants personID roleID on targetUnitID with scope "unit" (M10.6's own
// callers — registration's congregation-admin grant, the boot-time first-admin's future role
// grants — never need "subtree"; add a graphID param the day one does). Idempotent: a repeat grant
// identical to an existing active one (the unique index on subject/role/unit/scope/graph WHERE
// revoked_at IS NULL) is treated as success, not an error — matching go-oikumenea's own
// IsAssignmentConflict-as-success behaviour this replaces (registration/application/service.go's
// ensureGrant).
func (s *Store) InsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, grantedBy string) error {
	var grantedByArg any
	if grantedBy != "" {
		grantedByArg = grantedBy
	}
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope, granted_by)
		VALUES ($1, $2, $3, 'unit', $4)
		RETURNING id`, personID, roleID, targetUnitID, grantedByArg).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "authz_role_assignments_active_idx" {
			return nil
		}
		return err
	}
	return nil
}
