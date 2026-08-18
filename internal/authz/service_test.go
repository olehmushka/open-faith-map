// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"errors"
	"testing"

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

type noopClosure struct{}

func (noopClosure) IsAncestorOrSelf(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (noopClosure) IsAuthorityBearing(context.Context, string) (bool, error) { return false, nil }

func TestServiceRequireDeniesAbsentSubject(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{})
	err := svc.Require(context.Background(), domain.PermUnitRead, "unit-a")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("Require with no subject in context = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequireAllowsInstanceAdmin(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{admins: map[string]bool{"p1": true}})
	ctx := NewContext(context.Background(), Subject{PersonID: "p1"})
	if err := svc.Require(ctx, domain.PermInstanceAdminManage, "unit-a"); err != nil {
		t.Errorf("Require for instance admin = %v, want nil", err)
	}
}

func TestServiceRequireDeniesNonAdminInstanceScope(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{})
	ctx := NewContext(context.Background(), Subject{PersonID: "p1"})
	err := svc.Require(ctx, domain.PermInstanceAdminManage, "unit-a")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Errorf("Require = %v, want ErrPermissionDenied", err)
	}
}

func TestServiceRequirePanicsOnSystemContext(t *testing.T) {
	svc := NewService(domain.NewPDP(noopClosure{}), fakeStore{})
	defer func() {
		if recover() == nil {
			t.Error("Require with a SystemContext did not panic")
		}
	}()
	_ = svc.Require(SystemContext(context.Background()), domain.PermUnitRead, "unit-a")
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
