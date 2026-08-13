// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"strings"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// matchTaxon resolves a connector's free-text taxonHint to a religion_taxa RID via the
// operator-maintained congregationimport_taxon_aliases table. hint nil or matching nothing returns
// (_, false, nil) — never an error; the caller routes that to NEEDS_TAXON_REVIEW. The actual
// matching logic is findAliasMatch, a pure function split out so it's unit-testable without a live
// store (mirrors scripts/bootstrap-registration-org's own permissionsToAdd precedent).
func (s *Service) matchTaxon(ctx context.Context, sourceCode string, hint *string) (taxonID string, matched bool, err error) {
	if hint == nil {
		return "", false, nil
	}
	normalizedHint := normalizeAlias(*hint)
	if normalizedHint == "" {
		return "", false, nil
	}
	aliases, err := s.store.ListAliasesForMatching(ctx, sourceCode)
	if err != nil {
		return "", false, err
	}
	taxonID, matched = findTaxonAliasMatch(normalizedHint, aliases)
	return taxonID, matched, nil
}

// findTaxonAliasMatch checks whether any known alias appears as a SUBSTRING of the (already
// normalized) hint — hint is typically a full scraped name (e.g. ЄДР's legal entity name), not a
// short keyword, so an exact-match lookup would never fire against real scraped data.
// Source-scoped aliases are checked before global ones (aliases' own ordering, set by
// ListAliasesForMatching), so a source-specific override wins on overlap. Pure — no I/O — so it's
// directly unit-testable without a live store.
func findTaxonAliasMatch(normalizedHint string, aliases []domain.TaxonAlias) (taxonID string, matched bool) {
	for _, a := range aliases {
		if strings.Contains(normalizedHint, normalizeAlias(a.AliasText)) {
			return a.TaxonID, true
		}
	}
	return "", false
}

// normalizeAlias matches how congregationimport_taxon_aliases.alias_text is expected to be stored —
// lowercase, trimmed. Operator-entered aliases must be normalized the same way when created
// (application/provision.go's CreateTaxonAlias does this before calling the store), so a lookup and
// its matching insert always agree.
func normalizeAlias(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
