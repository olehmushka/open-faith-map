// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds the discovery module's business logic: the cache-first/live-fallback
// search, and the operator-only manual refresh tool. See docs/modules/discovery.md's redesign.
//
// M10.6 cutover: the live-fallback search and the operator refresh both call internal/religion's
// SearchSites directly (in-process) instead of building a service-principal go-oikumenea client per
// call — the "server's own call, never on behalf of the caller" framing stays true, it's just no
// longer an HTTP round-trip. requireOperator decides against internal/authz.Require.
package application

import (
	"context"
	"math"
	"slices"
	"strings"

	"github.com/olehmushka/open-faith-map/internal/authz"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/discovery/adapters"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
	"github.com/palantir/pkg/uuid"
)

// operatorPermission mirrors registration's IsOperator (M2.3) and content's content.manage — the
// same religionorg.manage grant, checked against the shared root unit instead of a single
// congregation's own unit.
const operatorPermission = authzdomain.PermReligionOrgManage

// ContentResolver looks up the published content_sites row for a congregation unit, if any — an
// interface-call cross-module dependency (conventions.md), not a raw cross-module SQL query, even
// though the underlying FK is real (docs/modules/discovery.md's DS-OFM-13 resolution). Optional:
// a nil ContentResolver just leaves CacheRow.ContentSiteID unset.
type ContentResolver interface {
	GetSiteByUnit(ctx context.Context, congregationUnitRID string) (siteID string, found bool, err error)
}

type Config struct {
	// RootUnitID is the same shared root unit registration/content already use
	// (internal/platform/seed.Resolve's RootUnitID) — the target of the operator-scoped Require check.
	RootUnitID string
}

type Service struct {
	store    *adapters.Repository
	content  ContentResolver
	religion *religionapplication.Service
	authzSvc *authz.Service
	cfg      Config
}

func NewService(store *adapters.Repository, content ContentResolver, religionSvc *religionapplication.Service, authzSvc *authz.Service, cfg Config) *Service {
	return &Service{store: store, content: content, religion: religionSvc, authzSvc: authzSvc, cfg: cfg}
}

// Search answers GET /search (DiscoveryPublicService — no token, D-AdminSurface). A bare or
// lat/lng/radius-only query is served from discovery_site_cache when the cache has any rows at
// all; anything else (a tradition/language/dayOfWeek/query filter, or an empty cache) goes live —
// the server's own in-process call, never on behalf of any specific caller, who has no token.
func (s *Service) Search(ctx context.Context, q domain.SearchQuery) ([]domain.CacheRow, error) {
	if !q.BypassesCache() {
		cached, err := s.store.SearchAll(ctx)
		if err != nil {
			return nil, err
		}
		if len(cached) > 0 {
			return filterByRadius(cached, q), nil
		}
	}
	return s.refreshFromLive(ctx, q)
}

// GetSiteByUnit answers the detail page's server-rendered fetch (M13.0's getSite endpoint) — always
// live, never cache: a direct link/refresh/crawler hit needs the current, correctly precision-
// coarsened record for exactly one congregation, not whatever radius search last happened to cache.
// Returns found=false if the unit has no public, non-hidden site (a private/unlisted/hidden site,
// or no site at all, are indistinguishable to an anonymous caller by design) — including when
// unitID isn't even a well-formed RID: org_unit_id is a real `uuid` column, so a malformed path
// parameter would otherwise reach Postgres and come back as a raw type-coercion error (500) rather
// than the same "nothing here" a syntactically valid but nonexistent id already gets (404).
func (s *Service) GetSiteByUnit(ctx context.Context, unitID string) (domain.CacheRow, bool, error) {
	if _, err := uuid.ParseUUID(unitID); err != nil {
		return domain.CacheRow{}, false, nil
	}
	sites, err := s.religion.SearchSites(ctx, religiondomain.DiscoveryQuery{UnitID: &unitID, Limit: 1})
	if err != nil {
		return domain.CacheRow{}, false, err
	}
	if len(sites) == 0 || sites[0].Latitude == nil || sites[0].Longitude == nil {
		return domain.CacheRow{}, false, nil
	}
	row := cacheRowFromDiscoverySite(sites[0])
	s.enrichContentSite(ctx, &row)
	return row, true, nil
}

func (s *Service) refreshFromLive(ctx context.Context, q domain.SearchQuery) ([]domain.CacheRow, error) {
	accessibility, err := parseAccessibility(q.Accessibility)
	if err != nil {
		// A genuine client error (bad accessibility= value) — unlike SearchSites' own error path
		// below, this must reach the caller as a real 400, not silently degrade to the cache.
		return nil, err
	}
	sites, err := s.religion.SearchSites(ctx, religiondomain.DiscoveryQuery{
		Lat: q.Lat, Lng: q.Lng, RadiusM: q.RadiusM,
		Religion:      derefOrEmpty(q.Tradition),
		Query:         derefOrEmpty(q.Query),
		Language:      q.Language,
		DayOfWeek:     q.DayOfWeek,
		Accessibility: accessibility,
		OnlineOnly:    q.OnlineOnly != nil && *q.OnlineOnly,
	})
	if err != nil {
		// Never blocks the anonymous caller on an upstream hiccup (discovery.md's invariants) —
		// whatever is already cached, even nothing, is still a valid answer.
		return s.store.SearchAll(ctx)
	}
	rows := make([]domain.CacheRow, 0, len(sites))
	for _, site := range sites {
		if site.Latitude == nil || site.Longitude == nil {
			continue // hidden sites are already excluded by SearchSites; defensive, not expected
		}
		row := cacheRowFromDiscoverySite(site)
		s.enrichContentSite(ctx, &row)
		persisted, err := s.store.UpsertRow(ctx, row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, persisted)
	}
	return rows, nil
}

