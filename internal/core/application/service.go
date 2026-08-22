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
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	auditlogapplication "github.com/olehmushka/open-faith-map/internal/auditlog/application"
	auditlogdomain "github.com/olehmushka/open-faith-map/internal/auditlog/domain"
	"github.com/olehmushka/open-faith-map/internal/authz"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	identityadapters "github.com/olehmushka/open-faith-map/internal/identity/adapters"
	identityapplication "github.com/olehmushka/open-faith-map/internal/identity/application"
	identitydomain "github.com/olehmushka/open-faith-map/internal/identity/domain"
	membershipadapters "github.com/olehmushka/open-faith-map/internal/membership/adapters"
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
	auditLog   *auditlogapplication.Service
	// pool is M11.8's own addition, alongside the six module Services above: MergePersons is the
	// first CoreSuperAdminService write that must span identity's, authz's, and membership's tables
	// in one atomic transaction, and none of those Services expose their own pool or store for a
	// caller to reuse — mirroring internal/identity/bootstrap.Run's cross-module transaction shape,
	// just triggered at runtime instead of at boot.
	pool *pgxpool.Pool
}

func NewService(
	directory *directoryapplication.Service,
	religion *religionapplication.Service,
	membership *membershipapplication.Service,
	identity *identityapplication.Service,
	refdata *refdataapplication.Service,
	authzSvc *authz.Service,
	auditLog *auditlogapplication.Service,
	pool *pgxpool.Pool,
) *Service {
	return &Service{
		directory: directory, religion: religion, membership: membership,
		identity: identity, refdata: refdata, authz: authzSvc, auditLog: auditLog,
		pool: pool,
	}
}

// requireSubject resolves ctx's subject or fails loud — the shared guard every mutating super-admin
// method below applies before touching a store, so a missing subject can never reach s.authz/s.identity
// with an empty grantedBy/revokedBy/actor (M11.2: once auditLog.Record hard-fails on a missing
// subject, leaving a mutation itself tolerant of one would split-fail: the write succeeds with a NULL
// actor while only the audit call errors).
func (s *Service) requireSubject(ctx context.Context) (authz.Subject, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return authz.Subject{}, authzdomain.ErrPermissionDenied
	}
	return subject, nil
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

// Audit action names — M11.2. Free text, not a DB enum (docs/architecture/decisions.md), since
// future milestones (M11.3 session revocation, M11.7 bulk role assignment, M11.8 person merge) each
// add their own without touching this table's schema.
const (
	auditActionGrantUnitRole        = "GRANT_UNIT_ROLE"
	auditActionRevokeRoleAssignment = "REVOKE_ROLE_ASSIGNMENT"
	auditActionGrantInstanceAdmin   = "GRANT_INSTANCE_ADMIN"
	auditActionRevokeInstanceAdmin  = "REVOKE_INSTANCE_ADMIN"
	auditActionDeactivateAccount    = "DEACTIVATE_ACCOUNT"
	auditActionReactivateAccount    = "REACTIVATE_ACCOUNT"
	auditActionRevokeSession        = "REVOKE_SESSION"
	auditActionUpdateProfile        = "UPDATE_PROFILE"
	auditActionCreateInvite         = "CREATE_INVITE"
	auditActionBulkGrantUnitRole    = "BULK_GRANT_UNIT_ROLE"
	auditActionMergePersons         = "MERGE_PERSONS"
	auditActionCreateApiKey         = "CREATE_API_KEY"
	// auditActionRevokeApiKey (self-revoke) is kept distinct from auditActionRevokeApiKeyAdmin
	// (admin-revoke) — M11.9 — so the audit trail visibly distinguishes "the owner revoked their own
	// key" from "an admin revoked it for them" (incident response), the same log-viewer surfacing
	// both to every caller.
	auditActionRevokeApiKey      = "REVOKE_API_KEY"
	auditActionRevokeApiKeyAdmin = "REVOKE_API_KEY_ADMIN"

	auditTargetRoleAssignment = "ROLE_ASSIGNMENT"
	auditTargetInstanceAdmin  = "INSTANCE_ADMIN"
	auditTargetAccount        = "ACCOUNT"
	auditTargetSession        = "SESSION"
	auditTargetPerson         = "PERSON"
	auditTargetApiKey         = "API_KEY"
)

