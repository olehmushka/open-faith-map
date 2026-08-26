// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"time"
)

var (
	ErrPersonNotFound        = errors.New("person not found")
	ErrAccountNotFound       = errors.New("account not found")
	ErrAccountDisabled       = errors.New("account is disabled")
	ErrAccountAlreadyExists  = errors.New("an active account already exists for this email")
	ErrIdentityNotFound      = errors.New("external identity not found")
	ErrIdentityConflict      = errors.New("external identity already linked to a different person")
	ErrInvalidExternalID     = errors.New("external identity requires both issuer and subject")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionRevoked        = errors.New("session is revoked")
	ErrInviteNotFound        = errors.New("invite not found")
	ErrInviteExpired         = errors.New("invite has expired")
	ErrInviteAlreadyAccepted = errors.New("invite has already been accepted")
	ErrCannotMergeSelf       = errors.New("cannot merge a person with itself")
	ErrAPIKeyNotFound        = errors.New("api key not found")
)

// Account status values — must match identity_accounts' CHECK constraint literals
// (migrations/0008_core_identity.sql).
const (
	AccountStatusActive   = "active"
	AccountStatusDisabled = "disabled"
)

// Person status values — must match identity_persons' CHECK constraint literals
// (migrations/0008_core_identity.sql). The column has existed since M10.1 but nothing wrote to it
// until M11.8's MergePersons, which sets a merged-away duplicate to PersonStatusDeactivated
// alongside soft-deleting it.
const (
	PersonStatusActive      = "active"
	PersonStatusDeactivated = "deactivated"
)

// APIKeyTokenPrefix marks a raw bearer token as API-key-shaped (M11.9) rather than a JWT — a JWT is
// always three dot-separated base64url segments, which never starts with this literal. Shared between
// internal/identity/application (which generates tokens with this prefix) and
// internal/identity/middleware (which branches on it before ever attempting JWT parsing), so both
// import this one package instead of duplicating the literal.
const APIKeyTokenPrefix = "ofm_"

// Invite status values — must match identity_invites' CHECK constraint literals
// (migrations/0018_core_invites.sql). No "expired"/"revoked" value: see that migration's own
// comment for why both are checked live instead of stored.
const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
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
	// LastActiveAt is M11.4's activity signal: the most recent identity_sessions.last_seen_at across
	// all of this person's account's sessions, revoked or not (same "revoked or not" read convention
	// GetSession already uses). Nil if the person has no account or that account has no sessions.
	// Only SearchPersons populates this today — GetPerson/GetPersons (non-admin CoreService reads)
	// leave it nil, so it stays absent for every caller besides the super-admin people list.
	LastActiveAt *time.Time
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

// Session is identity_sessions (M11.3, D-SessionTracking) — one row per NextAuth sign-in on the
// admin app. Mutable, unlike ExternalIdentity: LastSeenAt is bumped on the request path (throttled,
// see adapters.sessionTouchThrottle) and RevokedAt is set by RevokeSession.
type Session struct {
	ID          string
	AccountID   string
	Issuer      string
	DeviceLabel string // best-effort User-Agent captured at sign-in; may be empty
	CreatedAt   time.Time
	LastSeenAt  time.Time
	RevokedAt   *time.Time
}

// Invite is identity_invites (M11.6, D-InviteLinkMVP) — a shareable one-time link for a
// pre-provisioned Person+Account pair, generated by an admin and consumed by M10.2's existing JIT
// link-on-match logic on the invitee's first Google sign-in. TokenHash is the only form of the
// token ever persisted (see migrations/0018_core_invites.sql); the raw token is returned once, by
// CreateInvite, and never stored.
type Invite struct {
	ID         string
	PersonID   string
	AccountID  string
	Email      string
	Status     string // InviteStatusPending | InviteStatusAccepted
	InvitedBy  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AcceptedAt *time.Time
}

// APIKey is identity_api_keys (M11.9) — a second credential for an existing person, not a new
// principal type (authz.Subject's own doc comment, D-DirectTokenVerification, stays true).
// PermissionCodes is a fixed allowlist the owner chose at creation time; the effective permission set
// for a request authenticated via this key is that allowlist intersected with PersonID's LIVE authz
// grants at request time (internal/authz.Service.Require), never stored as a materialized set here.
// TokenHash is the only form of the raw secret ever persisted — see migrations/0019_core_api_keys.sql.
type APIKey struct {
	ID              string
	PersonID        string
	Label           string
	PermissionCodes []string
	CreatedAt       time.Time
	LastUsedAt      *time.Time
	RevokedAt       *time.Time
	RevokedBy       *string
}
