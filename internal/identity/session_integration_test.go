// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M11.3's D-SessionTracking against a real Postgres instance — see
// identity_integration_test.go's own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/identity/... -run TestSessionIntegration -v
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

func TestSessionIntegration(t *testing.T) {
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
		// Sessions cascade off the account delete (identity_sessions.account_id ON DELETE CASCADE);
		// accounts must go before persons (identity_accounts.person_id is ON DELETE RESTRICT).
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_accounts WHERE person_id = ANY($1)`, personIDs); err != nil {
			t.Errorf("cleanup: delete accounts: %v", err)
		}
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
	})

	svc := application.NewService(adapters.NewRepository(pool))
	const issuer = "urn:test:m11-3-issuer"

	insertPersonWithAccount := func(name, email string) (personID, accountID string) {
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
			VALUES ($1, 'M11.3', 'Test') RETURNING id`, name).Scan(&personID); err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		personIDs = append(personIDs, personID)
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_accounts (person_id, email)
			VALUES ($1, $2) RETURNING id`, personID, email).Scan(&accountID); err != nil {
			t.Fatalf("insert account for %s: %v", name, err)
		}
		return personID, accountID
	}

	eveID, eveAccountID := insertPersonWithAccount("M11.3 Eve Test", "eve-m11-3@example.test")
	frankID, frankAccountID := insertPersonWithAccount("M11.3 Frank Test", "frank-m11-3@example.test")

	// --- ListSessions is empty before any session exists.
	if sessions, err := svc.ListSessions(ctx, eveID); err != nil || len(sessions) != 0 {
		t.Fatalf("ListSessions(eve, none yet) = (%+v, %v), want (empty, nil)", sessions, err)
	}

	// --- RegisterSession creates a row; ListSessions/Touch both see it.
	sess, err := svc.RegisterSession(ctx, eveAccountID, issuer, "test-agent/1.0")
	if err != nil {
		t.Fatalf("RegisterSession(eve): %v", err)
	}
	if sess.AccountID != eveAccountID || sess.RevokedAt != nil {
		t.Fatalf("RegisterSession(eve) = %+v, want AccountID=%q, RevokedAt=nil", sess, eveAccountID)
	}

	sessions, err := svc.ListSessions(ctx, eveID)
	if err != nil || len(sessions) != 1 || sessions[0].ID != sess.ID {
		t.Fatalf("ListSessions(eve, one session) = (%+v, %v), want exactly [%q]", sessions, err, sess.ID)
	}

	if accountID, err := svc.Touch(ctx, sess.ID); err != nil || accountID != eveAccountID {
		t.Errorf("Touch(eve's session) = (%q, %v), want (%q, nil)", accountID, err, eveAccountID)
	}

	// --- A cross-account revoke attempt (frank revoking eve's session id) is rejected without
	// mutating the row — proves an admin-supplied personId can't be used to guess-revoke someone
	// else's session.
	if _, err := svc.RevokeSession(ctx, frankID, sess.ID); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("RevokeSession(frank, eve's sessionId) error = %v, want ErrSessionNotFound", err)
	}
	if accountID, err := svc.Touch(ctx, sess.ID); err != nil || accountID != eveAccountID {
		t.Errorf("Touch(eve's session) after a rejected cross-account revoke = (%q, %v), want still (%q, nil)", accountID, err, eveAccountID)
	}

	// --- The owning person can revoke it; a subsequent Touch reports ErrSessionRevoked.
	revoked, err := svc.RevokeSession(ctx, eveID, sess.ID)
	if err != nil || revoked.RevokedAt == nil {
		t.Fatalf("RevokeSession(eve, own session) = (%+v, %v), want RevokedAt set, nil error", revoked, err)
	}
	if _, err := svc.Touch(ctx, sess.ID); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Errorf("Touch(revoked session) error = %v, want ErrSessionRevoked", err)
	}
	if sessions, err := svc.ListSessions(ctx, eveID); err != nil || len(sessions) != 0 {
		t.Errorf("ListSessions(eve, after revoke) = (%+v, %v), want (empty, nil) — revoked sessions are not active", sessions, err)
	}

	// Revoke is idempotent.
	if _, err := svc.RevokeSession(ctx, eveID, sess.ID); err != nil {
		t.Errorf("RevokeSession(eve, already-revoked session) error = %v, want nil (idempotent)", err)
	}

	// --- Touch on an unknown session id.
	if _, err := svc.Touch(ctx, "00000000-0000-8000-8000-000000000000"); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("Touch(unknown session) error = %v, want ErrSessionNotFound", err)
	}

	// --- Self-scoped ListMySessions/RevokeMySession mirror the admin-scoped behavior for frank.
	frankSess, err := svc.RegisterSession(ctx, frankAccountID, issuer, "")
	if err != nil {
		t.Fatalf("RegisterSession(frank): %v", err)
	}
	if my, err := svc.ListMySessions(ctx, frankAccountID); err != nil || len(my) != 1 || my[0].ID != frankSess.ID {
		t.Fatalf("ListMySessions(frank) = (%+v, %v), want exactly [%q]", my, err, frankSess.ID)
	}
	if _, err := svc.RevokeMySession(ctx, eveAccountID, frankSess.ID); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("RevokeMySession(eve, frank's sessionId) error = %v, want ErrSessionNotFound", err)
	}
	if _, err := svc.RevokeMySession(ctx, frankAccountID, frankSess.ID); err != nil {
		t.Errorf("RevokeMySession(frank, own session) error = %v, want nil", err)
	}
}
