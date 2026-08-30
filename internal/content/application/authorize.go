// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// managePermission is content.manage's underlying internal/authz permission. As of M14.9
// (D-TenantSubdomains' U16 ruling), it is its own grantable code — migrations/0026_content_manage_permission.sql
// grants it to congregation-admin only — rather than a byproduct of registration-operator's
// religionorg.manage subtree grant on the shared root, which is what let any operator edit any
// congregation's site through M14.8. Same shape as internal/religion/application/authorize.go's
// own site.manage check (M13.2).
const managePermission = authzdomain.PermContentManage

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

// catalogManagePermission is content.catalog.manage's underlying internal/authz permission
// (M14.13, D-SitePatterns): platform-moderator authority, not a new authority concept — the same
// PermModerationStanding internal/moderation already checks
// (internal/moderation/application/authorize.go's requireModerate), reused here rather than
// duplicated.
const catalogManagePermission = authzdomain.PermModerationStanding

// requireCatalogManage asks internal/authz's PDP whether the request's subject holds
// catalogManagePermission on s.cfg.RootUnitID specifically — platform-wide, never scoped to a
// congregation unit (unlike requireManage above). Mirrors
// internal/moderation/application/authorize.go's requireModerate exactly.
func (s *Service) requireCatalogManage(ctx context.Context) error {
	if err := s.authzSvc.Require(ctx, catalogManagePermission, s.cfg.RootUnitID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}