func (s *Service) GrantUnitRole(ctx context.Context, personID, roleID, unitID string) error {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return err
	}
	assignmentID, err := s.authz.GrantUnitRole(ctx, personID, roleID, unitID, subject.PersonID)
	if err != nil {
		return err
	}
	return s.auditLog.Record(ctx, auditActionGrantUnitRole, auditTargetRoleAssignment, assignmentID,
		nil, map[string]any{"personId": personID, "roleId": roleID, "unitId": unitID})
}

// BulkGrantUnitRole grants roleID on unitID to every id in personIDs, atomically — M11.7. The store
// call is the entire transaction: it either returns every resulting assignment id (all committed) or
// a non-nil error with nothing committed, so the audit loop below only ever runs over already-durable
// rows. Best-effort across the loop (not abort-on-first-failure) so one transient Record failure
// doesn't blank out audit rows for the rest of an already-successful batch.
func (s *Service) BulkGrantUnitRole(ctx context.Context, personIDs []string, roleID, unitID string) error {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return err
	}
	assignmentIDs, err := s.authz.BulkGrantUnitRole(ctx, personIDs, roleID, unitID, subject.PersonID)
	if err != nil {
		return err
	}
	var errs []error
	for i, assignmentID := range assignmentIDs {
		if err := s.auditLog.Record(ctx, auditActionBulkGrantUnitRole, auditTargetRoleAssignment, assignmentID,
			nil, map[string]any{"personId": personIDs[i], "roleId": roleID, "unitId": unitID}); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// MergePreview is what PreviewMergePersons computes read-only, before mergePersons's own
// destructive write (M11.8) — the admin UI's "what will move" step. Field-for-field, this mirrors
// MergeResult below (Xxx"ToMove"/"ToRevokeAsRedundant" here vs. Xxx"Moved"/"XxxRevokedRedundant"
// there) so the same UI rendering logic can present either shape.
type MergePreview struct {
	SurvivorID                            string
	DuplicatePersonID                     string
	RoleAssignmentsToMove                 int
	RoleAssignmentsToRevokeAsRedundant    int
	MembershipsToMove                     int
	MembershipsToEndAsRedundant           int
	InstanceAdminWillMove                 bool
	InstanceAdminWillBeRevokedAsRedundant bool
	DuplicateHasActiveAccount             bool
	// AccountConflict is true when the survivor already has their own active account — in that case
	// the duplicate's account will be disabled (soft-merge) rather than moved, and its login stops
	// working (a decision made with the user, not the alternative of re-pointing the duplicate's
	// external identity onto the survivor's own account).
	AccountConflict bool
}

// MergeResult is what MergePersons actually did (M11.8) — the audit record's payload and the
// confirmation UI's own summary.
type MergeResult struct {
	SurvivorID                      string
	DuplicatePersonID               string
	RoleAssignmentsMoved            int
	RoleAssignmentsRevokedRedundant int
	MembershipsMoved                int
	MembershipsEnded                int
	InstanceAdminMoved              bool
	InstanceAdminRevokedRedundant   bool
	DuplicateAccountMoved           bool
	DuplicateAccountDisabled        bool
}

// PreviewMergePersons computes, read-only, what MergePersons would move/end for (survivorID,
// duplicateID) — no requireSubject gate beyond the route group's own RequireInstanceAdmin, same as
// SearchPersons/ListRoles above (a pure read carries no further per-call check on this service).
// Every count/flag here is computed by the same predicate the real mutation uses, so a preview can
// never disagree with what confirming it will actually do.
func (s *Service) PreviewMergePersons(ctx context.Context, survivorID, duplicateID string) (MergePreview, error) {
	if survivorID == duplicateID {
		return MergePreview{}, identitydomain.ErrCannotMergeSelf
	}
	if _, err := s.identity.GetPerson(ctx, survivorID); err != nil {
		return MergePreview{}, err
	}
	if _, err := s.identity.GetPerson(ctx, duplicateID); err != nil {
		return MergePreview{}, err
	}

	authzStore := authzadapters.NewStore(s.pool)
	roleAssignmentsToMove, roleAssignmentsToRevoke, err := authzStore.CountRepointableRoleAssignments(ctx, duplicateID, survivorID)
	if err != nil {
		return MergePreview{}, err
	}
	instanceAdminWillMove, instanceAdminWillRevoke, err := authzStore.PreviewRepointInstanceAdmin(ctx, duplicateID, survivorID)
	if err != nil {
		return MergePreview{}, err
	}

	membershipStore := membershipadapters.NewStore(s.pool)
	membershipsToMove, membershipsToEnd, err := membershipStore.CountRepointableMemberships(ctx, duplicateID, survivorID)
	if err != nil {
		return MergePreview{}, err
	}

	identityStore := identityadapters.NewStore(s.pool)
	duplicateHasActiveAccount, accountConflict, err := identityStore.PreviewMergeIdentity(ctx, survivorID, duplicateID)
	if err != nil {
		return MergePreview{}, err
	}

	return MergePreview{
		SurvivorID:                            survivorID,
		DuplicatePersonID:                     duplicateID,
		RoleAssignmentsToMove:                 roleAssignmentsToMove,
		RoleAssignmentsToRevokeAsRedundant:    roleAssignmentsToRevoke,
		MembershipsToMove:                     membershipsToMove,
		MembershipsToEndAsRedundant:           membershipsToEnd,
		InstanceAdminWillMove:                 instanceAdminWillMove,
		InstanceAdminWillBeRevokedAsRedundant: instanceAdminWillRevoke,
		DuplicateHasActiveAccount:             duplicateHasActiveAccount,
		AccountConflict:                       accountConflict,
	}, nil
}

// MergePersons reassigns duplicateID's active role-assignment and membership rows onto survivorID,
// moves or disables duplicateID's account (see internal/identity/adapters.MergePersonIdentity),
// soft-deletes the duplicate person, and audit-logs the merge — M11.8, "the riskiest of the nine"
// (docs/milestones.md). Everything below runs inside one transaction spanning identity's, authz's,
// and membership's own tables: each module's store still only touches its own tables by name (no
// cross-module SQL), but this method is the one place that binds all three to the same pgx.Tx and
// commits once — the same shape internal/identity/bootstrap.Run already established for the
// boot-time admin seed, just triggered at runtime. Out of scope, deliberately (matches this
// milestone's own written scope): registration/moderation/vouching/congregationimport rows, which
// reference person ids as opaque text with no FK and are never touched here.
func (s *Service) MergePersons(ctx context.Context, survivorID, duplicateID string) (MergeResult, error) {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return MergeResult{}, err
	}
	if survivorID == duplicateID {
		return MergeResult{}, identitydomain.ErrCannotMergeSelf
	}
	if _, err := s.identity.GetPerson(ctx, survivorID); err != nil {
		return MergeResult{}, err
	}
	if _, err := s.identity.GetPerson(ctx, duplicateID); err != nil {
		return MergeResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MergeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	authzStore := authzadapters.NewStore(tx)
	roleAssignmentsMoved, roleAssignmentsRevoked, err := authzStore.RepointRoleAssignments(ctx, duplicateID, survivorID, subject.PersonID)
	if err != nil {
		return MergeResult{}, err
	}
	instanceAdminMoved, instanceAdminRevoked, err := authzStore.RepointInstanceAdmin(ctx, duplicateID, survivorID, subject.PersonID)
	if err != nil {
		return MergeResult{}, err
	}

	membershipStore := membershipadapters.NewStore(tx)
	membershipsMoved, membershipsEnded, err := membershipStore.RepointMemberships(ctx, duplicateID, survivorID)
	if err != nil {
		return MergeResult{}, err
	}

	identityStore := identityadapters.NewStore(tx)
	accountMoved, accountDisabled, err := identityStore.MergePersonIdentity(ctx, survivorID, duplicateID)
	if err != nil {
		return MergeResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return MergeResult{}, err
	}

	result := MergeResult{
		SurvivorID:                      survivorID,
		DuplicatePersonID:               duplicateID,
		RoleAssignmentsMoved:            len(roleAssignmentsMoved),
		RoleAssignmentsRevokedRedundant: len(roleAssignmentsRevoked),
		MembershipsMoved:                len(membershipsMoved),
		MembershipsEnded:                len(membershipsEnded),
		InstanceAdminMoved:              instanceAdminMoved,
		InstanceAdminRevokedRedundant:   instanceAdminRevoked,
		DuplicateAccountMoved:           accountMoved,
		DuplicateAccountDisabled:        accountDisabled,
	}
	if err := s.auditLog.Record(ctx, auditActionMergePersons, auditTargetPerson, survivorID, nil, map[string]any{
		"duplicatePersonId":               duplicateID,
		"roleAssignmentsMoved":            result.RoleAssignmentsMoved,
		"roleAssignmentsRevokedRedundant": result.RoleAssignmentsRevokedRedundant,
		"membershipsMoved":                result.MembershipsMoved,
		"membershipsEnded":                result.MembershipsEnded,
		"instanceAdminMoved":              result.InstanceAdminMoved,
		"instanceAdminRevokedRedundant":   result.InstanceAdminRevokedRedundant,
		"duplicateAccountMoved":           result.DuplicateAccountMoved,
		"duplicateAccountDisabled":        result.DuplicateAccountDisabled,
	}); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) RevokeRoleAssignment(ctx context.Context, assignmentID string) error {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return err
	}
	before, err := s.authz.RevokeRoleAssignment(ctx, assignmentID, subject.PersonID)
	if err != nil {
		return err
	}
	return s.auditLog.Record(ctx, auditActionRevokeRoleAssignment, auditTargetRoleAssignment, assignmentID,
		map[string]any{"personId": before.PersonID, "roleId": before.RoleID, "unitId": before.TargetUnitID, "scope": string(before.Scope)}, nil)
}

