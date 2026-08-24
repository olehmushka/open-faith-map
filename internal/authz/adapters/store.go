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
	"github.com/jackc/pgx/v5/pgxpool"
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

// ListRoleAssignmentsByPerson lists personID's own active role assignments, across every unit — M11.5's
// self-service profile page. Mirrors ListRoleAssignmentsByUnit's query shape exactly, filtered on
// subject_person_id instead of target_unit_id; PersonName is always the caller's own name here, but
// scanned via the same domain.RoleAssignment shape rather than a narrower one, so both listings stay
// interchangeable for any future shared rendering.
func (s *Store) ListRoleAssignmentsByPerson(ctx context.Context, personID string) ([]domain.RoleAssignment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.subject_person_id, p.display_name, a.role_id, r.code, a.target_unit_id, a.scope, a.granted_at
		FROM openfaithmap.authz_role_assignments a
		JOIN openfaithmap.authz_roles r ON r.id = a.role_id
		JOIN openfaithmap.identity_persons p ON p.id = a.subject_person_id
		WHERE a.subject_person_id = $1 AND a.revoked_at IS NULL
		ORDER BY a.granted_at DESC`, personID)
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
// done" success reading. RETURNING the identity columns (M11.2) gives the caller a real "before"
// snapshot for the audit log with no second read — the values are unchanged by this UPDATE, only
// revoked_at/revoked_by are set.
func (s *Store) RevokeRoleAssignment(ctx context.Context, assignmentID, revokedBy string) (domain.RevokedRoleAssignment, error) {
	var revokedByArg any
	if revokedBy != "" {
		revokedByArg = revokedBy
	}
	var out domain.RevokedRoleAssignment
	var scope string
	err := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.authz_role_assignments
		SET revoked_at = now(), revoked_by = $2
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING id, subject_person_id, role_id, target_unit_id, scope`, assignmentID, revokedByArg,
	).Scan(&out.ID, &out.PersonID, &out.RoleID, &out.TargetUnitID, &scope)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RevokedRoleAssignment{}, domain.ErrAssignmentNotFound
		}
		return domain.RevokedRoleAssignment{}, err
	}
	out.Scope = domain.Scope(scope)
	return out, nil
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
// domain.ErrInstanceAdminGrantNotFound if personID holds no active grant. RETURNING the grant's own
// id (M11.2) gives the audit log the same target_id GrantInstanceAdmin already uses, rather than
// personID (which is also available, but the grant id is what identifies this specific plane
// membership, same as RevokeRoleAssignment returning the assignment's own id).
func (s *Store) RevokeInstanceAdmin(ctx context.Context, personID, revokedBy string) (domain.RevokedInstanceAdminGrant, error) {
	var revokedByArg any
	if revokedBy != "" {
		revokedByArg = revokedBy
	}
	var out domain.RevokedInstanceAdminGrant
	err := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.authz_instance_admins
		SET revoked_at = now(), revoked_by = $2
		WHERE person_id = $1 AND revoked_at IS NULL
		RETURNING id, person_id`, personID, revokedByArg,
	).Scan(&out.ID, &out.PersonID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RevokedInstanceAdminGrant{}, domain.ErrInstanceAdminGrantNotFound
		}
		return domain.RevokedInstanceAdminGrant{}, err
	}
	return out, nil
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

// InsertRoleAssignment grants personID roleID on targetUnitID at scope ("unit" or "subtree"),
// graphID required (and only meaningful) when scope is "subtree" — empty/"" for "unit"
// (M12.2: real subtree-grant provisioning, resolving U14; every caller before M12.2 only ever needed
// "unit", so this used to hardcode it). Idempotent: a repeat grant identical to an existing active
// one (the unique index on subject/role/unit/scope/graph WHERE revoked_at IS NULL) is treated as
// success, not an error — matching go-oikumenea's own IsAssignmentConflict-as-success behaviour this
// replaces (registration/application/service.go's ensureGrant). Returns the assignment's id either
// way (M11.2: the audit log needs a real target_id even on the idempotent-conflict path, so that path
// looks the existing row's id up rather than returning empty).
func (s *Store) InsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, scope, graphID, grantedBy string) (string, error) {
	var grantedByArg any
	if grantedBy != "" {
		grantedByArg = grantedBy
	}
	graphIDArg := nullableText(graphID)
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope, graph_id, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`, personID, roleID, targetUnitID, scope, graphIDArg, grantedByArg).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "authz_role_assignments_active_idx" {
			err := s.pool.QueryRow(ctx, `
				SELECT id FROM openfaithmap.authz_role_assignments
				WHERE subject_person_id = $1 AND role_id = $2 AND target_unit_id = $3 AND scope = $4
				  AND graph_id IS NOT DISTINCT FROM $5 AND revoked_at IS NULL`,
				personID, roleID, targetUnitID, scope, graphIDArg).Scan(&id)
			return id, err
		}
		return "", err
	}
	return id, nil
}

