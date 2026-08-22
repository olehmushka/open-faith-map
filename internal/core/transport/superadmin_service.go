// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"
	"encoding/json"
	"time"

	auditlogdomain "github.com/olehmushka/open-faith-map/internal/auditlog/domain"
	gencore "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/core"
	"github.com/olehmushka/open-faith-map/internal/core/application"
	"github.com/palantir/pkg/bearertoken"
	"github.com/palantir/pkg/datetime"
)

// defaultAuditLogPageSize/maxAuditLogPageSize mirror moderation's own unspecified-pageSize fallback
// and provisional ceiling (M7, docs/modules/hardening.md) — not data-tuned, same as moderation's.
const (
	defaultAuditLogPageSize = 50
	maxAuditLogPageSize     = 200
)

// SuperAdminService implements gencore.CoreSuperAdminService. Every route this type is registered
// under is wrapped, as a whole group, by internal/authz/transport.RequireInstanceAdmin
// (cmd/openfaithmap-api/register_core.go) — no per-method gate needed here (D-SuperAdminFold's
// amendment: one shared, hard-to-misuse enforcer, not a sixth hand-copied require*).
type SuperAdminService struct {
	app *application.Service
}

func NewSuperAdminService(app *application.Service) *SuperAdminService {
	return &SuperAdminService{app: app}
}

var _ gencore.CoreSuperAdminService = (*SuperAdminService)(nil)

func (s *SuperAdminService) SearchPersons(ctx context.Context, _ bearertoken.Token, queryArg *string, limitArg *int) (gencore.PersonPage, error) {
	persons, err := s.app.SearchPersons(ctx, derefStr(queryArg), derefInt(limitArg))
	if err != nil {
		return gencore.PersonPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.Person, len(persons))
	for i, p := range persons {
		out[i] = toAPIPerson(p)
	}
	return gencore.PersonPage{Persons: out}, nil
}

func (s *SuperAdminService) ListRoles(ctx context.Context, _ bearertoken.Token) (gencore.RolePage, error) {
	roles, err := s.app.ListRoles(ctx)
	if err != nil {
		return gencore.RolePage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.Role, len(roles))
	for i, r := range roles {
		out[i] = gencore.Role{Id: r.ID, Code: r.Code, Name: r.Name, Description: optionalStr(r.Description), IsBase: r.IsBase}
	}
	return gencore.RolePage{Roles: out}, nil
}