func (s *Service) ListInstanceAdmins(ctx context.Context) ([]authzdomain.InstanceAdminGrant, error) {
	return s.authz.ListInstanceAdmins(ctx)
}

func (s *Service) GrantInstanceAdmin(ctx context.Context, personID string) (authzdomain.InstanceAdminGrant, error) {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return authzdomain.InstanceAdminGrant{}, err
	}
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
	grant := authzdomain.InstanceAdminGrant{ID: id, PersonID: personID}
	for _, a := range admins {
		if a.ID == id {
			grant = a
			break
		}
	}
	// map[string]any, not the raw grant struct: authzdomain.InstanceAdminGrant carries no json tags,
	// so marshaling it directly would key the payload PersonId/PersonName (Go field casing) instead
	// of the personId/roleId/unitId camelCase every other call site's audit payload uses.
	if err := s.auditLog.Record(ctx, auditActionGrantInstanceAdmin, auditTargetInstanceAdmin, grant.ID, nil,
		map[string]any{"personId": grant.PersonID, "personName": grant.PersonName}); err != nil {
		return authzdomain.InstanceAdminGrant{}, err
	}
	return grant, nil
}

func (s *Service) RevokeInstanceAdmin(ctx context.Context, personID string) error {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return err
	}
	before, err := s.authz.RevokeInstanceAdmin(ctx, personID, subject.PersonID)
	if err != nil {
		return err
	}
	return s.auditLog.Record(ctx, auditActionRevokeInstanceAdmin, auditTargetInstanceAdmin, before.ID,
		map[string]any{"personId": before.PersonID}, nil)
}

