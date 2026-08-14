// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"
)

func TestIsLikelyChristian(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "Orthodox parish, real ЄДР-shaped name",
			in:   "РЕЛІГІЙНА ГРОМАДА УКРАЇНСЬКОЇ ПРАВОСЛАВНОЇ ЦЕРКВИ",
			want: true,
		},
		{
			name: "Baptist congregation",
			in:   "РЕЛІГІЙНА ГРОМАДА ЄВАНГЕЛЬСЬКИХ ХРИСТИЯН-БАПТИСТІВ \"БЛАГОДАТЬ\"",
			want: true,
		},
		{
			name: "UGCC parish naming an archeparchy",
			in:   "ПАРАФІЯ СВЯТОГО ЮРІЯ ЛЬВІВСЬКОЇ АРХІЄПАРХІЇ УКРАЇНСЬКОЇ ГРЕКО-КАТОЛИЦЬКОЇ ЦЕРКВИ",
			want: true,
		},
		{
			name: "Pentecostal, straight apostrophe",
			in:   "ЦЕРКВА ХРИСТИЯН ВІРИ ЄВАНГЕЛЬСЬКОЇ П'ЯТДЕСЯТНИКІВ",
			want: true,
		},
		{
			name: "Pentecostal, curly apostrophe (U+2019)",
			in:   "ЦЕРКВА ХРИСТИЯН ВІРИ ЄВАНГЕЛЬСЬКОЇ П’ЯТДЕСЯТНИКІВ",
			want: true,
		},
		{
			name: "Pentecostal, modifier-letter apostrophe (U+02BC)",
			in:   "ЦЕРКВА ХРИСТИЯН ВІРИ ЄВАНГЕЛЬСЬКОЇ ПʼЯТДЕСЯТНИКІВ",
			want: true,
		},
		{
			name: "Orthodox parish naming an eparchy in genitive case, UPC abbreviation — real false negative found live, fixed",
			in:   "РЕЛІГІЙНА ГРОМАДА СВЯТОГО ВЕЛИКОМУЧЕНИКА ПАНТЕЛЕЙМОНА ПАРАФІЇ ДНІПРОПЕТРОВСЬКОЇ ЄПАРХІЇ УПЦ",
			want: true,
		},
		{
			name: "OCU abbreviation (ПЦУ) alone, no other keyword present",
			in:   "РЕЛІГІЙНА ГРОМАДА СВЯТО-МИКОЛАЇВСЬКА ПЦУ С. ІВАНІВКА",
			want: true,
		},
		{
			name: "Jehovah's Witnesses — not caught here by design, caught downstream by checkExcluded",
			in:   "РЕЛІГІЙНА ГРОМАДА СВІДКІВ ЄГОВИ",
			want: false,
		},
		{
			name: "LDS/Mormon — not caught here by design, caught downstream by checkExcluded",
			in:   "РЕЛІГІЙНА ГРОМАДА ЦЕРКВИ ІСУСА ХРИСТА СВЯТИХ ОСТАННІХ ДНІВ",
			want: true, // contains "христ" (Христа) — a real, expected false positive for THIS filter;
			// it's still routed correctly downstream, since LDS resolves a real taxon and
			// checkExcluded rejects it with the specific D-Exclusions reason, never reaching this
			// filter at all in the real pipeline (see service.go's placement comment). Asserted here
			// only to document the keyword list's own limits honestly, not to claim this filter alone
			// would ever see this name in production.
		},
		{
			name: "Muslim community",
			in:   "МУСУЛЬМАНСЬКА РЕЛІГІЙНА ГРОМАДА",
			want: false,
		},
		{
			name: "Jewish community",
			in:   "ЄВРЕЙСЬКА РЕЛІГІЙНА ГРОМАДА \"ХАБАД\"",
			want: false,
		},
		{
			name: "Buddhist community",
			in:   "БУДДІЙСЬКА РЕЛІГІЙНА ГРОМАДА",
			want: false,
		},
		{
			name: "secular LLC, no religious markers at all",
			in:   "ТОВАРИСТВО З ОБМЕЖЕНОЮ ВІДПОВІДАЛЬНІСТЮ \"ГАРАНТ\"",
			want: false,
		},
		{
			name: "empty name",
			in:   "",
			want: false,
		},
		{
			name: "choir, not chapel — real word-confusion risk this list deliberately avoids",
			in:   "НАРОДНА КАПЕЛА БАНДУРИСТІВ",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLikelyChristian(tt.in); got != tt.want {
				t.Errorf("isLikelyChristian(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
