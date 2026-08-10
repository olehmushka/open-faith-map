// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"

	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/authorization"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

// moderatePermission is platform-moderator's underlying go-oikumenea permission (D-PlatformModerator
// — the ADR deliberately leaves the actual permission choice to M5's own scoping). go-oikumenea's
// permission catalog is closed and code-defined (go-oikumenea/internal/authorization/domain/
// permissions.go); a write of an unknown code is rejected server-side, so this reuses an existing
// permission rather than minting one. unit.lifecycle is not already held by registration-operator or
// congregation-admin (scripts/bootstrap-registration-org), so the two roles stay distinguishable at
// the PDP — unlike reusing religionorg.manage again, which every other module's operator/manage gate
// already relies on for its own distinct purpose. It's also the closest existing semantic fit:
// moderation's own suspend/archive action kinds parallel a unit's lifecycle state.
const moderatePermission = "unit.lifecycle"

// requireModerate asks go-oikumenea's real PDP (Authorize) whether callerPersonID holds
// moderatePermission on Config.RootUnitID specifically — the same target-scoped pattern
// registration's IsOperator/content's requireManage/discovery's requireOperator already use.
// moderation.read and moderation.act (docs/modules/moderation.md) both resolve to this one check —
// there is no second go-oikumenea permission distinguishing read from act in this design, matching
// how content/discovery already collapse their own read/manage distinctions to one PDP call.
func (s *Service) requireModerate(ctx context.Context, token, callerPersonID string) error {
	c, err := s.userClient(token)
	if err != nil {
		return err
	}
	rootUnitID := s.cfg.RootUnitID
	resp, err := c.Authorization.Authorize(ctx, authorization.AuthorizeRequest{
		SubjectPersonId: callerPersonID,
		Action:          moderatePermission,
		UnitId:          &rootUnitID,
	})
	if err != nil {
		if authorization.IsPermissionDenied(err) {
			return domain.ErrForbidden
		}
		return err
	}
	if !resp.Allow {
		return domain.ErrForbidden
	}
	return nil
}

// congregationAdminPermission mirrors content's own managePermission — the same religionorg.manage
// grant congregation-admin holds on its own unit, reused here (not moderatePermission) because
// filing an appeal is an act of the AFFECTED congregation's admin, never a moderator's own standing.
const congregationAdminPermission = "religionorg.manage"

// requireCongregationAdmin asks whether callerPersonID holds congregationAdminPermission on
// unitRID specifically — the same target-scoped pattern as requireModerate, just checked against a
// single congregation's unit instead of the shared root.
func (s *Service) requireCongregationAdmin(ctx context.Context, token, callerPersonID, unitRID string) error {
	c, err := s.userClient(token)
	if err != nil {
		return err
	}
	resp, err := c.Authorization.Authorize(ctx, authorization.AuthorizeRequest{
		SubjectPersonId: callerPersonID,
		Action:          congregationAdminPermission,
		UnitId:          &unitRID,
	})
	if err != nil {
		if authorization.IsPermissionDenied(err) {
			return domain.ErrForbidden
		}
		return err
	}
	if !resp.Allow {
		return domain.ErrForbidden
	}
	return nil
}
