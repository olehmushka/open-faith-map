// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package nominatim implements domain.Geocoder against the public OpenStreetMap Nominatim search
// API (https://nominatim.openstreetmap.org) — free, keyless, the same data source already behind
// this project's own Leaflet/OSM public map (M4).
//
// The HTTP request/response handling itself (structured query params, the real string-typed
// lat/lon response-shape gotcha, and the rate limiter Nominatim's own usage policy requires) is
// delegated to github.com/olehmushka/go-nominatim, extracted from this connector's own earlier
// implementation — see that package's own doc comment for the full real-world-verified findings.
// This package's own job is just domain.Geocoder's contract: translating domain.GeocodeQuery/
// domain.GeocodeResult to and from go-nominatim's Query/Result shapes.
//
// This package must only ever be called from an operator-triggered single lookup
// (application.SuggestCoordinates), never from RunConnector's bulk pipeline — go-nominatim's own
// rate.Limiter enforces Nominatim's 1 request/second usage policy mechanically, but staying
// single-lookup-only is still on the caller.
package nominatim

import (
	"context"
	"net/http"

	upstream "github.com/olehmushka/go-nominatim"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const Code = upstream.Code

type Geocoder struct {
	// BaseURL overrides upstream.DefaultBaseURL — read fresh on every Geocode call, so a caller
	// (e.g. a test pointing this at an httptest.Server) can set it any time after New.
	BaseURL string

	client *upstream.Client
}

// New constructs a Nominatim geocoder. httpClient nil defaults to go-nominatim's own bounded
// requestTimeout client — deliberately bounded, unlike uaedr's own explicit no-timeout client:
// this is a small, single-request lookup, not a multi-hundred-MB streaming download.
func New(httpClient *http.Client) *Geocoder {
	return &Geocoder{
		BaseURL: upstream.DefaultBaseURL,
		// Nominatim's own usage policy requires a real identifying User-Agent — same convention
		// uaedr/arrnc's own Citation().UserAgent already establishes for this module's other sources.
		client: upstream.New(httpClient, upstream.WithUserAgent(
			"openfaithmap-congregationimport/1.0 (operator-triggered address lookup, not a scrape)")),
	}
}

func (g *Geocoder) Code() string { return Code }

// Geocode issues one rate-limited, structured search request via go-nominatim. An empty result is
// a real "no match" (nil, nil), not an error — see domain.ErrGeocodeNoMatch for how the caller
// distinguishes that from a real failure.
func (g *Geocoder) Geocode(ctx context.Context, query domain.GeocodeQuery) (*domain.GeocodeResult, error) {
	g.client.BaseURL = g.BaseURL

	q := upstream.Query{}
	if query.Street != nil {
		q.Street = *query.Street
	}
	if query.Locality != nil {
		q.Locality = *query.Locality
	}
	if query.AdminArea1 != nil {
		q.AdminArea1 = *query.AdminArea1
	}
	if query.Country != nil {
		q.Country = *query.Country
	}

	result, found, err := g.client.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	var precision *string
	if result.Precision != "" {
		p := result.Precision
		precision = &p
	}

	return &domain.GeocodeResult{
		Latitude:    result.Latitude,
		Longitude:   result.Longitude,
		Precision:   precision,
		DisplayName: result.DisplayName,
		Provider:    Code,
	}, nil
}
