// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"fmt"

	"github.com/olehmushka/open-faith-map/internal/authz/domain"
)

// GrantStore fetches a subject's authority state, freshly, per call (D-InProcessAuthz's amendment:
// no grant cache — a stale ALLOW with no RLS backstop underneath it is the security bug this avoids)
// — and, since M10.6, writes new unit-scoped grants (the module's own tables, so it owns the write).
type GrantStore interface {
	IsActiveInstanceAdmin(ctx context.Context, personID string) (bool, error)
	ActiveGrantsForSubject(ctx context.Context, personID string) ([]domain.ActiveGrant, error)
	InsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, grantedBy string) (string, error)

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
	ListInstanceAdmins(ctx context.Context) ([]domain.InstanceAdminGrant, error)
	InsertInstanceAdmin(ctx context.Context, personID, grantedBy string) (string, error)
	RevokeInstanceAdmin(ctx context.Context, personID, revokedBy string) (domain.RevokedInstanceAdminGrant, error)
}

// Service is the module's composition: the pure PDP engine plus the store that fetches the authority
// state it decides over.
type Service struct {
	pdp   domain.PDP
	store GrantStore
}

func NewService(pdp domain.PDP, store GrantStore) *Service {
	return &Service{pdp: pdp, store: store}
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

// GrantUnitRole grants personID roleID on unitID, scope "unit" — M10.6's registration cutover is
// the first caller (approval-time congregation-admin grant). No epoch bump, no cache to invalidate
// (D-InProcessAuthz's amendment: grants are read fresh per request), so a grant is visible to the
// very next Require call with no extra step. Returns the assignment's id (M11.2: super-admin callers
// use it as the audit log's target_id).
func (s *Service) GrantUnitRole(ctx context.Context, personID, roleID, unitID, grantedByPersonID string) (string, error) {
	return s.store.InsertRoleAssignment(ctx, personID, roleID, unitID, grantedByPersonID)
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
