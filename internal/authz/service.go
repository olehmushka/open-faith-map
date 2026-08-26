// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz/adapters"
	"github.com/olehmushka/open-faith-map/internal/authz/domain"
)

// GrantStore fetches a subject's authority state, freshly, per call (D-InProcessAuthz's amendment:
// no grant cache — a stale ALLOW with no RLS backstop underneath it is the security bug this avoids)
// — and, since M10.6, writes new unit-scoped grants (the module's own tables, so it owns the write).
type GrantStore interface {
	IsActiveInstanceAdmin(ctx context.Context, personID string) (bool, error)
	ActiveGrantsForSubject(ctx context.Context, personID string) ([]domain.ActiveGrant, error)
	// InsertRoleAssignment's scope is "unit" or "subtree" (domain.Scope); graphID is required (and
	// only meaningful) when scope is "subtree" (M12.2, resolving U14 — see GrantUnitRole's own doc).
	// expiresAt is nil for a non-expiring grant (M12.3) — the PDP already enforces it, this call is
	// what finally lets a caller set it.
	InsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, scope, graphID, grantedBy string, expiresAt *time.Time) (string, error)
	// UpsertRoleAssignment is BulkGrantUnitRole's (M11.7) per-row statement, called in a loop inside
	// Service.inTx — a real upsert, not InsertRoleAssignment's catch-then-select, since a caught
	// 23505 inside an explicit multi-statement tx would abort the whole transaction. See the
	// adapter's own doc comment.
	UpsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, scope, graphID, grantedBy string, expiresAt *time.Time) (string, error)

	// The M10.7 super-admin surface: the role catalog, per-unit assignment listing/revocation, and
	// the instance-admin plane's own list/grant/revoke — all new at M10.7 (InsertInstanceAdmin
	// already existed, for the boot seed; nothing else on the instance-admin plane had a Service
	// wrapper until now). RevokeRoleAssignment/RevokeInstanceAdmin return the revoked row's identity
	// (M11.2) so the audit-log helper has a real "before" snapshot with no second read.
	ListRoles(ctx context.Context) ([]domain.Role, error)
	ListRoleAssignmentsByUnit(ctx context.Context, unitID string) ([]domain.RoleAssignment, error)
	// ListRoleAssignmentsByPerson is M11.5's self-service read: the caller's own active role
	// assignments across every unit, personID always the resolved subject's — never a request param.
	ListRoleAssignmentsByPerson(ctx context.Context, personID string) ([]domain.RoleAssignment, error)
	RevokeRoleAssignment(ctx context.Context, assignmentID, revokedBy string) (domain.RevokedRoleAssignment, error)
	// ClearRoleAssignmentExpiry clears an active assignment's expiresAt, leaving the grant itself
	// untouched — M12.3.
	ClearRoleAssignmentExpiry(ctx context.Context, assignmentID string) (domain.RevokedRoleAssignment, error)
	ListInstanceAdmins(ctx context.Context) ([]domain.InstanceAdminGrant, error)
	InsertInstanceAdmin(ctx context.Context, personID, grantedBy string) (string, error)
	RevokeInstanceAdmin(ctx context.Context, personID, revokedBy string) (domain.RevokedInstanceAdminGrant, error)
}

// Service is the module's composition: the pure PDP engine plus the store that fetches the authority
// state it decides over.
type Service struct {
	pdp   domain.PDP
	store GrantStore
	pool  *pgxpool.Pool
}

func NewService(pdp domain.PDP, store GrantStore, pool *pgxpool.Pool) *Service {
	return &Service{pdp: pdp, store: store, pool: pool}
}

