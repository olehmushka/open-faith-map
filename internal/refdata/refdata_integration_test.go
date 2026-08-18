// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves internal/refdata reads the exact M10.1-seeded country data against a real Postgres
// instance — see internal/directory/directory_integration_test.go's own header comment for the
// invocation. The live table actually holds 250 rows (Kosovo/XK included), not the 249
// migrations/0021_core_refdata.sql's own header comment claims — found live by this test, not
// assumed; the module code is correct, the migration comment is off by one.
package refdata_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/refdata/application"
)

func TestRefdataIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
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
	if len(countries) != 250 {
		t.Errorf("len(countries) = %d, want 250 (249 ISO-3166-1 + Kosovo/XK)", len(countries))
	}
	total := 0
	for _, c := range countries {
		total += len(c.Names)
	}
	if total != 1000 { // 250 countries x 4 locales
		t.Errorf("total name rows = %d, want 1000", total)
	}
	var sawUA bool
	for _, c := range countries {
		if c.Code == "UA" {
			sawUA = true
			if c.Names["ukr"] != "Україна" {
				t.Errorf("UA ukr name = %q, want Україна", c.Names["ukr"])
			}
		}
	}
	if !sawUA {
		t.Error("UA not found in ListCountries")
	}
}
