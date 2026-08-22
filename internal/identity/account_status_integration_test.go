// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M11.1's D-AccountStatusEnforcement against a real Postgres instance — see
// identity_integration_test.go's own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/identity/... -run TestAccountStatusIntegration -v
package identity_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/identity/adapters"
	"github.com/olehmushka/open-faith-map/internal/identity/application"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

func TestAccountStatusIntegration(t *testing.T) {
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

	var personIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		// Accounts must go before persons (identity_accounts.person_id is ON DELETE RESTRICT);
		// identity_external_identities cascades off the account delete.
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_accounts WHERE person_id = ANY($1)`, personIDs); err != nil {
			t.Errorf("cleanup: delete accounts: %v", err)
		}
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
	})

	svc := application.NewService(adapters.NewStore(pool))
	const issuer = "urn:test:m11-1-issuer"

	insertPerson := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
			VALUES ($1, 'M11.1', 'Test') RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		personIDs = append(personIDs, id)
		return id
	}

	// --- Carol: has an account, linked to one external identity.
	carolID := insertPerson("M11.1 Carol Test")
	var carolAccountID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_accounts (person_id, email)
		VALUES ($1, 'carol-m11-1@example.test') RETURNING id`, carolID).Scan(&carolAccountID); err != nil {
		t.Fatalf("insert carol account: %v", err)
	}
	const carolSubject = "carol-sub"
	if _, err := pool.Exec(ctx, `
		INSERT INTO openfaithmap.identity_external_identities (account_id, issuer, subject)
		VALUES ($1, $2, $3)`, carolAccountID, issuer, carolSubject); err != nil {
		t.Fatalf("insert carol external identity: %v", err)
	}

	// Resolve works while active.
	res, err := svc.Resolve(ctx, issuer, carolSubject)
	if err != nil {
		t.Fatalf("Resolve(active carol): %v", err)
	}
	if res.PersonID != carolID {
		t.Errorf("Resolve(active carol).PersonID = %q, want %q", res.PersonID, carolID)
	}

	// AccountStatus reports "active" while active.
	if status, found, err := svc.AccountStatus(ctx, carolID); err != nil || !found || status != domain.AccountStatusActive {
		t.Errorf("AccountStatus(carol) = (%q, %v, %v), want (%q, true, nil)", status, found, err, domain.AccountStatusActive)
	}

	// --- Deactivate: Resolve must now reject, LinkOnMatch must reject (not silently re-link or
	// double-provision a second account for the same person).
	if _, account, err := svc.Deactivate(ctx, carolID); err != nil || account.Status != domain.AccountStatusDisabled {
		t.Fatalf("Deactivate(carol) = (%+v, %v), want status=disabled, nil error", account, err)
	}
	if _, err := svc.Resolve(ctx, issuer, carolSubject); !errors.Is(err, domain.ErrIdentityNotFound) {
		t.Errorf("Resolve(disabled carol) error = %v, want ErrIdentityNotFound", err)
	}
	if _, err := svc.LinkOnMatch(ctx, carolID, issuer, carolSubject, "carol-m11-1@example.test"); !errors.Is(err, domain.ErrAccountDisabled) {
		t.Errorf("LinkOnMatch(disabled carol) error = %v, want ErrAccountDisabled", err)
	}
	var accountCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM openfaithmap.identity_accounts WHERE person_id = $1`, carolID).Scan(&accountCount); err != nil {
		t.Fatalf("count carol accounts: %v", err)
	}
	if accountCount != 1 {
		t.Errorf("carol has %d accounts after a rejected LinkOnMatch, want exactly 1 (no duplicate provisioning)", accountCount)
	}

	// Deactivate is idempotent.
	if _, account, err := svc.Deactivate(ctx, carolID); err != nil || account.Status != domain.AccountStatusDisabled {
		t.Errorf("Deactivate(already-disabled carol) = (%+v, %v), want status=disabled, nil error", account, err)
	}

	// --- Reactivate reverses it.
	if _, account, err := svc.Reactivate(ctx, carolID); err != nil || account.Status != domain.AccountStatusActive {
		t.Fatalf("Reactivate(carol) = (%+v, %v), want status=active, nil error", account, err)
	}
	if res, err := svc.Resolve(ctx, issuer, carolSubject); err != nil || res.PersonID != carolID {
		t.Errorf("Resolve(reactivated carol) = (%+v, %v), want carol resolved with no error", res, err)
	}
	// Reactivate is idempotent.
	if _, account, err := svc.Reactivate(ctx, carolID); err != nil || account.Status != domain.AccountStatusActive {
		t.Errorf("Reactivate(already-active carol) = (%+v, %v), want status=active, nil error", account, err)
	}

	// --- Dave: never had an account.
	daveID := insertPerson("M11.1 Dave Test")
	if status, found, err := svc.AccountStatus(ctx, daveID); err != nil || found || status != "" {
		t.Errorf("AccountStatus(dave, no account) = (%q, %v, %v), want (\"\", false, nil)", status, found, err)
	}
	if _, _, err := svc.Deactivate(ctx, daveID); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Errorf("Deactivate(dave, no account) error = %v, want ErrAccountNotFound", err)
	}
}
