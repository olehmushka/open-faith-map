// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M11.6's D-InviteLinkMVP against a real Postgres instance — see
// identity_integration_test.go's own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/identity/... -run TestInviteIntegration -v
package identity_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/identity/adapters"
	"github.com/olehmushka/open-faith-map/internal/identity/application"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

func TestInviteIntegration(t *testing.T) {
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
		// identity_invites references both accounts and persons (ON DELETE RESTRICT on both), so it
		// must go first; accounts must go before persons for the same reason.
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_invites WHERE person_id = ANY($1)`, personIDs); err != nil {
			t.Errorf("cleanup: delete invites: %v", err)
		}
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
	const issuer = "urn:test:m11-6-issuer"

	insertAdmin := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
			VALUES ($1, 'M11.6', 'Test') RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		personIDs = append(personIDs, id)
		return id
	}
	trackInvitePerson := func(invite domain.Invite) {
		personIDs = append(personIDs, invite.PersonID)
	}

	adminID := insertAdmin("M11.6 Admin Test")

	// --- CreateInvite: pre-provisions a Person+Account+Invite in one call, active status, pending.
	invite, rawToken, err := svc.CreateInvite(ctx, "grace-m11-6@example.test", "M11.6 Grace Test", adminID)
	if err != nil {
		t.Fatalf("CreateInvite(grace): %v", err)
	}
	trackInvitePerson(invite)
	if rawToken == "" {
		t.Fatal("CreateInvite(grace) returned an empty raw token")
	}
	if invite.Status != domain.InviteStatusPending {
		t.Errorf("CreateInvite(grace).Status = %q, want %q", invite.Status, domain.InviteStatusPending)
	}
	if invite.Email != "grace-m11-6@example.test" {
		t.Errorf("CreateInvite(grace).Email = %q, want the invited email", invite.Email)
	}
	var accountStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM openfaithmap.identity_accounts WHERE id = $1`, invite.AccountID).Scan(&accountStatus); err != nil {
		t.Fatalf("read grace's account status: %v", err)
	}
	if accountStatus != domain.AccountStatusActive {
		t.Errorf("grace's pre-provisioned account status = %q, want %q", accountStatus, domain.AccountStatusActive)
	}

	// --- CreateInvite rejects a second invite for the same (now active) email.
	if _, _, err := svc.CreateInvite(ctx, "grace-m11-6@example.test", "Duplicate Grace", adminID); !errors.Is(err, domain.ErrAccountAlreadyExists) {
		t.Errorf("CreateInvite(duplicate email) error = %v, want ErrAccountAlreadyExists", err)
	}

	// --- ResolveInvite: happy path returns the invitee's display info.
	info, err := svc.ResolveInvite(ctx, rawToken)
	if err != nil {
		t.Fatalf("ResolveInvite(grace's token): %v", err)
	}
	if info.DisplayName != "M11.6 Grace Test" || info.Email != "grace-m11-6@example.test" {
		t.Errorf("ResolveInvite(grace's token) = %+v, want grace's display name/email", info)
	}

	// --- ResolveInvite on an unknown token.
	if _, err := svc.ResolveInvite(ctx, "not-a-real-token"); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Errorf("ResolveInvite(bogus token) error = %v, want ErrInviteNotFound", err)
	}

	// --- LinkOnMatch's acceptance hook: a real JIT link flips the invite pending -> accepted, and a
	// second ResolveInvite call on the same token now reports ErrInviteAlreadyAccepted.
	const graceSubject = "grace-sub"
	if _, err := svc.LinkOnMatch(ctx, invite.PersonID, issuer, graceSubject, "grace-m11-6@example.test"); err != nil {
		t.Fatalf("LinkOnMatch(grace): %v", err)
	}
	var inviteStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM openfaithmap.identity_invites WHERE id = $1`, invite.ID).Scan(&inviteStatus); err != nil {
		t.Fatalf("read grace's invite status: %v", err)
	}
	if inviteStatus != domain.InviteStatusAccepted {
		t.Errorf("grace's invite status after LinkOnMatch = %q, want %q", inviteStatus, domain.InviteStatusAccepted)
	}
	if _, err := svc.ResolveInvite(ctx, rawToken); !errors.Is(err, domain.ErrInviteAlreadyAccepted) {
		t.Errorf("ResolveInvite(grace's token, after acceptance) error = %v, want ErrInviteAlreadyAccepted", err)
	}

	// --- A non-invite JIT link (no pending invite for that account) leaves MarkInviteAcceptedByAccount
	// a no-op — proved implicitly above already succeeding is not enough; assert directly that a
	// second, unrelated LinkOnMatch on an account with no invite at all does not error.
	henryID := insertAdmin("M11.6 Henry Test")
	var henryAccountID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_accounts (person_id, email)
		VALUES ($1, 'henry-m11-6@example.test') RETURNING id`, henryID).Scan(&henryAccountID); err != nil {
		t.Fatalf("insert henry account: %v", err)
	}
	if _, err := svc.LinkOnMatch(ctx, henryID, issuer, "henry-sub", "henry-m11-6@example.test"); err != nil {
		t.Errorf("LinkOnMatch(henry, no invite) error = %v, want nil (MarkInviteAcceptedByAccount is a no-op)", err)
	}

	// --- Expired invite: CreateInvite a fresh one, backdate expires_at directly (same technique
	// M11.4's own live verification used for last_seen_at), confirm ResolveInvite rejects it.
	expiredInvite, expiredToken, err := svc.CreateInvite(ctx, "ivy-m11-6@example.test", "M11.6 Ivy Test", adminID)
	if err != nil {
		t.Fatalf("CreateInvite(ivy): %v", err)
	}
	trackInvitePerson(expiredInvite)
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.identity_invites SET expires_at = $2 WHERE id = $1`,
		expiredInvite.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("backdate ivy's invite: %v", err)
	}
	if _, err := svc.ResolveInvite(ctx, expiredToken); !errors.Is(err, domain.ErrInviteExpired) {
		t.Errorf("ResolveInvite(ivy's expired token) error = %v, want ErrInviteExpired", err)
	}

	// --- Disabled account: a still-pending invite behind a deactivated account reads as not found —
	// no oracle leak, same "disabled reads as unknown" convention D-AccountStatusEnforcement's own
	// ResolveBySubject uses — and reuses M11.1's existing Deactivate as the invite's revocation path.
	janeInvite, janeToken, err := svc.CreateInvite(ctx, "jane-m11-6@example.test", "M11.6 Jane Test", adminID)
	if err != nil {
		t.Fatalf("CreateInvite(jane): %v", err)
	}
	trackInvitePerson(janeInvite)
	if _, _, err := svc.Deactivate(ctx, janeInvite.PersonID); err != nil {
		t.Fatalf("Deactivate(jane): %v", err)
	}
	if _, err := svc.ResolveInvite(ctx, janeToken); !errors.Is(err, domain.ErrAccountDisabled) {
		t.Errorf("ResolveInvite(jane's token, account disabled) error = %v, want ErrAccountDisabled", err)
	}
}
