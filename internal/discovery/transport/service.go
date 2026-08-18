// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M10.6: the caller's identity no longer arrives via a per-request whoami round-trip — authorization
// is decided by internal/authz.Require against the request's context-resolved subject (populated by
// internal/identity's authenticator middleware), so this layer no longer needs it at all.
package transport

import (
	"context"

	gendiscovery "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/discovery"
	"github.com/olehmushka/open-faith-map/internal/discovery/application"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
	"github.com/palantir/pkg/bearertoken"
)

// Service implements DiscoveryService — the header-authenticated operator tool (POST /refresh).
// Never called by the public map; openfaithmap-admin only.
type Service struct {
	appService *application.Service
}

func NewService(appService *application.Service) *Service {
	return &Service{appService: appService}
}

var _ gendiscovery.DiscoveryService = (*Service)(nil)

func (s *Service) Refresh(ctx context.Context, authHeader bearertoken.Token, requestArg gendiscovery.RefreshRegionRequest) (gendiscovery.RefreshResult, error) {
	count, err := s.appService.RefreshRegion(ctx, domain.RefreshRegion{
		MinLat: requestArg.MinLat,
		MinLng: requestArg.MinLng,
		MaxLat: requestArg.MaxLat,
		MaxLng: requestArg.MaxLng,
	})
	if err != nil {
		return gendiscovery.RefreshResult{}, mapErr(err)
	}
	return gendiscovery.RefreshResult{RefreshedCount: count}, nil
}