// cacheRowFromDiscoverySite projects a live religion.DiscoverySite hit onto the cache's own shape
// (M13.0) — every field CacheRow carries now comes straight from this one SearchSites call, closing
// the pre-existing gap where a cached/live row left tradition/language/day nil regardless of path.
func cacheRowFromDiscoverySite(site religiondomain.DiscoverySite) domain.CacheRow {
	return domain.CacheRow{
		ReligionSiteRID:     site.ID,
		CongregationUnitRID: site.OrgUnitID,
		Latitude:            site.Latitude,
		Longitude:           site.Longitude,
		Name:                site.Name,
		Address:             site.Address,
		TraditionTaxonID:    site.TraditionTaxonID,
		TraditionTaxonCode:  site.TraditionTaxonCode,
		TraditionTaxonName:  site.TraditionTaxonName,
		ServiceLanguages:    site.ServiceLanguages,
		ServiceDays:         site.ServiceDays,
		Attributes:          site.Attributes,
	}
}

func (s *Service) enrichContentSite(ctx context.Context, row *domain.CacheRow) {
	if s.content == nil {
		return
	}
	if id, found, err := s.content.GetSiteByUnit(ctx, row.CongregationUnitRID); err == nil && found {
		row.ContentSiteID = &id
	}
}

// RefreshRegion answers POST /refresh (DiscoveryService — header auth, operator tool only, not
// part of the public product surface).
func (s *Service) RefreshRegion(ctx context.Context, region domain.RefreshRegion) (int, error) {
	if err := s.requireOperator(ctx); err != nil {
		return 0, err
	}
	sites, err := s.religion.SearchSites(ctx, religiondomain.DiscoveryQuery{
		MinLat: &region.MinLat, MinLng: &region.MinLng, MaxLat: &region.MaxLat, MaxLng: &region.MaxLng,
	})
	if err != nil {
		return 0, err
	}
	for _, site := range sites {
		if site.Latitude == nil || site.Longitude == nil {
			continue
		}
		row := cacheRowFromDiscoverySite(site)
		s.enrichContentSite(ctx, &row)
		if _, err := s.store.UpsertRow(ctx, row); err != nil {
			return 0, err
		}
	}
	return len(sites), nil
}

// Facets answers GET /facets (DiscoveryPublicService — no token) — always live, same as
// GetSiteByUnit, since facets must reflect current data and cache-side filtering is out of scope
// for M13.1 anyway.
func (s *Service) Facets(ctx context.Context) (religiondomain.Facets, error) {
	return s.religion.SearchFacets(ctx)
}

// requireOperator asks internal/authz's PDP whether the request's subject (from ctx) holds
// operatorPermission on Config.RootUnitID specifically — the same target-scoped pattern
// registration's IsOperator (M2.3) and content's requireManage already use, here targeting the
// shared root since a region refresh has no single congregation unit of its own.
func (s *Service) requireOperator(ctx context.Context) error {
	if err := s.authzSvc.Require(ctx, operatorPermission, s.cfg.RootUnitID); err != nil {
		if err == authzdomain.ErrPermissionDenied {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

// parseAccessibility splits SearchQuery.Accessibility's comma-separated criterion keys and
// validates each against religiondomain.AccessibilityCriteria — an unrecognized key is a real
// client error (domain.ErrInvalidFilter), not a silent zero-result match. Returns nil, nil when raw
// is nil (no filter requested).
func parseAccessibility(raw *string) ([]string, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	keys := strings.Split(*raw, ",")
	for i, key := range keys {
		keys[i] = strings.TrimSpace(key)
		if !slices.Contains(religiondomain.AccessibilityCriteria, keys[i]) {
			return nil, domain.ErrInvalidFilter
		}
	}
	return keys, nil
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func filterByRadius(rows []domain.CacheRow, q domain.SearchQuery) []domain.CacheRow {
	if q.Lat == nil || q.Lng == nil || q.RadiusM == nil {
		return rows
	}
	out := make([]domain.CacheRow, 0, len(rows))
	for _, r := range rows {
		if r.Latitude == nil || r.Longitude == nil {
			continue
		}
		if haversineMeters(*q.Lat, *q.Lng, *r.Latitude, *r.Longitude) <= *q.RadiusM {
			out = append(out, r)
		}
	}
	return out
}

// haversineMeters is a client-side, non-PostGIS distance calc — deliberate: D-Facade says
// OpenFaithMap owns no location index of its own, so this cache is a flat disposable projection,
// not a spatial index. internal/religion's own PostGIS search (via the live fallback) does real
// radius/bbox filtering server-side; this only ever filters an already-cache-hit result set.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
