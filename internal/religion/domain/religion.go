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
	"strings"
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

// AccessibilityAttributes are the named, specific accessibility criteria a site can report —
// deliberately not a single "accessible" flag (scoped with the user at M13.0): each criterion is
// independently true/false/unknown so the public UI can show exactly what's confirmed rather than
// a vague badge.
type AccessibilityAttributes struct {
	StepFreeEntrance           bool `json:"stepFreeEntrance"`
	AccessibleRestroom         bool `json:"accessibleRestroom"`
	HearingLoop                bool `json:"hearingLoop"`
	SignLanguageInterpretation bool `json:"signLanguageInterpretation"`
	AccessibleParking          bool `json:"accessibleParking"`
	WheelchairSeating          bool `json:"wheelchairSeating"`
	BrailleOrLargePrint        bool `json:"brailleOrLargePrint"`
}

// SiteAttributes is religion_sites.attributes' Go shape (M13.0) — validated/shaped here in the
// application layer rather than a DB CHECK, matching this repo's existing convention for
// structured JSONB (e.g. directory_units.metadata). Every criterion defaults to false/unset until
// an operator sets it via the admin UI (M13.2) — absent, not a false claim of inaccessibility.
type SiteAttributes struct {
	Accessibility AccessibilityAttributes `json:"accessibility"`
	OnlineStream  bool                    `json:"onlineStream"`
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
	Attributes      SiteAttributes

	// The following are populated only by SearchSites (the public discovery search path, M13.0) —
	// every other Site-returning query (ListSitesByUnit, GetSiteRow, InsertSite) leaves them at
	// zero value, the same "populated lazily, not by every path" convention
	// discovery.CacheRow's own doc comment already uses.
	Name               string
	AdminArea1         string
	AdminArea2         string
	Locality           string
	Street             string
	HouseNumber        string
	PostalCode         string
	TraditionTaxonID   *string
	TraditionTaxonCode *string
	TraditionTaxonName *string
	ServiceLanguages   []string
	ServiceDays        []int
}

// DiscoverySite is SearchSites' public-safe projection: the coordinate is coarsened per
// PublicPrecision (nil lat/lng for `hidden`), never the exact value Site carries. Address text is
// coarsened the same way, via CoarsenAddress (M13.0) — see D-DiscoveryAddressPrecision. Name is
// shown regardless of precision (scoped with the user: a name alone carries no finer location
// signal than what's already public); Attributes passes through unfiltered — accessibility/
// online-stream flags aren't location-sensitive.
type DiscoverySite struct {
	ID                 string
	OrgUnitID          string
	SiteTypeID         string
	SiteTypeCode       string
	SiteTypeName       string
	PublicPrecision    string
	IsPrimary          bool
	Latitude           *float64
	Longitude          *float64
	Name               string
	Address            *string
	TraditionTaxonID   *string
	TraditionTaxonCode *string
	TraditionTaxonName *string
	ServiceLanguages   []string
	ServiceDays        []int
	Attributes         SiteAttributes
}

// DiscoveryQuery is SearchSites' input: an optional spatial window (radius XOR bbox), an optional
// taxon filter (via the taxonomy closure), an optional text query over unit code/name/alias, and an
// optional service-schedule filter (Language/DayOfWeek — matches a site with at least one
// religion_service_schedules row satisfying each filter given, independently of one another).
type DiscoveryQuery struct {
	Lat, Lng, RadiusM              *float64
	MinLat, MinLng, MaxLat, MaxLng *float64
	Religion                       string // taxon id; "" = no filter
	Query                          string
	Language                       *string // religion_service_schedules.language; nil = no filter
	DayOfWeek                      *int    // religion_service_schedules.day_of_week (0=Sunday..6=Saturday); nil = no filter
	UnitID                         *string // exact org_unit_id match — M13.0's single-site detail-page lookup; nil = no filter
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

// CoarsenAddress projects a site's structured address to display text at the same publish
// precision Coarsen already gates the coordinate at (D-DiscoveryAddressPrecision,
// docs/architecture/decisions.md): exact/street show the full address (street level is already
// ~11m via Coarsen — gating the *text* tighter than the *pin* adds no privacy); neighborhood/city
// show only locality + admin area 1, no street/house number; hidden returns ok=false (never
// reached by SearchSites in practice, since hidden sites are excluded from the result set itself —
// kept for GetSite-style single-site callers that might one day accept an ID with no filtering).
func CoarsenAddress(locality, adminArea1, adminArea2, street, houseNumber, postalCode, precision string) (line string, ok bool) {
	switch precision {
	case PrecisionHidden:
		return "", false
	case PrecisionNeighborhood, PrecisionCity:
		return joinNonEmpty(", ", locality, adminArea1), locality != "" || adminArea1 != ""
	default: // "", exact, street, or an unrecognized precision — the full address
		streetLine := joinNonEmpty(" ", street, houseNumber)
		line = joinNonEmpty(", ", streetLine, locality, adminArea2, adminArea1, postalCode)
		return line, line != ""
	}
}

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}
