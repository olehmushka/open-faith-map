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
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
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

func (s *PublicService) Search(ctx context.Context, latArg *float64, lngArg *float64, radiusMArg *float64, traditionArg *string, languageArg *string, dayOfWeekArg *int, queryArg *string, accessibilityArg *string, onlineOnlyArg *bool) (gendiscovery.SearchResult, error) {
	rows, err := s.appService.Search(ctx, domain.SearchQuery{
		Lat:           latArg,
		Lng:           lngArg,
		RadiusM:       radiusMArg,
		Tradition:     traditionArg,
		Language:      languageArg,
		DayOfWeek:     dayOfWeekArg,
		Query:         queryArg,
		Accessibility: accessibilityArg,
		OnlineOnly:    onlineOnlyArg,
	})
	if err != nil {
		return gendiscovery.SearchResult{}, mapErr(err)
	}
	return gendiscovery.SearchResult{Sites: toAPISites(rows)}, nil
}

// Facets answers GET /facets (M13.1) — backs the picker UI so it never offers a filter value that
// would zero out every result.
func (s *PublicService) Facets(ctx context.Context) (gendiscovery.FacetsResult, error) {
	facets, err := s.appService.Facets(ctx)
	if err != nil {
		return gendiscovery.FacetsResult{}, mapErr(err)
	}
	return toAPIFacets(facets), nil
}

// GetSite answers the per-congregation detail page's server-rendered fetch — always live (see
// application.Service.GetSiteByUnit's own doc for why), throwing SiteNotFound rather than a bare
// 404 so the generated TS client can distinguish "no discoverable site" from a transport error.
func (s *PublicService) GetSite(ctx context.Context, unitIdArg string) (gendiscovery.DiscoverySite, error) {
	row, found, err := s.appService.GetSiteByUnit(ctx, unitIdArg)
	if err != nil {
		return gendiscovery.DiscoverySite{}, err
	}
	if !found {
		return gendiscovery.DiscoverySite{}, gendiscovery.NewSiteNotFound()
	}
	return toAPISite(row), nil
}

func toAPISites(rows []domain.CacheRow) []gendiscovery.DiscoverySite {
	out := make([]gendiscovery.DiscoverySite, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPISite(r))
	}
	return out
}

func toAPIFacets(f religiondomain.Facets) gendiscovery.FacetsResult {
	traditions := make([]gendiscovery.TraditionFacet, 0, len(f.Traditions))
	for _, t := range f.Traditions {
		traditions = append(traditions, gendiscovery.TraditionFacet{
			TaxonId: t.TaxonID, TaxonCode: t.TaxonCode, TaxonName: t.TaxonName,
		})
	}
	languages := f.Languages
	if languages == nil {
		languages = []string{}
	}
	return gendiscovery.FacetsResult{Traditions: traditions, Languages: languages}
}

func toAPIAttributes(a religiondomain.SiteAttributes) gendiscovery.SiteAttributes {
	return gendiscovery.SiteAttributes{
		Accessibility: gendiscovery.Accessibility{
			StepFreeEntrance:           a.Accessibility.StepFreeEntrance,
			AccessibleRestroom:         a.Accessibility.AccessibleRestroom,
			HearingLoop:                a.Accessibility.HearingLoop,
			SignLanguageInterpretation: a.Accessibility.SignLanguageInterpretation,
			AccessibleParking:          a.Accessibility.AccessibleParking,
			WheelchairSeating:          a.Accessibility.WheelchairSeating,
			BrailleOrLargePrint:        a.Accessibility.BrailleOrLargePrint,
		},
		OnlineStream: a.OnlineStream,
	}
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
		Name:                r.Name,
		Address:             r.Address,
		TraditionTaxonId:    r.TraditionTaxonID,
		TraditionTaxonCode:  r.TraditionTaxonCode,
		TraditionTaxonName:  r.TraditionTaxonName,
		ServiceLanguages:    languages,
		ServiceDays:         days,
		Attributes:          toAPIAttributes(r.Attributes),
		RefreshedAt:         datetime.DateTime(r.RefreshedAt),
	}
}