// AccountStatusNone is the wire value for "this person has never had a login attached" — not an
// identity_accounts.status value (there is no row to have one), so it lives here rather than in
// internal/identity/domain.
const AccountStatusNone = "none"

// AccountStatus is core's own read-model for an M11.1 account-status check — the super-admin person
// detail page's deactivate/reactivate action. LastActiveAt is M11.4's addition, nil for
// AccountStatusNone and for an account that has never had a session.
type AccountStatus struct {
	PersonID     string
	Status       string
	LastActiveAt *time.Time
}

func (s *Service) GetAccountStatus(ctx context.Context, personID string) (AccountStatus, error) {
	status, lastActiveAt, found, err := s.identity.AccountStatus(ctx, personID)
	if err != nil {
		return AccountStatus{}, err
	}
	if !found {
		status = AccountStatusNone
	}
	return AccountStatus{PersonID: personID, Status: status, LastActiveAt: lastActiveAt}, nil
}

func (s *Service) DeactivateAccount(ctx context.Context, personID string) (AccountStatus, error) {
	if _, err := s.requireSubject(ctx); err != nil {
		return AccountStatus{}, err
	}
	before, after, err := s.identity.Deactivate(ctx, personID)
	if err != nil {
		return AccountStatus{}, err
	}
	if err := s.auditLog.Record(ctx, auditActionDeactivateAccount, auditTargetAccount, personID,
		map[string]any{"status": before.Status}, map[string]any{"status": after.Status}); err != nil {
		return AccountStatus{}, err
	}
	return AccountStatus{PersonID: personID, Status: after.Status}, nil
}

