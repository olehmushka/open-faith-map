// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the refdata module's one type: the static country reference list. Per
// D-StaticRefData, the actual data (refdata_countries/refdata_country_names) was already seeded
// byte-for-byte from the live database at M10.1 (migrations/0012_core_refdata.sql) — this module is
// a read-only Go surface over it, replacing hermenea and go-oikumenea's geo.Country.
package domain

// Country is one ISO-3166-1 alpha-2 country with its four locale names (eng/spa/por/ukr). Names is
// keyed by locale exactly as refdata_country_names.locale stores it — internal/congregationimport's
// matchCountry (countrymatch.go's findCountryMatch) ranges over the values only, never the keys, so
// the locale set does not need to be part of this type's contract.
type Country struct {
	ID    string
	Code  string // ISO-3166-1 alpha-2
	Name  string // English name (refdata_countries.name)
	Names map[string]string
}
