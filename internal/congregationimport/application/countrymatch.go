// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"strings"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/geo"
)

// matchCountry resolves a connector's CountryHint (a plain country name, e.g. arrnc's hardcoded
// "Argentina") to a real go-oikumenea country RID — an exact name match, deliberately simpler than
// matchTaxon/matchJurisdiction's substring-against-operator-alias-table approach: unlike a scraped
// taxon/jurisdiction name, CountryHint is never a full legal name to mine a keyword out of, it is
// already exactly the country's own name. Found live (2026-08-14): this hint was being computed by
// arrnc and then silently dropped — nothing ever read it, so every one of ~29.6k Argentina
// candidates needed the operator to set country by hand before SuggestCoordinates had any chance of
// resolving an address, even though the country was already a known, deterministic fact at ingest
// time.
func (s *Service) matchCountry(ctx context.Context, c *oikumenea.Client, hint *string) (countryID string, matched bool, err error) {
	if hint == nil {
		return "", false, nil
	}
	normalizedHint := normalizeAlias(*hint)
	if normalizedHint == "" {
		return "", false, nil
	}
	countries, err := c.Geo.ListCountries(ctx)
	if err != nil {
		return "", false, err
	}
	countryID, matched = findCountryMatch(normalizedHint, countries.Countries)
	return countryID, matched, nil
}

// findCountryMatch mirrors findTaxonAliasMatch's own "pure function split out for testability"
// shape — an exact (not substring) match against any of a country's locale->name values, since
// (unlike a taxon/jurisdiction hint) CountryHint is already exactly a country's own name, never a
// longer string to mine a keyword out of. Pure — no I/O — so it's directly unit-testable without a
// live go-oikumenea client.
func findCountryMatch(normalizedHint string, countries []geo.Country) (countryID string, matched bool) {
	for _, country := range countries {
		for _, name := range country.Name {
			if strings.ToLower(strings.TrimSpace(name)) == normalizedHint {
				return country.Id, true
			}
		}
	}
	return "", false
}
