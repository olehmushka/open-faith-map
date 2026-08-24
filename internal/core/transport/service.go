// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated core.CoreService and core.CoreSuperAdminService
// (Conjure server interfaces): translates Conjure structs <-> the underlying modules' domain types
// and maps domain errors to this contract's typed Conjure errors. Mirrors internal/content/transport's
// shape — the subject arrives via context (authz.SubjectFromContext, populated by the identity
// middleware), never re-derived from authHeader here.
package transport

import (
	"context"
	"time"

	"github.com/olehmushka/open-faith-map/internal/authz"
	gencore "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/core"
	"github.com/olehmushka/open-faith-map/internal/core/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	identitydomain "github.com/olehmushka/open-faith-map/internal/identity/domain"
	membershipdomain "github.com/olehmushka/open-faith-map/internal/membership/domain"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
	"github.com/palantir/pkg/bearertoken"
	"github.com/palantir/pkg/datetime"
)

type Service struct {
	app *application.Service
}

func NewService(app *application.Service) *Service {
	return &Service{app: app}
}

var _ gencore.CoreService = (*Service)(nil)

func (s *Service) Whoami(ctx context.Context, _ bearertoken.Token) (gencore.Whoami, error) {
	who, err := s.app.Whoami(ctx)
	if err != nil {
		return gencore.Whoami{}, mapErr(err, errCtx{})
	}
	return gencore.Whoami{
		PersonId: who.PersonID, AccountId: who.AccountID, Email: who.Email, IsInstanceAdmin: who.IsInstanceAdmin,
	}, nil
}

func (s *Service) RegisterSession(ctx context.Context, _ bearertoken.Token, requestArg gencore.RegisterSessionRequest) (gencore.Session, error) {
	sess, err := s.app.RegisterSession(ctx, derefStr(requestArg.DeviceLabel))
	if err != nil {
		return gencore.Session{}, mapErr(err, errCtx{})
	}
	return toAPISession(sess, ""), nil
}

func (s *Service) ListMySessions(ctx context.Context, _ bearertoken.Token) (gencore.SessionPage, error) {
	sessions, err := s.app.ListMySessions(ctx)
	if err != nil {
		return gencore.SessionPage{}, mapErr(err, errCtx{})
	}
	subject, _ := authz.SubjectFromContext(ctx)
	out := make([]gencore.Session, len(sessions))
	for i, sess := range sessions {
		out[i] = toAPISession(sess, subject.SessionID)
	}
	return gencore.SessionPage{Sessions: out}, nil
}

func (s *Service) RevokeMySession(ctx context.Context, _ bearertoken.Token, sessionIdArg string) error {
	if err := s.app.RevokeMySession(ctx, sessionIdArg); err != nil {
		return mapErr(err, errCtx{SessionID: sessionIdArg})
	}
	return nil
}

func (s *Service) UpdateMyProfile(ctx context.Context, _ bearertoken.Token, requestArg gencore.UpdateMyProfileRequest) (gencore.Person, error) {
	p, err := s.app.UpdateMyProfile(ctx, requestArg.DisplayName)
	if err != nil {
		return gencore.Person{}, mapErr(err, errCtx{})
	}
	return toAPIPerson(p), nil
}

func (s *Service) ListPermissionCatalog(ctx context.Context, _ bearertoken.Token) (gencore.PermissionCodePage, error) {
	codes, err := s.app.ListPermissionCatalog(ctx)
	if err != nil {
		return gencore.PermissionCodePage{}, mapErr(err, errCtx{})
	}
	return gencore.PermissionCodePage{Codes: codes}, nil
}

func (s *Service) ListMyApiKeys(ctx context.Context, _ bearertoken.Token) (gencore.ApiKeyPage, error) {
	keys, err := s.app.ListMyApiKeys(ctx)
	if err != nil {
		return gencore.ApiKeyPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.ApiKey, len(keys))
	for i, k := range keys {
		out[i] = toAPIApiKey(k)
	}
	return gencore.ApiKeyPage{ApiKeys: out}, nil
}

func (s *Service) CreateApiKey(ctx context.Context, _ bearertoken.Token, requestArg gencore.CreateApiKeyRequest) (gencore.CreateApiKeyResult, error) {
	result, err := s.app.CreateApiKey(ctx, requestArg.Label, requestArg.PermissionCodes)
	if err != nil {
		return gencore.CreateApiKeyResult{}, mapErr(err, errCtx{})
	}
	return gencore.CreateApiKeyResult{
		Id: result.ID, Label: result.Label, PermissionCodes: result.PermissionCodes,
		Token: result.Token, CreatedAt: datetime.DateTime(result.CreatedAt),
	}, nil
}

