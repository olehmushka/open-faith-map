// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package nominatim implements domain.Geocoder against the public OpenStreetMap Nominatim search
// API (https://nominatim.openstreetmap.org) — free, keyless, the same data source already behind
// this project's own Leaflet/OSM public map (M4).
//
// Real, verified against the live endpoint (2026-08-14, not assumed): a structured query for
// Villa María/Córdoba/Argentina returned a real match (lat -32.4106245, lon -63.2435809). Also hit
// real connection resets/timeouts on repeated calls seconds apart from the same host — the public
// endpoint is a best-effort free community service, not a guaranteed-uptime API; this package's own
// bounded client timeout and the caller's own error handling (transport/errors.go's mapErr passes
// a non-nil, non-ErrGeocodeNoMatch error straight through) both exist because of that, not
// speculatively.
//
// Real, verified response shape: a JSON array, `lat`/`lon` are STRINGS (not numbers) — a real
// gotcha, checked directly against the live response, not assumed from the docs.
//
// Real, mechanically-enforced rate limit: Nominatim's own usage policy
// (https://operations.osmfoundation.org/policies/nominatim/) caps general use at 1 request/second
// and explicitly forbids bulk/systematic querying — enforced here with a real rate.Limiter, not
// just a comment, and this package must only ever be called from an operator-triggered single
// lookup (application.SuggestCoordinates), never from RunConnector's bulk pipeline.
package nominatim

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	"golang.org/x/time/rate"
)

const Code = "nominatim"

// defaultBaseURL is a field, not hardcoded inline into requests — so a future self-hosted
// Nominatim/Photon instance can be swapped in later without an interface change, same reasoning
// uaedr/arrnc already apply to their own configurable source URLs.
const defaultBaseURL = "https://nominatim.openstreetmap.org"

const requestTimeout = 10 * time.Second

type Geocoder struct {
	BaseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// New constructs a Nominatim geocoder. httpClient nil defaults to a client with requestTimeout —
// deliberately bounded, unlike uaedr's own explicit no-timeout client: this is a small,
// single-request lookup, not a multi-hundred-MB streaming download.
func New(httpClient *http.Client) *Geocoder {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: requestTimeout}
	}
	return &Geocoder{
		BaseURL:    defaultBaseURL,
		httpClient: httpClient,
		limiter:    rate.NewLimiter(rate.Every(time.Second), 1),
	}
}

func (g *Geocoder) Code() string { return Code }

// nominatimResult mirrors the real fields this package actually uses from Nominatim's own
// jsonv2 response shape — lat/lon are strings in the real response, not numbers.
type nominatimResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	AddressType string `json:"addresstype"`
}

// Geocode issues one rate-limited, structured search request. An empty result array is a real "no
// match" (nil, nil), not an error — see domain.ErrGeocodeNoMatch for how the caller distinguishes
// that from a real failure.
func (g *Geocoder) Geocode(ctx context.Context, query domain.GeocodeQuery) (*domain.GeocodeResult, error) {
	if err := g.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("nominatim: rate limiter: %w", err)
	}

	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("limit", "1")
	if query.Street != nil && *query.Street != "" {
		q.Set("street", *query.Street)
	}
	if query.Locality != nil && *query.Locality != "" {
		q.Set("city", *query.Locality)
	}
	if query.AdminArea1 != nil && *query.AdminArea1 != "" {
		q.Set("state", *query.AdminArea1)
	}
	if query.Country != nil && *query.Country != "" {
		q.Set("country", *query.Country)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.BaseURL+"/search?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("nominatim: build request: %w", err)
	}
	// Nominatim's own usage policy requires a real identifying User-Agent — same convention
	// uaedr/arrnc's own Citation().UserAgent already establishes for this module's other sources.
	req.Header.Set("User-Agent", "openfaithmap-congregationimport/1.0 (operator-triggered address lookup, not a scrape)")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nominatim: GET %s: %w", req.URL.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("nominatim: GET %s: unexpected status %s: %s", req.URL.Path, resp.Status, body)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("nominatim: decode response: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("nominatim: parse lat %q: %w", results[0].Lat, err)
	}
	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("nominatim: parse lon %q: %w", results[0].Lon, err)
	}

	var precision *string
	if p := results[0].AddressType; p != "" {
		precision = &p
	} else if p := results[0].Type; p != "" {
		precision = &p
	}

	return &domain.GeocodeResult{
		Latitude:    lat,
		Longitude:   lon,
		Precision:   precision,
		DisplayName: results[0].DisplayName,
		Provider:    Code,
	}, nil
}
