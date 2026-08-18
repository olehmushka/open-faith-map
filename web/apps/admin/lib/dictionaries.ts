// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: small, bounded picker lists (religion taxa, countries) — M10.7 repoints this from
// go-oikumenea (lib/oikumenea.ts, deleted this milestone) to openfaithmap-api's own core.conjure.yml
// surface (lib/core.ts). This got strictly simpler, not just repointed: religion_taxa.name is a
// plain string (no per-locale name table was ported, D-CorePortScope), and refdata_country_names'
// locale column already uses this app's own locale codes (en/es/pt/uk) directly — no ISO-639-3
// translation table to maintain anymore.
import "server-only";

import * as core from "./core";

// D-Exclusions (architecture/decisions.md), same codes as
// internal/registration/domain.ExcludedTaxonCodes — the authoritative check runs server-side in
// openfaithmap-api/congregationimport regardless; filtering them out of a picker is a UX nicety
// only. Shared across every picker in this app (register/page.tsx used to keep its own copy).
export const EXCLUDED_TAXON_CODES = new Set(["russian_orthodox_church", "jehovahs_witnesses", "lds_church"]);

export type PickerOption = { id: string; name: string };

/**
 * Every non-excluded religion taxon, sorted — for a plain <select>, same shape register/page.tsx
 * already uses for its own tradition picker (the real taxonomy is far smaller, seeded ~130 rows).
 */
export async function listTaxaForPicker(): Promise<PickerOption[]> {
  const taxa = await core.listTaxa(undefined, 500);
  return taxa
    .filter((taxon) => !EXCLUDED_TAXON_CODES.has(taxon.code))
    .map((taxon) => ({ id: taxon.id, name: taxon.name }))
    .sort((a, b) => a.name.localeCompare(b.name));
}

/** Every country, localized and sorted — same shape register/page.tsx already uses. */
export async function listCountriesForPicker(locale: string): Promise<PickerOption[]> {
  const countries = await core.listCountries();
  return countries
    .map((c) => ({ id: c.id, name: c.names[locale] ?? c.name }))
    .sort((a, b) => a.name.localeCompare(b.name));
}
