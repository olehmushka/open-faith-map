// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

func TestFindJurisdictionAliasMatch(t *testing.T) {
	lviv := domain.JurisdictionAlias{AliasText: "львівської архієпархії", JurisdictionUnitID: "unit-lviv-archeparchy"}
	sourceScoped := domain.JurisdictionAlias{SourceCode: strPtr("ua-edr"), AliasText: "одеської єпархії", JurisdictionUnitID: "unit-source-scoped"}
	globalOverride := domain.JurisdictionAlias{AliasText: "одеської єпархії", JurisdictionUnitID: "unit-global"}

	tests := []struct {
		name        string
		hint        string
		aliases     []domain.JurisdictionAlias
		wantID      string
		wantMatched bool
	}{
		{
			name:        "a real UGCC legal name embeds the eparchy",
			hint:        normalizeAlias("...ПАРАФІЯ СВЯТОГО ЮРА ЛЬВІВСЬКОЇ АРХІЄПАРХІЇ УКРАЇНСЬКОЇ ГРЕКО-КАТОЛИЦЬКОЇ ЦЕРКВИ"),
			aliases:     []domain.JurisdictionAlias{lviv},
			wantID:      "unit-lviv-archeparchy",
			wantMatched: true,
		},
		{
			name:        "an independent-polity name (Baptist) correctly has no jurisdiction match",
			hint:        normalizeAlias("РЕЛІГІЙНА ГРОМАДА ЄВАНГЕЛЬСЬКИХ ХРИСТИЯН-БАПТИСТІВ \"БЛАГОДАТЬ\""),
			aliases:     []domain.JurisdictionAlias{lviv},
			wantMatched: false,
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
			aliases:     []domain.JurisdictionAlias{lviv},
			wantMatched: false,
		},
		{
			name:        "a source-scoped alias earlier in the list wins over a global one later",
			hint:        normalizeAlias("...ОДЕСЬКОЇ ЄПАРХІЇ УКРАЇНСЬКОЇ ПРАВОСЛАВНОЇ ЦЕРКВИ"),
			aliases:     []domain.JurisdictionAlias{sourceScoped, globalOverride},
			wantID:      "unit-source-scoped",
			wantMatched: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotMatched := findJurisdictionAliasMatch(tt.hint, tt.aliases)
			if gotMatched != tt.wantMatched || (tt.wantMatched && gotID != tt.wantID) {
				t.Errorf("findJurisdictionAliasMatch(%q, ...) = (%q, %v), want (%q, %v)", tt.hint, gotID, gotMatched, tt.wantID, tt.wantMatched)
			}
		})
	}
}
