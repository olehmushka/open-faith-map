// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is core's orchestration layer — a thin fan-in over the modules M10.1-M10.5
// already built (internal/{identity,directory,religion,membership,refdata,authz}), giving
// openfaithmap-admin its own Conjure surface (M10.7) for what it used to reach through a sibling
// go-oikumenea instance. Owns no new tables and no new domain logic of its own beyond one thing: the
// authorization gate on CreateChildOrg. internal/religion carries zero authorization logic by design
// (D-InProcessAuthz) — every consumer module gates its own writes before calling in (registration,
// content, moderation, vouching, congregationimport all already do this), and core.CoreService is
// this admin app's own consumer module, so it follows the same convention rather than a special case.
package application

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/authz"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	identityapplication "github.com/olehmushka/open-faith-map/internal/identity/application"
	identitydomain "github.com/olehmushka/open-faith-map/internal/identity/domain"
	membershipapplication "github.com/olehmushka/open-faith-map/internal/membership/application"
	membershipdomain "github.com/olehmushka/open-faith-map/internal/membership/domain"
	refdataapplication "github.com/olehmushka/open-faith-map/internal/refdata/application"
	refdatadomain "github.com/olehmushka/open-faith-map/internal/refdata/domain"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
)

type Service struct {
	directory  *directoryapplication.Service
	religion   *religionapplication.Service
	membership *membershipapplication.Service
	identity   *identityapplication.Service
	refdata    *refdataapplication.Service
	authz      *authz.Service
}

func NewService(
	directory *directoryapplication.Service,
	religion *religionapplication.Service,
	membership *membershipapplication.Service,
	identity *identityapplication.Service,
	refdata *refdataapplication.Service,
	authzSvc *authz.Service,
) *Service {
	return &Service{
		directory: directory, religion: religion, membership: membership,
		identity: identity, refdata: refdata, authz: authzSvc,
	}
}

// ---------------------------------------------------------------- whoami

// Whoami is the resolved subject plus its instance-admin standing — CoreService's own replacement
// for go-oikumenea's identityFederation.whoami(), now backed by the same subject the identity
// middleware already resolved for this request (authz.SubjectFromContext) rather than a second
// round trip.
type Whoami struct {
	PersonID        string
	AccountID       string
	Email           string
	IsInstanceAdmin bool
}

func (s *Service) Whoami(ctx context.Context) (Whoami, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok {
		return Whoami{}, authzdomain.ErrPermissionDenied
	}
	// RequireInstanceAdmin would deny-and-return for a non-admin; whoami needs the boolean either
	// way, so ask the store-backed check directly via a denied Require rather than treating denial
	// as an error.
	isAdmin := s.authz.RequireInstanceAdmin(ctx) == nil
	return Whoami{PersonID: subject.PersonID, AccountID: subject.AccountID, Email: subject.Email, IsInstanceAdmin: isAdmin}, nil
}

// ---------------------------------------------------------------- units (read-only; no per-call
// gate beyond the session default-auth already requires — matches the pre-cutover admin app, which
// never scoped these particular reads to the caller's own grants either)

func (s *Service) GetUnit(ctx context.Context, id string) (directorydomain.Unit, error) {
	return s.directory.GetUnit(ctx, id)
}

func (s *Service) ListUnits(ctx context.Context, query string, limit int) ([]directorydomain.Unit, error) {
	return s.directory.ListUnits(ctx, query, limit)
}

func (s *Service) UnitAncestors(ctx context.Context, unitID string) ([]directorydomain.UnitRef, error) {
	return s.directory.Ancestors(ctx, unitID, directorydomain.CanonicalGraphCode)
}

// ---------------------------------------------------------------- religion catalog (read-only, same
// no-per-call-gate reasoning as units — register/page.tsx's own use of these needs to work for a
// freshly-signed-in caller who holds no grant on anything yet)

func (s *Service) ListTaxa(ctx context.Context, query string, limit int) ([]religiondomain.Taxon, error) {
	return s.religion.ListTaxa(ctx, query, limit)
}

func (s *Service) GetTaxon(ctx context.Context, id string) (religiondomain.Taxon, error) {
	return s.religion.GetTaxon(ctx, id)
}

// OrgKind is core's own read-model for religionapplication.ListOrgKinds' result — religion's
// adapters.OrgKind type is that module's own adapter-layer concern (its Store method's return
// shape), not something other modules' application layers should import directly.
type OrgKind struct {
	ID   string
	Code string
	Name string
}

func (s *Service) ListOrgKinds(ctx context.Context) ([]OrgKind, error) {
	kinds, err := s.religion.ListOrgKinds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OrgKind, len(kinds))
	for i, k := range kinds {
		out[i] = OrgKind{ID: k.ID, Code: k.Code, Name: k.Name}
	}
	return out, nil
}

func (s *Service) GetOrgProfile(ctx context.Context, unitID string) (religiondomain.OrgProfile, error) {
	return s.religion.GetOrgProfile(ctx, unitID)
}

// CreateChildOrg is CoreService's one gated write — requireReligionOrgManage on the parent unit,
// same shape every other consumer module already uses before calling into internal/religion, since
// internal/religion itself carries no authorization logic (D-InProcessAuthz).
func (s *Service) CreateChildOrg(ctx context.Context, parentUnitID, code, name string, orgKindID, primaryTaxonID *string) (religiondomain.OrgProfile, error) {
	if err := s.authz.Require(ctx, authzdomain.PermReligionOrgManage, parentUnitID); err != nil {
		return religiondomain.OrgProfile{}, err
	}
	return s.religion.CreateChildOrg(ctx, parentUnitID, code, name, orgKindID, primaryTaxonID)
}

