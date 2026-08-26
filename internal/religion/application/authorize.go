// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// managePermission is site.manage's underlying internal/authz permission (M13.2) — a previously-
// unused catalog entry (internal/authz/domain/permissions.go), already seeded to congregation-admin
// and registration-operator on exactly the same target-scoped shape content.manage already uses
// (D-PlatformModerator), just checked against a different permission code: PermSiteManage rather
// than content's PermReligionOrgManage, since GetSiteByUnit/UpdateSiteAttributes act on
// religion_sites specifically, not the broader "manage this congregation" grant.
const managePermission = authzdomain.PermSiteManage

// requireManage asks internal/authz's PDP whether the request's subject (from ctx) holds
// managePermission specifically on unitID — never an untargeted "holds it anywhere" check
// (architecture/conventions.md, D-PlatformModerator). Religion previously carried zero
// authorization logic (every write went through some caller's own application layer); this is
// religion's first direct authenticated entrypoint of its own (M13.2), so it needs its own gate,
// mirroring internal/content/application/authorize.go's requireManage almost exactly.
func (s *Service) requireManage(ctx context.Context, unitID string) error {
	if err := s.authzSvc.Require(ctx, managePermission, unitID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}
