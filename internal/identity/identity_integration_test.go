// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves GetPerson/GetPersons/SearchPersons (M10.7, added for core.conjure.yml's admin-app surface)
// against a real Postgres instance — see internal/directory/directory_integration_test.go's own
// header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/identity/... -run TestIdentityPersonReadsIntegration -v
package identity_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/identity/adapters"
	"github.com/olehmushka/open-faith-map/internal/identity/application"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

func TestIdentityPersonReadsIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close) // registered first: LIFO runs DB cleanup (below) before the pool closes.

	var personIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
	})

	svc := application.NewService(adapters.NewStore(pool))

	var aliceID, bobID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M10.7 Alice Test', 'Alice', 'Test') RETURNING id`).Scan(&aliceID); err != nil {
		t.Fatalf("insert alice: %v", err)
	}
	personIDs = append(personIDs, aliceID)
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M10.7 Bob Test', 'Bob', 'Test') RETURNING id`).Scan(&bobID); err != nil {
		t.Fatalf("insert bob: %v", err)
	}
	personIDs = append(personIDs, bobID)

	// --- GetPerson
	alice, err := svc.GetPerson(ctx, aliceID)
	if err != nil {
		t.Fatalf("GetPerson(alice): %v", err)
	}
	if alice.DisplayName != "M10.7 Alice Test" {
		t.Errorf("GetPerson(alice).DisplayName = %q, want %q", alice.DisplayName, "M10.7 Alice Test")
	}
	if _, err := svc.GetPerson(ctx, "00000002-0001-0001-0001-000000000000"); err != domain.ErrPersonNotFound {
		t.Errorf("GetPerson(unknown) error = %v, want ErrPersonNotFound", err)
	}

	// --- GetPersons: batched read returns both, in whatever order, none missing.
	batch, err := svc.GetPersons(ctx, []string{aliceID, bobID})
	if err != nil {
		t.Fatalf("GetPersons: %v", err)
	}
	if len(batch) != 2 {
		t.Fatalf("GetPersons = %+v, want exactly 2 persons", batch)
	}
	seen := map[string]bool{}
	for _, p := range batch {
		seen[p.ID] = true
	}
	if !seen[aliceID] || !seen[bobID] {
		t.Errorf("GetPersons(%v) = %+v, missing one of the requested ids", []string{aliceID, bobID}, batch)
	}
	if empty, err := svc.GetPersons(ctx, nil); err != nil || len(empty) != 0 {
		t.Errorf("GetPersons(nil) = (%+v, %v), want (nil, nil)", empty, err)
	}

	// --- SearchPersons: substring match on display_name finds exactly the M10.7 test rows.
	results, err := svc.SearchPersons(ctx, "M10.7", 10)
	if err != nil {
		t.Fatalf("SearchPersons: %v", err)
	}
	seen = map[string]bool{}
	for _, p := range results {
		seen[p.ID] = true
	}
	if !seen[aliceID] || !seen[bobID] {
		t.Errorf("SearchPersons(%q) = %+v, want it to include both test persons", "M10.7", results)
	}
}