// BulkInsertRoleAssignments grants roleID on targetUnitID to every personID in one batch, all inside
// a single transaction (M11.7's own explicit ask). Deliberately does NOT reuse InsertRoleAssignment's
// catch-23505-then-SELECT body: once multiple statements share one explicit pgx.Tx, a caught 23505
// aborts the whole transaction (Postgres error 25P02, "current transaction is aborted") — every
// subsequent statement, including the rest of the batch, would fail until rollback. Uses a real
// upsert instead, so the idempotent-conflict path never raises an error at all. The ON CONFLICT
// target must match authz_role_assignments_active_idx's own partial-index predicate exactly
// (migrations/0009_core_authz.sql) — that index is a bare CREATE UNIQUE INDEX, not a named
// constraint, so ON CONFLICT ON CONSTRAINT can't resolve it. DO UPDATE (not DO NOTHING) is required
// so RETURNING id still fires on the conflict branch. Requires a pool-bound Store, same as
// InsertPersonAccountInvite (internal/identity/adapters/store.go).
func (s *Store) BulkInsertRoleAssignments(ctx context.Context, personIDs []string, roleID, targetUnitID, scope, graphID, grantedBy string) ([]string, error) {
	pool, ok := s.pool.(*pgxpool.Pool)
	if !ok {
		return nil, errors.New("BulkInsertRoleAssignments requires a pool-bound Store")
	}
	var grantedByArg any
	if grantedBy != "" {
		grantedByArg = grantedBy
	}
	graphIDArg := nullableText(graphID)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ids := make([]string, 0, len(personIDs))
	for _, personID := range personIDs {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope, graph_id, granted_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (subject_person_id, role_id, target_unit_id, scope, graph_id) WHERE revoked_at IS NULL
			DO UPDATE SET updated_at = now()
			RETURNING id`, personID, roleID, targetUnitID, scope, graphIDArg, grantedByArg).Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ids, nil
}

// CountRepointableRoleAssignments previews RepointRoleAssignments' own two-statement effect
// (M11.8) without mutating anything: how many of duplicateID's active role assignments would move
// onto survivorID untouched, versus how many would instead be revoked as redundant because
// survivorID already holds an identical active grant (same role/unit/scope/graph). Mirrors
// RepointRoleAssignments' NOT EXISTS predicate exactly so the preview and the real mutation can
// never disagree.
func (s *Store) CountRepointableRoleAssignments(ctx context.Context, duplicateID, survivorID string) (toMove, toRevoke int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE NOT EXISTS (
				SELECT 1 FROM openfaithmap.authz_role_assignments s
				WHERE s.subject_person_id = $2 AND s.role_id = ra.role_id AND s.target_unit_id = ra.target_unit_id
				  AND s.scope = ra.scope AND s.graph_id IS NOT DISTINCT FROM ra.graph_id AND s.revoked_at IS NULL
			)),
			count(*) FILTER (WHERE EXISTS (
				SELECT 1 FROM openfaithmap.authz_role_assignments s
				WHERE s.subject_person_id = $2 AND s.role_id = ra.role_id AND s.target_unit_id = ra.target_unit_id
				  AND s.scope = ra.scope AND s.graph_id IS NOT DISTINCT FROM ra.graph_id AND s.revoked_at IS NULL
			))
		FROM openfaithmap.authz_role_assignments ra
		WHERE ra.subject_person_id = $1 AND ra.revoked_at IS NULL`, duplicateID, survivorID,
	).Scan(&toMove, &toRevoke)
	return toMove, toRevoke, err
}

