// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"strings"

	refdatadomain "github.com/olehmushka/open-faith-map/internal/refdata/domain"
)

// matchCountry resolves a connector's CountryHint (a plain country name, e.g. arrnc's hardcoded
// "Argentina") to a real country RID via internal/refdata — an exact name match, deliberately
// simpler than matchTaxon/matchJurisdiction's substring-against-operator-alias-table approach:
// unlike a scraped taxon/jurisdiction name, CountryHint is never a full legal name to mine a keyword
// out of, it is already exactly the country's own name. ctx is expected to already carry an
// authz.SystemContext marker (RunConnector's own doc comment) — refdata itself checks nothing, but
// the marker documents that this read has no human subject behind it, same as every other read in
// this file's own call chain.
func (s *Service) matchCountry(ctx context.Context, hint *string) (countryID string, matched bool, err error) {
	if hint == nil {
		return "", false, nil
	}
	normalizedHint := normalizeAlias(*hint)
	if normalizedHint == "" {
		return "", false, nil
	}
	countries, err := s.refdata.ListCountries(ctx)
	if err != nil {
		return "", false, err
	}
	countryID, matched = findCountryMatch(normalizedHint, countries)
	return countryID, matched, nil
}

// findCountryMatch mirrors findTaxonAliasMatch's own "pure function split out for testability"
// shape — an exact (not substring) match against any of a country's locale->name values, since
// (unlike a taxon/jurisdiction hint) CountryHint is already exactly a country's own name, never a
// longer string to mine a keyword out of. Pure — no I/O — so it's directly unit-testable without a
// live database.
func findCountryMatch(normalizedHint string, countries []refdatadomain.Country) (countryID string, matched bool) {
	for _, country := range countries {
		for _, name := range country.Names {
			if strings.ToLower(strings.TrimSpace(name)) == normalizedHint {
				return country.ID, true
			}
		}
	}
	return "", false
}
