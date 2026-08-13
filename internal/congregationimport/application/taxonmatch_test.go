// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

func TestFindTaxonAliasMatch(t *testing.T) {
	baptist := domain.TaxonAlias{AliasText: "баптист", TaxonID: "taxon-baptist"}
	ugcc := domain.TaxonAlias{AliasText: "греко-католицької", TaxonID: "taxon-ugcc"}
	sourceScoped := domain.TaxonAlias{SourceCode: strPtr("ua-edr"), AliasText: "свідків єгови", TaxonID: "taxon-source-scoped"}
	globalOverride := domain.TaxonAlias{AliasText: "свідків єгови", TaxonID: "taxon-global"}

	tests := []struct {
		name        string
		hint        string
		aliases     []domain.TaxonAlias
		wantID      string
		wantMatched bool
	}{
		{
			name:        "substring match against a full legal name",
			hint:        normalizeAlias("РЕЛІГІЙНА ГРОМАДА ЄВАНГЕЛЬСЬКИХ ХРИСТИЯН-БАПТИСТІВ \"БЛАГОДАТЬ\""),
			aliases:     []domain.TaxonAlias{baptist, ugcc},
			wantID:      "taxon-baptist",
			wantMatched: true,
		},
		{
			name:        "case-insensitive because both sides are normalized before comparison",
			hint:        normalizeAlias("...ЛЬВІВСЬКОЇ АРХІЄПАРХІЇ УКРАЇНСЬКОЇ ГРЕКО-КАТОЛИЦЬКОЇ ЦЕРКВИ"),
			aliases:     []domain.TaxonAlias{baptist, ugcc},
			wantID:      "taxon-ugcc",
			wantMatched: true,
		},
		{
			name:        "no aliases at all",
			hint:        normalizeAlias("anything"),
			aliases:     nil,
			wantMatched: false,
		},
		{
			name:        "empty hint never matches",
			hint:        "",
			aliases:     []domain.TaxonAlias{baptist},
			wantMatched: false,
		},
		{
			name:        "no alias substring present",
			hint:        normalizeAlias("ТОВАРИСТВО З ОБМЕЖЕНОЮ ВІДПОВІДАЛЬНІСТЮ"),
			aliases:     []domain.TaxonAlias{baptist, ugcc},
			wantMatched: false,
		},
		{
			name:        "a source-scoped alias earlier in the list wins over a global one later",
			hint:        normalizeAlias("РЕЛІГІЙНА ГРОМАДА СВІДКІВ ЄГОВИ"),
			aliases:     []domain.TaxonAlias{sourceScoped, globalOverride},
			wantID:      "taxon-source-scoped",
			wantMatched: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotMatched := findTaxonAliasMatch(tt.hint, tt.aliases)
			if gotMatched != tt.wantMatched || (tt.wantMatched && gotID != tt.wantID) {
				t.Errorf("findTaxonAliasMatch(%q, ...) = (%q, %v), want (%q, %v)", tt.hint, gotID, gotMatched, tt.wantID, tt.wantMatched)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