// inTx runs fn against a Repository bound to a fresh transaction — BulkGrantUnitRole's own need
// (M11.7 + M12.2's scope param), the one write in this module that spans multiple statements.
// Mirrors directory/application/service.go's own inTx helper.
func (s *Service) inTx(ctx context.Context, fn func(store GrantStore) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(adapters.NewRepository(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Require answers "may the request's subject perform action on unitID", subject resolved from ctx —
// never a parameter, so this can't become an oracle over an arbitrary subject by a call-site mistake
// (the same defect class this repo already fixed at M2.3 and M3). Panics if ctx carries a
// SystemContext marker: a background/system context must never reach a request-scoped authorization
// check, and a stripped-but-somehow-present marker here means a bug in this package's own callers,
// not a normal deny.
func (s *Service) Require(ctx context.Context, action domain.Permission, unitID string) error {
	if isSystemContext(ctx) {
		panic("authz: Require called with a SystemContext — system contexts must never reach a request-scoped authorization check")
	}
	subject, ok := SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return domain.ErrPermissionDenied
	}
	// M11.9: an API-key-authenticated subject may exercise at most its key's stored allowlist, on top
	// of (never instead of) whatever its owning person actually holds — this is the "allowlist ∩ live
	// grants" intersection's first half, checked as a cheap short-circuit before the second half (the
	// unmodified enforce/decide path below, which already answers "does PersonID currently hold
	// action"). subject.APIKeyPermissionCodes == nil means this is an ordinary session-based request,
	// not narrowed at all.
	if subject.APIKeyPermissionCodes != nil && !slices.Contains(subject.APIKeyPermissionCodes, string(action)) {
		return domain.ErrPermissionDenied
	}
	return s.enforce(ctx, subject.PersonID, action, unitID)
}

// DecideFor answers the same question for an arbitrary subjectPersonID rather than ctx's own
// subject. Reserved for the one super-admin "what can this person do" screen (M10.8); callers must
// gate access to this method itself on the instance-admin plane (Require the caller holds
// instance.admin.manage, or an equivalent instance-scope permission, before calling DecideFor) — it
// is not itself an authorization check.
func (s *Service) DecideFor(ctx context.Context, subjectPersonID string, action domain.Permission, unitID string) error {
	return s.enforce(ctx, subjectPersonID, action, unitID)
}

// ExplainDecision answers the same question as DecideFor, but with the PDP's full explain trace
// populated (Allow, Via, DenyReason) instead of a bare error. Reserved for the super-admin "why
// does this user have this access" debugging screen (M12.4); same caller-gating requirement as
// DecideFor: callers must gate access to this method itself on the instance-admin plane before
// calling it — it is not itself an authorization check.
func (s *Service) ExplainDecision(ctx context.Context, subjectPersonID string, action domain.Permission, unitID string) (domain.Decision, error) {
	return s.decide(ctx, subjectPersonID, action, unitID, true)
}

func (s *Service) enforce(ctx context.Context, subjectPersonID string, action domain.Permission, unitID string) error {
	d, err := s.decide(ctx, subjectPersonID, action, unitID, false)
	if err != nil {
		return err
	}
	if !d.Allow {
		return domain.ErrPermissionDenied
	}
	return nil
}

// GrantUnitRole grants personID roleID on unitID at scope (domain.ScopeUnit or domain.ScopeSubtree),
// graphID required (and only meaningful) when scope is ScopeSubtree — M10.6's registration cutover
// was the first caller (approval-time congregation-admin grant, always ScopeUnit); M12.2 adds real
// scope=subtree provisioning, resolving U14 (docs/milestones-2026-08-07-2026-08-26.md): before this, every real/test grant
// was hardcoded to scope='unit', so D-UnitMoveDualScope's dual-parent unit.edges.manage check could
// never pass for a non-root move — subtree was fully implemented in the PDP but unprovisionable
// through any surface. No epoch bump, no cache to invalidate (D-InProcessAuthz's amendment: grants
// are read fresh per request), so a grant is visible to the very next Require call with no extra
// step. Returns the assignment's id (M11.2: super-admin callers use it as the audit log's target_id).
// expiresAt is nil for a non-expiring grant (M12.3).
func (s *Service) GrantUnitRole(ctx context.Context, personID, roleID, unitID string, scope domain.Scope, graphID, grantedByPersonID string, expiresAt *time.Time) (string, error) {
	return s.store.InsertRoleAssignment(ctx, personID, roleID, unitID, string(scope), graphID, grantedByPersonID, expiresAt)
}

// BulkGrantUnitRole grants roleID on unitID to every personID in one batch, atomically, at scope
// (M11.7 + M12.2's scope param), all with the same expiresAt (M12.3, nil for non-expiring). Returns
// the resulting assignment ids in the same order as personIDs, so callers can pair each id back to
// the person it belongs to (e.g. for per-row audit logging) with no extra lookup. No dedup of
// personIDs: the store's upsert already absorbs a duplicate id harmlessly, and a same-order,
// same-length 1:1 pairing with the input is simpler to reason about than reordering logic.
func (s *Service) BulkGrantUnitRole(ctx context.Context, personIDs []string, roleID, unitID string, scope domain.Scope, graphID, grantedByPersonID string, expiresAt *time.Time) ([]string, error) {
	if len(personIDs) == 0 {
		return nil, domain.ErrEmptyPersonIDs
	}
	var ids []string
	err := s.inTx(ctx, func(store GrantStore) error {
		ids = make([]string, 0, len(personIDs))
		for _, personID := range personIDs {
			id, err := store.UpsertRoleAssignment(ctx, personID, roleID, unitID, string(scope), graphID, grantedByPersonID, expiresAt)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// RequireInstanceAdmin is the shared, hard-to-misuse enforcer for the instance-admin plane
// (D-SuperAdminFold's amendment: "one shared, hard-to-misuse enforcer... not a sixth hand-copied
// require*"). Mirrors Require's shape (subject from ctx, panics on SystemContext) but checks
// IsActiveInstanceAdmin directly — there is no unit dimension to the instance-admin plane. Wired as
// route-group middleware over CoreSuperAdminService (internal/authz/transport), not called
// per-handler, so no future super-admin endpoint can be added without inheriting the check.
func (s *Service) RequireInstanceAdmin(ctx context.Context) error {
	if isSystemContext(ctx) {
		panic("authz: RequireInstanceAdmin called with a SystemContext — system contexts must never reach a request-scoped authorization check")
	}
	subject, ok := SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return domain.ErrPermissionDenied
	}
	// M11.9: an API-key subject is denied outright, unconditionally — never checked against
	// IsActiveInstanceAdmin at all, even when the key's owner genuinely is an instance admin. The
	// instance-admin plane is PDP.Decide's "allow everything" bypass; letting a key ride it would make
	// its allowlist meaningless the moment the owner is later granted instance-admin, with no
	// re-issuance and no visible change to the key itself. CreateApiKey correspondingly refuses to
	// let a key's allowlist contain any instance-scope permission code, since it could never actually
	// be exercised through this hard deny.
	if subject.APIKeyPermissionCodes != nil {
		return domain.ErrPermissionDenied
	}
	isAdmin, err := s.store.IsActiveInstanceAdmin(ctx, subject.PersonID)
	if err != nil {
		return fmt.Errorf("authz: fetch instance-admin state: %w", err)
	}
	if !isAdmin {
		return domain.ErrPermissionDenied
	}
	return nil
}

// ListRoles returns the grantable role catalog — M10.7's super-admin role-grants screen.
func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.store.ListRoles(ctx)
}

// ListRoleAssignmentsByUnit lists unitID's active role assignments — M10.7's super-admin
// role-grants screen.
func (s *Service) ListRoleAssignmentsByUnit(ctx context.Context, unitID string) ([]domain.RoleAssignment, error) {
	return s.store.ListRoleAssignmentsByUnit(ctx, unitID)
}

// ListRoleAssignmentsByPerson returns personID's own active role assignments across every unit —
// M11.5's self-service profile page. Unlike ListRoleAssignmentsByUnit, callers must derive personID
// from the resolved request subject only, never a client-supplied argument.
func (s *Service) ListRoleAssignmentsByPerson(ctx context.Context, personID string) ([]domain.RoleAssignment, error) {
	return s.store.ListRoleAssignmentsByPerson(ctx, personID)
}

// RevokeRoleAssignment revokes assignmentID, recording revokedByPersonID. Returns the revoked row's
// identity (M11.2: the audit log's "before" snapshot).
func (s *Service) RevokeRoleAssignment(ctx context.Context, assignmentID, revokedByPersonID string) (domain.RevokedRoleAssignment, error) {
	return s.store.RevokeRoleAssignment(ctx, assignmentID, revokedByPersonID)
}

// ClearRoleAssignmentExpiry clears assignmentID's expiresAt, leaving the grant itself untouched —
// M12.3. Returns the assignment's identity (mirroring RevokeRoleAssignment's own "before" snapshot
// shape) so the audit-log helper needs no second read.
func (s *Service) ClearRoleAssignmentExpiry(ctx context.Context, assignmentID string) (domain.RevokedRoleAssignment, error) {
	return s.store.ClearRoleAssignmentExpiry(ctx, assignmentID)
}

// ListInstanceAdmins returns every active instance-admin grant — M10.7's super-admin people screen.
func (s *Service) ListInstanceAdmins(ctx context.Context) ([]domain.InstanceAdminGrant, error) {
	return s.store.ListInstanceAdmins(ctx)
}

// GrantInstanceAdmin grants personID the instance-admin plane, recording grantedByPersonID.
// Callers must gate access to this method itself on RequireInstanceAdmin — it is not itself an
// authorization check, same convention as DecideFor.
func (s *Service) GrantInstanceAdmin(ctx context.Context, personID, grantedByPersonID string) (string, error) {
	return s.store.InsertInstanceAdmin(ctx, personID, grantedByPersonID)
}

// RevokeInstanceAdmin revokes personID's active instance-admin grant, recording revokedByPersonID.
// Returns the revoked grant's identity (M11.2: the audit log's "before" snapshot).
func (s *Service) RevokeInstanceAdmin(ctx context.Context, personID, revokedByPersonID string) (domain.RevokedInstanceAdminGrant, error) {
	return s.store.RevokeInstanceAdmin(ctx, personID, revokedByPersonID)
}

func (s *Service) decide(ctx context.Context, subjectPersonID string, action domain.Permission, unitID string, explain bool) (domain.Decision, error) {
	isAdmin, err := s.store.IsActiveInstanceAdmin(ctx, subjectPersonID)
	if err != nil {
		return domain.Decision{}, fmt.Errorf("authz: fetch instance-admin state: %w", err)
	}
	var grants []domain.ActiveGrant
	if !isAdmin {
		grants, err = s.store.ActiveGrantsForSubject(ctx, subjectPersonID)
		if err != nil {
			return domain.Decision{}, fmt.Errorf("authz: fetch active grants: %w", err)
		}
	}
	return s.pdp.Decide(ctx, domain.DecisionInput{
		Grants: grants, IsInstanceAdmin: isAdmin, Action: string(action), UnitID: unitID, Explain: explain,
	})
}
