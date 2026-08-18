// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

// moderatePermission is platform-moderator's underlying internal/authz permission
// (D-PlatformModerator, resolved at M5) — same constant value as moderation's own moderatePermission,
// duplicated here rather than imported: this repo's real, observed convention is each module holding
// its own copy of this check (content's requireManage and moderation's requireCongregationAdmin are
// already independent copies of the identical shape), not importing another module's application
// package.
const moderatePermission = authzdomain.PermUnitLifecycle

// requireModerate asks internal/authz's PDP whether the request's subject (from ctx) holds
// moderatePermission on Config.RootUnitID specifically. moderation.read and moderation.act
// (docs/modules/vouching.md) both resolve to this one check — there is no second permission
// distinguishing read from act, matching moderation's own collapse of the same distinction.
func (s *Service) requireModerate(ctx context.Context) error {
	if err := s.authzSvc.Require(ctx, moderatePermission, s.cfg.RootUnitID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

// congregationStandingPermission mirrors content's managePermission/moderation's
// congregationAdminPermission — the same religionorg.manage grant congregation-admin holds on its
// own unit.
const congregationStandingPermission = authzdomain.PermReligionOrgManage

// requireCongregationStanding asks whether the request's subject (from ctx) holds
// congregationStandingPermission on unitRID specifically — the target-scoped check createVouch uses
// to prove the caller (guarantor) administers SOME congregation, per vouching.md: "there is no
// relationship requirement between guarantor and claim." unitRID here is the caller's own
// guarantorCongregationUnitId, never the claim's own congregationUnitId.
func (s *Service) requireCongregationStanding(ctx context.Context, unitRID string) error {
	if err := s.authzSvc.Require(ctx, congregationStandingPermission, unitRID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}