func (s *Service) ReactivateAccount(ctx context.Context, personID string) (AccountStatus, error) {
	if _, err := s.requireSubject(ctx); err != nil {
		return AccountStatus{}, err
	}
	before, account, err := s.identity.Reactivate(ctx, personID)
	if err != nil {
		return AccountStatus{}, err
	}
	if err := s.auditLog.Record(ctx, auditActionReactivateAccount, auditTargetAccount, personID,
		map[string]any{"status": before.Status}, map[string]any{"status": account.Status}); err != nil {
		return AccountStatus{}, err
	}
	return AccountStatus{PersonID: personID, Status: account.Status}, nil
}

// ---------------------------------------------------------------- sessions (M11.3, D-SessionTracking)

// RegisterSession creates the identity_sessions row backing a just-completed NextAuth sign-in.
// Self-scoped (the caller registers their own session, never someone else's) and deliberately NOT
// audit-logged — a sign-in is not an admin action, and internal/identity/middleware's
// sessionExemptRoutes lets this one request through without a session id of its own to present
// (there being nothing yet to present). The issuer is read off the caller's own verified bearer
// (subject.Issuer), not a client-supplied request field — the middleware already established it.
func (s *Service) RegisterSession(ctx context.Context, deviceLabel string) (identitydomain.Session, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.AccountID == "" {
		return identitydomain.Session{}, authzdomain.ErrPermissionDenied
	}
	return s.identity.RegisterSession(ctx, subject.AccountID, subject.Issuer, deviceLabel)
}

// ListSessions returns personID's active sessions (admin-scoped) — CoreSuperAdminService.listSessions.
func (s *Service) ListSessions(ctx context.Context, personID string) ([]identitydomain.Session, error) {
	return s.identity.ListSessions(ctx, personID)
}

// RevokeSession revokes one of personID's sessions (admin-scoped) — CoreSuperAdminService.revokeSession.
func (s *Service) RevokeSession(ctx context.Context, personID, sessionID string) error {
	if _, err := s.requireSubject(ctx); err != nil {
		return err
	}
	before, err := s.identity.RevokeSession(ctx, personID, sessionID)
	if err != nil {
		return err
	}
	return s.auditLog.Record(ctx, auditActionRevokeSession, auditTargetSession, sessionID,
		map[string]any{"revokedAt": nil}, map[string]any{"revokedAt": before.RevokedAt})
}

// ListMySessions returns the caller's own active sessions (self-scoped) — CoreService.listMySessions.
func (s *Service) ListMySessions(ctx context.Context) ([]identitydomain.Session, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.AccountID == "" {
		return nil, authzdomain.ErrPermissionDenied
	}
	return s.identity.ListMySessions(ctx, subject.AccountID)
}

// RevokeMySession revokes one of the caller's own sessions (self-scoped) —
// CoreService.revokeMySession. Still audit-logged (M11.2: every mutation this arc adds is, whether
// admin- or self-initiated) — the actor and target happen to be the same person here.
func (s *Service) RevokeMySession(ctx context.Context, sessionID string) error {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return err
	}
	if subject.AccountID == "" {
		return authzdomain.ErrPermissionDenied
	}
	before, err := s.identity.RevokeMySession(ctx, subject.AccountID, sessionID)
	if err != nil {
		return err
	}
	return s.auditLog.Record(ctx, auditActionRevokeSession, auditTargetSession, sessionID,
		map[string]any{"revokedAt": nil}, map[string]any{"revokedAt": before.RevokedAt})
}

