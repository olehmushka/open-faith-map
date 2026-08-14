// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"strings"
)

// christianKeywords is a POSITIVE, source-agnostic pre-filter (D-Scope: OpenFaithMap is
// Christian-only, docs/architecture/decisions.md) for Ukrainian-language legal/registered names — a
// bounded allow-list, deliberately not a blacklist of every non-Christian religion/sect name
// variant (open-ended, easy to miss entries for). ~99% recall is the bar, not 100%: a false
// NEGATIVE here (a real Christian name none of these words match) is a no-op — it reaches
// NEEDS_TAXON_REVIEW exactly as it does today, same as before this filter existed. A false POSITIVE
// (a non-Christian name matches) is also harmless in the safe direction — it's just
// under-filtering, i.e. today's status quo (falls through to manual review), not a wrongful
// auto-reject. The one failure mode to actually minimize is a real Christian name matching
// nothing, so keep additions liberal, removals conservative.
//
// Word STEMS, truncated before the case-ending vowel, since Ukrainian noun/adjective declension
// means the bare nominative form is often NOT a substring of how a name actually inflects it (e.g.
// "церква" is not a substring of "церкви"/"церкві"/"церквою" — real forms a registered name will
// use — so the stem "церкв" is used instead, matching all of them). Same normalization discipline
// taxonmatch.go's normalizeAlias already established, extended for morphology, not just casing.
//
// "капел" is deliberately NOT included as a stand-in for "каплиця" (chapel) — "капела" (choir /
// musical ensemble, e.g. "народна капела") is a real, unrelated secular Ukrainian word sharing only
// a prefix, not a spelling variant of the same word. "каплиц" (the actual chapel stem) is used
// instead.
//
// Deliberately Ukrainian-only, not i18n'd — per-language lists for future connectors (Brazil,
// Argentina, OSM) are a real, separate follow-up once those connectors exist, not built
// speculatively now (this repo's own bias against unjustified infrastructure).
var christianKeywords = []string{
	"христ",        // христова, християн(и/ство/ська) — broadest positive marker
	"правосл",      // православна (Orthodox)
	"катол",        // католицька, incl. Greek-Catholic/UGCC
	"євангел",      // євангельська/євангелістська (Evangelical)
	"баптист",      // баптист(ів/ська)
	"лютеран",      // лютеранська (Lutheran)
	"англікан",     // англіканська (Anglican)
	"адвент",       // адвентист(ів/ська)
	"пятдесятник",  // Pentecostal — apostrophe already stripped before matching, see below
	"пятидесятник", // alt. spelling seen in real registrations, also apostrophe-stripped
	"харизмат",     // харизматична (charismatic movement)
	"методист",     // методист(ів/ська)
	"пресвітер",    // пресвітеріанська (Presbyterian) — also the clergy title itself, still safe
	"апостольськ",  // апостольська — common Pentecostal/independent self-description
	"церкв",        // церква/церкви/церкві/церквою/... — single broadest institutional marker
	"парафі",       // парафія/парафії/парафію/парафією (parish) — stemmed the same way as церкв
	// above; "парафія" (the bare nominative) was the ORIGINAL entry here and is a real bug, found
	// live against a real 30,721-record run: it does not match "парафії" (genitive), the form that
	// actually appears in most real registered names ("...ПАРАФІЇ ЛЬВІВСЬКОЇ АРХІЄПАРХІЇ...") —
	// exactly the declension trap this file's own doc comment already warned about for "церква",
	// just not applied consistently here on the first pass.
	"собор",  // cathedral
	"каплиц", // каплиця/каплиці/каплицю (chapel)
	"храм",   // temple/church building
	"єпархі", // єпархія/єпархії (eparchy/diocese) — Orthodox/Greek-Catholic administrative unit,
	// found missing live: real names routinely name an eparchy without also saying
	// "православна"/"католицька" in the same short legal name.
	"упц", // Українська Православна Церква — near-ubiquitous abbreviation in real registered
	// names; found missing live — spelled-out "православ" alone doesn't cover it.
	"пцу", // Православна Церква України — the other major Orthodox body's own equally common
	// abbreviation, same reasoning as УПЦ above.
}

// apostropheStripper normalizes the several real Unicode apostrophe glyphs Ukrainian orthography
// uses interchangeably for words like "п'ятдесятник" (straight ', curly ’, modifier letter ʼ) so a
// source's encoding choice can never silently produce a false negative — checked directly against
// this real failure mode, not assumed away.
var apostropheStripper = strings.NewReplacer("'", "", "’", "", "ʼ", "", "`", "")

// isLikelyChristian checks name against christianKeywords, case-insensitive substring — pure, no
// I/O, directly unit-testable, mirroring findTaxonAliasMatch's own style and reusing
// normalizeAlias's lowercase+trim exactly.
func isLikelyChristian(name string) bool {
	n := apostropheStripper.Replace(normalizeAlias(name))
	for _, kw := range christianKeywords {
		if strings.Contains(n, kw) {
			return true
		}
	}
	return false
}
