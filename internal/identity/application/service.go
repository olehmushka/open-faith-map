// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the identity module's orchestration layer — resolving a verified
// (issuer, subject) to a PDP subject, and D-JIT's link-on-match (never person-creating) provisioning.
package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// InviteTTL is how long a generated invite link stays valid (M11.6) — not configurable: no existing
// precedent in this codebase for exposing this class of tuning knob as an env var (same reasoning
// adapters.sessionTouchThrottle's own doc comment gives).
const InviteTTL = 7 * 24 * time.Hour

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
	InsertPersonAccountInvite(ctx context.Context, displayName, email, tokenHash, invitedBy string, expiresAt time.Time) (domain.Invite, error)
	GetInviteByTokenHash(ctx context.Context, tokenHash string) (domain.Invite, error)
	MarkInviteAcceptedByAccount(ctx context.Context, accountID string) error
	InsertApiKey(ctx context.Context, personID, label, tokenHash string, permissionCodes []string) (domain.APIKey, error)
	GetApiKeyByTokenHash(ctx context.Context, tokenHash string) (domain.APIKey, error)
	TouchApiKeyLastUsed(ctx context.Context, apiKeyID string) error
	ListApiKeysByPerson(ctx context.Context, personID string) ([]domain.APIKey, error)
	ListApiKeysByPersonIncludingRevoked(ctx context.Context, personID string) ([]domain.APIKey, error)
	RevokeApiKey(ctx context.Context, apiKeyID, personID, revokedBy string) (domain.APIKey, error)
	ResolveByAPIKey(ctx context.Context, tokenHash string) (domain.Resolution, []string, error)
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
	// M11.6: the exact moment JIT actually links the account is when a pre-provisioned invite (if
	// any) counts as accepted — not when the invitee merely views the accept-invite landing page.
	// Idempotent no-op for every non-invite sign-in (the overwhelming majority).
	if err := s.store.MarkInviteAcceptedByAccount(ctx, account.ID); err != nil {
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

// ---------------------------------------------------------------- invites (M11.6, D-InviteLinkMVP)

// InviteInfo is what a valid, still-pending invite reveals to its own not-yet-authenticated
// invitee — deliberately just enough for the accept-invite landing page's welcome message, nothing
// that would help an outsider probe for valid tokens (the invite itself, before this call succeeds,
// already proves possession).
type InviteInfo struct {
	DisplayName string
	Email       string
}

// CreateInvite pre-provisions a Person+Account for email/displayName (the same InsertPerson ->
// InsertAccount sequence LinkOnMatch's own fallback path already uses, just invoked proactively
// instead of lazily on first sign-in) and generates a one-time invite link token. Fails
// ErrAccountAlreadyExists up front if an active account for this email already exists, rather than
// letting InsertAccount hit identity_accounts_email_active_idx's unique-violation. Returns the raw
// token exactly once — only its hash is ever persisted (identity_invites.token_hash).
func (s *Service) CreateInvite(ctx context.Context, email, displayName, invitedByPersonID string) (domain.Invite, string, error) {
	if _, err := s.store.GetActiveAccountByEmail(ctx, email); err == nil {
		return domain.Invite{}, "", domain.ErrAccountAlreadyExists
	} else if !errors.Is(err, domain.ErrAccountNotFound) {
		return domain.Invite{}, "", err
	}

	rawToken, tokenHash, err := newInviteToken()
	if err != nil {
		return domain.Invite{}, "", err
	}
	// One transaction (Store.InsertPersonAccountInvite) for all three inserts — a failure partway
	// through must not leave an orphaned Person+Account with no Invite, which would permanently
	// block any future invite to this email (the GetActiveAccountByEmail check above would keep
	// seeing it as taken).
	invite, err := s.store.InsertPersonAccountInvite(ctx, displayName, email, tokenHash, invitedByPersonID, time.Now().Add(InviteTTL))
	if err != nil {
		return domain.Invite{}, "", err
	}
	return invite, rawToken, nil
}

// ResolveInvite validates rawToken and returns the invitee's display info, or one of
// ErrInviteNotFound / ErrInviteAlreadyAccepted / ErrInviteExpired / ErrAccountDisabled. Deliberately
// read-only: acceptance itself only ever happens via LinkOnMatch's own hook, never from viewing this
// page.
func (s *Service) ResolveInvite(ctx context.Context, rawToken string) (InviteInfo, error) {
	invite, err := s.store.GetInviteByTokenHash(ctx, hashInviteToken(rawToken))
	if err != nil {
		return InviteInfo{}, err
	}
	if invite.Status == domain.InviteStatusAccepted {
		return InviteInfo{}, domain.ErrInviteAlreadyAccepted
	}
	if time.Now().After(invite.ExpiresAt) {
		return InviteInfo{}, domain.ErrInviteExpired
	}
	person, err := s.store.GetPerson(ctx, invite.PersonID)
	if err != nil {
		return InviteInfo{}, err
	}
	account, err := s.store.GetActiveAccountByPerson(ctx, invite.PersonID)
	if err != nil {
		return InviteInfo{}, err
	}
	// Reusing the account's own status, not a separate "revoked" invite status (see
	// migrations/0018_core_invites.sql): deactivating the pre-provisioned account via M11.1's
	// existing action naturally invalidates a bad invite too.
	if account.Status == domain.AccountStatusDisabled {
		return InviteInfo{}, domain.ErrAccountDisabled
	}
	return InviteInfo{DisplayName: person.DisplayName, Email: invite.Email}, nil
}

// newInviteToken generates a random 32-byte token (base64url-encoded for the link) and its SHA-256
// hash (the only form ever persisted) — deliberately not a signed HMAC/JWT like
// internal/platform/devtoken.go's dev-only pattern: that would need a new production secret and
// still couldn't be truly one-time without a DB row anyway, so a random-token-plus-stored-hash (the
// same defensive shape password-reset tokens use elsewhere) is simpler and no less secure.
func newInviteToken() (rawToken, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = base64.RawURLEncoding.EncodeToString(buf)
	return rawToken, hashInviteToken(rawToken), nil
}

func hashInviteToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// -------------------------------------------------------------------- API keys (M11.9)

// CreateApiKey mints a new API key for personID, scoped to permissionCodes (the owner's chosen
// allowlist — validated against the closed catalog and rejected if it contains any instance-scope
// code, since RequireInstanceAdmin hard-denies every API-key-authenticated subject regardless of
// allowlist contents, so an instance-scope code could never actually be exercised through a key).
// Returns the raw token exactly once — only its hash is ever persisted (identity_api_keys.token_hash).
func (s *Service) CreateApiKey(ctx context.Context, personID, label string, permissionCodes []string) (domain.APIKey, string, error) {
	for _, code := range permissionCodes {
		if !authzdomain.IsKnownPermission(code) {
			return domain.APIKey{}, "", domain.ErrUnknownPermissionCode
		}
		if authzdomain.IsInstanceScope(code) {
			return domain.APIKey{}, "", domain.ErrUnknownPermissionCode
		}
	}
	rawToken, tokenHash, err := newAPIKeyToken()
	if err != nil {
		return domain.APIKey{}, "", err
	}
	key, err := s.store.InsertApiKey(ctx, personID, label, tokenHash, permissionCodes)
	if err != nil {
		return domain.APIKey{}, "", err
	}
	return key, rawToken, nil
}

// ResolveByAPIKey hashes rawToken and resolves it to the owning person's PDP subject plus the key's
// stored permission-code allowlist — internal/identity/middleware's ResolveByAPIKey hook. Best-effort
// bumps the key's last-used timestamp on success; a touch failure never fails the request (matching
// TouchSession's own fire-and-forget-adjacent posture, just without even surfacing the error).
func (s *Service) ResolveByAPIKey(ctx context.Context, rawToken string) (domain.Resolution, []string, error) {
	res, permissionCodes, err := s.store.ResolveByAPIKey(ctx, hashAPIKeyToken(rawToken))
	if err != nil {
		return domain.Resolution{}, nil, err
	}
	return res, permissionCodes, nil
}

// ListMyApiKeys returns personID's own active keys (self-scoped, M11.9).
func (s *Service) ListMyApiKeys(ctx context.Context, personID string) ([]domain.APIKey, error) {
	return s.store.ListApiKeysByPerson(ctx, personID)
}

// RevokeMyApiKey revokes one of the caller's own keys (self-scoped, M11.9) — RevokeApiKey's own
// person_id-scoped WHERE clause means a foreign apiKeyID resolves as ErrAPIKeyNotFound, not a
// different error that would leak whether the id exists at all.
func (s *Service) RevokeMyApiKey(ctx context.Context, personID, apiKeyID string) (domain.APIKey, error) {
	return s.store.RevokeApiKey(ctx, apiKeyID, personID, personID)
}

// ListApiKeysByPerson is the admin-oversight read: personID's full key history, active and revoked
// (M11.9).
func (s *Service) ListApiKeysByPerson(ctx context.Context, personID string) ([]domain.APIKey, error) {
	return s.store.ListApiKeysByPersonIncludingRevoked(ctx, personID)
}

// RevokeApiKey revokes personID's apiKeyID on an admin's behalf (M11.9, admin oversight — incident
// response without waiting on the owner). revokedByPersonID is the acting admin, recorded distinctly
// from personID (the owner) in the returned row's RevokedBy.
func (s *Service) RevokeApiKey(ctx context.Context, personID, apiKeyID, revokedByPersonID string) (domain.APIKey, error) {
	return s.store.RevokeApiKey(ctx, apiKeyID, personID, revokedByPersonID)
}

// newAPIKeyToken generates a random 32-byte token, base64url-encoded and prefixed with
// domain.APIKeyTokenPrefix so internal/identity/middleware can detect an API-key-shaped bearer without
// a DB round-trip or a doomed JWT-parse attempt, and its SHA-256 hash (the only form ever persisted) —
// the same random-token-plus-stored-hash shape newInviteToken already uses.
func newAPIKeyToken() (rawToken, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = domain.APIKeyTokenPrefix + base64.RawURLEncoding.EncodeToString(buf)
	return rawToken, hashAPIKeyToken(rawToken), nil
}

func hashAPIKeyToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
