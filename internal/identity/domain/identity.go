// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"time"
)

var (
	ErrPersonNotFound    = errors.New("person not found")
	ErrAccountNotFound   = errors.New("account not found")
	ErrAccountDisabled   = errors.New("account is disabled")
	ErrIdentityNotFound  = errors.New("external identity not found")
	ErrIdentityConflict  = errors.New("external identity already linked to a different person")
	ErrInvalidExternalID = errors.New("external identity requires both issuer and subject")
)

// Account status values — must match identity_accounts' CHECK constraint literals
// (migrations/0008_core_identity.sql).
const (
	AccountStatusActive   = "active"
	AccountStatusDisabled = "disabled"
)

// Person is identity_persons — the durable PDP subject. Trimmed relative to go-oikumenea's own
// person module (D-CorePortScope): no birthdate/sex/country-of-birth, none of it used anywhere in
// this repo.
type Person struct {
	ID          string
	Code        string // optional stable external id, unique among active
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Account is identity_accounts — an optional login attachment to exactly one person. Tokens/
// passwords are never stored; auth is delegated to Google (D-DirectTokenVerification).
type Account struct {
	ID        string
	PersonID  string
	Email     string // citext; unique among active when set
	Status    string // AccountStatusActive | AccountStatusDisabled
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ExternalIdentity is identity_external_identities — a verified (issuer, subject) login point
// federating to one account. Immutable once created; an unlink is a hard delete.
type ExternalIdentity struct {
	ID        string
	AccountID string
	Issuer    string
	Subject   string
	CreatedAt time.Time
}

func (e ExternalIdentity) Validate() error {
	if e.Issuer == "" || e.Subject == "" {
		return ErrInvalidExternalID
	}
	return nil
}

// Resolution is what a successful identity lookup or JIT link yields: enough to build an
// authz.Subject without the caller needing the full Person/Account rows.
type Resolution struct {
	PersonID  string
	AccountID string
	Email     string
}
