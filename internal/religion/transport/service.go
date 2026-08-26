// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated religion.ReligionService (Conjure server interface):
// translates Conjure structs <-> domain types and maps domain errors to this module's typed
// Conjure errors. Religion's first transport layer (M13.2) — every prior consumer called
// application.Service in-process; this is the first direct HTTP entrypoint.
package transport

import (
	"context"

	genreligion "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/religion"
	"github.com/olehmushka/open-faith-map/internal/religion/application"
	"github.com/olehmushka/open-faith-map/internal/religion/domain"
	"github.com/palantir/pkg/bearertoken"
)

type Service struct {
	appService *application.Service
}

func NewService(appService *application.Service) *Service {
	return &Service{appService: appService}
}

var _ genreligion.ReligionService = (*Service)(nil)

func (s *Service) GetSite(ctx context.Context, authHeader bearertoken.Token, unitIdArg string) (genreligion.Site, error) {
	site, err := s.appService.GetSiteByUnit(ctx, unitIdArg)
	if err != nil {
		return genreligion.Site{}, mapErr(err)
	}
	return toAPISite(site), nil
}

func (s *Service) UpdateSiteAttributes(ctx context.Context, authHeader bearertoken.Token, unitIdArg string, requestArg genreligion.UpdateSiteAttributesRequest) (genreligion.Site, error) {
	site, err := s.appService.UpdateSiteAttributes(ctx, unitIdArg, fromAPIAttributes(requestArg.Attributes))
	if err != nil {
		return genreligion.Site{}, mapErr(err)
	}
	return toAPISite(site), nil
}

func toAPIAttributes(a domain.SiteAttributes) genreligion.SiteAttributes {
	return genreligion.SiteAttributes{
		Accessibility: genreligion.Accessibility{
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

func fromAPIAttributes(a genreligion.SiteAttributes) domain.SiteAttributes {
	return domain.SiteAttributes{
		Accessibility: domain.AccessibilityAttributes{
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

func toAPISite(s domain.Site) genreligion.Site {
	return genreligion.Site{
		Id:              s.ID,
		OrgUnitId:       s.OrgUnitID,
		LocationId:      s.LocationID,
		SiteTypeId:      s.SiteTypeID,
		SiteTypeCode:    s.SiteTypeCode,
		SiteTypeName:    s.SiteTypeName,
		Visibility:      s.Visibility,
		PublicPrecision: s.PublicPrecision,
		IsPrimary:       s.IsPrimary,
		Latitude:        s.Latitude,
		Longitude:       s.Longitude,
		Attributes:      toAPIAttributes(s.Attributes),
	}
}
