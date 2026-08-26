// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/olehmushka/open-faith-map/internal/authz/domain"
)

type fakeStore struct {
	admins map[string]bool
	grants map[string][]domain.ActiveGrant
}

func (f fakeStore) IsActiveInstanceAdmin(_ context.Context, personID string) (bool, error) {
	return f.admins[personID], nil
}

func (f fakeStore) ActiveGrantsForSubject(_ context.Context, personID string) ([]domain.ActiveGrant, error) {
	return f.grants[personID], nil
}

func (f fakeStore) InsertRoleAssignment(_ context.Context, _, _, _, _, _, _ string, _ *time.Time) (string, error) {
	return "", nil
}

func (f fakeStore) UpsertRoleAssignment(_ context.Context, _, _, _, _, _, _ string, _ *time.Time) (string, error) {
	return "", nil
}

func (f fakeStore) ListRoles(context.Context) ([]domain.Role, error) { return nil, nil }

func (f fakeStore) ListRoleAssignmentsByUnit(context.Context, string) ([]domain.RoleAssignment, error) {
	return nil, nil
}

func (f fakeStore) ListRoleAssignmentsByPerson(context.Context, string) ([]domain.RoleAssignment, error) {
	return nil, nil
}

func (f fakeStore) RevokeRoleAssignment(context.Context, string, string) (domain.RevokedRoleAssignment, error) {
	return domain.RevokedRoleAssignment{}, nil
}

func (f fakeStore) ClearRoleAssignmentExpiry(context.Context, string) (domain.RevokedRoleAssignment, error) {
	return domain.RevokedRoleAssignment{}, nil
}

func (f fakeStore) ListInstanceAdmins(context.Context) ([]domain.InstanceAdminGrant, error) {
	return nil, nil
}

