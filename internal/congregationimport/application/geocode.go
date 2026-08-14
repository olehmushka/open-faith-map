// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// SuggestCoordinates looks up approximate coordinates for a staged candidate's address via the
// configured Geocoder — ADVISORY ONLY, mirrors matchJurisdiction's own "never auto-applied"
// invariant exactly: this never writes to the store. The operator must still call EditCandidate to
// actually persist Latitude/Longitude, the same way a matched suggestedJurisdictionUnitId still
// requires an explicit jurisdictionUnitId on ApproveCandidate.
func (s *Service) SuggestCoordinates(ctx context.Context, token, callerPersonID, id string) (domain.GeocodeResult, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return domain.GeocodeResult{}, err
	}
	if s.geocoder == nil {
		return domain.GeocodeResult{}, fmt.Errorf("congregationimport: no geocoding provider configured")
	}

	cand, err := s.store.GetCandidate(ctx, id)
	if err != nil {
		return domain.GeocodeResult{}, err
	}

	query := domain.GeocodeQuery{
		Street:     cand.Street,
		Locality:   cand.Locality,
		AdminArea1: cand.AdminArea1,
		Country:    s.resolveCountryName(ctx, cand.CountryID),
	}

	result, err := s.geocoder.Geocode(ctx, query)
	if err != nil {
		return domain.GeocodeResult{}, fmt.Errorf("congregationimport: geocode: %w", err)
	}
	if result == nil {
		return domain.GeocodeResult{}, domain.ErrGeocodeNoMatch
	}
	return *result, nil
}

// resolveCountryName best-effort maps a candidate's go-oikumenea countryId RID to a real country
// name for the geocoder's structured query — read-only, so the service principal is the right
// caller (same precedent checkExcluded's ancestor walk and dedup's SearchSites already use, not
// the operator's own token). Deliberately never blocks or fails the whole lookup: a country name
// materially helps Nominatim's own structured query resolve correctly, but its absence shouldn't
// prevent a locality/state-level match from at least being attempted.
func (s *Service) resolveCountryName(ctx context.Context, countryID *string) *string {
	if countryID == nil {
		return nil
	}
	c, err := s.serviceClient(ctx)
	if err != nil {
		return nil
	}
	countries, err := c.Geo.ListCountries(ctx)
	if err != nil {
		return nil
	}
	for _, country := range countries.Countries {
		if country.Id == *countryID {
			name := country.Name["eng"]
			if name == "" {
				for _, v := range country.Name {
					name = v
					break
				}
			}
			if name == "" {
				return nil
			}
			return &name
		}
	}
	return nil
}
