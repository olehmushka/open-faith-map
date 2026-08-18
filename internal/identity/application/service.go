// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the identity module's orchestration layer — resolving a verified
// (issuer, subject) to a PDP subject, and D-JIT's link-on-match (never person-creating) provisioning.
package application

import (
	"context"
	"errors"
	"strings"

	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// Store is the subset of internal/identity/adapters.Store this service needs.
type Store interface {
	GetActiveAccountByPerson(ctx context.Context, personID string) (domain.Account, error)
	GetActiveAccountByEmail(ctx context.Context, email string) (domain.Account, error)
	InsertAccount(ctx context.Context, personID, email string) (domain.Account, error)
	ResolveBySubject(ctx context.Context, issuer, subject string) (domain.Resolution, error)
	InsertIdentity(ctx context.Context, accountID, issuer, subject string) (domain.ExternalIdentity, error)
	GetActivePersonByCode(ctx context.Context, code string) (domain.Person, error)
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
