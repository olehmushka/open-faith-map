// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

// moderatePermission is platform-moderator's underlying internal/authz permission. D-PlatformModerator's
// M5 addendum originally reused unit.lifecycle here, since go-oikumenea's permission catalog was closed
// pre-port and couldn't mint a new moderation.* code. M12.0 splits this out into its own dedicated
// PermModerationStanding: that constraint is gone now that the catalog is this repo's own Go code
// (D-InProcessAuthz), and reusing unit.lifecycle would have meant every platform-moderator silently
// gained M12.1's real setUnitState/deleteUnit power over every unit under root, as a side effect of two
// unrelated features sharing one permission code — not an intended widening of moderator authority.
const moderatePermission = authzdomain.PermModerationStanding

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
