// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: small, bounded picker lists (religion taxa, countries) fetched straight from
// go-oikumenea, same D-Facade reasoning as lib/jurisdiction.ts's own file comment — no
// openfaithmap-api mirror, no new backend endpoint, since these are go-oikumenea-native
// dictionaries an operator just needs to pick from, not data OpenFaithMap owns.
import "server-only";

import { oikumenea } from "./oikumenea";

// D-Exclusions (architecture/decisions.md), same codes as
// internal/registration/domain.ExcludedTaxonCodes — the authoritative check runs server-side in
// openfaithmap-api/congregationimport regardless; filtering them out of a picker is a UX nicety
// only. Shared across every picker in this app (register/page.tsx used to keep its own copy).
export const EXCLUDED_TAXON_CODES = new Set(["russian_orthodox_church", "jehovahs_witnesses", "lds_church"]);

// go-oikumenea's own locale codes are ISO 639-3; this app's URL-facing locales are ISO 639-1.
const OIKUMENEA_LOCALE: Record<string, string> = { en: "eng", uk: "ukr", es: "spa", pt: "por" };

export type PickerOption = { id: string; name: string };

function localizedName(name: Record<string, string>, oikumeneaLocale: string, fallback: string): string {
  return name[oikumeneaLocale] ?? name["eng"] ?? Object.values(name)[0] ?? fallback;
}

/**
 * Every non-excluded religion taxon, localized and sorted — for a plain <select>, same shape
 * register/page.tsx already uses for its own tradition picker (500 is that page's own bound too;
 * the real taxonomy is far smaller, seeded ~130 rows).
 */
export async function listTaxaForPicker(locale: string): Promise<PickerOption[]> {
  const oikumeneaLocale = OIKUMENEA_LOCALE[locale] ?? "eng";
  const client = await oikumenea();
  const taxaPage = await client.religion.listTaxa(undefined, undefined, undefined, undefined, undefined, 500);
  return taxaPage.taxa
    .filter((taxon) => !EXCLUDED_TAXON_CODES.has(taxon.code))
    .map((taxon) => ({ id: taxon.id, name: localizedName(taxon.name, oikumeneaLocale, taxon.code) }))
    .sort((a, b) => a.name.localeCompare(b.name));
}

/** Every country, localized and sorted — same shape register/page.tsx already uses. */
export async function listCountriesForPicker(locale: string): Promise<PickerOption[]> {
  const oikumeneaLocale = OIKUMENEA_LOCALE[locale] ?? "eng";
  const client = await oikumenea();
  const countries = await client.geo.listCountries();
  return countries.countries
    .map((c) => ({ id: c.id, name: localizedName(c.name, oikumeneaLocale, c.code) }))
    .sort((a, b) => a.name.localeCompare(b.name));
}
