// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

// moderatePermission is platform-moderator's underlying internal/authz permission (D-PlatformModerator
// — the ADR deliberately leaves the actual permission choice to M5's own scoping). unit.lifecycle is
// not already held by registration-operator or congregation-admin (migrations/0022_core_seed.sql), so
// the two roles stay distinguishable at the PDP — unlike reusing religionorg.manage again, which every
// other module's operator/manage gate already relies on for its own distinct purpose. It's also the
// closest existing semantic fit: moderation's own suspend/archive action kinds parallel a unit's
// lifecycle state.
const moderatePermission = authzdomain.PermUnitLifecycle

// requireModerate asks internal/authz's PDP whether the request's subject (from ctx) holds
// moderatePermission on Config.RootUnitID specifically — the same target-scoped pattern
// registration's IsOperator/content's requireManage/discovery's requireOperator already use.
// moderation.read and moderation.act (docs/modules/moderation.md) both resolve to this one check —
// there is no second permission distinguishing read from act in this design, matching how
// content/discovery already collapse their own read/manage distinctions to one PDP call.
func (s *Service) requireModerate(ctx context.Context) error {
	if err := s.authzSvc.Require(ctx, moderatePermission, s.cfg.RootUnitID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

// congregationAdminPermission mirrors content's own managePermission — the same religionorg.manage
// grant congregation-admin holds on its own unit, reused here (not moderatePermission) because
// filing an appeal is an act of the AFFECTED congregation's admin, never a moderator's own standing.
const congregationAdminPermission = authzdomain.PermReligionOrgManage

// requireCongregationAdmin asks whether the request's subject (from ctx) holds
// congregationAdminPermission on unitRID specifically — the same target-scoped pattern as
// requireModerate, just checked against a single congregation's unit instead of the shared root.
func (s *Service) requireCongregationAdmin(ctx context.Context, unitRID string) error {
	if err := s.authzSvc.Require(ctx, congregationAdminPermission, unitRID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}
