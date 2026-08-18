// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the religion module's types: taxonomy lookups, the per-unit org profile and
// classification, and the discovery-search substrate (sites). Ported from
// ../go-oikumenea/internal/religion/domain, trimmed per D-CorePortScope: no clergy/affiliation, no
// policy-kind/classification/org-kind CRUD (this repo's own consumer modules only ever read the
// seeded catalogs, never manage them — see migrations/0018_core_religion.sql's seed rows), no
// dynamic taxon create/update/reparent (the taxonomy is a static, curated seed, also from 0018).
package domain

import (
	"errors"
	"math"
	"time"
)

var (
	ErrTaxonNotFound         = errors.New("religion: taxon not found")
	ErrProfileNotFound       = errors.New("religion: org profile not found")
	ErrChildCreationExcluded = errors.New("religion: parent excludes child creation")
)

// PolicyExcludesChildCreation is the policy-kind code that blocks creating child organizations
// beneath a unit (religion_policy_kinds seed row, migrations/0018_core_religion.sql:379).
const PolicyExcludesChildCreation = "excludes_child_creation"

// Taxon is one node in the static, curated faith taxonomy (religion_taxa).
type Taxon struct {
	ID        string
	ParentID  *string // nil = root religion
	RankID    string
	RankCode  string
	Code      string
	Name      string
	SortOrder *int
}

// OrgProfile is the 1:1 faith attributes of a religious-body unit (religion_org_profiles).
type OrgProfile struct {
	UnitID          string
	OrgKindID       string // "" = unset
	ShortCode       string // "" = unset
	Classifications []OrgClassification
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OrgClassification is a tradition tag on a unit (religion_org_classifications).
type OrgClassification struct {
	ID        string
	UnitID    string
	TaxonID   string
	TaxonCode string
	TaxonName string
	IsPrimary bool
	CreatedAt time.Time
}

// Site is a religious body's physical/online presence (religion_sites), joined to its location's
// EXACT coordinate — callers that expose this to an anonymous caller MUST run it through Coarsen
// first (SearchSites already does; a direct GetSite/ListUnitSites caller must too).
type Site struct {
	ID              string
	OrgUnitID       string
	LocationID      string
	SiteTypeID      string
	SiteTypeCode    string
	SiteTypeName    string
	Visibility      string
	PublicPrecision string
	IsPrimary       bool
	Latitude        float64
	Longitude       float64
}

// DiscoverySite is SearchSites' public-safe projection: the coordinate is coarsened per
// PublicPrecision (nil lat/lng for `hidden`), never the exact value Site carries.
type DiscoverySite struct {
	ID              string
	OrgUnitID       string
	SiteTypeID      string
	SiteTypeCode    string
	SiteTypeName    string
	PublicPrecision string
	IsPrimary       bool
	Latitude        *float64
	Longitude       *float64
}

// DiscoveryQuery is SearchSites' input: an optional spatial window (radius XOR bbox), an optional
// taxon filter (via the taxonomy closure), and an optional text query over unit code/name/alias.
type DiscoveryQuery struct {
	Lat, Lng, RadiusM              *float64
	MinLat, MinLng, MaxLat, MaxLng *float64
	Religion                       string // taxon id; "" = no filter
	Query                          string
	Limit                          int
}

// PublicPrecision values (religion_sites CHECK, migrations/0018_core_religion.sql:534).
const (
	PrecisionExact        = "exact"
	PrecisionStreet       = "street"
	PrecisionNeighborhood = "neighborhood"
	PrecisionCity         = "city"
	PrecisionHidden       = "hidden"
)

// precisionDecimals maps a publish precision to the number of decimal places a coordinate is
// rounded to (~11m / ~110m / ~1.1km at the equator for street/neighborhood/city). Ported verbatim
// from ../go-oikumenea/internal/religion/domain/discovery.go:214.
var precisionDecimals = map[string]int{PrecisionStreet: 4, PrecisionNeighborhood: 3, PrecisionCity: 2}

// Coarsen projects an exact (lat,lng) to the publish precision: exact returns the coordinate
// unchanged, street/neighborhood/city round to decreasing decimal places, hidden returns ok=false
// (the coordinate must be omitted). Used for a single site's public projection (e.g. a direct
// GetSite read) — SearchSites' own position-oracle fix additionally filters `hidden` out of the
// result set entirely and snaps the predicate geometry itself; see adapters.Store.SearchSites.
func Coarsen(lat, lng float64, precision string) (rlat, rlng float64, ok bool) {
	switch precision {
	case "", PrecisionExact:
		return lat, lng, true
	case PrecisionHidden:
		return 0, 0, false
	default:
		d, known := precisionDecimals[precision]
		if !known {
			return lat, lng, true
		}
		f := math.Pow(10, float64(d))
		return math.Round(lat*f) / f, math.Round(lng*f) / f, true
	}
}