func (s *SuperAdminService) ListRoleAssignmentsByUnit(ctx context.Context, _ bearertoken.Token, unitIdArg string) (gencore.RoleAssignmentPage, error) {
	assignments, err := s.app.ListRoleAssignmentsByUnit(ctx, unitIdArg)
	if err != nil {
		return gencore.RoleAssignmentPage{}, mapErr(err, errCtx{UnitID: unitIdArg})
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

func (s *SuperAdminService) GrantUnitRole(ctx context.Context, _ bearertoken.Token, requestArg gencore.GrantUnitRoleRequest) error {
	if err := s.app.GrantUnitRole(ctx, requestArg.PersonId, requestArg.RoleId, requestArg.UnitId); err != nil {
		return mapErr(err, errCtx{PersonID: requestArg.PersonId, UnitID: requestArg.UnitId})
	}
	return nil
}

func (s *SuperAdminService) RevokeRoleAssignment(ctx context.Context, _ bearertoken.Token, assignmentIdArg string) error {
	if err := s.app.RevokeRoleAssignment(ctx, assignmentIdArg); err != nil {
		return mapErr(err, errCtx{AssignmentID: assignmentIdArg})
	}
	return nil
}

func (s *SuperAdminService) ListInstanceAdmins(ctx context.Context, _ bearertoken.Token) (gencore.InstanceAdminPage, error) {
	admins, err := s.app.ListInstanceAdmins(ctx)
	if err != nil {
		return gencore.InstanceAdminPage{}, mapErr(err, errCtx{})
	}
	out := make([]gencore.InstanceAdminGrant, len(admins))
	for i, a := range admins {
		out[i] = gencore.InstanceAdminGrant{Id: a.ID, PersonId: a.PersonID, PersonName: a.PersonName, GrantedAt: datetime.DateTime(a.GrantedAt)}
	}
	return gencore.InstanceAdminPage{Admins: out}, nil
}

func (s *SuperAdminService) GrantInstanceAdmin(ctx context.Context, _ bearertoken.Token, requestArg gencore.GrantInstanceAdminRequest) (gencore.InstanceAdminGrant, error) {
	grant, err := s.app.GrantInstanceAdmin(ctx, requestArg.PersonId)
	if err != nil {
		return gencore.InstanceAdminGrant{}, mapErr(err, errCtx{PersonID: requestArg.PersonId})
	}
	return gencore.InstanceAdminGrant{Id: grant.ID, PersonId: grant.PersonID, PersonName: grant.PersonName, GrantedAt: datetime.DateTime(grant.GrantedAt)}, nil
}

func (s *SuperAdminService) RevokeInstanceAdmin(ctx context.Context, _ bearertoken.Token, personIdArg string) error {
	if err := s.app.RevokeInstanceAdmin(ctx, personIdArg); err != nil {
		return mapErr(err, errCtx{PersonID: personIdArg})
	}
	return nil
}

func (s *SuperAdminService) GetAccountStatus(ctx context.Context, _ bearertoken.Token, personIdArg string) (gencore.AccountStatus, error) {
	status, err := s.app.GetAccountStatus(ctx, personIdArg)
	if err != nil {
		return gencore.AccountStatus{}, mapErr(err, errCtx{PersonID: personIdArg})
	}
	return gencore.AccountStatus{PersonId: status.PersonID, Status: status.Status}, nil
}

func (s *SuperAdminService) DeactivateAccount(ctx context.Context, _ bearertoken.Token, personIdArg string) (gencore.AccountStatus, error) {
	status, err := s.app.DeactivateAccount(ctx, personIdArg)
	if err != nil {
		return gencore.AccountStatus{}, mapErr(err, errCtx{PersonID: personIdArg})
	}
	return gencore.AccountStatus{PersonId: status.PersonID, Status: status.Status}, nil
}

func (s *SuperAdminService) ReactivateAccount(ctx context.Context, _ bearertoken.Token, personIdArg string) (gencore.AccountStatus, error) {
	status, err := s.app.ReactivateAccount(ctx, personIdArg)
	if err != nil {
		return gencore.AccountStatus{}, mapErr(err, errCtx{PersonID: personIdArg})
	}
	return gencore.AccountStatus{PersonId: status.PersonID, Status: status.Status}, nil
}

// ListAuditLog uses the same real keyset-pagination trick as moderation's ListReports (M7,
// docs/modules/hardening.md): query pageSize+1 rows, trim the extra one, encode its cursor as
// nextPageToken — so the caller learns whether a next page exists with no second round trip.
func (s *SuperAdminService) ListAuditLog(
	ctx context.Context, _ bearertoken.Token,
	actorPersonIdArg, targetKindArg, targetIdArg *string,
	fromArg, toArg *datetime.DateTime,
	pageSizeArg *int, pageTokenArg *string,
) (gencore.AuditLogPage, error) {
	var after *auditlogdomain.PageCursor
	if pageTokenArg != nil {
		c, err := decodeAuditCursor(*pageTokenArg)
		if err != nil {
			return gencore.AuditLogPage{}, gencore.NewInvalidPageToken()
		}
		after = &c
	}
	filter := application.AuditLogFilter{
		ActorPersonID: derefStr(actorPersonIdArg),
		TargetKind:    derefStr(targetKindArg),
		TargetID:      derefStr(targetIdArg),
	}
	if fromArg != nil {
		t := time.Time(*fromArg)
		filter.From = &t
	}
	if toArg != nil {
		t := time.Time(*toArg)
		filter.To = &t
	}
	pageSize := auditLogPageSizeOrDefault(pageSizeArg)
	entries, err := s.app.ListAuditLog(ctx, filter, pageSize+1, after)
	if err != nil {
		return gencore.AuditLogPage{}, mapErr(err, errCtx{})
	}
	var nextToken *string
	if len(entries) > pageSize {
		last := entries[pageSize-1]
		t := encodeAuditCursor(auditlogdomain.PageCursor{CreatedAt: last.CreatedAt, ID: last.ID})
		nextToken = &t
		entries = entries[:pageSize]
	}
	out := make([]gencore.AuditLogEntry, len(entries))
	for i, e := range entries {
		out[i] = toAPIAuditLogEntry(e)
	}
	return gencore.AuditLogPage{Entries: out, NextPageToken: nextToken}, nil
}

func auditLogPageSizeOrDefault(p *int) int {
	if p == nil || *p <= 0 {
		return defaultAuditLogPageSize
	}
	if *p > maxAuditLogPageSize {
		return maxAuditLogPageSize
	}
	return *p
}

func toAPIAuditLogEntry(e auditlogdomain.Entry) gencore.AuditLogEntry {
	entry := gencore.AuditLogEntry{
		Id: e.ID, ActorPersonId: optionalStr(e.ActorPersonID), ActorPersonName: optionalStr(e.ActorPersonName),
		Action: e.Action, TargetKind: e.TargetKind, TargetId: e.TargetID, CreatedAt: datetime.DateTime(e.CreatedAt),
	}
	if len(e.Before) > 0 {
		var v any = json.RawMessage(e.Before)
		entry.Before = &v
	}
	if len(e.After) > 0 {
		var v any = json.RawMessage(e.After)
		entry.After = &v
	}
	return entry
}
