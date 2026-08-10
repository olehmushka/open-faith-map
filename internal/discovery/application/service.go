// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds the discovery module's business logic: the cache-first/live-fallback
// search, and the operator-only manual refresh tool. See docs/modules/discovery.md's redesign.
package application

import (
	"context"
	"math"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/authorization"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/discovery/adapters"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
)

// operatorPermission mirrors registration's IsOperator (M2.3) and content's content.manage — the
// same religionorg.manage grant, reused rather than minted as a new go-oikumenea permission, just
// checked against the shared root unit instead of a single congregation's own unit.
const operatorPermission = "religionorg.manage"

// ContentResolver looks up the published content_sites row for a congregation unit, if any — an
// interface-call cross-module dependency (conventions.md), not a raw cross-module SQL query, even
// though the underlying FK is real (docs/modules/discovery.md's DS-OFM-13 resolution). Optional:
// a nil ContentResolver just leaves CacheRow.ContentSiteID unset.
type ContentResolver interface {
	GetSiteByUnit(ctx context.Context, congregationUnitRID string) (siteID string, found bool, err error)
}

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
	// RootUnitID is the same shared root unit registration/content already use
	// (scripts/bootstrap-registration-org) — the target of the operator-scoped Authorize check.
	RootUnitID string
	// ServicePrincipal configures the server's own go-oikumenea call for Search's cache-miss
	// fallback — never a forwarded caller token, since GET /search has no caller token to forward.
	ServicePrincipal coreintegration.Config
}

type Service struct {
	store   *adapters.Store
	content ContentResolver
	cfg     Config
}

func NewService(store *adapters.Store, content ContentResolver, cfg Config) *Service {
	return &Service{store: store, content: content, cfg: cfg}
}

func (s *Service) serviceClient(ctx context.Context) (*oikumenea.Client, error) {
	return coreintegration.NewServiceClient(ctx, s.cfg.ServicePrincipal)
}

func (s *Service) userClient(token string) (*oikumenea.Client, error) {
	return coreintegration.NewUserClient(s.cfg.OikumeneaBaseURL, token, s.cfg.OikumeneaInsecureSkipVerify)
}

// Search answers GET /search (DiscoveryPublicService — no token, D-AdminSurface). A bare or
// lat/lng/radius-only query is served from discovery_site_cache when the cache has any rows at
// all; anything else (a tradition/language/dayOfWeek/query filter, or an empty cache) goes live via
// the service principal — the server's own call, never on behalf of the caller, who has no token.
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

func (s *Service) refreshFromLive(ctx context.Context, q domain.SearchQuery) ([]domain.CacheRow, error) {
	c, err := s.serviceClient(ctx)
	if err != nil {
		return nil, err
	}
	page, err := c.Religion.SearchSites(ctx,
		q.Lat, q.Lng, q.RadiusM, nil, nil, nil, nil, q.Tradition, q.Language, q.DayOfWeek, nil, q.Query, nil)
	if err != nil {
		// Never blocks the anonymous caller on an upstream hiccup (discovery.md's invariants) —
		// whatever is already cached, even nothing, is still a valid answer.
		return s.store.SearchAll(ctx)
	}
	rows := make([]domain.CacheRow, 0, len(page.Sites))
	for _, site := range page.Sites {
		row := domain.CacheRow{
			ReligionSiteRID:     site.Id,
			CongregationUnitRID: site.OrgUnitId,
			Latitude:            site.Latitude,
			Longitude:           site.Longitude,
		}
		s.enrichContentSite(ctx, &row)
		persisted, err := s.store.UpsertRow(ctx, row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, persisted)
	}
	return rows, nil
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
func (s *Service) RefreshRegion(ctx context.Context, token, callerPersonID string, region domain.RefreshRegion) (int, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return 0, err
	}
	c, err := s.serviceClient(ctx)
	if err != nil {
		return 0, err
	}
	page, err := c.Religion.SearchSites(ctx,
		nil, nil, nil, &region.MinLat, &region.MinLng, &region.MaxLat, &region.MaxLng, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return 0, err
	}
	for _, site := range page.Sites {
		row := domain.CacheRow{
			ReligionSiteRID:     site.Id,
			CongregationUnitRID: site.OrgUnitId,
			Latitude:            site.Latitude,
			Longitude:           site.Longitude,
		}
		s.enrichContentSite(ctx, &row)
		if _, err := s.store.UpsertRow(ctx, row); err != nil {
			return 0, err
		}
	}
	return len(page.Sites), nil
}

// requireOperator asks go-oikumenea's real PDP whether the caller holds operatorPermission on
// Config.RootUnitID specifically — the same target-scoped pattern registration's IsOperator (M2.3)
// and content's requireManage already use, here targeting the shared root since a region refresh
// has no single congregation unit of its own.
func (s *Service) requireOperator(ctx context.Context, token, callerPersonID string) error {
	c, err := s.userClient(token)
	if err != nil {
		return err
	}
	rootUnitID := s.cfg.RootUnitID
	resp, err := c.Authorization.Authorize(ctx, authorization.AuthorizeRequest{
		SubjectPersonId: callerPersonID,
		Action:          operatorPermission,
		UnitId:          &rootUnitID,
	})
	if err != nil {
		if authorization.IsPermissionDenied(err) {
			return domain.ErrForbidden
		}
		return err
	}
	if !resp.Allow {
		return domain.ErrForbidden
	}
	return nil
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
// not a spatial index. go-oikumenea's own PostGIS search (via the live fallback) does real
// radius/bbox filtering server-side; this only ever filters an already-cache-hit result set.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