// UpdateMyProfile sets the caller's own display name (self-scoped) — CoreService.updateMyProfile.
// personID always comes from requireSubject's resolved subject, never a request argument — a
// deliberate BOLA/IDOR defense: there is no way for this endpoint to be pointed at anyone else's
// person row. Still audit-logged (M11.2's every-mutation convention, same reasoning
// RevokeMySession's own doc comment gives): actor and target happen to be the same person here.
func (s *Service) UpdateMyProfile(ctx context.Context, displayName string) (identitydomain.Person, error) {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return identitydomain.Person{}, err
	}
	before, err := s.identity.GetPerson(ctx, subject.PersonID)
	if err != nil {
		return identitydomain.Person{}, err
	}
	after, err := s.identity.UpdateMyProfile(ctx, subject.PersonID, displayName)
	if err != nil {
		return identitydomain.Person{}, err
	}
	if err := s.auditLog.Record(ctx, auditActionUpdateProfile, auditTargetPerson, subject.PersonID,
		map[string]any{"displayName": before.DisplayName}, map[string]any{"displayName": after.DisplayName}); err != nil {
		return identitydomain.Person{}, err
	}
	return after, nil
}

// ListMyRoleAssignments returns the caller's own active role assignments across every unit
// (self-scoped) — CoreService.listMyRoleAssignments. Pure read, no audit (same reasoning
// ListMySessions already documents); personID again comes only from the resolved subject.
func (s *Service) ListMyRoleAssignments(ctx context.Context) ([]authzdomain.RoleAssignment, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return nil, authzdomain.ErrPermissionDenied
	}
	return s.authz.ListRoleAssignmentsByPerson(ctx, subject.PersonID)
}

// AuditLogFilter narrows ListAuditLog by actor/target/date — every field optional, ANDed together
// when set. Core's own copy of auditlogdomain.Filter's fields (not a type alias) so this file's
// transport-facing surface doesn't leak an internal/auditlog import requirement onto its callers,
// same reasoning OrgKind above gives for not re-exporting religion's own adapter-layer type.
type AuditLogFilter struct {
	ActorPersonID string
	TargetKind    string
	TargetID      string
	From          *time.Time
	To            *time.Time
}

// ListAuditLog answers the super-admin audit-log viewer's listAuditLog (M11.2) — pageSize+1 is the
// caller's responsibility, same convention as moderation.Service.ListReports: transport trims the
// extra row and encodes nextPageToken from it.
func (s *Service) ListAuditLog(ctx context.Context, filter AuditLogFilter, pageSize int, after *auditlogdomain.PageCursor) ([]auditlogdomain.Entry, error) {
	return s.auditLog.List(ctx, auditlogdomain.Filter{
		ActorPersonID: filter.ActorPersonID, TargetKind: filter.TargetKind, TargetID: filter.TargetID,
		From: filter.From, To: filter.To,
	}, pageSize, after)
}

// ---------------------------------------------------------------- invites (M11.6, D-InviteLinkMVP)

// InviteResult is CoreSuperAdminService.invitePerson's response — Token is the bare, one-time raw
// token, not a full URL: the backend has no notion of the admin app's own public origin, so the
// Next.js layer builds the shareable link from its own known origin.
type InviteResult struct {
	PersonID  string
	AccountID string
	Token     string
	ExpiresAt time.Time
}

// InvitePerson pre-provisions a Person+Account for email/displayName and generates a one-time
// invite link (admin-scoped, requires CoreSuperAdminService's route-group gate) — CoreSuperAdminService.invitePerson.
func (s *Service) InvitePerson(ctx context.Context, email, displayName string) (InviteResult, error) {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return InviteResult{}, err
	}
	invite, rawToken, err := s.identity.CreateInvite(ctx, email, displayName, subject.PersonID)
	if err != nil {
		return InviteResult{}, err
	}
	if err := s.auditLog.Record(ctx, auditActionCreateInvite, auditTargetPerson, invite.PersonID, nil,
		map[string]any{"email": email, "displayName": displayName}); err != nil {
		return InviteResult{}, err
	}
	return InviteResult{PersonID: invite.PersonID, AccountID: invite.AccountID, Token: rawToken, ExpiresAt: invite.ExpiresAt}, nil
}

// ResolveInvite validates an invite token for its own not-yet-authenticated invitee —
// CoreService.resolveInvite, the one endpoint in this arc reachable with no session at all
// (internal/identity/middleware's anonymousRoutes). Deliberately no requireSubject call and no audit
// log: a pure read, and the caller has no subject to resolve yet.
func (s *Service) ResolveInvite(ctx context.Context, token string) (identityapplication.InviteInfo, error) {
	return s.identity.ResolveInvite(ctx, token)
}

// ---------------------------------------------------------------- API keys (M11.9)

