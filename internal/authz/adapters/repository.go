// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is authz's Postgres store: one indexed join over authz_role_assignments per
// request (D-InProcessAuthz's amendment — no grant cache, grants are read fresh every call) plus the
// instance-admin plane's point lookups. sqlc-generated (docs/architecture/decisions.md's D-Stack) —
// queries live in queries/authz.sql, generated code in authzsql/. BulkInsertRoleAssignments' own
// pool-Begin/Commit is gone: it moved to the service layer's inTx helper (matching directory's own
// convention), so this Repository accepts any db.DBTX uniformly.
package adapters

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/authz/adapters/authzsql"
	"github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *authzsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: authzsql.New(conn)}
}

func nullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// IsActiveInstanceAdmin reports whether personID currently holds an active instance-admin grant.
func (r *Repository) IsActiveInstanceAdmin(ctx context.Context, personID string) (bool, error) {
	return r.q.IsActiveInstanceAdmin(ctx, personID)
}

// HasActiveInstanceAdmin reports whether any active instance admin exists at all — the boot seed's
// idempotency gate.
func (r *Repository) HasActiveInstanceAdmin(ctx context.Context) (bool, error) {
	return r.q.HasActiveInstanceAdmin(ctx)
}

// InsertInstanceAdmin grants personID the instance-admin plane. grantedBy is empty for bootstrap
// (NULL — D-SeedBootstrap: the first admin has no grantor).
func (r *Repository) InsertInstanceAdmin(ctx context.Context, personID, grantedBy string) (string, error) {
	return r.q.InsertInstanceAdmin(ctx, authzsql.InsertInstanceAdminParams{PersonID: personID, GrantedBy: nullableText(grantedBy)})
}

func toRole(row authzsql.ListRolesRow) domain.Role {
	return domain.Role{ID: row.ID, Code: row.Code, Name: row.Name, Description: row.Description, IsBase: row.IsBase}
}

// ListRoles returns the grantable role catalog — M10.7's super-admin role-grants screen's role
// picker.
func (r *Repository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.q.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRole(row))
	}
	return out, nil
}

// GetRoleByCode resolves a role by its stable code (e.g. "platform-moderator") — used by
// internal/platform/seed.Resolve at boot time instead of a hardcoded UUID constant.
func (r *Repository) GetRoleByCode(ctx context.Context, code string) (domain.Role, error) {
	row, err := r.q.GetRoleByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Role{}, domain.ErrRoleNotFound
	}
	if err != nil {
		return domain.Role{}, err
	}
	return domain.Role{ID: row.ID, Code: row.Code, Name: row.Name, Description: row.Description, IsBase: row.IsBase}, nil
}

// ListRoleAssignmentsByUnit lists unitID's active role assignments, with the subject's display name
// denormalized in — M10.7's super-admin role-grants screen.
func (r *Repository) ListRoleAssignmentsByUnit(ctx context.Context, unitID string) ([]domain.RoleAssignment, error) {
	rows, err := r.q.ListRoleAssignmentsByUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.RoleAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RoleAssignment{
			ID: row.ID, PersonID: row.SubjectPersonID, PersonName: row.DisplayName, RoleID: row.RoleID,
			RoleCode: row.RoleCode, TargetUnitID: row.TargetUnitID, Scope: domain.Scope(row.Scope), GrantedAt: row.GrantedAt,
			ExpiresAt: db.NullableTime(row.ExpiresAt),
		})
	}
	return out, nil
}

// ListRoleAssignmentsByPerson lists personID's own active role assignments, across every unit — M11.5's
// self-service profile page. Mirrors ListRoleAssignmentsByUnit's query shape exactly, filtered on
// subject_person_id instead of target_unit_id.
func (r *Repository) ListRoleAssignmentsByPerson(ctx context.Context, personID string) ([]domain.RoleAssignment, error) {
	rows, err := r.q.ListRoleAssignmentsByPerson(ctx, personID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.RoleAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RoleAssignment{
			ID: row.ID, PersonID: row.SubjectPersonID, PersonName: row.DisplayName, RoleID: row.RoleID,
			RoleCode: row.RoleCode, TargetUnitID: row.TargetUnitID, Scope: domain.Scope(row.Scope), GrantedAt: row.GrantedAt,
			ExpiresAt: db.NullableTime(row.ExpiresAt),
		})
	}
	return out, nil
}