// RepointRoleAssignments moves every one of duplicateID's active role assignments onto survivorID
// (M11.8's MergePersons). A set-based two-statement pattern rather than BulkInsertRoleAssignments'
// per-row loop: the input here is "every active row belonging to one person," which SQL already
// expresses as one predicate. The first UPDATE repoints whatever doesn't collide with a grant
// survivorID already holds; the second revokes whatever is still left under duplicateID afterward
// (i.e. exactly the rows that collided) as redundant, not deleted — matching this codebase's
// existing revoke-not-delete convention (RevokeRoleAssignment). Must run inside the same tx as the
// caller's other MergePersons store calls (core/application.Service.MergePersons), so this method
// takes no Begin/Commit itself and works against whatever s.pool is bound to.
func (s *Store) RepointRoleAssignments(ctx context.Context, duplicateID, survivorID, actorID string) (movedIDs, revokedIDs []string, err error) {
	moveRows, err := s.pool.Query(ctx, `
		UPDATE openfaithmap.authz_role_assignments ra
		SET subject_person_id = $2, updated_at = now()
		WHERE ra.subject_person_id = $1 AND ra.revoked_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM openfaithmap.authz_role_assignments s
		    WHERE s.subject_person_id = $2 AND s.role_id = ra.role_id AND s.target_unit_id = ra.target_unit_id
		      AND s.scope = ra.scope AND s.graph_id IS NOT DISTINCT FROM ra.graph_id AND s.revoked_at IS NULL
		  )
		RETURNING ra.id`, duplicateID, survivorID)
	if err != nil {
		return nil, nil, err
	}
	movedIDs, err = pgx.CollectRows(moveRows, pgx.RowTo[string])
	if err != nil {
		return nil, nil, err
	}

	var revokedByArg any
	if actorID != "" {
		revokedByArg = actorID
	}
	revokeRows, err := s.pool.Query(ctx, `
		UPDATE openfaithmap.authz_role_assignments
		SET revoked_at = now(), revoked_by = $2
		WHERE subject_person_id = $1 AND revoked_at IS NULL
		RETURNING id`, duplicateID, revokedByArg)
	if err != nil {
		return nil, nil, err
	}
	revokedIDs, err = pgx.CollectRows(revokeRows, pgx.RowTo[string])
	if err != nil {
		return nil, nil, err
	}
	return movedIDs, revokedIDs, nil
}

// PreviewRepointInstanceAdmin previews RepointInstanceAdmin's effect (M11.8) without mutating
// anything: whether duplicateID's active instance-admin grant (if any) would move onto survivorID,
// or would instead be revoked as redundant because survivorID already holds one.
func (s *Store) PreviewRepointInstanceAdmin(ctx context.Context, duplicateID, survivorID string) (willMove, willRevoke bool, err error) {
	var duplicateIsAdmin, survivorIsAdmin bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM openfaithmap.authz_instance_admins WHERE person_id = $1 AND revoked_at IS NULL)`,
		duplicateID).Scan(&duplicateIsAdmin); err != nil {
		return false, false, err
	}
	if !duplicateIsAdmin {
		return false, false, nil
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM openfaithmap.authz_instance_admins WHERE person_id = $1 AND revoked_at IS NULL)`,
		survivorID).Scan(&survivorIsAdmin); err != nil {
		return false, false, err
	}
	return !survivorIsAdmin, survivorIsAdmin, nil
}

// RepointInstanceAdmin moves duplicateID's active instance-admin grant (if any) onto survivorID —
// same repoint-or-revoke-redundant shape as RepointRoleAssignments, simplified to the plane's own
// single-key uniqueness (authz_instance_admins_person_active_idx has no compound key to match).
func (s *Store) RepointInstanceAdmin(ctx context.Context, duplicateID, survivorID, actorID string) (moved, revoked bool, err error) {
	var movedID string
	err = s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.authz_instance_admins a
		SET person_id = $2, updated_at = now()
		WHERE a.person_id = $1 AND a.revoked_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM openfaithmap.authz_instance_admins s WHERE s.person_id = $2 AND s.revoked_at IS NULL
		  )
		RETURNING a.id`, duplicateID, survivorID).Scan(&movedID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, false, err
	}
	moved = movedID != ""

	var revokedByArg any
	if actorID != "" {
		revokedByArg = actorID
	}
	var revokedID string
	err = s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.authz_instance_admins
		SET revoked_at = now(), revoked_by = $2
		WHERE person_id = $1 AND revoked_at IS NULL
		RETURNING id`, duplicateID, revokedByArg).Scan(&revokedID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, false, err
	}
	revoked = revokedID != ""
	return moved, revoked, nil
}

// nullableText converts an empty string (this package's own "no value" convention for an optional
// text column) to nil, so InsertRoleAssignment/BulkInsertRoleAssignments write a real SQL NULL for
// graph_id when scope is "unit" rather than an empty-string literal.
func nullableText(s string) any {
	if s == "" {
		return nil
	}
	return s
}