// ---------------------------------------------------------------- refdata (read-only catalog, same
// no-per-call-gate reasoning)

func (s *Service) ListCountries(ctx context.Context) ([]refdatadomain.Country, error) {
	return s.refdata.ListCountries(ctx)
}

// ---------------------------------------------------------------- membership + persons (my-congregation's
// unitId is always derived server-side from the caller's own approved registration record — never
// client-supplied — so this needs no additional per-call gate beyond the session itself, matching the
// pre-cutover admin app's own behaviour for this exact call sequence)

func (s *Service) ListMembershipsByUnit(ctx context.Context, unitID string) ([]membershipdomain.Membership, error) {
	return s.membership.ListMembershipsByUnit(ctx, unitID)
}

func (s *Service) GetPerson(ctx context.Context, id string) (identitydomain.Person, error) {
	return s.identity.GetPerson(ctx, id)
}

func (s *Service) GetPersons(ctx context.Context, ids []string) ([]identitydomain.Person, error) {
	return s.identity.GetPersons(ctx, ids)
}

// ---------------------------------------------------------------- super-admin surface (every method
// below is only reachable through CoreSuperAdminService, gated as a whole route group by
// internal/authz/transport.RequireInstanceAdmin — no per-call gate needed here, same reasoning
// D-SuperAdminFold's amendment gives for building one structural enforcer instead of a sixth
// hand-copied require*)

func (s *Service) SearchPersons(ctx context.Context, query string, limit int) ([]identitydomain.Person, error) {
	return s.identity.SearchPersons(ctx, query, limit)
}

func (s *Service) ListRoles(ctx context.Context) ([]authzdomain.Role, error) {
	return s.authz.ListRoles(ctx)
}

func (s *Service) ListRoleAssignmentsByUnit(ctx context.Context, unitID string) ([]authzdomain.RoleAssignment, error) {
	return s.authz.ListRoleAssignmentsByUnit(ctx, unitID)
}

func (s *Service) GrantUnitRole(ctx context.Context, personID, roleID, unitID string) error {
	subject, _ := authz.SubjectFromContext(ctx)
	return s.authz.GrantUnitRole(ctx, personID, roleID, unitID, subject.PersonID)
}

func (s *Service) RevokeRoleAssignment(ctx context.Context, assignmentID string) error {
	subject, _ := authz.SubjectFromContext(ctx)
	return s.authz.RevokeRoleAssignment(ctx, assignmentID, subject.PersonID)
}

func (s *Service) ListInstanceAdmins(ctx context.Context) ([]authzdomain.InstanceAdminGrant, error) {
	return s.authz.ListInstanceAdmins(ctx)
}

func (s *Service) GrantInstanceAdmin(ctx context.Context, personID string) (authzdomain.InstanceAdminGrant, error) {
	subject, _ := authz.SubjectFromContext(ctx)
	id, err := s.authz.GrantInstanceAdmin(ctx, personID, subject.PersonID)
	if err != nil {
		return authzdomain.InstanceAdminGrant{}, err
	}
	// GrantInstanceAdmin's store call returns only the new row's id; read the row back so the
	// caller gets the same shape ListInstanceAdmins returns, PersonName included.
	admins, err := s.authz.ListInstanceAdmins(ctx)
	if err != nil {
		return authzdomain.InstanceAdminGrant{}, err
	}
	for _, a := range admins {
		if a.ID == id {
			return a, nil
		}
	}
	return authzdomain.InstanceAdminGrant{ID: id, PersonID: personID}, nil
}

func (s *Service) RevokeInstanceAdmin(ctx context.Context, personID string) error {
	subject, _ := authz.SubjectFromContext(ctx)
	return s.authz.RevokeInstanceAdmin(ctx, personID, subject.PersonID)
}

// AccountStatusNone is the wire value for "this person has never had a login attached" — not an
// identity_accounts.status value (there is no row to have one), so it lives here rather than in
// internal/identity/domain.
const AccountStatusNone = "none"

// AccountStatus is core's own read-model for an M11.1 account-status check — the super-admin person
// detail page's deactivate/reactivate action.
type AccountStatus struct {
	PersonID string
	Status   string
}

func (s *Service) GetAccountStatus(ctx context.Context, personID string) (AccountStatus, error) {
	status, found, err := s.identity.AccountStatus(ctx, personID)
	if err != nil {
		return AccountStatus{}, err
	}
	if !found {
		status = AccountStatusNone
	}
	return AccountStatus{PersonID: personID, Status: status}, nil
}

func (s *Service) DeactivateAccount(ctx context.Context, personID string) (AccountStatus, error) {
	account, err := s.identity.Deactivate(ctx, personID)
	if err != nil {
		return AccountStatus{}, err
	}
	return AccountStatus{PersonID: personID, Status: account.Status}, nil
}

func (s *Service) ReactivateAccount(ctx context.Context, personID string) (AccountStatus, error) {
	account, err := s.identity.Reactivate(ctx, personID)
	if err != nil {
		return AccountStatus{}, err
	}
	return AccountStatus{PersonID: personID, Status: account.Status}, nil
}
