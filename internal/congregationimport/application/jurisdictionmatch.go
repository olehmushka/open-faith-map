// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"strings"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// matchJurisdiction resolves a connector's free-text jurisdictionHint to a go-oikumenea jurisdiction
// Unit RID via the operator-maintained congregationimport_jurisdiction_aliases table — same
// substring-match discipline as matchTaxon, for the same reason (a hint is typically a full scraped
// legal name, e.g. one that literally names an eparchy, not a short keyword). The actual matching
// logic is findJurisdictionAliasMatch, a pure function split out so it's unit-testable without a
// live store.
//
// This is deliberately advisory only, never a status gate: D-JurisdictionUnits already decided
// jurisdiction is operator-assigned at approval time and never inferred, so a match here only
// populates Candidate.SuggestedJurisdictionUnitID for the operator's own review — it is never applied
// automatically, and an unmatched hint never blocks approval the way an unmatched taxon hint does
// (most independent-polity congregations have no jurisdiction at all, by design).
func (s *Service) matchJurisdiction(ctx context.Context, sourceCode string, hint *string) (jurisdictionUnitID string, matched bool, err error) {
	if hint == nil {
		return "", false, nil
	}
	normalizedHint := normalizeAlias(*hint)
	if normalizedHint == "" {
		return "", false, nil
	}
	aliases, err := s.store.ListJurisdictionAliasesForMatching(ctx, sourceCode)
	if err != nil {
		return "", false, err
	}
	jurisdictionUnitID, matched = findJurisdictionAliasMatch(normalizedHint, aliases)
	return jurisdictionUnitID, matched, nil
}

// findJurisdictionAliasMatch mirrors findTaxonAliasMatch exactly (taxonmatch.go) — substring match
// against the (already normalized) hint, source-scoped aliases checked before global ones. Pure —
// no I/O — so it's directly unit-testable without a live store.
func findJurisdictionAliasMatch(normalizedHint string, aliases []domain.JurisdictionAlias) (jurisdictionUnitID string, matched bool) {
	for _, a := range aliases {
		if strings.Contains(normalizedHint, normalizeAlias(a.AliasText)) {
			return a.JurisdictionUnitID, true
		}
	}
	return "", false
}
