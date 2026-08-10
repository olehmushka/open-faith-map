// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"

	gendiscovery "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/discovery"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/discovery/application"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
	"github.com/palantir/pkg/bearertoken"
)

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
}

// Service implements DiscoveryService — the header-authenticated operator tool (POST /refresh).
// Never called by the public map; openfaithmap-admin only.
type Service struct {
	oikumeneaBaseURL  string
	oikumeneaInsecure bool
	appService        *application.Service
}

func NewService(appService *application.Service, cfg Config) *Service {
	return &Service{appService: appService, oikumeneaBaseURL: cfg.OikumeneaBaseURL, oikumeneaInsecure: cfg.OikumeneaInsecureSkipVerify}
}

var _ gendiscovery.DiscoveryService = (*Service)(nil)

// whoami resolves the caller's own go-oikumenea person RID from their forwarded token — never
// trusts a client-supplied id (mirrors internal/content/transport's identical helper).
func (s *Service) whoami(ctx context.Context, token bearertoken.Token) (string, error) {
	c, err := coreintegration.NewUserClient(s.oikumeneaBaseURL, string(token), s.oikumeneaInsecure)
	if err != nil {
		return "", err
	}
	who, err := c.IdentityFederation.Whoami(ctx)
	if err != nil {
		return "", err
	}
	return who.PersonId, nil
}

func (s *Service) Refresh(ctx context.Context, authHeader bearertoken.Token, requestArg gendiscovery.RefreshRegionRequest) (gendiscovery.RefreshResult, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gendiscovery.RefreshResult{}, err
	}
	count, err := s.appService.RefreshRegion(ctx, string(authHeader), personID, domain.RefreshRegion{
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
