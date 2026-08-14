// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package osm reads OpenStreetMap's Overpass API for Christian places of worship
// (amenity=place_of_worship + religion=christian) and yields candidate religious organizations.
//
// Real, verified source (checked live, 2026-08-14, not assumed). Two Overpass mirrors were checked
// for robots.txt before either was used:
//   - overpass-api.de (the main, OSM-Foundation-run instance): robots.txt disallows /api/ — the
//     exact path the query endpoint (/api/interpreter) lives at. NOT used.
//   - overpass.kumi.systems (Private.coffee): robots.txt returns 404 — no restriction stated —
//     confirmed live, and the operator's own published policy is explicitly welcoming ("no rate
//     limit enforced; users should notify operators before large-scale projects"). Used as the
//     default endpoint, overridable via Connector.BaseURL. overpass.osm.ch independently confirmed
//     the same 404-no-robots.txt absence, as a second data point.
//
// A real end-to-end query against overpass.kumi.systems for all of Uruguay
// (area["ISO3166-1"="UY"][admin_level=2]) returned 200 OK in ~20s: 566 real elements (290 node, 274
// way, 2 relation — every way/relation carried a real `center` object from `out center`, 0 missing),
// with real tag shapes confirming every field mapping below: `name` (78 of 566 elements — ~14%,
// genuinely common, not a rare edge case — had none at all), `denomination` (catholic/roman_catholic
// both appear for the same real denomination — a real vocabulary inconsistency, not a parsing bug),
// `religion=christian`, `addr:street`/`addr:housenumber`/`addr:city`/`addr:country`, and a real
// `diocese` tag on at least one element.
//
// Much simpler execution shape than either ua-edr or ar-rnc: Overpass has no pagination, and a
// handful of countries' worth of place-of-worship data is small (566 for all of Uruguay alone) —
// this connector queries once per configured country on the first Fetch call, keeps every result in
// memory, and serves batches via plain integer-offset slicing, the same shape ar-rnc already uses
// for its own much-smaller-than-uaedr source.
//
// Real, live finding (2026-08-14, a real operator run through the admin UI): a single whole-country
// query for Colombia FAILED — reproduced directly against the same mirror moments later: HTTP 504,
// body "runtime error: ... Dispatcher_Client::request_read_and_idx::timeout. The server is probably
// too busy to handle your request." The identical Uruguay query, re-run at the same time, completed
// in 6.5s — this is a real per-country capacity limit on the free mirror, not a bug in the query
// shape or country selection, and not every configured country needs the same treatment. Colombia's
// own real real-time behavior was also inconsistent across attempts: one failure came back as a
// clean 504 (caught by the status check below), another as `200 OK` with an HTML error page in the
// body instead of JSON (NOT caught by a bare status check — surfaced as a confusing low-level
// "invalid character '<'" JSON-decode error until this was fixed to detect it explicitly).
//
// Fix: only Colombia is configured with a countryConfig.Grid (see below) — its query is split into
// several smaller bounding-box regions (still intersected with the real country polygon via the
// SAME area["ISO3166-1"=...] filter, so results stay geographically accurate; the bbox only bounds
// how much of that polygon one single request has to search), each queried and paced separately.
// Every other configured country keeps the original single whole-country query, unchanged — grid
// splitting is real, measured overhead (more requests, more courtesy-delay time) with no evidence
// any of them need it yet; add a Grid for one only once it's actually observed to time out the same
// way, not preemptively.
package osm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const Code = "osm"

// batchSize mirrors uaedr/arrnc's own choice — bounds how many records one Fetch call returns.
const batchSize = 500

// defaultBaseURL is overpass.kumi.systems, chosen over the main overpass-api.de mirror specifically
// because the latter's own robots.txt disallows /api/ (see package doc comment) — this one has no
// robots.txt at all and a published policy welcoming reasonable programmatic use.
const defaultBaseURL = "https://overpass.kumi.systems/api/interpreter"

