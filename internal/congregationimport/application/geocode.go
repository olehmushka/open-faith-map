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

	country := s.resolveCountryName(ctx, cand.CountryID)

	// Broadening fallback: real scraped addresses (ar-rnc especially — see connector.go's own doc
	// comment on its free-text "address" column) are often too specific/malformed for a structured
	// street-level match — found live against candidate 9ec02b15... (street "Sede central: Lago
	// Mascardi S/N y Paso Córdoba, Allen" + locality "Departamento General Roca", a department, not
	// the actual city) where the full query genuinely returns no match but dropping Street, then
	// Locality too, still resolves to at least a town/state-level point. Each step is strictly a
	// SUBSET of the previous query's fields, so trying them in order can only ever trade precision
	// for a match, never contradict a more specific one. Still advisory only — GeocodeResult.Precision
	// carries the provider's own place type, so the operator can see a coarse match for what it is
	// before trusting it.
	candidates := []domain.GeocodeQuery{
		{Street: cand.Street, Locality: cand.Locality, AdminArea1: cand.AdminArea1, Country: country},
		{Locality: cand.Locality, AdminArea1: cand.AdminArea1, Country: country},
		{AdminArea1: cand.AdminArea1, Country: country},
	}
	queries := dedupNonEmptyQueries(candidates)
	if len(queries) == 0 {
		return domain.GeocodeResult{}, domain.ErrGeocodeNoMatch
	}
	for _, query := range queries {
		result, err := s.geocoder.Geocode(ctx, query)
		if err != nil {
			return domain.GeocodeResult{}, fmt.Errorf("congregationimport: geocode: %w", err)
		}
		if result != nil {
			return *result, nil
		}
	}
	return domain.GeocodeResult{}, domain.ErrGeocodeNoMatch
}

// dedupNonEmptyQueries drops any query with no field set at all (a candidate missing every address
// field would otherwise send an empty request to the provider) and collapses immediate duplicates
// (e.g. Street already nil makes the first two broadening steps identical) — keeps the operator's
// single "Suggest coordinates" click to the minimum number of real, distinct rate-limited requests.
func dedupNonEmptyQueries(queries []domain.GeocodeQuery) []domain.GeocodeQuery {
	out := make([]domain.GeocodeQuery, 0, len(queries))
	for _, q := range queries {
		if q.Street == nil && q.Locality == nil && q.AdminArea1 == nil && q.Country == nil {
			continue
		}
		if len(out) > 0 && q == out[len(out)-1] {
			continue
		}
		out = append(out, q)
	}
	return out
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