// RevokeRoleAssignment soft-revokes assignmentID — sets revoked_at/revoked_by, only if it is
// currently active. Returns domain.ErrAssignmentNotFound if it was already revoked or never existed.
func (r *Repository) RevokeRoleAssignment(ctx context.Context, assignmentID, revokedBy string) (domain.RevokedRoleAssignment, error) {
	row, err := r.q.RevokeRoleAssignment(ctx, authzsql.RevokeRoleAssignmentParams{ID: assignmentID, RevokedBy: nullableText(revokedBy)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RevokedRoleAssignment{}, domain.ErrAssignmentNotFound
	}
	if err != nil {
		return domain.RevokedRoleAssignment{}, err
	}
	return domain.RevokedRoleAssignment{ID: row.ID, PersonID: row.SubjectPersonID, RoleID: row.RoleID, TargetUnitID: row.TargetUnitID, Scope: domain.Scope(row.Scope)}, nil
}

// ClearRoleAssignmentExpiry clears assignmentID's expires_at, leaving the grant itself untouched —
// M12.3. Returns domain.ErrAssignmentNotFound if it was already revoked or never existed, mirroring
// RevokeRoleAssignment's own not-found handling.
func (r *Repository) ClearRoleAssignmentExpiry(ctx context.Context, assignmentID string) (domain.RevokedRoleAssignment, error) {
	row, err := r.q.ClearRoleAssignmentExpiry(ctx, assignmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RevokedRoleAssignment{}, domain.ErrAssignmentNotFound
	}
	if err != nil {
		return domain.RevokedRoleAssignment{}, err
	}
	return domain.RevokedRoleAssignment{ID: row.ID, PersonID: row.SubjectPersonID, RoleID: row.RoleID, TargetUnitID: row.TargetUnitID, Scope: domain.Scope(row.Scope)}, nil
}

// ListInstanceAdmins returns every active instance-admin grant, with the subject's display name
// denormalized in — M10.7's super-admin people/instance-admins screen.
func (r *Repository) ListInstanceAdmins(ctx context.Context) ([]domain.InstanceAdminGrant, error) {
	rows, err := r.q.ListInstanceAdmins(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.InstanceAdminGrant, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.InstanceAdminGrant{ID: row.ID, PersonID: row.PersonID, PersonName: row.DisplayName, GrantedAt: row.GrantedAt})
	}
	return out, nil
}

// RevokeInstanceAdmin soft-revokes personID's active instance-admin grant, if any. Returns
// domain.ErrInstanceAdminGrantNotFound if personID holds no active grant.
func (r *Repository) RevokeInstanceAdmin(ctx context.Context, personID, revokedBy string) (domain.RevokedInstanceAdminGrant, error) {
	row, err := r.q.RevokeInstanceAdmin(ctx, authzsql.RevokeInstanceAdminParams{PersonID: personID, RevokedBy: nullableText(revokedBy)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RevokedInstanceAdminGrant{}, domain.ErrInstanceAdminGrantNotFound
	}
	if err != nil {
		return domain.RevokedInstanceAdminGrant{}, err
	}
	return domain.RevokedInstanceAdminGrant{ID: row.ID, PersonID: row.PersonID}, nil
}

// ActiveGrantsForSubject fetches every active, unexpired role assignment for personID with its
// role's full permission set, joined with the assignment's graph code when scope='subtree'. Grouped
// in Go (one ActiveGrant per assignment id) since a role carries many permissions.
func (r *Repository) ActiveGrantsForSubject(ctx context.Context, personID string) ([]domain.ActiveGrant, error) {
	rows, err := r.q.ActiveGrantsForSubject(ctx, personID)
	if err != nil {
		return nil, err
	}

	byAssignment := map[string]*domain.ActiveGrant{}
	var order []string
	for _, row := range rows {
		graphID, _ := row.GraphID.(string)
		g, ok := byAssignment[row.ID]
		if !ok {
			g = &domain.ActiveGrant{
				AssignmentID: row.ID,
				RoleID:       row.RoleID,
				RoleCode:     row.RoleCode,
				TargetUnitID: row.TargetUnitID,
				Scope:        domain.Scope(row.Scope),
				GraphID:      graphID,
				GraphCode:    row.GraphCode,
				Perms:        map[domain.Permission]struct{}{},
			}
			byAssignment[row.ID] = g
			order = append(order, row.ID)
		}
		g.Perms[domain.Permission(row.PermissionCode)] = struct{}{}
	}

	out := make([]domain.ActiveGrant, 0, len(order))
	for _, id := range order {
		out = append(out, *byAssignment[id])
	}
	return out, nil
}

// InsertRoleAssignment grants personID roleID on targetUnitID at scope ("unit" or "subtree"),
// graphID required (and only meaningful) when scope is "subtree" — empty/"" for "unit". Idempotent:
// a repeat grant identical to an existing active one (the unique index on
// subject/role/unit/scope/graph WHERE revoked_at IS NULL) is treated as success, not an error.
// Returns the assignment's id either way (the audit log needs a real target_id even on the
// idempotent-conflict path, so that path looks the existing row's id up rather than returning empty).
func (r *Repository) InsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, scope, graphID, grantedBy string, expiresAt *time.Time) (string, error) {
	graphIDArg := nullableText(graphID)
	id, err := r.q.InsertRoleAssignment(ctx, authzsql.InsertRoleAssignmentParams{
		SubjectPersonID: personID, RoleID: roleID, TargetUnitID: targetUnitID, Scope: scope,
		GraphID: graphIDArg, GrantedBy: nullableText(grantedBy), ExpiresAt: db.NullableTimeArg(expiresAt),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "authz_role_assignments_active_idx" {
			return r.q.GetActiveRoleAssignmentID(ctx, authzsql.GetActiveRoleAssignmentIDParams{
				SubjectPersonID: personID, RoleID: roleID, TargetUnitID: targetUnitID, Scope: scope, GraphID: graphIDArg,
			})
		}
		return "", err
	}
	return id, nil
}

// UpsertRoleAssignment is BulkInsertRoleAssignments' per-row statement (see
// authz/application/service.go's inTx-wrapped loop) — a real upsert rather than
// InsertRoleAssignment's catch-then-select, since a caught 23505 inside an explicit multi-statement
// tx would abort the whole transaction.
func (r *Repository) UpsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, scope, graphID, grantedBy string, expiresAt *time.Time) (string, error) {
	return r.q.UpsertRoleAssignment(ctx, authzsql.UpsertRoleAssignmentParams{
		SubjectPersonID: personID, RoleID: roleID, TargetUnitID: targetUnitID, Scope: scope,
		GraphID: nullableText(graphID), GrantedBy: nullableText(grantedBy), ExpiresAt: db.NullableTimeArg(expiresAt),
	})
}