// queryTimeoutSeconds is the Overpass QL [timeout:] value requested server-side, not the HTTP
// client's own timeout — a real full-Uruguay query completed in ~20s against a 90s budget, so this
// leaves real headroom for larger countries (Colombia) without being needlessly long.
const queryTimeoutSeconds = 90

// courtesyDelay is paused between EVERY request this connector makes (not just between countries —
// a country with a Grid configured issues more than one) — not required by any enforced rate limit
// (overpass.kumi.systems publishes none), but a deliberate courtesy given its own "notify before
// large-scale projects" policy: this connector issues a handful of requests per run, not a bulk
// crawl, and pacing them costs nothing.
const courtesyDelay = 2 * time.Second

// robotsCheckedAt records when this connector's robots.txt checks (see package doc comment and
// Citation) actually happened — a fixed historical fact, not something recomputed on every call.
var robotsCheckedAt = time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

// regionBBox is a real-world bounding box (degrees) used to chunk one country's query into several
// smaller ones. South/West/North/East, matching Overpass QL's own (south,west,north,east) bbox
// filter argument order.
type regionBBox struct {
	South, West, North, East float64
}

// regionGrid splits a country's BBox into Rows×Cols equal-sized cells for querying separately — see
// splitGrid. BBox is deliberately mainland-only where a country's real extent includes a small,
// remote outlying territory (e.g. Colombia's San Andrés y Providencia archipelago, ~700km from the
// mainland, near Nicaragua) — including it would waste grid cells on open ocean between the two and
// skew the grid toward the empty gap rather than the actually populated mainland. A real, documented
// gap for now, not a silent one: that territory's own congregations (if any are mapped at all) are
// simply not covered by this connector until a dedicated extra cell is added for it.
type regionGrid struct {
	BBox       regionBBox
	Rows, Cols int
}

// countryConfig is one configured country's fixed identity: Name (the plain English string
// CountryHint hands to matchCountry's exact-name match against go-oikumenea's Geo.ListCountries,
// countrymatch.go — same "plain literal string, no ISO-code layer" pattern ar-rnc already uses for
// "Argentina") and an optional Grid for countries whose single whole-country query has been
// observed to time out on the free mirror (see package doc comment) — nil means "one query, no
// split," the default and today's behavior for every country but Colombia.
type countryConfig struct {
	Name string
	Grid *regionGrid
}

// countries maps an ISO 3166-1 alpha-2 code (as configured on Connector.CountryCodes) to its fixed
// config. Deliberately a small, explicit map rather than a general ISO-3166 library: Name must match
// go-oikumenea's own locale-keyed country name exactly, which a generic library's formatting isn't
// guaranteed to. Default scope is the D-Scope rollout countries with no confirmed dedicated registry
// (Uruguay, Paraguay, Colombia, Chile) — see New's own doc comment for why. Adding a new country
// requires adding it here explicitly; New rejects any configured code not present, rather than
// silently skipping or guessing a name.
//
// Colombia's Grid bbox (real, live-verified 2026-08-14 via a direct Overpass
// relation["ISO3166-1"="CO"][admin_level=2]; out bb; query, not guessed): mainland bounds roughly
// -4.3..12.6°lat, -79.1..-66.8°lon, tighter than the full administrative relation's own reported
// bbox (-4.23..16.05, -82.12..-66.85) specifically to exclude the San Andrés outlier (see
// regionGrid's own doc comment). 3 rows × 2 cols (6 cells) is a first-cut heuristic sized to
// comfortably beat the timeout actually observed on the whole-country query, not exhaustively
// tuned — adjust if a cell is still found to time out.
var countries = map[string]countryConfig{
	"UY": {Name: "Uruguay"},
	"PY": {Name: "Paraguay"},
	"CO": {Name: "Colombia", Grid: &regionGrid{
		BBox: regionBBox{South: -4.3, West: -79.1, North: 12.6, East: -66.8},
		Rows: 3, Cols: 2,
	}},
	"CL": {Name: "Chile"},
}

