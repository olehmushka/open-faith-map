// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

type fakeGrantStore struct {
	admins map[string]bool
}

func (f fakeGrantStore) IsActiveInstanceAdmin(_ context.Context, personID string) (bool, error) {
	return f.admins[personID], nil
}
func (f fakeGrantStore) ActiveGrantsForSubject(context.Context, string) ([]domain.ActiveGrant, error) {
	return nil, nil
}
func (f fakeGrantStore) InsertRoleAssignment(context.Context, string, string, string, string) (string, error) {
	return "", nil
}
func (f fakeGrantStore) ListRoles(context.Context) ([]domain.Role, error) { return nil, nil }
func (f fakeGrantStore) ListRoleAssignmentsByUnit(context.Context, string) ([]domain.RoleAssignment, error) {
	return nil, nil
}
func (f fakeGrantStore) RevokeRoleAssignment(context.Context, string, string) (domain.RevokedRoleAssignment, error) {
	return domain.RevokedRoleAssignment{}, nil
}
func (f fakeGrantStore) ListInstanceAdmins(context.Context) ([]domain.InstanceAdminGrant, error) {
	return nil, nil
}
func (f fakeGrantStore) InsertInstanceAdmin(context.Context, string, string) (string, error) {
	return "", nil
}
func (f fakeGrantStore) RevokeInstanceAdmin(context.Context, string, string) (domain.RevokedInstanceAdminGrant, error) {
	return domain.RevokedInstanceAdminGrant{}, nil
}

type noopClosure struct{}

func (noopClosure) IsAncestorOrSelf(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (noopClosure) IsAuthorityBearing(context.Context, string) (bool, error) { return false, nil }

func TestRequireInstanceAdminDeniesBeforeNext(t *testing.T) {
	svc := authz.NewService(domain.NewPDP(noopClosure{}), fakeGrantStore{})
	mw := RequireInstanceAdmin(svc)

	ctx := authz.NewContext(context.Background(), authz.Subject{PersonID: "p1"}) // not an admin
	req := httptest.NewRequest(http.MethodGet, "/core/v1/super-admin/people", nil).WithContext(ctx)
	rw := httptest.NewRecorder()

	var nextCalled bool
	mw(rw, req, wrouter.RequestVals{}, func(http.ResponseWriter, *http.Request, wrouter.RequestVals) {
		nextCalled = true
	})

	if nextCalled {
		t.Error("next was called for a non-admin subject, want denied before next")
	}
	if rw.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusForbidden)
	}
}

func TestRequireInstanceAdminAllowsRealInstanceAdmin(t *testing.T) {
	svc := authz.NewService(domain.NewPDP(noopClosure{}), fakeGrantStore{admins: map[string]bool{"p1": true}})
	mw := RequireInstanceAdmin(svc)

	ctx := authz.NewContext(context.Background(), authz.Subject{PersonID: "p1"})
	req := httptest.NewRequest(http.MethodGet, "/core/v1/super-admin/people", nil).WithContext(ctx)
	rw := httptest.NewRecorder()

	var nextCalled bool
	mw(rw, req, wrouter.RequestVals{}, func(http.ResponseWriter, *http.Request, wrouter.RequestVals) {
		nextCalled = true
	})

	if !nextCalled {
		t.Error("next was not called for a real instance admin, want allowed")
	}
}

func TestRequireInstanceAdminDeniesAnonymous(t *testing.T) {
	svc := authz.NewService(domain.NewPDP(noopClosure{}), fakeGrantStore{})
	mw := RequireInstanceAdmin(svc)

	req := httptest.NewRequest(http.MethodGet, "/core/v1/super-admin/people", nil) // no subject in context
	rw := httptest.NewRecorder()

	var nextCalled bool
	mw(rw, req, wrouter.RequestVals{}, func(http.ResponseWriter, *http.Request, wrouter.RequestVals) {
		nextCalled = true
	})

	if nextCalled {
		t.Error("next was called with no subject in context, want denied before next")
	}
	if rw.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusForbidden)
	}
}
