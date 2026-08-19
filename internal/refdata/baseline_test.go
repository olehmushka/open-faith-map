// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M10.9's country-parity proof: diffs testdata/oikumenea_baseline_countries.json — captured live
// against the still-running oikumenea schema immediately before M10.8's teardown (see
// testdata/README.md for full provenance) — against internal/refdata.Service.ListCountries()
// post-teardown. Matched by code, not id: M10.1 minted OpenFaithMap's own RIDs, never reused
// oikumenea's (D-OwnRIDs). Same DATABASE_URL-gated invocation as refdata_integration_test.go.
package refdata_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/refdata/application"
)

type baselineCountry struct {
	Code  string            `json:"code"`
	Name  string            `json:"name"`
	Names map[string]string `json:"names"`
}

func TestCountryBaselineParity(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	raw, err := os.ReadFile("testdata/oikumenea_baseline_countries.json")
	if err != nil {
		t.Fatalf("read baseline fixture: %v", err)
	}
	var baseline []baselineCountry
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatalf("parse baseline fixture: %v", err)
	}
	if len(baseline) != 250 {
		t.Fatalf("baseline fixture has %d rows, want 250 (sanity check on the fixture itself, not the live data)", len(baseline))
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	countries, err := application.NewService(pool).ListCountries(ctx)
	if err != nil {
		t.Fatalf("ListCountries: %v", err)
	}

	byCode := make(map[string]int, len(countries)) // code -> index, for exact-set-equality + O(1) lookup
	for i, c := range countries {
		byCode[c.Code] = i
	}

	if len(countries) != len(baseline) {
		t.Errorf("ListCountries returned %d countries, baseline has %d", len(countries), len(baseline))
	}

	seenCodes := make(map[string]bool, len(baseline))
	for _, b := range baseline {
		seenCodes[b.Code] = true
		idx, ok := byCode[b.Code]
		if !ok {
			t.Errorf("baseline code %s missing from live ListCountries", b.Code)
			continue
		}
		live := countries[idx]
		if live.Name != b.Name {
			t.Errorf("%s: Name = %q, baseline had %q", b.Code, live.Name, b.Name)
		}
		for locale, wantName := range b.Names {
			if gotName := live.Names[locale]; gotName != wantName {
				t.Errorf("%s: Names[%q] = %q, baseline had %q", b.Code, locale, gotName, wantName)
			}
		}
		if len(live.Names) != len(b.Names) {
			t.Errorf("%s: live has %d locale names, baseline has %d", b.Code, len(live.Names), len(b.Names))
		}
	}

	// The other direction of exact-set-equality: nothing live that wasn't in the baseline either.
	for _, c := range countries {
		if !seenCodes[c.Code] {
			t.Errorf("live code %s not present in the pre-teardown baseline", c.Code)
		}
	}
}