// CountRepointableRoleAssignments previews RepointRoleAssignments' own two-statement effect
// (M11.8) without mutating anything.
func (r *Repository) CountRepointableRoleAssignments(ctx context.Context, duplicateID, survivorID string) (toMove, toRevoke int, err error) {
	row, err := r.q.CountRepointableRoleAssignments(ctx, authzsql.CountRepointableRoleAssignmentsParams{SurvivorID: survivorID, DuplicateID: duplicateID})
	if err != nil {
		return 0, 0, err
	}
	return int(row.ToMove), int(row.ToRevoke), nil
}

// RepointRoleAssignments moves every one of duplicateID's active role assignments onto survivorID
// (M11.8's MergePersons). Must run inside the same tx as the caller's other MergePersons store
// calls (core/application.Service.MergePersons), so this method takes no Begin/Commit itself.
func (r *Repository) RepointRoleAssignments(ctx context.Context, duplicateID, survivorID, actorID string) (movedIDs, revokedIDs []string, err error) {
	movedIDs, err = r.q.RepointMoveRoleAssignments(ctx, authzsql.RepointMoveRoleAssignmentsParams{SurvivorID: survivorID, DuplicateID: duplicateID})
	if err != nil {
		return nil, nil, err
	}
	revokedIDs, err = r.q.RepointRevokeRoleAssignments(ctx, authzsql.RepointRevokeRoleAssignmentsParams{DuplicateID: duplicateID, RevokedBy: nullableText(actorID)})
	if err != nil {
		return nil, nil, err
	}
	return movedIDs, revokedIDs, nil
}

// PreviewRepointInstanceAdmin previews RepointInstanceAdmin's effect (M11.8) without mutating
// anything.
func (r *Repository) PreviewRepointInstanceAdmin(ctx context.Context, duplicateID, survivorID string) (willMove, willRevoke bool, err error) {
	duplicateIsAdmin, err := r.q.IsActiveInstanceAdmin(ctx, duplicateID)
	if err != nil {
		return false, false, err
	}
	if !duplicateIsAdmin {
		return false, false, nil
	}
	survivorIsAdmin, err := r.q.IsActiveInstanceAdmin(ctx, survivorID)
	if err != nil {
		return false, false, err
	}
	return !survivorIsAdmin, survivorIsAdmin, nil
}

// RepointInstanceAdmin moves duplicateID's active instance-admin grant (if any) onto survivorID —
// same repoint-or-revoke-redundant shape as RepointRoleAssignments.
func (r *Repository) RepointInstanceAdmin(ctx context.Context, duplicateID, survivorID, actorID string) (moved, revoked bool, err error) {
	movedID, err := r.q.RepointMoveInstanceAdmin(ctx, authzsql.RepointMoveInstanceAdminParams{SurvivorID: survivorID, DuplicateID: duplicateID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, false, err
	}
	moved = movedID != ""

	revokedID, err := r.q.RepointRevokeInstanceAdmin(ctx, authzsql.RepointRevokeInstanceAdminParams{DuplicateID: duplicateID, RevokedBy: nullableText(actorID)})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, false, err
	}
	revoked = revokedID != ""
	return moved, revoked, nil
}
