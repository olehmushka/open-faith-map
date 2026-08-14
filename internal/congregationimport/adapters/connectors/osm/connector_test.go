// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package osm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// fixtureElement builds a minimal overpassElement for tests — a node by default (coordinatesOf's
// simpler path); tests needing a way/relation set Type explicitly.
func fixtureElement(id int64, name string, tags map[string]string) overpassElement {
	lat, lon := -34.9, -56.2
	all := map[string]string{"amenity": "place_of_worship", "religion": "christian"}
	if name != "" {
		all["name"] = name
	}
	for k, v := range tags {
		all[k] = v
	}
	return overpassElement{Type: "node", ID: id, Lat: &lat, Lon: &lon, Tags: all}
}

// TestLoadMultiCountryAndFilters spins up a fake Overpass server that inspects the query's ISO code
// (embedded in the `data` query param) and returns country-specific fixtures — confirms (a) each
// country is queried once, (b) results carry the right literal CountryHint-feeding Country, (c) a
// nameless element is filtered out before ever becoming a record, (d) a name:es-only element is kept
// via the locale fallback.
func TestLoadMultiCountryAndFilters(t *testing.T) {
	var queriedCodes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := r.URL.Query().Get("data")
		var resp overpassResponse
		switch {
		case strings.Contains(data, `"UY"`):
			queriedCodes = append(queriedCodes, "UY")
			resp.Elements = []overpassElement{
				fixtureElement(1, "Iglesia Uno", nil),
				fixtureElement(2, "", nil), // no name tag at all — must be filtered out
			}
		case strings.Contains(data, `"PY"`):
			queriedCodes = append(queriedCodes, "PY")
			resp.Elements = []overpassElement{
				fixtureElement(3, "", map[string]string{"name:es": "Iglesia Tres"}), // locale fallback
			}
		default:
			t.Fatalf("unexpected query, no known ISO code found: %s", data)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := New(srv.URL, []string{"UY", "PY"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	records, err := c.load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := queriedCodes, []string{"UY", "PY"}; !equalStrings(got, want) {
		t.Fatalf("queried codes = %v, want %v (each country queried once, in order)", got, want)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2 (element 2 has no name tag and must be filtered out)", len(records))
	}
	if records[0].Element.ID != 1 || records[0].Country != "Uruguay" {
		t.Fatalf("record 0 = %+v, want element 1 tagged Uruguay", records[0])
	}
	if records[1].Element.ID != 3 || records[1].Country != "Paraguay" {
		t.Fatalf("record 1 = %+v, want element 3 (name:es fallback) tagged Paraguay", records[1])
	}
}

// TestLoadSplitsColombiaOnly confirms the real 2026-08-14 fix: Colombia (configured with a Grid)
// issues one request per grid cell (3*2=6), each scoped to a distinct bbox, while an unsplit
// country in the same run (Uruguay) still issues exactly one whole-country request.
func TestLoadSplitsColombiaOnly(t *testing.T) {
	var uyRequests, coRequests int
	var coBBoxes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := r.URL.Query().Get("data")
		switch {
		case strings.Contains(data, `"UY"`):
			uyRequests++
			if strings.Contains(data, "(area.a)(") {
				t.Fatalf("UY (unsplit) query unexpectedly carries a bbox clause: %s", data)
			}
		case strings.Contains(data, `"CO"`):
			coRequests++
			const prefix = "(area.a)("
			start := strings.Index(data, prefix)
			if start == -1 {
				t.Fatalf("CO (split) query is missing its bbox clause: %s", data)
			}
			afterPrefix := start + len(prefix)
			end := strings.Index(data[afterPrefix:], ")")
			if end == -1 {
				t.Fatalf("CO (split) query's bbox clause is unterminated: %s", data)
			}
			coBBoxes = append(coBBoxes, data[afterPrefix:afterPrefix+end])
		default:
			t.Fatalf("unexpected query: %s", data)
		}
		_ = json.NewEncoder(w).Encode(overpassResponse{})
	}))
	defer srv.Close()

	c, err := New(srv.URL, []string{"UY", "CO"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.load(context.Background()); err != nil {
		t.Fatal(err)
	}

	if uyRequests != 1 {
		t.Fatalf("UY requests = %d, want exactly 1 (no Grid configured)", uyRequests)
	}
	wantCOCells := countries["CO"].Grid.Rows * countries["CO"].Grid.Cols
	if coRequests != wantCOCells {
		t.Fatalf("CO requests = %d, want %d (Rows*Cols)", coRequests, wantCOCells)
	}
	distinct := make(map[string]bool, len(coBBoxes))
	for _, b := range coBBoxes {
		distinct[b] = true
	}
	if len(distinct) != wantCOCells {
		t.Fatalf("CO issued %d requests but only %d distinct bboxes — cells are not actually covering different regions", coRequests, len(distinct))
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestQueryCountryRequestShape confirms the actual HTTP request: GET, the ISO code and the
// amenity/religion filters present in the query string, and a non-empty User-Agent.
func TestQueryCountryRequestShape(t *testing.T) {
	var gotMethod, gotUA string
	var gotData string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotUA = r.Header.Get("User-Agent")
		gotData = r.URL.Query().Get("data")
		_ = json.NewEncoder(w).Encode(overpassResponse{})
	}))
	defer srv.Close()

	c, err := New(srv.URL, []string{"CL"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.queryRegion(context.Background(), "CL", nil); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", gotMethod)
	}
	if gotUA == "" {
		t.Fatal("User-Agent header was empty")
	}
	for _, want := range []string{`"CL"`, `amenity`, `place_of_worship`, `religion`, `christian`} {
		if !strings.Contains(gotData, want) {
			t.Fatalf("query %q does not contain %q", gotData, want)
		}
	}
}

// TestFetchMultiBatch mirrors arrnc's own equivalent test: this connector loads everything into
// memory up front and slices an already-materialized slice, so the failure mode to guard against is
// an off-by-one at a batch boundary, not ua-edr's reopen-and-reskip cursor-doubling class of bug.
func TestFetchMultiBatch(t *testing.T) {
	const wantTotal = 1200 // > 2*batchSize, forcing 3 Fetch calls (500, 500, 200)
	c := &Connector{}
	c.loadOnce.Do(func() {}) // mark as already loaded — skip real HTTP
	for i := 0; i < wantTotal; i++ {
		c.records = append(c.records, osmRecord{
			Element: overpassElement{Type: "node", ID: int64(i)},
			Country: "Uruguay",
		})
	}

	ctx := context.Background()
	var cursor *string
	seenIDs := make(map[string]bool)
	calls := 0
	for {
		batch, next, err := c.Fetch(ctx, cursor)
		if err != nil {
			t.Fatalf("Fetch (call %d): %v", calls+1, err)
		}
		calls++
		for _, r := range batch {
			if seenIDs[r.SourceRecordID] {
				t.Fatalf("SourceRecordID %s yielded more than once (call %d)", r.SourceRecordID, calls)
			}
			seenIDs[r.SourceRecordID] = true
		}
		if next == nil {
			break
		}
		cursor = next
		if calls > 10 {
			t.Fatal("too many Fetch calls — cursor likely stuck")
		}
	}

	if len(seenIDs) != wantTotal {
		t.Fatalf("got %d distinct records across %d calls, want %d", len(seenIDs), calls, wantTotal)
	}
	if calls != 3 {
		t.Fatalf("got %d Fetch calls, want exactly 3 (500,500,200)", calls)
	}
}

func TestFetchSingleBatchExhaustion(t *testing.T) {
	c := &Connector{}
	c.loadOnce.Do(func() {})
	c.records = []osmRecord{
		{Element: overpassElement{Type: "node", ID: 1}, Country: "Uruguay"},
		{Element: overpassElement{Type: "way", ID: 2}, Country: "Uruguay"},
	}
	batch, next, err := c.Fetch(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 2 {
		t.Fatalf("got %d records, want 2", len(batch))
	}
	if next != nil {
		t.Fatalf("got non-nil next cursor %q for a fully-exhausted single batch", *next)
	}
	if batch[0].SourceRecordID != "node/1" || batch[1].SourceRecordID != "way/2" {
		t.Fatalf("SourceRecordIDs = %q, %q — want node/1, way/2", batch[0].SourceRecordID, batch[1].SourceRecordID)
	}
}

func TestCoordinatesOf(t *testing.T) {
	lat, lon := -34.9, -56.2
	node := overpassElement{Type: "node", Lat: &lat, Lon: &lon}
	if gotLat, gotLon := coordinatesOf(node); gotLat == nil || gotLon == nil || *gotLat != lat || *gotLon != lon {
		t.Fatalf("node coordinates = %v, %v, want %v, %v", gotLat, gotLon, lat, lon)
	}

	way := overpassElement{Type: "way", Center: &overpassCenter{Lat: lat, Lon: lon}}
	if gotLat, gotLon := coordinatesOf(way); gotLat == nil || gotLon == nil || *gotLat != lat || *gotLon != lon {
		t.Fatalf("way center coordinates = %v, %v, want %v, %v", gotLat, gotLon, lat, lon)
	}

	wayNoCenter := overpassElement{Type: "way"}
	if gotLat, gotLon := coordinatesOf(wayNoCenter); gotLat != nil || gotLon != nil {
		t.Fatalf("way with no center should yield nil, nil coordinates, got %v, %v", gotLat, gotLon)
	}
}

func TestElementName(t *testing.T) {
	cases := []struct {
		name string
		tags map[string]string
		want string
	}{
		{"bare name", map[string]string{"name": "Iglesia Uno"}, "Iglesia Uno"},
		{"name:es fallback", map[string]string{"name:es": "Iglesia Dos"}, "Iglesia Dos"},
		{"name:pt fallback", map[string]string{"name:pt": "Igreja Tres"}, "Igreja Tres"},
		{"no name at all", map[string]string{"denomination": "catholic"}, ""},
		{"prefers bare name over locale", map[string]string{"name": "A", "name:es": "B"}, "A"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := elementName(tc.tags); got != tc.want {
				t.Fatalf("elementName(%v) = %q, want %q", tc.tags, got, tc.want)
			}
		})
	}
}

// TestNormalize confirms the full field mapping, including the real, live-verified
// denomination/diocese vocabulary and the fallback chains when those tags are absent.
func TestNormalize(t *testing.T) {
	c := &Connector{}

	t.Run("full tags", func(t *testing.T) {
		rec := osmRecord{
			Country: "Uruguay",
			Element: overpassElement{
				Type:   "way",
				ID:     37519466,
				Center: &overpassCenter{Lat: -34.8518342, Lon: -56.2287703},
				Tags: map[string]string{
					"name":             "Parroquia San Francisco de Asís",
					"denomination":     "roman_catholic",
					"diocese":          "Arquidiócesis de Montevideo",
					"addr:street":      "Vitoria",
					"addr:housenumber": "59",
					"addr:city":        "Montevideo",
					"addr:postcode":    "12900",
					"religion":         "christian",
				},
			},
		}
		payload, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		nc, err := c.Normalize(domain.RawRecord{RawPayload: payload})
		if err != nil {
			t.Fatal(err)
		}
		if nc.Name != "Parroquia San Francisco de Asís" {
			t.Fatalf("Name = %q", nc.Name)
		}
		if nc.TaxonHint == nil || *nc.TaxonHint != "roman_catholic" {
			t.Fatalf("TaxonHint = %v, want denomination tag", nc.TaxonHint)
		}
		if nc.JurisdictionHint == nil || *nc.JurisdictionHint != "Arquidiócesis de Montevideo" {
			t.Fatalf("JurisdictionHint = %v, want diocese tag", nc.JurisdictionHint)
		}
		if nc.CountryHint == nil || *nc.CountryHint != "Uruguay" {
			t.Fatalf("CountryHint = %v", nc.CountryHint)
		}
		if nc.Latitude == nil || nc.Longitude == nil || *nc.Latitude != -34.8518342 || *nc.Longitude != -56.2287703 {
			t.Fatalf("coordinates = %v, %v — want the way's own center", nc.Latitude, nc.Longitude)
		}
		if nc.Street == nil || *nc.Street != "Vitoria" {
			t.Fatalf("Street = %v", nc.Street)
		}
		if nc.HouseNumber == nil || *nc.HouseNumber != "59" {
			t.Fatalf("HouseNumber = %v", nc.HouseNumber)
		}
		if nc.Locality == nil || *nc.Locality != "Montevideo" {
			t.Fatalf("Locality = %v", nc.Locality)
		}
		if nc.PostalCode == nil || *nc.PostalCode != "12900" {
			t.Fatalf("PostalCode = %v", nc.PostalCode)
		}
	})

	t.Run("no denomination or diocese falls back to name", func(t *testing.T) {
		rec := osmRecord{
			Country: "Chile",
			Element: overpassElement{
				Type: "node",
				ID:   1,
				Lat:  floatPtr(-33.4),
				Lon:  floatPtr(-70.6),
				Tags: map[string]string{"name": "Iglesia Bautista Central"},
			},
		}
		payload, _ := json.Marshal(rec)
		nc, err := c.Normalize(domain.RawRecord{RawPayload: payload})
		if err != nil {
			t.Fatal(err)
		}
		if nc.TaxonHint == nil || *nc.TaxonHint != "Iglesia Bautista Central" {
			t.Fatalf("TaxonHint = %v, want fallback to Name", nc.TaxonHint)
		}
		if nc.JurisdictionHint == nil || *nc.JurisdictionHint != "Iglesia Bautista Central" {
			t.Fatalf("JurisdictionHint = %v, want fallback to Name", nc.JurisdictionHint)
		}
		if nc.Street != nil {
			t.Fatalf("Street = %v, want nil (no addr:street tag)", nc.Street)
		}
	})
}

func floatPtr(f float64) *float64 { return &f }

func TestNewValidation(t *testing.T) {
	if _, err := New("", nil, nil); err == nil {
		t.Fatal("New with no country codes should error")
	}
	if _, err := New("", []string{"XX"}, nil); err == nil {
		t.Fatal("New with an unknown country code should error")
	}
	c, err := New("", []string{"UY"}, nil)
	if err != nil {
		t.Fatalf("New with a known country code should succeed: %v", err)
	}
	if c.BaseURL != defaultBaseURL {
		t.Fatalf("BaseURL = %q, want default %q when unset", c.BaseURL, defaultBaseURL)
	}
}

// TestQueryCountryErrorStatus confirms a non-200 response is surfaced as an error, not silently
// treated as zero results.
func TestQueryCountryErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, err := New(srv.URL, []string{"UY"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.queryRegion(context.Background(), "UY", nil); err == nil {
		t.Fatal("expected an error on a non-200 response, got nil")
	}
}

// TestQueryRegionHTMLErrorPage is the regression test for the real 2026-08-14 finding: this mirror
// sometimes answers an overloaded query with 200 OK and an HTML error page instead of JSON — must
// be caught explicitly, not surfaced as a confusing raw JSON-decode error.
func TestQueryRegionHTMLErrorPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html><body><p>Error: runtime error: the server is probably too busy.</p></body></html>`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, []string{"CO"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.queryRegion(context.Background(), "CO", nil)
	if err == nil {
		t.Fatal("expected an error for an HTML response, got nil")
	}
	if !strings.Contains(err.Error(), "non-JSON") {
		t.Fatalf("error = %q, want it to explicitly call out a non-JSON response, not a raw decode error", err.Error())
	}
}

// TestSplitGrid confirms the pure grid-splitting math: cell count, and that adjacent cells share an
// edge with no gap or overlap.
func TestSplitGrid(t *testing.T) {
	b := regionBBox{South: 0, West: 0, North: 9, East: 4}
	cells := splitGrid(b, 3, 2)
	if len(cells) != 6 {
		t.Fatalf("got %d cells, want 3*2=6", len(cells))
	}
	// Row-major: cell 0 is the southwest-most cell.
	if cells[0].South != 0 || cells[0].North != 3 || cells[0].West != 0 || cells[0].East != 2 {
		t.Fatalf("cells[0] = %+v, want {South:0 North:3 West:0 East:2}", cells[0])
	}
	// Last cell reaches the original bbox's own North/East exactly (no rounding gap).
	last := cells[len(cells)-1]
	if last.North != b.North || last.East != b.East {
		t.Fatalf("last cell = %+v, does not reach the original bbox's North/East (%v, %v)", last, b.North, b.East)
	}
	// Adjacent rows share an edge exactly (cell 2's South == cell 0's North).
	if cells[2].South != cells[0].North {
		t.Fatalf("cells[2].South = %v, want == cells[0].North = %v (a gap or overlap between rows)", cells[2].South, cells[0].North)
	}
}

// TestClone is the regression test for the real 2026-08-14 staleness bug: a long-lived instance's
// loadOnce-cached records must never leak into a fresh run. The fake server returns a DIFFERENT
// element count on its second call — a clone sharing the original's cache would still report the
// first call's result; a genuinely fresh instance re-queries and sees the current one.
func TestClone(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		n := 1
		if calls > 1 {
			n = 2
		}
		var resp overpassResponse
		for i := 0; i < n; i++ {
			resp.Elements = append(resp.Elements, fixtureElement(int64(i), "Iglesia", nil))
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	original, err := New(srv.URL, []string{"UY"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	batch, _, err := original.Fetch(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 1 {
		t.Fatalf("original fetched %d records, want 1", len(batch))
	}

	clone := original.Clone()
	batch2, _, err := clone.Fetch(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch2) != 2 {
		t.Fatalf("clone fetched %d records, want 2 (a fresh query, not the original's cached 1-element result)", len(batch2))
	}
}

// TestWithParameters covers osm's one ConnectorConfigurable knob: countryCodes.
func TestWithParameters(t *testing.T) {
	base, err := New("", []string{"UY"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid override", func(t *testing.T) {
		configured, err := base.WithParameters(map[string]string{"countryCodes": "PY,CL"})
		if err != nil {
			t.Fatal(err)
		}
		c := configured.(*Connector)
		if len(c.CountryCodes) != 2 || c.CountryCodes[0] != "PY" || c.CountryCodes[1] != "CL" {
			t.Fatalf("CountryCodes = %v, want [PY CL]", c.CountryCodes)
		}
		if c.BaseURL != base.BaseURL {
			t.Fatalf("BaseURL = %q, want inherited %q (deployment config, not a per-run choice)", c.BaseURL, base.BaseURL)
		}
	})

	t.Run("unrecognized key rejected", func(t *testing.T) {
		if _, err := base.WithParameters(map[string]string{"countryCode": "PY"}); err == nil {
			t.Fatal("expected an error for an unrecognized parameter key, got nil")
		}
	})

	t.Run("unknown country code rejected", func(t *testing.T) {
		if _, err := base.WithParameters(map[string]string{"countryCodes": "XX"}); err == nil {
			t.Fatal("expected an error for an unknown country code, got nil")
		}
	})

	t.Run("blank countryCodes value rejected", func(t *testing.T) {
		if _, err := base.WithParameters(map[string]string{"countryCodes": " , "}); err == nil {
			t.Fatal("expected an error for an effectively-empty countryCodes value, got nil")
		}
	})

	t.Run("empty parameters map behaves like no override", func(t *testing.T) {
		configured, err := base.WithParameters(map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		c := configured.(*Connector)
		if len(c.CountryCodes) != 1 || c.CountryCodes[0] != "UY" {
			t.Fatalf("CountryCodes = %v, want unchanged [UY]", c.CountryCodes)
		}
	})
}
