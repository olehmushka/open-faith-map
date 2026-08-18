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
	InsertRoleAssignment(ctx context.Context, personID, roleID, targetUnitID, grantedBy string) error
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
// very next Require call with no extra step.
func (s *Service) GrantUnitRole(ctx context.Context, personID, roleID, unitID, grantedByPersonID string) error {
	return s.store.InsertRoleAssignment(ctx, personID, roleID, unitID, grantedByPersonID)
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
