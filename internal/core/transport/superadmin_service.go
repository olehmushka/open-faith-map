// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"

	gencore "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/core"
	"github.com/olehmushka/open-faith-map/internal/core/application"
	"github.com/palantir/pkg/bearertoken"
	"github.com/palantir/pkg/datetime"
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
