// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the identity module's Postgres store. Hand-written pgx (matches
// internal/registration/adapters' convention for a small, focused query surface).
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so a Store can be bound either to the pool
// for normal request-scoped calls or to a single pgx.Tx for the boot-time admin seed's atomic
// person+account+identity+instance-admin write (internal/identity/bootstrap).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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

func (s *Store) GetActiveAccountByPerson(ctx context.Context, personID string) (domain.Account, error) {
	return s.scanAccount(s.pool.QueryRow(ctx, `
		SELECT id, person_id, COALESCE(email::text, ''), created_at, updated_at
		FROM openfaithmap.identity_accounts
		WHERE person_id = $1 AND deleted_at IS NULL`, personID))
}

func (s *Store) GetActiveAccountByEmail(ctx context.Context, email string) (domain.Account, error) {
	// email is citext: this comparison is case-insensitive, and the partial unique active-index
	// (identity_accounts_email_active_idx) makes "the single account" true by construction — no
	// ambiguous-match case to resolve in Go.
	return s.scanAccount(s.pool.QueryRow(ctx, `
		SELECT id, person_id, COALESCE(email::text, ''), created_at, updated_at
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
		RETURNING id, person_id, COALESCE(email::text, ''), created_at, updated_at`, personID, emailArg))
}

func (s *Store) scanAccount(row pgx.Row) (domain.Account, error) {
	var a domain.Account
	if err := row.Scan(&a.ID, &a.PersonID, &a.Email, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return a, nil
}

// ResolveBySubject maps a verified (issuer, subject) to the account+person it federates to.
func (s *Store) ResolveBySubject(ctx context.Context, issuer, subject string) (domain.Resolution, error) {
	var out domain.Resolution
	err := s.pool.QueryRow(ctx, `
		SELECT a.person_id, a.id, COALESCE(a.email::text, '')
		FROM openfaithmap.identity_external_identities x
		JOIN openfaithmap.identity_accounts a ON a.id = x.account_id AND a.deleted_at IS NULL
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