// splitGrid divides a bbox into rows×cols equal-sized cells, row-major order. Pure function, no I/O
// — directly unit-testable. rows/cols below 1 are treated as 1 (no split on that axis).
func splitGrid(b regionBBox, rows, cols int) []regionBBox {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	latStep := (b.North - b.South) / float64(rows)
	lonStep := (b.East - b.West) / float64(cols)
	cells := make([]regionBBox, 0, rows*cols)
	for r := 0; r < rows; r++ {
		for cIdx := 0; cIdx < cols; cIdx++ {
			cells = append(cells, regionBBox{
				South: b.South + float64(r)*latStep,
				North: b.South + float64(r+1)*latStep,
				West:  b.West + float64(cIdx)*lonStep,
				East:  b.West + float64(cIdx+1)*lonStep,
			})
		}
	}
	return cells
}

// overpassElement is one Overpass JSON result element (a node, way, or relation).
type overpassElement struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    *float64          `json:"lat,omitempty"`
	Lon    *float64          `json:"lon,omitempty"`
	Center *overpassCenter   `json:"center,omitempty"`
	Tags   map[string]string `json:"tags"`
}

// overpassCenter is present on way/relation elements when the query requests `out center` — a
// node's own coordinates live directly on Lat/Lon instead, never here.
type overpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// overpassResponse is the top-level Overpass JSON shape ([out:json] output format).
type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

// osmRecord is this connector's own RawRecord payload shape: the source element plus which
// configured country's query produced it — Country is carried here (not re-derived from the
// element's own tags in Normalize) because individual OSM elements rarely carry a reliable
// addr:country tag, but the connector always knows which country's area it queried.
type osmRecord struct {
	Element overpassElement `json:"element"`
	Country string          `json:"country"`
}

// Connector queries the Overpass API once per configured country on the first Fetch call, caching
// every result in memory (loadOnce) — see package doc comment for why this is safe at the real
// scale involved (566 elements for all of Uruguay alone; even summed across several countries this
// stays orders of magnitude smaller than uaedr's streaming design exists to handle).
type Connector struct {
	BaseURL      string
	CountryCodes []string

	httpClient *http.Client

	loadOnce sync.Once
	loadErr  error
	records  []osmRecord
}

