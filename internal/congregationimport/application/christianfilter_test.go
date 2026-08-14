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
		// --- Spanish cases (arrnc, Argentina's Registro Nacional de Cultos) — real names/shapes
		// found live in the actual 30,178-row export, 2026-08-14.
		{
			name: "Evangelical, unaccented — the dominant real spelling (10,055 of ~10,836 real hits)",
			in:   "IGLESIA EVANGELICA PENTECOSTAL",
			want: true,
		},
		{
			name: "Evangelical, accented — real but the minority spelling; must match identically to the unaccented form",
			in:   "IGLESIA EVANGÉLICA PENTECOSTAL",
			want: true,
		},
		{
			name: "Evangelistic ministry — real miss the short 'evangel' stem exists specifically to fix ('evangelistico' does not contain 'evangelic')",
			in:   "MINISTERIO EVANGELISTICO INTERNACIONAL JERUSALEN",
			want: true,
		},
		{
			name: "Assemblies of God — real, high-volume denomination name with no other matching keyword",
			in:   "UNION DE LAS ASAMBLEAS DE DIOS - FILIAL 672",
			want: true,
		},
		{
			name: "Baptist, real shape",
			in:   "IGLESIA BAUTISTA DE FE",
			want: true,
		},
		{
			name: "Church of the Nazarene, real denomination",
			in:   "IGLESIA DEL NAZARENO ARGENTINA",
			want: true,
		},
		{
			name: "Jehovah's Witnesses — not caught here by design, caught downstream by checkExcluded (real name)",
			in:   "ASOCIACION DE LOS TESTIGOS DE JEHOVA",
			want: false,
		},
		{
			name: "Bahá'í — a real, expected false positive for THIS filter (contains 'asamblea'), same accepted trade-off documented on the 'asamblea' keyword entry itself; still routed correctly downstream once an operator seeds a Bahá'í taxon alias, or otherwise waits in NEEDS_TAXON_REVIEW rather than being auto-rejected — asserted here to document the list's own known limits honestly, mirroring the LDS/Mormon case above",
			in:   "ASAMBLEA ESPIRITUAL DE LOS BAHAIS DE ARGENTINA - FILIAL 1",
			want: true,
		},
		{
			name: "Spiritism — real name, correctly not Christian",
			in:   "CONFEDERACION ESPIRITISTA ARGENTINA",
			want: false,
		},
		{
			name: "secular civil association, no religious markers at all",
			in:   "ASOCIACION DE BENEFICENCIA SIRIANA",
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
