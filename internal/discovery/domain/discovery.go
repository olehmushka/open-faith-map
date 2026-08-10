// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the discovery module's business types — no framework/SQL/HTTP imports.
package domain

import (
	"errors"
	"time"
)

// CacheRow is one denormalized, disposable projection of a go-oikumenea religion_sites row —
// rebuildable at any time, never a system of record (D-Facade). Not every column can be populated
// from go-oikumenea's searchSites response alone: TraditionTaxonID/ServiceLanguages/ServiceDays
// need a second round-trip per site (GetEffectiveClassifications / service-schedule reads) this
// module does not make yet (see docs/modules/discovery.md's open seams) — they stay nil/empty on a
// lazily-cached row until that resolver exists.
type CacheRow struct {
	ID                  string
	ReligionSiteRID     string
	CongregationUnitRID string
	ContentSiteID       *string
	Latitude            *float64
	Longitude           *float64
	TraditionTaxonID    *string
	ServiceLanguages    []string
	ServiceDays         []int
	RefreshedAt         time.Time
}

// SearchQuery is GET /search's parsed request. Tradition/Language/DayOfWeek/Query all bypass the
// local cache entirely and go live — the cache today only reliably holds Latitude/Longitude (see
// CacheRow's doc), so filtering a cache hit by any of them would silently under-return. Only a
// bare or lat/lng/radius-only query can ever be served from the cache.
type SearchQuery struct {
	Lat       *float64
	Lng       *float64
	RadiusM   *float64
	Tradition *string
	Language  *string
	DayOfWeek *int
	Query     *string
}

// BypassesCache reports whether q must always be answered live (never from discovery_site_cache).
func (q SearchQuery) BypassesCache() bool {
	return q.Tradition != nil || q.Language != nil || q.DayOfWeek != nil || q.Query != nil
}

type RefreshRegion struct {
	MinLat, MinLng, MaxLat, MaxLng float64
}

var ErrForbidden = errors.New("caller does not hold the operator's grant on the root unit")
