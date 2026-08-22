// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M11.5's self-service display-name update against a real Postgres instance — see
// identity_integration_test.go's own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/identity/... -run TestProfileIntegration -v
package identity_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/identity/adapters"
	"github.com/olehmushka/open-faith-map/internal/identity/application"
)

func TestProfileIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	var personID string
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, personID); err != nil {
			t.Errorf("cleanup: delete person: %v", err)
		}
	})

	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.5 Grace Test', 'M11.5', 'Test') RETURNING id`).Scan(&personID); err != nil {
		t.Fatalf("insert person: %v", err)
	}

	svc := application.NewService(adapters.NewStore(pool))

	before, err := svc.GetPerson(ctx, personID)
	if err != nil {
		t.Fatalf("GetPerson(before update): %v", err)
	}
	if before.DisplayName != "M11.5 Grace Test" {
		t.Fatalf("GetPerson(before update).DisplayName = %q, want %q", before.DisplayName, "M11.5 Grace Test")
	}

	after, err := svc.UpdateMyProfile(ctx, personID, "M11.5 Grace Updated")
	if err != nil {
		t.Fatalf("UpdateMyProfile: %v", err)
	}
	if after.DisplayName != "M11.5 Grace Updated" {
		t.Errorf("UpdateMyProfile.DisplayName = %q, want %q", after.DisplayName, "M11.5 Grace Updated")
	}
	if !after.UpdatedAt.After(before.UpdatedAt) && !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("UpdateMyProfile.UpdatedAt = %v, want >= before's %v", after.UpdatedAt, before.UpdatedAt)
	}

	reread, err := svc.GetPerson(ctx, personID)
	if err != nil {
		t.Fatalf("GetPerson(after update): %v", err)
	}
	if reread.DisplayName != "M11.5 Grace Updated" {
		t.Errorf("GetPerson(after update).DisplayName = %q, want %q — write did not persist", reread.DisplayName, "M11.5 Grace Updated")
	}

	// A second, unrelated person's display name is untouched by the first's update.
	var otherID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.5 Henry Test', 'M11.5', 'Test') RETURNING id`).Scan(&otherID); err != nil {
		t.Fatalf("insert other person: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, otherID); err != nil {
			t.Errorf("cleanup: delete other person: %v", err)
		}
	})
	other, err := svc.GetPerson(ctx, otherID)
	if err != nil {
		t.Fatalf("GetPerson(other): %v", err)
	}
	if other.DisplayName != "M11.5 Henry Test" {
		t.Errorf("GetPerson(other).DisplayName = %q, want unchanged %q", other.DisplayName, "M11.5 Henry Test")
	}
}
