// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the identity module's Postgres store. Hand-written pgx (matches
// internal/registration/adapters' convention for a small, focused query surface).
package adapters

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// sessionTouchThrottle is TouchSession's minimum interval between last_seen_at writes — M11.3's
// build-time decision to keep the per-request session check off the write hot path (see
// docs/milestones.md's M11.3 row and D-SessionTracking's Consequences). Not configurable: no
// existing precedent in this codebase for exposing this class of tuning knob as an env var.
const sessionTouchThrottle = 60 * time.Second

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so a Store can be bound either to the pool
// for normal request-scoped calls or to a single pgx.Tx for the boot-time admin seed's atomic
// person+account+identity+instance-admin write (internal/identity/bootstrap).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Store struct {
	pool Querier
}

func NewStore(pool Querier) *Store {
	return &Store{pool: pool}
}

func (s *Store) GetActivePersonByCode(ctx context.Context, code string) (domain.Person, error) {
	return s.scanPerson(s.pool.QueryRow(ctx, `
		SELECT id, code, display_name, created_at, updated_at
		FROM openfaithmap.identity_persons
		WHERE code = $1 AND deleted_at IS NULL`, code))
}

func (s *Store) InsertPerson(ctx context.Context, p domain.Person) (domain.Person, error) {
	var code any
	if p.Code != "" {
		code = p.Code
	}
	return s.scanPerson(s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (code, display_name)
		VALUES ($1, $2)
		RETURNING id, code, display_name, created_at, updated_at`, code, p.DisplayName))
}

func (s *Store) scanPerson(row pgx.Row) (domain.Person, error) {
	var p domain.Person
	var code *string
	if err := row.Scan(&p.ID, &code, &p.DisplayName, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Person{}, domain.ErrPersonNotFound
		}
		return domain.Person{}, err
	}
	if code != nil {
		p.Code = *code
	}
	return p, nil
}

// GetPerson reads a single person by id — M10.7's core.conjure.yml GetPerson/detail-screen read.
func (s *Store) GetPerson(ctx context.Context, id string) (domain.Person, error) {
	return s.scanPerson(s.pool.QueryRow(ctx, `
		SELECT id, code, display_name, created_at, updated_at
		FROM openfaithmap.identity_persons
		WHERE id = $1 AND deleted_at IS NULL`, id))
}

// GetPersons is the batched form of GetPerson — replaces admin's my-congregation page N+1
// per-member GetPerson loop with one round trip (M10.7). Returns whatever subset of ids exists;
// callers must not assume the result is ordered like or the same length as ids.
func (s *Store) GetPersons(ctx context.Context, ids []string) ([]domain.Person, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, code, display_name, created_at, updated_at
		FROM openfaithmap.identity_persons
		WHERE id = ANY($1) AND deleted_at IS NULL`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Person
	for rows.Next() {
		p, err := s.scanPerson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SearchPersons is the M10.7 super-admin people screen's search — case-insensitive substring match
// on display name or code, capped at limit (default/max 50).
// SearchPersons backs the super-admin people list, so unlike GetPerson/GetPersons (shared with
// non-admin CoreService reads) it also computes each result's M11.4 last-active signal in the same
// round trip — a revoked-inclusive MAX(last_seen_at) per account, not filtered by
// ListActiveSessionsByAccount's own revoked_at IS NULL predicate. Scanned separately from
// scanPerson (an extra column) rather than changing that shared helper, so GetPerson/GetPersons stay
// untouched and don't pay for the join.
func (s *Store) SearchPersons(ctx context.Context, query string, limit int) ([]domain.Person, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.code, p.display_name, p.created_at, p.updated_at, las.last_active
		FROM openfaithmap.identity_persons p
		LEFT JOIN openfaithmap.identity_accounts a ON a.person_id = p.id AND a.deleted_at IS NULL
		LEFT JOIN (
			SELECT account_id, MAX(last_seen_at) AS last_active
			FROM openfaithmap.identity_sessions
			GROUP BY account_id
		) las ON las.account_id = a.id
		WHERE p.deleted_at IS NULL
		  AND ($1 = '' OR p.display_name ILIKE '%' || $1 || '%' OR p.code ILIKE '%' || $1 || '%')
		ORDER BY p.display_name
		LIMIT $2`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Person
	for rows.Next() {
		var p domain.Person
		var code *string
		if err := rows.Scan(&p.ID, &code, &p.DisplayName, &p.CreatedAt, &p.UpdatedAt, &p.LastActiveAt); err != nil {
			return nil, err
		}
		if code != nil {
			p.Code = *code
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetActiveAccountByPerson returns personID's account regardless of its status — "Active" here means
// only "not soft-deleted" (deleted_at IS NULL), same convention as GetActiveAccountByEmail below.
// Callers that care whether the account is usable must check the returned Account.Status themselves
// (see application.Service.LinkOnMatch for why filtering status out in SQL here would be wrong).
func (s *Store) GetActiveAccountByPerson(ctx context.Context, personID string) (domain.Account, error) {
	return s.scanAccount(s.pool.QueryRow(ctx, `
		SELECT id, person_id, COALESCE(email::text, ''), status, created_at, updated_at
		FROM openfaithmap.identity_accounts
		WHERE person_id = $1 AND deleted_at IS NULL`, personID))
}

func (s *Store) GetActiveAccountByEmail(ctx context.Context, email string) (domain.Account, error) {
	// email is citext: this comparison is case-insensitive, and the partial unique active-index
	// (identity_accounts_email_active_idx) makes "the single account" true by construction — no
	// ambiguous-match case to resolve in Go.
	return s.scanAccount(s.pool.QueryRow(ctx, `
		SELECT id, person_id, COALESCE(email::text, ''), status, created_at, updated_at
		FROM openfaithmap.identity_accounts
		WHERE email = $1 AND deleted_at IS NULL`, email))
}

func (s *Store) InsertAccount(ctx context.Context, personID, email string) (domain.Account, error) {
	var emailArg any
	if email != "" {
		emailArg = email
	}
	return s.scanAccount(s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_accounts (person_id, email)
		VALUES ($1, $2)
		RETURNING id, person_id, COALESCE(email::text, ''), status, created_at, updated_at`, personID, emailArg))
}

// SetAccountStatus sets accountID's status (domain.AccountStatusActive/AccountStatusDisabled) and
// returns the updated row — backs application.Service.Deactivate/Reactivate.
func (s *Store) SetAccountStatus(ctx context.Context, accountID, status string) (domain.Account, error) {
	return s.scanAccount(s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.identity_accounts
		SET status = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, person_id, COALESCE(email::text, ''), status, created_at, updated_at`, accountID, status))
}

func (s *Store) scanAccount(row pgx.Row) (domain.Account, error) {
	var a domain.Account
	if err := row.Scan(&a.ID, &a.PersonID, &a.Email, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return a, nil
}

// ResolveBySubject maps a verified (issuer, subject) to the account+person it federates to. A
// disabled account (D-AccountStatusEnforcement) resolves as ErrIdentityNotFound, same as an unknown
// one — no oracle leak, and it reuses the authenticator's existing uniform-401 path.
func (s *Store) ResolveBySubject(ctx context.Context, issuer, subject string) (domain.Resolution, error) {
	var out domain.Resolution
	err := s.pool.QueryRow(ctx, `
		SELECT a.person_id, a.id, COALESCE(a.email::text, '')
		FROM openfaithmap.identity_external_identities x
		JOIN openfaithmap.identity_accounts a ON a.id = x.account_id AND a.deleted_at IS NULL AND a.status = 'active'
		WHERE x.issuer = $1 AND x.subject = $2`, issuer, subject,
	).Scan(&out.PersonID, &out.AccountID, &out.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Resolution{}, domain.ErrIdentityNotFound
	}
	return out, err
}

// InsertIdentity links (issuer, subject) to accountID. Callers must pre-check via ResolveBySubject —
// a unique-violation would poison the surrounding transaction, so this assumes the caller has already
// established no conflicting row exists.
func (s *Store) InsertIdentity(ctx context.Context, accountID, issuer, subject string) (domain.ExternalIdentity, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_external_identities (account_id, issuer, subject)
		VALUES ($1, $2, $3)
		RETURNING id`, accountID, issuer, subject,
	).Scan(&id)
	if err != nil {
		return domain.ExternalIdentity{}, err
	}
	return domain.ExternalIdentity{ID: id, AccountID: accountID, Issuer: issuer, Subject: subject}, nil
}

// InsertSession creates a new identity_sessions row (M11.3) — one per NextAuth sign-in. deviceLabel
// is best-effort (User-Agent captured at sign-in) and may be empty.
func (s *Store) InsertSession(ctx context.Context, accountID, issuer, deviceLabel string) (domain.Session, error) {
	var deviceLabelArg any
	if deviceLabel != "" {
		deviceLabelArg = deviceLabel
	}
	return s.scanSession(s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_sessions (account_id, issuer, device_label)
		VALUES ($1, $2, $3)
		RETURNING id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at`,
		accountID, issuer, deviceLabelArg))
}

// GetSession reads a single session by id, revoked or not — callers that care whether it's usable
// check the returned Session.RevokedAt themselves (same convention GetActiveAccountByPerson uses
// for Account.Status).
func (s *Store) GetSession(ctx context.Context, sessionID string) (domain.Session, error) {
	return s.scanSession(s.pool.QueryRow(ctx, `
		SELECT id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at
		FROM openfaithmap.identity_sessions
		WHERE id = $1`, sessionID))
}

// ListActiveSessionsByAccount returns accountID's not-yet-revoked sessions, most recently active
// first — backs application.Service.ListSessions/ListMySessions.
func (s *Store) ListActiveSessionsByAccount(ctx context.Context, accountID string) ([]domain.Session, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at
		FROM openfaithmap.identity_sessions
		WHERE account_id = $1 AND revoked_at IS NULL
		ORDER BY last_seen_at DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Session
	for rows.Next() {
		sess, err := s.scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// TouchSession validates sessionID (ErrSessionNotFound / ErrSessionRevoked) and, on the happy path,
// bumps last_seen_at — but only when the existing value is more than sessionTouchThrottle stale, so
// a burst of requests from one still-fresh session costs one read, not one write, per request. This
// backs the per-request check in internal/identity/middleware.Authenticator.Handle.
func (s *Store) TouchSession(ctx context.Context, sessionID string) (domain.Session, error) {
	sess, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return domain.Session{}, err
	}
	if sess.RevokedAt != nil {
		return sess, domain.ErrSessionRevoked
	}
	if time.Since(sess.LastSeenAt) <= sessionTouchThrottle {
		return sess, nil
	}
	return s.scanSession(s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.identity_sessions
		SET last_seen_at = now()
		WHERE id = $1
		RETURNING id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at`, sessionID))
}

