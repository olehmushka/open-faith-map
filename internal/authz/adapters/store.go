// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is authz's Postgres store: one indexed join over authz_role_assignments per
// request (D-InProcessAuthz's amendment — no grant cache, grants are read fresh every call) plus the
// instance-admin plane's point lookups. Hand-written pgx, matching this repo's own convention for a
// small, single-purpose query surface (see internal/registration/adapters).
package adapters

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz/domain"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so a Store can be bound either to the pool
// for normal request-scoped Require calls or to a single pgx.Tx for the boot-time admin seed's
// atomic instance-admin grant (internal/identity/bootstrap).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
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

// ClosureStore implements domain.ClosurePort against directory's tables directly — a straight SQL
// join, no Go dependency on internal/directory (D-InProcessAuthz amendment #4: internal/authz
// imports no other module). Live until internal/directory registers its own adapter at M10.4; this
// type is a placeholder callers may swap for one directory provides once that module exists.
type ClosureStore struct {
	pool *pgxpool.Pool
}

func NewClosureStore(pool *pgxpool.Pool) *ClosureStore {
	return &ClosureStore{pool: pool}
}

func (c *ClosureStore) IsAncestorOrSelf(ctx context.Context, graphID, ancestorUnitID, descendantUnitID string) (bool, error) {
	var reachable bool
	err := c.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM openfaithmap.directory_unit_closure
			WHERE graph_id = $1 AND ancestor_id = $2 AND descendant_id = $3
		)`, graphID, ancestorUnitID, descendantUnitID).Scan(&reachable)
	return reachable, err
}

func (c *ClosureStore) IsAuthorityBearing(ctx context.Context, graphID string) (bool, error) {
	var bearing bool
	err := c.pool.QueryRow(ctx, `
		SELECT is_authority_bearing FROM openfaithmap.directory_graphs WHERE id = $1
	`, graphID).Scan(&bearing)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return bearing, err
}
