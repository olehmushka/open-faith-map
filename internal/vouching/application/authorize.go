// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"

	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/authorization"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

// moderatePermission is platform-moderator's underlying go-oikumenea permission
// (D-PlatformModerator, resolved at M5) — same constant value as moderation's own moderatePermission,
// duplicated here rather than imported: this repo's real, observed convention is each module holding
// its own copy of this check (content's requireManage and moderation's requireCongregationAdmin are
// already independent copies of the identical shape), not importing another module's application
// package.
const moderatePermission = "unit.lifecycle"

// requireModerate asks go-oikumenea's real PDP (Authorize) whether callerPersonID holds
// moderatePermission on Config.RootUnitID specifically. moderation.read and moderation.act
// (docs/modules/vouching.md) both resolve to this one check — there is no second go-oikumenea
// permission distinguishing read from act, matching moderation's own collapse of the same
// distinction.
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

// congregationStandingPermission mirrors content's managePermission/moderation's
// congregationAdminPermission — the same religionorg.manage grant congregation-admin holds on its
// own unit.
const congregationStandingPermission = "religionorg.manage"

// requireCongregationStanding asks whether callerPersonID holds congregationStandingPermission on
// unitRID specifically — the target-scoped check createVouch uses to prove the caller (guarantor)
// administers SOME congregation, per vouching.md: "there is no relationship requirement between
// guarantor and claim." unitRID here is the caller's own guarantorCongregationUnitId, never the
// claim's own congregationUnitId.
func (s *Service) requireCongregationStanding(ctx context.Context, token, callerPersonID, unitRID string) error {
	c, err := s.userClient(token)
	if err != nil {
		return err
	}
	resp, err := c.Authorization.Authorize(ctx, authorization.AuthorizeRequest{
		SubjectPersonId: callerPersonID,
		Action:          congregationStandingPermission,
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