// New constructs an osm connector. countryCodes must be non-empty and every code must be present in
// countries — an unknown code is a construction error (fail loudly), not a silent skip, since a
// silently-dropped country would look exactly like "ran successfully, found nothing there."
func New(baseURL string, countryCodes []string, httpClient *http.Client) (*Connector, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if len(countryCodes) == 0 {
		return nil, errors.New("osm: at least one country code must be configured")
	}
	for _, code := range countryCodes {
		if _, ok := countries[code]; !ok {
			return nil, fmt.Errorf("osm: unknown country code %q — add it to countries before using it", code)
		}
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Connector{BaseURL: baseURL, CountryCodes: countryCodes, httpClient: httpClient}, nil
}

func (c *Connector) Code() string { return Code }

// Clone returns a fresh Connector for one RunConnector call — same fixed BaseURL/CountryCodes/
// httpClient, zero loadOnce/records. Real bug fixed by this (found live 2026-08-14): without Clone,
// a second RunConnector call with no parameters against the same long-lived, boot-registered
// instance would silently reuse the FIRST run's cached elements forever via loadOnce, never
// re-querying Overpass — WithParameters already got a fresh instance for the parameterized path,
// but a plain parameterless re-run needs this too.
func (c *Connector) Clone() domain.Connector {
	return &Connector{BaseURL: c.BaseURL, CountryCodes: c.CountryCodes, httpClient: c.httpClient}
}

// WithParameters implements domain.ConnectorConfigurable — recognizes exactly one key,
// "countryCodes" (a comma-separated ISO 3166-1 alpha-2 list, same convention as OSM_COUNTRY_CODES),
// and returns a fresh Connector scoped to just this run with that override applied. Any other key
// is a construction error (fail loudly — an operator's typo in a UI form deserves the same clear
// error a misconfigured env var would get, not a silently-ignored parameter). BaseURL/httpClient
// are always inherited from the receiver — they're fixed deployment config, not a per-run choice
// (see New's own doc comment for why).
func (c *Connector) WithParameters(params map[string]string) (domain.Connector, error) {
	countryCodes := c.CountryCodes
	for key, value := range params {
		switch key {
		case "countryCodes":
			var codes []string
			for _, code := range strings.Split(value, ",") {
				if code = strings.TrimSpace(code); code != "" {
					codes = append(codes, code)
				}
			}
			if len(codes) == 0 {
				return nil, fmt.Errorf("osm: parameter \"countryCodes\" must not be empty (got %q)", value)
			}
			countryCodes = codes
		default:
			return nil, fmt.Errorf("osm: unrecognized run parameter %q", key)
		}
	}
	return New(c.BaseURL, countryCodes, c.httpClient)
}

// Citation records what was actually checked before this connector was allowed to run
// (D-CongregationImport's own "decision #4" discipline) — both mirrors' robots.txt findings, honestly,
// not just the one actually used.
func (c *Connector) Citation() domain.SourceCitation {
	robotsURL := "https://overpass.kumi.systems/robots.txt"
	checkedAt := robotsCheckedAt
	rateLimitNotes := "No robots.txt exists on this mirror (404, confirmed live 2026-08-14) and the " +
		"operator (Private.coffee / Kumi Systems) publishes no enforced rate limit, asking only that " +
		"large-scale projects notify them first — this connector issues one query per configured " +
		"country (more for a country configured with a Grid, e.g. Colombia's 6 bbox-chunked queries — " +
		"see the package doc comment for why), a handful of requests per run, not a bulk crawl, with a " +
		"deliberate pause between every one of them (courtesyDelay). The main OSM-Foundation-run " +
		"mirror (overpass-api.de) disallows /api/ in its own robots.txt and is deliberately NOT used " +
		"here."
	return domain.SourceCitation{
		RobotsTxtURL:    &robotsURL,
		RobotsCheckedAt: &checkedAt,
		UserAgent:       "openfaithmap-congregationimport/1.0 (structured Overpass API query, not a scrape)",
		RateLimitNotes:  &rateLimitNotes,
		Notes: "OpenStreetMap data via the Overpass API (Overpass QL), queried per configured country " +
			"(or per bbox-chunked region, for a country configured with a Grid): " +
			`nwr["amenity"="place_of_worship"]["religion"="christian"] within an ISO3166-1 area match, ` +
			"out center tags. Default endpoint https://overpass.kumi.systems/api/interpreter. " +
			"ODbL-licensed OpenStreetMap data, (c) OpenStreetMap contributors.",
	}
}

// Fetch loads every configured country's results (once, via ensureLoaded) then serves batchSize-
// sized batches by plain integer-offset slicing — cursor is that offset as a decimal string, same
// shape as arrnc's own Fetch. No reopen-and-reskip step exists here, so ua-edr's real cursor-
// doubling bug class cannot occur in this connector.
func (c *Connector) Fetch(ctx context.Context, cursor *string) (batch []domain.RawRecord, nextCursor *string, err error) {
	if err := c.ensureLoaded(ctx); err != nil {
		return nil, nil, err
	}

	offset := 0
	if cursor != nil && *cursor != "" {
		offset, err = strconv.Atoi(*cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("osm: invalid cursor %q: %w", *cursor, err)
		}
	}
	if offset >= len(c.records) {
		return nil, nil, nil
	}

	end := offset + batchSize
	if end > len(c.records) {
		end = len(c.records)
	}

	now := time.Now()
	batch = make([]domain.RawRecord, 0, end-offset)
	for _, rec := range c.records[offset:end] {
		payload, merr := json.Marshal(rec)
		if merr != nil {
			return nil, nil, fmt.Errorf("osm: marshal raw payload: %w", merr)
		}
		batch = append(batch, domain.RawRecord{
			SourceRecordID: fmt.Sprintf("%s/%d", rec.Element.Type, rec.Element.ID),
			RawPayload:     payload,
			FetchedAt:      now,
		})
	}

	if end >= len(c.records) {
		return batch, nil, nil // source exhausted
	}
	next := strconv.Itoa(end)
	return batch, &next, nil
}

// Normalize maps one osmRecord's raw JSON payload onto the common candidate shape. Unlike ua-edr
// (no address at all) or ar-rnc (address text but no coordinates), OSM commonly carries BOTH real
// address tags and real coordinates — Latitude/Longitude are populated directly here (from a node's
// own lat/lon, or a way/relation's `center`), which means a fresh osm candidate lands in StatusStaged
// directly rather than StatusNeedsGeocode (application/service.go's own nil-check on these two
// fields already does this, unchanged — osm is simply the first connector to exercise that path for
// real).
//
// Name is never empty here: load already filters out any element with no usable name tag before it
// ever becomes a RawRecord (see elementName/load) — a deliberate data-quality floor, not a silent
// drop the operator would otherwise expect to see counted in recordsFetched.
func (c *Connector) Normalize(raw domain.RawRecord) (domain.NormalizedCandidate, error) {
	var rec osmRecord
	if err := json.Unmarshal(raw.RawPayload, &rec); err != nil {
		return domain.NormalizedCandidate{}, fmt.Errorf("osm: unmarshal raw payload: %w", err)
	}
	tags := rec.Element.Tags
	name := elementName(tags)

	hint := name
	if d := tags["denomination"]; d != "" {
		hint = d
	}

	var jurisdictionHint *string
	if d := tags["diocese"]; d != "" {
		jurisdictionHint = &d
	} else if p := tags["parish"]; p != "" {
		jurisdictionHint = &p
	} else {
		jurisdictionHint = &name
	}

	country := rec.Country
	nc := domain.NormalizedCandidate{
		Name:             name,
		TaxonHint:        &hint,
		JurisdictionHint: jurisdictionHint,
		CountryHint:      &country,
	}
	nc.Latitude, nc.Longitude = coordinatesOf(rec.Element)
	if v := tags["addr:street"]; v != "" {
		nc.Street = &v
	}
	if v := tags["addr:housenumber"]; v != "" {
		nc.HouseNumber = &v
	}
	if v := tags["addr:city"]; v != "" {
		nc.Locality = &v
	}
	if v := tags["addr:state"]; v != "" {
		nc.AdminArea1 = &v
	}
	if v := tags["addr:postcode"]; v != "" {
		nc.PostalCode = &v
	}
	return nc, nil
}

// coordinatesOf returns a node's own lat/lon directly, or a way/relation's `center` (present because
// every query below requests `out center`) — nil/nil in the rare case neither is available (Overpass
// omits center when a way's geometry can't be resolved), which correctly falls back to
// StatusNeedsGeocode downstream, no special handling needed here.
func coordinatesOf(el overpassElement) (lat, lon *float64) {
	if el.Type == "node" {
		return el.Lat, el.Lon
	}
	if el.Center != nil {
		latVal, lonVal := el.Center.Lat, el.Center.Lon
		return &latVal, &lonVal
	}
	return nil, nil
}

// elementName picks the best available name tag — the bare `name` tag first, then a small set of
// locale-specific fallbacks relevant to the target countries' languages. Returns "" when no
// name-shaped tag exists at all, which load uses to filter the element out before it ever becomes a
// RawRecord.
func elementName(tags map[string]string) string {
	if v := tags["name"]; v != "" {
		return v
	}
	for _, k := range []string{"name:es", "name:pt", "name:en"} {
		if v := tags[k]; v != "" {
			return v
		}
	}
	return ""
}

// ensureLoaded queries every configured country exactly once per Connector instance, regardless of
// how many Fetch calls follow — same loadOnce shape as arrnc.ensureLoaded.
func (c *Connector) ensureLoaded(ctx context.Context) error {
	c.loadOnce.Do(func() {
		c.records, c.loadErr = c.load(ctx)
	})
	return c.loadErr
}

// load queries every configured country in turn — one request for a country with no Grid
// configured, or one request per splitGrid cell for one that has (Colombia, today) — pausing
// courtesyDelay between EVERY request, not just between countries, now that a single country can
// issue several. Filters out any element with no usable name (elementName returns "") before it
// ever becomes a record — an anonymous point tagged only amenity/religion/denomination isn't
// reviewable the way this pipeline expects.
func (c *Connector) load(ctx context.Context) ([]osmRecord, error) {
	var out []osmRecord
	first := true
	for _, code := range c.CountryCodes {
		cfg := countries[code] // New already validated every code is present
		regions := []*regionBBox{nil}
		if cfg.Grid != nil {
			cells := splitGrid(cfg.Grid.BBox, cfg.Grid.Rows, cfg.Grid.Cols)
			regions = make([]*regionBBox, len(cells))
			for i := range cells {
				regions[i] = &cells[i]
			}
		}
		for _, region := range regions {
			if !first {
				if err := sleepCtx(ctx, courtesyDelay); err != nil {
					return nil, err
				}
			}
			first = false
			elements, err := c.queryRegion(ctx, code, region)
			if err != nil {
				return nil, fmt.Errorf("osm: query %s (%s) region %v: %w", cfg.Name, code, region, err)
			}
			for _, el := range elements {
				if elementName(el.Tags) == "" {
					continue
				}
				out = append(out, osmRecord{Element: el, Country: cfg.Name})
			}
		}
	}
	return out, nil
}

// queryRegion issues one Overpass QL query for isoCode's administrative area, optionally further
// bounded to bbox (nil means the whole country — real, live-verified query shape 2026-08-14:
// area["ISO3166-1"="UY"][admin_level=2] resolved and returned all 566 real Uruguay elements in ~20s
// against this same timeout). Combining an area filter AND a bbox filter on the same statement is
// valid Overpass QL (both apply as AND) — bbox only bounds how much of the real country polygon one
// request has to search, it never lets a result fall outside the actual country's own boundary.
func (c *Connector) queryRegion(ctx context.Context, isoCode string, bbox *regionBBox) ([]overpassElement, error) {
	bboxClause := ""
	if bbox != nil {
		bboxClause = fmt.Sprintf("(%f,%f,%f,%f)", bbox.South, bbox.West, bbox.North, bbox.East)
	}
	query := fmt.Sprintf(
		"[out:json][timeout:%d];\n"+
			`area["ISO3166-1"="%s"][admin_level=2]->.a;`+"\n"+
			`nwr["amenity"="place_of_worship"]["religion"="christian"](area.a)%s;`+"\n"+
			"out center tags;",
		queryTimeoutSeconds, isoCode, bboxClause,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("osm: build request for %s: %w", c.BaseURL, err)
	}
	q := req.URL.Query()
	q.Set("data", query)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", c.Citation().UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osm: GET %s: %w", req.URL.String(), err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osm: GET %s: unexpected status %s", c.BaseURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("osm: read Overpass response for %s: %w", isoCode, err)
	}
	// Real, live finding (2026-08-14): this mirror sometimes answers an overloaded/failed query with
	// `200 OK` and an HTML error page in the body instead of JSON — a bare status-code check doesn't
	// catch that case, and json.Decode's own error ("invalid character '<' looking for beginning of
	// value") gives no hint what actually went wrong. Detect it explicitly and say so.
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '<' {
		snippet := string(trimmed)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return nil, fmt.Errorf("osm: Overpass returned a non-JSON (HTML) response for %s despite HTTP 200 — likely a server-side overload/error page: %s", isoCode, snippet)
	}

	var parsed overpassResponse
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return nil, fmt.Errorf("osm: decode Overpass response for %s: %w", isoCode, err)
	}
	return parsed.Elements, nil
}

// sleepCtx pauses for d, or returns ctx's error early if it's cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
