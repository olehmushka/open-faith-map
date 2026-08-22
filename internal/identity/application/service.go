// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the identity module's orchestration layer — resolving a verified
// (issuer, subject) to a PDP subject, and D-JIT's link-on-match (never person-creating) provisioning.
package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// Store is the subset of internal/identity/adapters.Store this service needs.
type Store interface {
	GetActiveAccountByPerson(ctx context.Context, personID string) (domain.Account, error)
	GetActiveAccountByEmail(ctx context.Context, email string) (domain.Account, error)
	InsertAccount(ctx context.Context, personID, email string) (domain.Account, error)
	SetAccountStatus(ctx context.Context, accountID, status string) (domain.Account, error)
	ResolveBySubject(ctx context.Context, issuer, subject string) (domain.Resolution, error)
	InsertIdentity(ctx context.Context, accountID, issuer, subject string) (domain.ExternalIdentity, error)
	GetActivePersonByCode(ctx context.Context, code string) (domain.Person, error)
	GetPerson(ctx context.Context, id string) (domain.Person, error)
	GetPersons(ctx context.Context, ids []string) ([]domain.Person, error)
	SearchPersons(ctx context.Context, query string, limit int) ([]domain.Person, error)
	UpdateDisplayName(ctx context.Context, personID, displayName string) (domain.Person, error)
	InsertSession(ctx context.Context, accountID, issuer, deviceLabel string) (domain.Session, error)
	GetSession(ctx context.Context, sessionID string) (domain.Session, error)
	ListActiveSessionsByAccount(ctx context.Context, accountID string) ([]domain.Session, error)
	TouchSession(ctx context.Context, sessionID string) (domain.Session, error)
	RevokeSession(ctx context.Context, sessionID string) (domain.Session, error)
	LastActiveAtByAccount(ctx context.Context, accountID string) (*time.Time, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Resolve maps a verified (issuer, subject) to the account+person it federates to, or
// domain.ErrIdentityNotFound.
func (s *Service) Resolve(ctx context.Context, issuer, subject string) (domain.Resolution, error) {
	return s.store.ResolveBySubject(ctx, issuer, subject)
}

// LinkOnMatch is called ONLY after the caller has already matched a token claim to an EXISTING
// person (D-JIT). It NEVER creates a person — only reuses-or-creates that person's account and links
// the verified identity to it.
func (s *Service) LinkOnMatch(ctx context.Context, personID, issuer, subject, email string) (domain.Resolution, error) {
	id := domain.ExternalIdentity{Issuer: issuer, Subject: subject}
	if err := id.Validate(); err != nil {
		return domain.Resolution{}, err
	}

	account, err := s.store.GetActiveAccountByPerson(ctx, personID)
	if errors.Is(err, domain.ErrAccountNotFound) {
		account, err = s.store.InsertAccount(ctx, personID, email)
	}
	if err != nil {
		return domain.Resolution{}, err
	}
	// A disabled account must never be re-linked or reused by JIT (D-AccountStatusEnforcement) — a
	// freshly inserted account is always active, so this only ever rejects the reuse path.
	if account.Status == domain.AccountStatusDisabled {
		return domain.Resolution{}, domain.ErrAccountDisabled
	}

	// Pre-check rather than insert-then-recover: a unique-violation on
	// identity_external_identities_issuer_subject_idx would poison the surrounding transaction.
	existing, err := s.store.ResolveBySubject(ctx, issuer, subject)
	switch {
	case err == nil:
		if existing.PersonID != personID {
			return domain.Resolution{}, domain.ErrIdentityConflict
		}
		return existing, nil // already linked to this person — idempotent
	case !errors.Is(err, domain.ErrIdentityNotFound):
		return domain.Resolution{}, err
	}

	if _, err := s.store.InsertIdentity(ctx, account.ID, issuer, subject); err != nil {
		return domain.Resolution{}, err
	}
	return domain.Resolution{PersonID: personID, AccountID: account.ID, Email: account.Email}, nil
}

// PersonIDByAccountEmail backs D-JIT's attribute arm: the person behind the single active account
// carrying this IdP-asserted email, and whether one was found.
func (s *Service) PersonIDByAccountEmail(ctx context.Context, email string) (string, bool, error) {
	if strings.TrimSpace(email) == "" {
		return "", false, nil
	}
	account, err := s.store.GetActiveAccountByEmail(ctx, email)
	if errors.Is(err, domain.ErrAccountNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return account.PersonID, true, nil
}

// PersonIDByCode backs D-JIT's default arm: a token claim value matched directly against
// identity_persons.code.
func (s *Service) PersonIDByCode(ctx context.Context, code string) (string, bool, error) {
	if strings.TrimSpace(code) == "" {
		return "", false, nil
	}
	p, err := s.store.GetActivePersonByCode(ctx, code)
	if errors.Is(err, domain.ErrPersonNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return p.ID, true, nil
}

// GetPerson reads a single person by id — M10.7's core.conjure.yml CoreService.GetPerson.
func (s *Service) GetPerson(ctx context.Context, id string) (domain.Person, error) {
	return s.store.GetPerson(ctx, id)
}

// GetPersons is the batched read backing my-congregation's member roster (M10.7) — one round trip
// instead of a per-member GetPerson loop.
func (s *Service) GetPersons(ctx context.Context, ids []string) ([]domain.Person, error) {
	return s.store.GetPersons(ctx, ids)
}

// SearchPersons backs the M10.7 super-admin people screen's search box.
func (s *Service) SearchPersons(ctx context.Context, query string, limit int) ([]domain.Person, error) {
	return s.store.SearchPersons(ctx, query, limit)
}

// UpdateMyProfile sets personID's display name — M11.5's self-service profile page. A plain
// delegate to the store, no subject resolution here: the caller (internal/core/application) must
// derive personID from the request's own resolved subject, never a client-supplied argument, same
// division of responsibility RevokeMySession already uses for accountID.
func (s *Service) UpdateMyProfile(ctx context.Context, personID, displayName string) (domain.Person, error) {
	return s.store.UpdateDisplayName(ctx, personID, displayName)
}

// AccountStatus reports personID's account status plus its M11.4 last-active signal, or found=false
// if the person has no account yet (not an error) — backs the M11.1/M11.4 super-admin person detail
// page's account-status display.
func (s *Service) AccountStatus(ctx context.Context, personID string) (status string, lastActiveAt *time.Time, found bool, err error) {
	account, err := s.store.GetActiveAccountByPerson(ctx, personID)
	if errors.Is(err, domain.ErrAccountNotFound) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}
	lastActiveAt, err = s.store.LastActiveAtByAccount(ctx, account.ID)
	if err != nil {
		return "", nil, false, err
	}
	return account.Status, lastActiveAt, true, nil
}

// Deactivate disables personID's account (D-AccountStatusEnforcement), rejecting further
// authentication for it — see ResolveBySubject and LinkOnMatch. Idempotent: deactivating an
// already-disabled account just re-asserts the state. Returns both the pre- and post-update account
// (M11.2: the super-admin caller uses before/after to log a real audit-trail snapshot with no second
// read — setStatus already holds the pre-update row before mutating it).
func (s *Service) Deactivate(ctx context.Context, personID string) (before, after domain.Account, err error) {
	return s.setStatus(ctx, personID, domain.AccountStatusDisabled)
}

// Reactivate re-enables personID's account. Idempotent, mirroring Deactivate.
func (s *Service) Reactivate(ctx context.Context, personID string) (before, after domain.Account, err error) {
	return s.setStatus(ctx, personID, domain.AccountStatusActive)
}

func (s *Service) setStatus(ctx context.Context, personID, status string) (before, after domain.Account, err error) {
	before, err = s.store.GetActiveAccountByPerson(ctx, personID)
	if err != nil {
		return domain.Account{}, domain.Account{}, err
	}
	after, err = s.store.SetAccountStatus(ctx, before.ID, status)
	if err != nil {
		return domain.Account{}, domain.Account{}, err
	}
	return before, after, nil
}

// RegisterSession creates a new session row for accountID (M11.3, D-SessionTracking) — called once
// right after a NextAuth sign-in, before the resulting session id is usable as X-Session-Id on any
// other request. That bootstrapping order is why this one endpoint is the sole entry in
// internal/identity/middleware's sessionExemptRoutes: it must be reachable without an
// already-existing session to present.
func (s *Service) RegisterSession(ctx context.Context, accountID, issuer, deviceLabel string) (domain.Session, error) {
	return s.store.InsertSession(ctx, accountID, issuer, deviceLabel)
}

// ListSessions returns personID's active sessions (admin-scoped, M11.3).
func (s *Service) ListSessions(ctx context.Context, personID string) ([]domain.Session, error) {
	account, err := s.store.GetActiveAccountByPerson(ctx, personID)
	if err != nil {
		return nil, err
	}
	return s.store.ListActiveSessionsByAccount(ctx, account.ID)
}

// RevokeSession revokes sessionID on personID's behalf (admin-scoped, M11.3). Rejects a sessionID
// belonging to a different person's account as ErrSessionNotFound — an admin-supplied path segment
// must not let a cross-account guess revoke someone else's session.
func (s *Service) RevokeSession(ctx context.Context, personID, sessionID string) (domain.Session, error) {
	account, err := s.store.GetActiveAccountByPerson(ctx, personID)
	if err != nil {
		return domain.Session{}, err
	}
	sess, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return domain.Session{}, err
	}
	if sess.AccountID != account.ID {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return s.store.RevokeSession(ctx, sessionID)
}

// ListMySessions returns the caller's own active sessions (self-scoped, M11.3).
func (s *Service) ListMySessions(ctx context.Context, accountID string) ([]domain.Session, error) {
	return s.store.ListActiveSessionsByAccount(ctx, accountID)
}

// RevokeMySession revokes one of the caller's own sessions (self-scoped, M11.3) — same
// account-ownership check as RevokeSession, scoped to the caller's own accountID instead of an
// admin-supplied personId.
func (s *Service) RevokeMySession(ctx context.Context, accountID, sessionID string) (domain.Session, error) {
	sess, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return domain.Session{}, err
	}
	if sess.AccountID != accountID {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return s.store.RevokeSession(ctx, sessionID)
}

// Touch implements internal/identity/middleware.SessionChecker — the per-request check backing
// D-SessionTracking. Returns the session's accountID so Handle can cross-check it against the
// bearer-resolved account: a session id that resolves to a DIFFERENT account than the verified
// bearer must still 401, not silently pass.
func (s *Service) Touch(ctx context.Context, sessionID string) (string, error) {
	sess, err := s.store.TouchSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return sess.AccountID, nil
}
