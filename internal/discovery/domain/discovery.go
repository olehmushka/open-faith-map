// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the discovery module's business types — no framework/SQL/HTTP imports.
package domain

import (
	"errors"
	"time"

	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// CacheRow is one denormalized, disposable projection of internal/religion's discovery search —
// rebuildable at any time, never a system of record (D-Facade). M13.0 closed the pre-existing gap
// this doc comment used to describe: Name/Address/TraditionTaxon*/ServiceLanguages/ServiceDays/
// Attributes are now populated on every refresh (refreshFromLive, RefreshRegion), cached or live,
// straight from religiondomain.DiscoverySite — no second round-trip needed, since M10.6's
// in-process cutover means SearchSites already returns the enriched projection in one call.
type CacheRow struct {
	ID                  string
	ReligionSiteRID     string
	CongregationUnitRID string
	ContentSiteID       *string
	Latitude            *float64
	Longitude           *float64
	Name                string
	Address             *string
	TraditionTaxonID    *string
	TraditionTaxonCode  *string
	TraditionTaxonName  *string
	ServiceLanguages    []string
	ServiceDays         []int
	Attributes          religiondomain.SiteAttributes
	RefreshedAt         time.Time
}

// SearchQuery is GET /search's parsed request. Tradition/Language/DayOfWeek/Query all bypass the
// local cache entirely and go live, even though M13.0 made CacheRow reliably carry tradition/
// language/day data on every write: this type's own filterByRadius (application/service.go) has no
// matching predicate for them yet, and an older row cached before M13.0 shipped still has them
// empty until its next refresh — building real cache-side filtering for these is M13.1's job, not
// this one's. Only a bare or lat/lng/radius-only query can ever be served from the cache today.
type SearchQuery struct {
	Lat           *float64
	Lng           *float64
	RadiusM       *float64
	Tradition     *string
	Language      *string
	DayOfWeek     *int
	Query         *string
	Accessibility *string // comma-separated AccessibilityCriteria keys (M13.1); nil = no filter
	OnlineOnly    *bool   // M13.1; nil/false = no filter
}

// BypassesCache reports whether q must always be answered live (never from discovery_site_cache).
// Accessibility/OnlineOnly (M13.1) join Tradition/Language/DayOfWeek/Query here for the same reason
// those four already do: cache-side filtering for any of them remains a separate, still-open gap
// (docs/modules/discovery.md's "Open seams") — building it is explicitly not part of M13.1.
func (q SearchQuery) BypassesCache() bool {
	return q.Tradition != nil || q.Language != nil || q.DayOfWeek != nil || q.Query != nil ||
		q.Accessibility != nil || (q.OnlineOnly != nil && *q.OnlineOnly)
}

type RefreshRegion struct {
	MinLat, MinLng, MaxLat, MaxLng float64
}

var ErrForbidden = errors.New("caller does not hold the operator's grant on the root unit")

// ErrInvalidFilter is returned when a SearchQuery.Accessibility value isn't one of
// religiondomain.AccessibilityCriteria's known keys (M13.1) — a real client error rather than a
// silent zero-result match, the same "fail loudly" fix applied to the pre-existing tradition bug.
var ErrInvalidFilter = errors.New("unrecognized accessibility criterion")