func (f fakeStore) InsertInstanceAdmin(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (f fakeStore) RevokeInstanceAdmin(_ context.Context, _, _ string) (domain.RevokedInstanceAdminGrant, error) {
	return domain.RevokedInstanceAdminGrant{}, nil
}

type noopClosure struct{}

func (noopClosure) IsAncestorOrSelf(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (noopClosure) IsAuthorityBearing(context.Context, string) (bool, error) { return false, nil }

func TestServiceRequireDeniesAbsentSubject(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{}, nil)
	err := svc.Require(context.Background(), domain.PermUnitRead, "unit-a")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("Require with no subject in context = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequireAllowsInstanceAdmin(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{admins: map[string]bool{"p1": true}}, nil)
	ctx := NewContext(context.Background(), Subject{PersonID: "p1"})
	if err := svc.Require(ctx, domain.PermInstanceAdminManage, "unit-a"); err != nil {
		t.Errorf("Require for instance admin = %v, want nil", err)
	}
}

func TestServiceRequireDeniesNonAdminInstanceScope(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{}, nil)
	ctx := NewContext(context.Background(), Subject{PersonID: "p1"})
	err := svc.Require(ctx, domain.PermInstanceAdminManage, "unit-a")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("Require = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequirePanicsOnSystemContext(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{}, nil)
	defer func() {
		if recover() == nil {
			t.Error("Require with a SystemContext did not panic")
		}
	}()
	_ = svc.Require(SystemContext(context.Background()), domain.PermUnitRead, "unit-a")
}

func TestServiceRequireInstanceAdminDeniesAbsentSubject(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{}, nil)
	err := svc.RequireInstanceAdmin(context.Background())
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("RequireInstanceAdmin with no subject in context = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequireInstanceAdminDeniesNonAdmin(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{}, nil)
	ctx := NewContext(context.Background(), Subject{PersonID: "p1"})
	if err := svc.RequireInstanceAdmin(ctx); !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("RequireInstanceAdmin for a non-admin = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequireInstanceAdminAllowsInstanceAdmin(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{admins: map[string]bool{"p1": true}}, nil)
	ctx := NewContext(context.Background(), Subject{PersonID: "p1"})
	if err := svc.RequireInstanceAdmin(ctx); err != nil {
		t.Errorf("RequireInstanceAdmin for an instance admin = %v, want nil", err)
	}
}

func TestServiceRequireInstanceAdminPanicsOnSystemContext(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{}, nil)
	defer func() {
		if recover() == nil {
			t.Error("RequireInstanceAdmin with a SystemContext did not panic")
		}
	}()
	_ = svc.RequireInstanceAdmin(SystemContext(context.Background()))
}

// -------------------------------------------------------------------- M11.9: API-key subject guards

func TestServiceRequireAllowsAPIKeyWithinAllowlistAndLiveGrants(t *testing.T) {
	store := fakeStore{grants: map[string][]domain.ActiveGrant{
		"p1": {{TargetUnitID: "unit-a", Scope: domain.ScopeUnit, Perms: map[domain.Permission]struct{}{domain.PermPersonRead: {}}}},
	}}
	svc := NewService(domain.NewPDP(noopClosure{}), store, nil)
	ctx := NewContext(context.Background(), Subject{PersonID: "p1", APIKeyPermissionCodes: []string{string(domain.PermPersonRead)}})
	if err := svc.Require(ctx, domain.PermPersonRead, "unit-a"); err != nil {
		t.Errorf("Require for an API key in-allowlist and in-grants = %v, want nil", err)
	}
}

func TestServiceRequireDeniesAPIKeyOutsideAllowlist(t *testing.T) {
	store := fakeStore{grants: map[string][]domain.ActiveGrant{
		"p1": {{TargetUnitID: "unit-a", Scope: domain.ScopeUnit, Perms: map[domain.Permission]struct{}{
			domain.PermPersonRead: {}, domain.PermPersonUpdate: {},
		}}},
	}}
	svc := NewService(domain.NewPDP(noopClosure{}), store, nil)
	// The owner actually holds PermPersonUpdate too, but the key's own allowlist never granted it —
	// the allowlist bounds even an owner with broader live grants.
	ctx := NewContext(context.Background(), Subject{PersonID: "p1", APIKeyPermissionCodes: []string{string(domain.PermPersonRead)}})
	err := svc.Require(ctx, domain.PermPersonUpdate, "unit-a")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("Require for an API key outside its allowlist = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequireDeniesAPIKeyInAllowlistButOutsideLiveGrants(t *testing.T) {
	// Allowlist is deliberately broader than what the owner currently holds — the intersection's
	// second half (the unmodified enforce/decide path) still applies.
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{}, nil)
	ctx := NewContext(context.Background(), Subject{PersonID: "p1", APIKeyPermissionCodes: []string{string(domain.PermPersonRead)}})
	err := svc.Require(ctx, domain.PermPersonRead, "unit-a")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("Require for an API key in-allowlist but with no live grant = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequireDeniesEmptyAllowlistAPIKey(t *testing.T) {
	store := fakeStore{grants: map[string][]domain.ActiveGrant{
		"p1": {{TargetUnitID: "unit-a", Scope: domain.ScopeUnit, Perms: map[domain.Permission]struct{}{domain.PermPersonRead: {}}}},
	}}
	svc := NewService(domain.NewPDP(noopClosure{}), store, nil)
	// Non-nil but empty: a legitimately zero-permission key, not "unset".
	ctx := NewContext(context.Background(), Subject{PersonID: "p1", APIKeyPermissionCodes: []string{}})
	err := svc.Require(ctx, domain.PermPersonRead, "unit-a")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("Require for an empty-allowlist API key = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequireInstanceAdminDeniesAPIKeyEvenForActualAdmin(t *testing.T) {
	// p1 genuinely is an instance admin, but the request came in via an API key — the instance-admin
	// plane must never be reachable through a key, allowlist or not.
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{admins: map[string]bool{"p1": true}}, nil)
	ctx := NewContext(context.Background(), Subject{PersonID: "p1", APIKeyPermissionCodes: []string{string(domain.PermInstanceAdminManage)}})
	err := svc.RequireInstanceAdmin(ctx)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("RequireInstanceAdmin for an API-key subject (even an admin owner) = %v, want ErrPermissionDenied", err)
	}
}

func TestStripSystemMarkerRemovesTheMarker(t *testing.T) {
	ctx := SystemContext(context.Background())
	if !isSystemContext(ctx) {
		t.Fatal("SystemContext did not mark ctx")
	}
	stripped := StripSystemMarker(ctx)
	if isSystemContext(stripped) {
		t.Error("StripSystemMarker left the marker in place")
	}
}
