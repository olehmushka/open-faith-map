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
// Originally Ukrainian-only; extended 2026-08-14 with a Spanish-language block (arrnc, Argentina's
// Registro Nacional de Cultos — the connector this extension was built for) once that connector
// actually existed, per this file's own original plan. A single merged list stays correct with no
// per-source dispatch: the Ukrainian (Cyrillic) and Spanish (Latin) stems occupy disjoint Unicode
// ranges, so cross-language false positives are structurally impossible, not just unlikely — no
// interface change, isLikelyChristian's signature is unchanged.
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

	// --- Spanish block (arrnc, Argentina's Registro Nacional de Cultos) — every stem below was
	// checked against the real, live-downloaded 30,178-row export before being added (grep counts,
	// 2026-08-14), the same evidence-based discipline the Ukrainian block above already used.
	// Diacritics are stripped before matching (see diacriticStripper) — a real, confirmed finding:
	// "evangelica" (unaccented) outnumbers "evangélica" 10,055 to 781 in the live data, so matching
	// only the accented form would silently miss most real matches.
	"iglesia",  // church — the single broadest institutional marker (15,664 real hits)
	"cristian", // cristiano/cristiana (Christian) — broadest positive marker, mirrors "христ" above
	"evangel",  // evangélico/evangélica/evangelismo/evangelista/evangelístico — deliberately the
	// short root, not "evangelic": a real miss found live, "evangelístico" (evangelistic) does NOT
	// contain "evangelic" as a substring (...evangeli-STICO, not ...evangeli-CO), and it's a common
	// real self-description ("MINISTERIO EVANGELISTICO INTERNACIONAL...").
	"catolic",   // católica/católico, incl. Eastern/Greek-Catholic bodies still using the word
	"bautist",   // bautista (Baptist)
	"luteran",   // luterana (Lutheran)
	"anglic",    // anglicana (Anglican)
	"advent",    // adventista (Adventist)
	"pentecost", // pentecostal
	"carismat",  // carismática (charismatic movement) — rare in this source (2 real hits) but real
	"metodist",  // metodista (Methodist)
	"presbiter", // presbiteriana (Presbyterian) — also the clergy title itself, same as "пресвітер"
	"apostolic", // apostólica — common Pentecostal/independent self-description
	"congregac", // congregación (congregation)
	"parroqui",  // parroquia (parish)
	"capilla",   // chapel
	"templo",    // temple/church building
	"catedral",  // cathedral
	"diocesis",  // diócesis (diocese) — also covers "arquidiócesis" (archdiocese) as a substring,
	// no separate entry needed
	"nazaren", // nazareno/nazarena — Church of the Nazarene, a real denomination with real
	// presence in this source (231 real hits)
	"menonit",  // menonita (Mennonite)
	"reformad", // reformada (Reformed church)
	"santidad", // holiness (the holiness movement)
	"ortodox",  // ortodoxa (Orthodox)
	"mision",   // misión (mission) — a very common real self-description among independent
	// evangelical congregations in this source (4,327 real hits)
	"ministerio", // ministerio (ministry) — a near-ubiquitous real evangelical self-description
	// ("MINISTERIO EVANGELISTICO...", "AGAPE MINISTERIO INTERNACIONAL..."); a small number of real
	// Espiritista entries also contain it (confirmed live, 13 of 1,882 real hits) — the same
	// accepted safe-direction trade-off already documented above (a false positive here just means
	// one extra manual review, never a wrongful auto-reject), not a new risk category.
	"asamblea", // asamblea (assembly) — "UNION DE LAS ASAMBLEAS DE DIOS" (Assemblies of God) alone
	// accounts for hundreds of real branch rows in this source with no other matching keyword; a
	// small number of real Bahá'í ("Asamblea Espiritual...") and Espiritista entries also contain
	// it (confirmed live, 49 of 4,846 real hits) — same accepted trade-off as "ministerio" above.
	"jesus", // jesús — common in real independent-evangelical self-descriptions
	// ("ASOCIACION CIVIL JESUS VIENE", "MINISTERIO INTERNACIONAL JESÚS ES MI AYUDA"); a small
	// number of real Espiritista/rabbinical entries also contain it (confirmed live, 10 of 1,099
	// real hits) — same accepted trade-off.
}

// diacriticStripper normalizes the small, fixed set of Spanish accented characters (already
// lowercase — normalizeAlias runs first) so a source's inconsistent real-world accenting can never
// silently produce a false negative — the Spanish-language equivalent of apostropheStripper below,
// found necessary by the exact same kind of live evidence (arrnc's real export mixes "evangelica"
// and "evangélica" roughly 13-to-1, confirmed by direct count, 2026-08-14).
var diacriticStripper = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
)

// apostropheStripper normalizes the several real Unicode apostrophe glyphs Ukrainian orthography
// uses interchangeably for words like "п'ятдесятник" (straight ', curly ’, modifier letter ʼ) so a
// source's encoding choice can never silently produce a false negative — checked directly against
// this real failure mode, not assumed away.
var apostropheStripper = strings.NewReplacer("'", "", "’", "", "ʼ", "", "`", "")

// isLikelyChristian checks name against christianKeywords, case-insensitive substring — pure, no
// I/O, directly unit-testable, mirroring findTaxonAliasMatch's own style and reusing
// normalizeAlias's lowercase+trim exactly.
func isLikelyChristian(name string) bool {
	n := diacriticStripper.Replace(apostropheStripper.Replace(normalizeAlias(name)))
	for _, kw := range christianKeywords {
		if strings.Contains(n, kw) {
			return true
		}
	}
	return false
}
