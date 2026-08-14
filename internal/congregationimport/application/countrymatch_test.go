// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"

	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/geo"
)

func TestFindCountryMatch(t *testing.T) {
	argentina := geo.Country{Id: "country-ar", Name: map[string]string{"eng": "Argentina", "spa": "Argentina"}}
	ukraine := geo.Country{Id: "country-ua", Name: map[string]string{"eng": "Ukraine", "ukr": "Україна"}}
	countries := []geo.Country{argentina, ukraine}

	tests := []struct {
		name        string
		hint        string
		wantID      string
		wantMatched bool
	}{
		{
			name:        "exact name match, normalized",
			hint:        normalizeAlias("Argentina"),
			wantID:      "country-ar",
			wantMatched: true,
		},
		{
			name:        "matches against any configured locale, not just eng",
			hint:        normalizeAlias("Україна"),
			wantID:      "country-ua",
			wantMatched: true,
		},
		{
			name:        "a substring is not enough — unlike taxon/jurisdiction hints",
			hint:        normalizeAlias("Argentine Republic"),
			wantMatched: false,
		},
		{
			name:        "empty hint never matches",
			hint:        "",
			wantMatched: false,
		},
		{
			name:        "no country configured at all",
			hint:        normalizeAlias("Brazil"),
			wantMatched: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotMatched := findCountryMatch(tt.hint, countries)
			if gotMatched != tt.wantMatched || (tt.wantMatched && gotID != tt.wantID) {
				t.Errorf("findCountryMatch(%q, ...) = (%q, %v), want (%q, %v)", tt.hint, gotID, gotMatched, tt.wantID, tt.wantMatched)
			}
		})
	}
}
