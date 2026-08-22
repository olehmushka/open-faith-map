// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"

	gencore "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/core"
	"github.com/olehmushka/open-faith-map/internal/core/application"
)

// PublicService implements gencore.CorePublicService — M11.6's genuinely anonymous surface,
// mirroring internal/content/transport.PublicService's shape. No bearertoken.Token param on its one
// method: CorePublicService carries no default-auth (api/core.conjure.yml), so Conjure's own
// generated interface doesn't ask for one, unlike every method on Service/SuperAdminService above.
type PublicService struct {
	app *application.Service
}

func NewPublicService(app *application.Service) *PublicService {
	return &PublicService{app: app}
}

var _ gencore.CorePublicService = (*PublicService)(nil)

func (s *PublicService) ResolveInvite(ctx context.Context, requestArg gencore.ResolveInviteRequest) (gencore.InviteInfo, error) {
	info, err := s.app.ResolveInvite(ctx, requestArg.Token)
	if err != nil {
		return gencore.InviteInfo{}, mapErr(err, errCtx{})
	}
	return gencore.InviteInfo{DisplayName: info.DisplayName, Email: info.Email}, nil
}