// LastActiveAtByAccount returns accountID's most recent session activity — MAX(last_seen_at) across
// all of its sessions, revoked or not (M11.4's revoked-inclusive decision: an admin revoking a
// session shouldn't retroactively erase the historical fact that the person was active). Nil if the
// account has never had a session. Backs application.Service.AccountStatus for the person detail page
// (SearchPersons computes the same signal itself, batched, for the people list).
func (s *Store) LastActiveAtByAccount(ctx context.Context, accountID string) (*time.Time, error) {
	var lastActive *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT MAX(last_seen_at)
		FROM openfaithmap.identity_sessions
		WHERE account_id = $1`, accountID).Scan(&lastActive)
	return lastActive, err
}

// RevokeSession sets revoked_at. Idempotent: revoking an already-revoked session is a no-op that
// still returns the current row (matching SetAccountStatus's own idempotency).
func (s *Store) RevokeSession(ctx context.Context, sessionID string) (domain.Session, error) {
	return s.scanSession(s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.identity_sessions
		SET revoked_at = COALESCE(revoked_at, now())
		WHERE id = $1
		RETURNING id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at`, sessionID))
}

func (s *Store) scanSession(row pgx.Row) (domain.Session, error) {
	var sess domain.Session
	var deviceLabel *string
	if err := row.Scan(&sess.ID, &sess.AccountID, &sess.Issuer, &deviceLabel, &sess.CreatedAt, &sess.LastSeenAt, &sess.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	if deviceLabel != nil {
		sess.DeviceLabel = *deviceLabel
	}
	return sess, nil
}