// ListPermissionCatalog returns the closed unit-scoped permission catalog (excluding instance-scope
// codes — see UnitScopedPermissionCodes' own doc comment) — CoreService.listPermissionCatalog. Static,
// self-scoped, no audit (a pure read, same reasoning ListMySessions/ListMyRoleAssignments give).
func (s *Service) ListPermissionCatalog(ctx context.Context) ([]string, error) {
	if _, err := s.requireSubject(ctx); err != nil {
		return nil, err
	}
	return authzdomain.UnitScopedPermissionCodes(), nil
}

// CreateApiKeyResult is CoreService.createApiKey's response — Token is the bare, one-time raw secret,
// the same "returned exactly once, only its hash is ever persisted" shape InviteResult already uses.
type CreateApiKeyResult struct {
	ID              string
	Label           string
	PermissionCodes []string
	Token           string
	CreatedAt       time.Time
}

// CreateApiKey mints a new API key for the caller, scoped to permissionCodes (self-scoped) —
// CoreService.createApiKey. permissionCodes is validated against the closed catalog (and rejected if
// it contains any instance-scope code) by s.identity.CreateApiKey itself.
func (s *Service) CreateApiKey(ctx context.Context, label string, permissionCodes []string) (CreateApiKeyResult, error) {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return CreateApiKeyResult{}, err
	}
	key, rawToken, err := s.identity.CreateApiKey(ctx, subject.PersonID, label, permissionCodes)
	if err != nil {
		return CreateApiKeyResult{}, err
	}
	if err := s.auditLog.Record(ctx, auditActionCreateApiKey, auditTargetApiKey, key.ID, nil,
		map[string]any{"label": label, "permissionCodes": permissionCodes}); err != nil {
		return CreateApiKeyResult{}, err
	}
	return CreateApiKeyResult{ID: key.ID, Label: key.Label, PermissionCodes: key.PermissionCodes, Token: rawToken, CreatedAt: key.CreatedAt}, nil
}

// ListMyApiKeys returns the caller's own active API keys (self-scoped) — CoreService.listMyApiKeys.
// Pure read, no audit (same reasoning ListMySessions already documents).
func (s *Service) ListMyApiKeys(ctx context.Context) ([]identitydomain.APIKey, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return nil, authzdomain.ErrPermissionDenied
	}
	return s.identity.ListMyApiKeys(ctx, subject.PersonID)
}

// RevokeMyApiKey revokes one of the caller's own API keys (self-scoped) — CoreService.revokeMyApiKey.
// s.identity.RevokeMyApiKey's own person_id-scoped WHERE clause means a foreign apiKeyID resolves as
// ErrAPIKeyNotFound, not a different error that would leak whether the id exists at all.
func (s *Service) RevokeMyApiKey(ctx context.Context, apiKeyID string) error {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return err
	}
	before, err := s.identity.RevokeMyApiKey(ctx, subject.PersonID, apiKeyID)
	if err != nil {
		return err
	}
	return s.auditLog.Record(ctx, auditActionRevokeApiKey, auditTargetApiKey, apiKeyID,
		map[string]any{"revokedAt": nil}, map[string]any{"revokedAt": before.RevokedAt})
}

// ListApiKeys returns personID's full key history, active and revoked (admin-scoped, incident-
// response visibility) — CoreSuperAdminService.listApiKeys.
func (s *Service) ListApiKeys(ctx context.Context, personID string) ([]identitydomain.APIKey, error) {
	return s.identity.ListApiKeysByPerson(ctx, personID)
}

// RevokeApiKey revokes one of personID's API keys on an admin's behalf (admin-scoped, incident
// response) — CoreSuperAdminService.revokeApiKey. revokedByPersonID (the acting admin) is recorded
// distinctly from personID (the owner) via auditActionRevokeApiKeyAdmin.
func (s *Service) RevokeApiKey(ctx context.Context, personID, apiKeyID string) error {
	subject, err := s.requireSubject(ctx)
	if err != nil {
		return err
	}
	before, err := s.identity.RevokeApiKey(ctx, personID, apiKeyID, subject.PersonID)
	if err != nil {
		return err
	}
	return s.auditLog.Record(ctx, auditActionRevokeApiKeyAdmin, auditTargetApiKey, apiKeyID,
		map[string]any{"revokedAt": nil}, map[string]any{"revokedAt": before.RevokedAt})
}
