// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// managePermission is content.manage's underlying internal/authz permission (M3, D-PlatformModerator's
// pattern applied to content). Reused, not newly minted: congregation-admin already holds
// religionorg.manage on its own unit (migrations/0022_core_seed.sql), the same permission
// registration's own operator gate reuses — just checked against a different unit (a specific
// site's congregation unit, never the shared root).
const managePermission = authzdomain.PermReligionOrgManage

// requireManage asks internal/authz's PDP whether the request's subject (from ctx) holds
// managePermission specifically on unitRID — never an untargeted "holds it anywhere" check
// (architecture/conventions.md, D-PlatformModerator). M10.6: no token, no client — the subject comes
// from context, resolved by internal/identity's authenticator middleware.
func (s *Service) requireManage(ctx context.Context, unitRID string) error {
	if err := s.authzSvc.Require(ctx, managePermission, unitRID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}
