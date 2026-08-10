// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"

	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/authorization"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// managePermission is content.manage's underlying go-oikumenea permission (M3, D-PlatformModerator's
// pattern applied to content). Reused, not newly minted: congregation-admin already holds
// religionorg.manage on its own unit (scripts/bootstrap-registration-org), the same permission
// registration's own operator gate reuses — just checked against a different unit (a specific
// site's congregation unit, never the shared root).
const managePermission = "religionorg.manage"

// requireManage asks go-oikumenea's real PDP (Authorize) whether callerPersonID holds
// managePermission specifically on unitRID — never an untargeted "holds it anywhere" check
// (architecture/conventions.md, D-PlatformModerator). Authorize itself requires the caller to
// already hold assignment.read reaching unitRID, no self-exemption — congregation-admin's role
// grants that (scripts/bootstrap-registration-org, M3's own fix to that role; see its comment for
// the M2.3 precedent this mirrors).
func (s *Service) requireManage(ctx context.Context, token, callerPersonID, unitRID string) error {
	c, err := s.client(token)
	if err != nil {
		return err
	}
	resp, err := c.Authorization.Authorize(ctx, authorization.AuthorizeRequest{
		SubjectPersonId: callerPersonID,
		Action:          managePermission,
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
