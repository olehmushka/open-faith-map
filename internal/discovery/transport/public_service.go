// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated discovery.DiscoveryPublicService and
// discovery.DiscoveryService (Conjure server interfaces).
package transport

import (
	"context"

	gendiscovery "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/discovery"
	"github.com/olehmushka/open-faith-map/internal/discovery/application"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
	"github.com/palantir/pkg/datetime"
)

// PublicService implements DiscoveryPublicService — genuinely anonymous, no bearertoken.Token
// parameter anywhere (D-AdminSurface: openfaithmap-web holds no session to forward).
type PublicService struct {
	appService *application.Service
}

func NewPublicService(appService *application.Service) *PublicService {
	return &PublicService{appService: appService}
}

var _ gendiscovery.DiscoveryPublicService = (*PublicService)(nil)

func (s *PublicService) Search(ctx context.Context, latArg *float64, lngArg *float64, radiusMArg *float64, traditionArg *string, languageArg *string, dayOfWeekArg *int, queryArg *string) (gendiscovery.SearchResult, error) {
	rows, err := s.appService.Search(ctx, domain.SearchQuery{
		Lat:       latArg,
		Lng:       lngArg,
		RadiusM:   radiusMArg,
		Tradition: traditionArg,
		Language:  languageArg,
		DayOfWeek: dayOfWeekArg,
		Query:     queryArg,
	})
	if err != nil {
		return gendiscovery.SearchResult{}, err
	}
	return gendiscovery.SearchResult{Sites: toAPISites(rows)}, nil
}

func toAPISites(rows []domain.CacheRow) []gendiscovery.DiscoverySite {
	out := make([]gendiscovery.DiscoverySite, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPISite(r))
	}
	return out
}

func toAPISite(r domain.CacheRow) gendiscovery.DiscoverySite {
	languages := r.ServiceLanguages
	if languages == nil {
		languages = []string{}
	}
	days := r.ServiceDays
	if days == nil {
		days = []int{}
	}
	return gendiscovery.DiscoverySite{
		Id:                  r.ID,
		ReligionSiteRid:     r.ReligionSiteRID,
		CongregationUnitRid: r.CongregationUnitRID,
		ContentSiteId:       r.ContentSiteID,
		Latitude:            r.Latitude,
		Longitude:           r.Longitude,
		TraditionTaxonId:    r.TraditionTaxonID,
		ServiceLanguages:    languages,
		ServiceDays:         days,
		RefreshedAt:         datetime.DateTime(r.RefreshedAt),
	}
}