func (s *Service) RevokeMyApiKey(ctx context.Context, _ bearertoken.Token, apiKeyIdArg string) error {
	if err := s.app.RevokeMyApiKey(ctx, apiKeyIdArg); err != nil {
		return mapErr(err, errCtx{ApiKeyID: apiKeyIdArg})
	}
	return nil
}

func (s *Service) ListMyRoleAssignments(ctx context.Context, _ bearertoken.Token) (gencore.RoleAssignmentPage, error) {
	assignments, err := s.app.ListMyRoleAssignments(ctx)
	if err != nil {
		return gencore.RoleAssignmentPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.RoleAssignment, len(assignments))
	for i, a := range assignments {
		out[i] = gencore.RoleAssignment{
			Id: a.ID, PersonId: a.PersonID, PersonName: a.PersonName, RoleId: a.RoleID, RoleCode: a.RoleCode,
			TargetUnitId: a.TargetUnitID, Scope: string(a.Scope), GrantedAt: datetime.DateTime(a.GrantedAt),
		}
	}
	return gencore.RoleAssignmentPage{Assignments: out}, nil
}

func (s *Service) GetUnit(ctx context.Context, _ bearertoken.Token, unitIdArg string) (gencore.Unit, error) {
	unit, err := s.app.GetUnit(ctx, unitIdArg)
	if err != nil {
		return gencore.Unit{}, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	return toAPIUnit(unit), nil
}

func (s *Service) ListUnits(ctx context.Context, _ bearertoken.Token, queryArg *string, limitArg *int) (gencore.UnitPage, error) {
	units, err := s.app.ListUnits(ctx, derefStr(queryArg), derefInt(limitArg))
	if err != nil {
		return gencore.UnitPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.Unit, len(units))
	for i, u := range units {
		out[i] = toAPIUnit(u)
	}
	return gencore.UnitPage{Units: out}, nil
}

func (s *Service) UnitAncestors(ctx context.Context, _ bearertoken.Token, unitIdArg string) (gencore.UnitRefPage, error) {
	refs, err := s.app.UnitAncestors(ctx, unitIdArg)
	if err != nil {
		return gencore.UnitRefPage{}, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	out := make([]gencore.UnitRef, len(refs))
	for i, r := range refs {
		out[i] = gencore.UnitRef{Id: r.ID, Code: r.Code, Name: r.Name, Depth: r.Depth}
	}
	return gencore.UnitRefPage{Units: out}, nil
}

func (s *Service) ListTaxa(ctx context.Context, _ bearertoken.Token, queryArg *string, limitArg *int) (gencore.TaxonPage, error) {
	taxa, err := s.app.ListTaxa(ctx, derefStr(queryArg), derefInt(limitArg))
	if err != nil {
		return gencore.TaxonPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.Taxon, len(taxa))
	for i, t := range taxa {
		out[i] = toAPITaxon(t)
	}
	return gencore.TaxonPage{Taxa: out}, nil
}

func (s *Service) GetTaxon(ctx context.Context, _ bearertoken.Token, taxonIdArg string) (gencore.Taxon, error) {
	t, err := s.app.GetTaxon(ctx, taxonIdArg)
	if err != nil {
		return gencore.Taxon{}, mapErr(err, errCtx{TaxonID: taxonIdArg})
	}
	return toAPITaxon(t), nil
}

func (s *Service) ListOrgKinds(ctx context.Context, _ bearertoken.Token) (gencore.OrgKindPage, error) {
	kinds, err := s.app.ListOrgKinds(ctx)
	if err != nil {
		return gencore.OrgKindPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.OrgKind, len(kinds))
	for i, k := range kinds {
		out[i] = gencore.OrgKind{Id: k.ID, Code: k.Code, Name: k.Name}
	}
	return gencore.OrgKindPage{OrgKinds: out}, nil
}

func (s *Service) GetOrgProfile(ctx context.Context, _ bearertoken.Token, unitIdArg string) (gencore.OrgProfile, error) {
	p, err := s.app.GetOrgProfile(ctx, unitIdArg)
	if err != nil {
		return gencore.OrgProfile{}, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	return toAPIOrgProfile(p), nil
}

func (s *Service) CreateChildOrg(ctx context.Context, _ bearertoken.Token, requestArg gencore.CreateChildOrgRequest) (gencore.OrgProfile, error) {
	p, err := s.app.CreateChildOrg(ctx, requestArg.ParentUnitId, requestArg.Code, requestArg.Name, requestArg.OrgKindId, requestArg.PrimaryTaxonId)
	if err != nil {
		return gencore.OrgProfile{}, mapErr(err, errCtx{UnitID: requestArg.ParentUnitId})
	}
	return toAPIOrgProfile(p), nil
}

// M12.1 — unit lifecycle CRUD.

func (s *Service) CreateUnit(ctx context.Context, _ bearertoken.Token, requestArg gencore.CreateUnitRequest) (gencore.Unit, error) {
	unit, err := s.app.CreateUnit(ctx, requestArg.ParentUnitId, requestArg.Code, requestArg.Name, toInt16Ptr(requestArg.Level))
	if err != nil {
		return gencore.Unit{}, mapErr(err, errCtx{UnitID: requestArg.ParentUnitId})
	}
	return toAPIUnit(unit), nil
}

func (s *Service) UpdateUnit(ctx context.Context, _ bearertoken.Token, unitIdArg string, requestArg gencore.UpdateUnitRequest) (gencore.Unit, error) {
	unit, err := s.app.UpdateUnit(ctx, unitIdArg, requestArg.Name, requestArg.Code, toInt16Ptr(requestArg.Level))
	if err != nil {
		return gencore.Unit{}, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	return toAPIUnit(unit), nil
}

func (s *Service) SetUnitState(ctx context.Context, _ bearertoken.Token, unitIdArg string, requestArg gencore.SetUnitStateRequest) (gencore.Unit, error) {
	state := directorydomain.State(requestArg.State)
	switch state {
	case directorydomain.StateActive, directorydomain.StateSuspended, directorydomain.StateArchived:
	default:
		return gencore.Unit{}, gencore.NewInvalidUnitState()
	}
	unit, err := s.app.SetUnitState(ctx, unitIdArg, state)
	if err != nil {
		return gencore.Unit{}, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	return toAPIUnit(unit), nil
}

func (s *Service) DeleteUnit(ctx context.Context, _ bearertoken.Token, unitIdArg string) error {
	if _, err := s.app.DeleteUnit(ctx, unitIdArg); err != nil {
		return mapErr(err, errCtx{UnitID: unitIdArg})
	}
	return nil
}

// M12.2 — generic unit move/reparent.

func (s *Service) MoveUnit(ctx context.Context, _ bearertoken.Token, unitIdArg string, requestArg gencore.MoveUnitRequest) (gencore.UnitMoveJob, error) {
	job, err := s.app.MoveUnit(ctx, unitIdArg, requestArg.NewParentUnitId, derefStr(requestArg.GraphCode))
	if err != nil {
		return gencore.UnitMoveJob{}, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	return toAPIUnitMoveJob(job), nil
}

func (s *Service) GetUnitMoveStatus(ctx context.Context, _ bearertoken.Token, unitIdArg string, graphCodeArg *string) (*gencore.UnitMoveJob, error) {
	job, err := s.app.GetUnitMoveStatus(ctx, unitIdArg, derefStr(graphCodeArg))
	if err != nil {
		return nil, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	if job == nil {
		return nil, nil
	}
	out := toAPIUnitMoveJob(*job)
	return &out, nil
}

func toAPIUnitMoveJob(j directorydomain.MoveJob) gencore.UnitMoveJob {
	return gencore.UnitMoveJob{
		Id: j.ID, GraphId: j.GraphID, UnitId: j.UnitID,
		OldParentUnitId: j.OldParentUnitID, NewParentUnitId: j.NewParentUnitID,
		Status: string(j.Status), PerformedByPersonId: j.PerformedByPersonID, Error: j.Error,
		CreatedAt: datetime.DateTime(j.CreatedAt), UpdatedAt: datetime.DateTime(j.UpdatedAt),
	}
}

func (s *Service) ListCountries(ctx context.Context, _ bearertoken.Token) (gencore.CountryPage, error) {
	countries, err := s.app.ListCountries(ctx)
	if err != nil {
		return gencore.CountryPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.Country, len(countries))
	for i, c := range countries {
		out[i] = gencore.Country{Id: c.ID, Code: c.Code, Name: c.Name, Names: c.Names}
	}
	return gencore.CountryPage{Countries: out}, nil
}

func (s *Service) ListMembershipsByUnit(ctx context.Context, _ bearertoken.Token, unitIdArg string) (gencore.MembershipPage, error) {
	memberships, err := s.app.ListMembershipsByUnit(ctx, unitIdArg)
	if err != nil {
		return gencore.MembershipPage{}, mapErr(err, errCtx{UnitID: unitIdArg})
	}
	out := make([]gencore.Membership, len(memberships))
	for i, m := range memberships {
		out[i] = toAPIMembership(m)
	}
	return gencore.MembershipPage{Memberships: out}, nil
}

func (s *Service) GetPerson(ctx context.Context, _ bearertoken.Token, personIdArg string) (gencore.Person, error) {
	p, err := s.app.GetPerson(ctx, personIdArg)
	if err != nil {
		return gencore.Person{}, mapErr(err, errCtx{PersonID: personIdArg})
	}
	return toAPIPerson(p), nil
}

func (s *Service) GetPersons(ctx context.Context, _ bearertoken.Token, requestArg gencore.GetPersonsRequest) (gencore.PersonPage, error) {
	persons, err := s.app.GetPersons(ctx, requestArg.PersonIds)
	if err != nil {
		return gencore.PersonPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.Person, len(persons))
	for i, p := range persons {
		out[i] = toAPIPerson(p)
	}
	return gencore.PersonPage{Persons: out}, nil
}

// ---------------------------------------------------------------- conversions

func toAPIUnit(u directorydomain.Unit) gencore.Unit {
	var level *int
	if u.Level != nil {
		l := int(*u.Level)
		level = &l
	}
	return gencore.Unit{
		Id: u.ID, Code: optionalStr(u.Code), Name: u.Name, Level: level, State: string(u.State),
		CreatedAt: datetime.DateTime(u.CreatedAt), UpdatedAt: datetime.DateTime(u.UpdatedAt),
	}
}

func toAPITaxon(t religiondomain.Taxon) gencore.Taxon {
	return gencore.Taxon{
		Id: t.ID, ParentId: t.ParentID, RankId: t.RankID, RankCode: t.RankCode,
		Code: t.Code, Name: t.Name, SortOrder: t.SortOrder,
	}
}

func toAPIOrgProfile(p religiondomain.OrgProfile) gencore.OrgProfile {
	classifications := make([]gencore.OrgClassification, len(p.Classifications))
	for i, c := range p.Classifications {
		classifications[i] = gencore.OrgClassification{
			Id: c.ID, UnitId: c.UnitID, TaxonId: c.TaxonID, TaxonCode: c.TaxonCode, TaxonName: c.TaxonName,
			IsPrimary: c.IsPrimary, CreatedAt: datetime.DateTime(c.CreatedAt),
		}
	}
	return gencore.OrgProfile{
		UnitId: p.UnitID, OrgKindId: optionalStr(p.OrgKindID), ShortCode: optionalStr(p.ShortCode),
		Classifications: classifications,
	}
}

func toAPIMembership(m membershipdomain.Membership) gencore.Membership {
	return gencore.Membership{
		Id: m.ID, PersonId: m.PersonID, UnitId: m.UnitID, PositionId: m.PositionID,
		Status: m.Status, EffectiveFrom: datetime.DateTime(m.EffectiveFrom),
	}
}

func toAPIPerson(p identitydomain.Person) gencore.Person {
	return gencore.Person{
		Id: p.ID, Code: optionalStr(p.Code), DisplayName: p.DisplayName,
		CreatedAt: datetime.DateTime(p.CreatedAt), UpdatedAt: datetime.DateTime(p.UpdatedAt),
		LastActiveAt: optionalDateTime(p.LastActiveAt),
	}
}

// optionalDateTime converts an optional time.Time (M11.4's revoked-inclusive last-active signal) to
// the conjure-generated *datetime.DateTime an optional<datetime> field wants, nil-safe.
func optionalDateTime(t *time.Time) *datetime.DateTime {
	if t == nil {
		return nil
	}
	dt := datetime.DateTime(*t)
	return &dt
}

// toAPISession converts one identity_sessions row (M11.3). currentSessionID is the caller's own
// authz.Subject.SessionID — compared against sess.ID to compute IsCurrent server-side, rather than
// asking the client to know which of its own sessions it's presently using.
func toAPISession(sess identitydomain.Session, currentSessionID string) gencore.Session {
	return gencore.Session{
		Id: sess.ID, DeviceLabel: optionalStr(sess.DeviceLabel),
		CreatedAt: datetime.DateTime(sess.CreatedAt), LastSeenAt: datetime.DateTime(sess.LastSeenAt),
		IsCurrent: currentSessionID != "" && sess.ID == currentSessionID,
	}
}

// toAPIApiKey converts one identity_api_keys row (M11.9) — metadata only, matching gencore.ApiKey's
// own shape (no token/token-hash field exists on that type at all, so this converter cannot leak the
// secret regardless of caller).
func toAPIApiKey(k identitydomain.APIKey) gencore.ApiKey {
	return gencore.ApiKey{
		Id: k.ID, Label: k.Label, PermissionCodes: k.PermissionCodes,
		CreatedAt: datetime.DateTime(k.CreatedAt), LastUsedAt: optionalDateTime(k.LastUsedAt),
		RevokedAt: optionalDateTime(k.RevokedAt),
	}
}

func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func toInt16Ptr(i *int) *int16 {
	if i == nil {
		return nil
	}
	l := int16(*i)
	return &l
}
