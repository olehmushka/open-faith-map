// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the identity module's Postgres store. sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/identity.sql, generated code
// in identitysql/. InsertPersonAccountInvite keeps its own pool-Begin/Commit (same shape as before):
// requires a pool-bound Repository, since it composes three inserts atomically.
package adapters

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/identity/adapters/identitysql"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

// sessionTouchThrottle is TouchSession's minimum interval between last_seen_at writes — M11.3's
// build-time decision to keep the per-request session check off the write hot path.
const sessionTouchThrottle = 60 * time.Second

type Repository struct {
	conn db.DBTX
	q    *identitysql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{conn: conn, q: identitysql.New(conn)}
}

func nullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func fromNullableText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// asString handles the "interface{}" columns sqlc couldn't statically type through a
// COALESCE(email::text, '') expression over a citext column — the COALESCE guarantees a non-null
// string at runtime regardless.
func asString(v any) string {
	s, _ := v.(string)
	return s
}

// asNullableTime handles the same "interface{}" fallback for a genuinely nullable aggregate
// (MAX(last_seen_at) with no cast) — nil stays nil, a real value comes back as time.Time.
func asNullableTime(v any) *time.Time {
	if v == nil {
		return nil
	}
	if t, ok := v.(time.Time); ok {
		return &t
	}
	return nil
}

func toPerson(id string, code pgtype.Text, displayName string, createdAt, updatedAt time.Time) domain.Person {
	return domain.Person{ID: id, Code: fromNullableText(code), DisplayName: displayName, CreatedAt: createdAt, UpdatedAt: updatedAt}
}

func (r *Repository) GetActivePersonByCode(ctx context.Context, code string) (domain.Person, error) {
	row, err := r.q.GetActivePersonByCode(ctx, nullableText(code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Person{}, domain.ErrPersonNotFound
		}
		return domain.Person{}, err
	}
	return toPerson(row.ID, row.Code, row.DisplayName, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) InsertPerson(ctx context.Context, p domain.Person) (domain.Person, error) {
	row, err := r.q.InsertPerson(ctx, identitysql.InsertPersonParams{Code: nullableText(p.Code), DisplayName: p.DisplayName})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Person{}, domain.ErrPersonNotFound
		}
		return domain.Person{}, err
	}
	return toPerson(row.ID, row.Code, row.DisplayName, row.CreatedAt, row.UpdatedAt), nil
}

// GetPerson reads a single person by id — M10.7's core.conjure.yml GetPerson/detail-screen read.
func (r *Repository) GetPerson(ctx context.Context, id string) (domain.Person, error) {
	row, err := r.q.GetPerson(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Person{}, domain.ErrPersonNotFound
		}
		return domain.Person{}, err
	}
	return toPerson(row.ID, row.Code, row.DisplayName, row.CreatedAt, row.UpdatedAt), nil
}

// GetPersons is the batched form of GetPerson — replaces admin's my-congregation page N+1
// per-member GetPerson loop with one round trip (M10.7). Returns whatever subset of ids exists;
// callers must not assume the result is ordered like or the same length as ids.
func (r *Repository) GetPersons(ctx context.Context, ids []string) ([]domain.Person, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.q.GetPersons(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Person, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPerson(row.ID, row.Code, row.DisplayName, row.CreatedAt, row.UpdatedAt))
	}
	return out, nil
}

// UpdateDisplayName sets personID's display_name — M11.5's self-service profile page. Callers must
// resolve personID from the request's own subject, never a client-supplied argument.
func (r *Repository) UpdateDisplayName(ctx context.Context, personID, displayName string) (domain.Person, error) {
	row, err := r.q.UpdateDisplayName(ctx, identitysql.UpdateDisplayNameParams{ID: personID, DisplayName: displayName})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Person{}, domain.ErrPersonNotFound
		}
		return domain.Person{}, err
	}
	return toPerson(row.ID, row.Code, row.DisplayName, row.CreatedAt, row.UpdatedAt), nil
}

// SearchPersons is the M10.7 super-admin people screen's search — case-insensitive substring match
// on display name or code, capped at limit (default/max 50).
func (r *Repository) SearchPersons(ctx context.Context, query string, limit int) ([]domain.Person, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	rows, err := r.q.SearchPersons(ctx, identitysql.SearchPersonsParams{Query: query, LimitCount: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Person, 0, len(rows))
	for _, row := range rows {
		p := toPerson(row.ID, row.Code, row.DisplayName, row.CreatedAt, row.UpdatedAt)
		p.LastActiveAt = asNullableTime(row.LastActive)
		out = append(out, p)
	}
	return out, nil
}

func toAccount(id, personID string, email any, status string, createdAt, updatedAt time.Time) domain.Account {
	return domain.Account{ID: id, PersonID: personID, Email: asString(email), Status: status, CreatedAt: createdAt, UpdatedAt: updatedAt}
}

// GetActiveAccountByPerson returns personID's account regardless of its status — "Active" here means
// only "not soft-deleted" (deleted_at IS NULL). Callers that care whether the account is usable must
// check the returned Account.Status themselves.
func (r *Repository) GetActiveAccountByPerson(ctx context.Context, personID string) (domain.Account, error) {
	row, err := r.q.GetActiveAccountByPerson(ctx, personID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return toAccount(row.ID, row.PersonID, row.Email, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetActiveAccountByEmail(ctx context.Context, email string) (domain.Account, error) {
	row, err := r.q.GetActiveAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return toAccount(row.ID, row.PersonID, row.Email, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) InsertAccount(ctx context.Context, personID, email string) (domain.Account, error) {
	row, err := r.q.InsertAccount(ctx, identitysql.InsertAccountParams{PersonID: personID, Email: nullableText(email)})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return toAccount(row.ID, row.PersonID, row.Email, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

// SetAccountStatus sets accountID's status and returns the updated row.
func (r *Repository) SetAccountStatus(ctx context.Context, accountID, status string) (domain.Account, error) {
	row, err := r.q.SetAccountStatus(ctx, identitysql.SetAccountStatusParams{ID: accountID, Status: status})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return toAccount(row.ID, row.PersonID, row.Email, row.Status, row.CreatedAt, row.UpdatedAt), nil
}

// ResolveBySubject maps a verified (issuer, subject) to the account+person it federates to. A
// disabled account resolves as ErrIdentityNotFound, same as an unknown one.
func (r *Repository) ResolveBySubject(ctx context.Context, issuer, subject string) (domain.Resolution, error) {
	row, err := r.q.ResolveBySubject(ctx, identitysql.ResolveBySubjectParams{Issuer: issuer, Subject: subject})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Resolution{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.Resolution{}, err
	}
	return domain.Resolution{PersonID: row.PersonID, AccountID: row.AccountID, Email: asString(row.Email)}, nil
}

// InsertIdentity links (issuer, subject) to accountID. Callers must pre-check via ResolveBySubject.
func (r *Repository) InsertIdentity(ctx context.Context, accountID, issuer, subject string) (domain.ExternalIdentity, error) {
	id, err := r.q.InsertIdentity(ctx, identitysql.InsertIdentityParams{AccountID: accountID, Issuer: issuer, Subject: subject})
	if err != nil {
		return domain.ExternalIdentity{}, err
	}
	return domain.ExternalIdentity{ID: id, AccountID: accountID, Issuer: issuer, Subject: subject}, nil
}

func toSession(row identitysql.OpenfaithmapIdentitySession) domain.Session {
	return domain.Session{
		ID: row.ID, AccountID: row.AccountID, Issuer: row.Issuer, DeviceLabel: fromNullableText(row.DeviceLabel),
		CreatedAt: row.CreatedAt, LastSeenAt: row.LastSeenAt, RevokedAt: db.NullableTime(row.RevokedAt),
	}
}

// InsertSession creates a new identity_sessions row (M11.3) — one per NextAuth sign-in. deviceLabel
// is best-effort (User-Agent captured at sign-in) and may be empty.
func (r *Repository) InsertSession(ctx context.Context, accountID, issuer, deviceLabel string) (domain.Session, error) {
	row, err := r.q.InsertSession(ctx, identitysql.InsertSessionParams{AccountID: accountID, Issuer: issuer, DeviceLabel: nullableText(deviceLabel)})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return toSession(row), nil
}

// GetSession reads a single session by id, revoked or not.
func (r *Repository) GetSession(ctx context.Context, sessionID string) (domain.Session, error) {
	row, err := r.q.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return toSession(row), nil
}

// ListActiveSessionsByAccount returns accountID's not-yet-revoked sessions, most recently active
// first.
func (r *Repository) ListActiveSessionsByAccount(ctx context.Context, accountID string) ([]domain.Session, error) {
	rows, err := r.q.ListActiveSessionsByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSession(row))
	}
	return out, nil
}

// TouchSession validates sessionID (ErrSessionNotFound / ErrSessionRevoked) and, on the happy path,
// bumps last_seen_at — but only when the existing value is more than sessionTouchThrottle stale.
func (r *Repository) TouchSession(ctx context.Context, sessionID string) (domain.Session, error) {
	sess, err := r.GetSession(ctx, sessionID)
	if err != nil {
		return domain.Session{}, err
	}
	if sess.RevokedAt != nil {
		return sess, domain.ErrSessionRevoked
	}
	if time.Since(sess.LastSeenAt) <= sessionTouchThrottle {
		return sess, nil
	}
	row, err := r.q.UpdateSessionLastSeen(ctx, sessionID)
	if err != nil {
		return domain.Session{}, err
	}
	return toSession(row), nil
}

// LastActiveAtByAccount returns accountID's most recent session activity, revoked-inclusive. Nil if
// the account has never had a session.
func (r *Repository) LastActiveAtByAccount(ctx context.Context, accountID string) (*time.Time, error) {
	v, err := r.q.LastActiveAtByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return asNullableTime(v), nil
}

// RevokeSession sets revoked_at. Idempotent: revoking an already-revoked session is a no-op that
// still returns the current row.
func (r *Repository) RevokeSession(ctx context.Context, sessionID string) (domain.Session, error) {
	row, err := r.q.RevokeSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return toSession(row), nil
}

func toInvite(id, personID, accountID, email, status, invitedBy string, expiresAt, createdAt time.Time, acceptedAt pgtype.Timestamptz) domain.Invite {
	return domain.Invite{
		ID: id, PersonID: personID, AccountID: accountID, Email: email, Status: status, InvitedBy: invitedBy,
		ExpiresAt: expiresAt, CreatedAt: createdAt, AcceptedAt: db.NullableTime(acceptedAt),
	}
}

// InsertInvite creates a new identity_invites row (M11.6) — tokenHash is the caller's already-hashed
// token; this store method never sees or persists the raw value.
func (r *Repository) InsertInvite(ctx context.Context, personID, accountID, email, tokenHash, invitedBy string, expiresAt time.Time) (domain.Invite, error) {
	row, err := r.q.InsertInvite(ctx, identitysql.InsertInviteParams{
		PersonID: personID, AccountID: accountID, Email: email, TokenHash: tokenHash, InvitedBy: invitedBy, ExpiresAt: expiresAt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Invite{}, domain.ErrInviteNotFound
		}
		return domain.Invite{}, err
	}
	return toInvite(row.ID, row.PersonID, row.AccountID, row.Email, row.Status, row.InvitedBy, row.ExpiresAt, row.CreatedAt, row.AcceptedAt), nil
}

// InsertPersonAccountInvite runs the three inserts CreateInvite needs — Person, Account, Invite — in
// ONE transaction, the same atomicity shape internal/identity/bootstrap's own person+account+
// identity+instance-admin write already uses. Requires the Repository to be pool-bound (ok fails
// only if this Repository is already itself bound to a pgx.Tx, which never happens on the normal
// request path).
func (r *Repository) InsertPersonAccountInvite(ctx context.Context, displayName, email, tokenHash, invitedBy string, expiresAt time.Time) (domain.Invite, error) {
	pool, ok := r.conn.(*pgxpool.Pool)
	if !ok {
		return domain.Invite{}, errors.New("InsertPersonAccountInvite requires a pool-bound Repository")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return domain.Invite{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txRepo := NewRepository(tx)
	person, err := txRepo.InsertPerson(ctx, domain.Person{DisplayName: displayName})
	if err != nil {
		return domain.Invite{}, err
	}
	account, err := txRepo.InsertAccount(ctx, person.ID, email)
	if err != nil {
		return domain.Invite{}, err
	}
	invite, err := txRepo.InsertInvite(ctx, person.ID, account.ID, email, tokenHash, invitedBy, expiresAt)
	if err != nil {
		return domain.Invite{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Invite{}, err
	}
	return invite, nil
}

// PreviewMergeIdentity previews MergePersonIdentity's effect (M11.8) without mutating anything.
func (r *Repository) PreviewMergeIdentity(ctx context.Context, survivorID, duplicateID string) (duplicateHasActiveAccount, accountConflict bool, err error) {
	if _, err := r.GetActiveAccountByPerson(ctx, duplicateID); err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	if _, err := r.GetActiveAccountByPerson(ctx, survivorID); err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return true, false, nil
		}
		return false, false, err
	}
	return true, true, nil
}

// MergePersonIdentity does the identity-side work of M11.8's MergePersons. Must run inside the
// caller's own tx (core/application.Service.MergePersons); no Begin/Commit here.
func (r *Repository) MergePersonIdentity(ctx context.Context, survivorID, duplicateID string) (accountMoved, accountDisabled bool, err error) {
	duplicateAccount, err := r.GetActiveAccountByPerson(ctx, duplicateID)
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		// No account to move or disable — falls through to the person soft-delete below.
	case err != nil:
		return false, false, err
	default:
		_, survivorErr := r.GetActiveAccountByPerson(ctx, survivorID)
		switch {
		case errors.Is(survivorErr, domain.ErrAccountNotFound):
			if _, err := r.q.MoveAccountToPerson(ctx, identitysql.MoveAccountToPersonParams{ID: duplicateAccount.ID, PersonID: survivorID}); err != nil {
				return false, false, err
			}
			accountMoved = true
		case survivorErr != nil:
			return false, false, survivorErr
		default:
			if _, err := r.q.DisableAccount(ctx, identitysql.DisableAccountParams{ID: duplicateAccount.ID, Status: domain.AccountStatusDisabled}); err != nil {
				return false, false, err
			}
			if _, err := r.q.RevokeAccountSessions(ctx, duplicateAccount.ID); err != nil {
				return false, false, err
			}
			accountDisabled = true
		}
	}

	if _, err := r.q.DeactivatePerson(ctx, identitysql.DeactivatePersonParams{ID: duplicateID, Status: domain.PersonStatusDeactivated}); err != nil {
		return accountMoved, accountDisabled, err
	}
	return accountMoved, accountDisabled, nil
}

// GetInviteByTokenHash looks up an invite by its hashed token, pending or accepted.
func (r *Repository) GetInviteByTokenHash(ctx context.Context, tokenHash string) (domain.Invite, error) {
	row, err := r.q.GetInviteByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Invite{}, domain.ErrInviteNotFound
		}
		return domain.Invite{}, err
	}
	return toInvite(row.ID, row.PersonID, row.AccountID, row.Email, row.Status, row.InvitedBy, row.ExpiresAt, row.CreatedAt, row.AcceptedAt), nil
}

// MarkInviteAcceptedByAccount flips accountID's pending invite (if any) to accepted. Idempotent
// no-op if accountID has no pending invite.
func (r *Repository) MarkInviteAcceptedByAccount(ctx context.Context, accountID string) error {
	_, err := r.q.MarkInviteAcceptedByAccount(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func toAPIKey(id, personID, label string, permissionCodes []string, createdAt time.Time, lastUsedAt, revokedAt pgtype.Timestamptz, revokedBy pgtype.Text) domain.APIKey {
	key := domain.APIKey{ID: id, PersonID: personID, Label: label, PermissionCodes: permissionCodes, CreatedAt: createdAt, LastUsedAt: db.NullableTime(lastUsedAt), RevokedAt: db.NullableTime(revokedAt)}
	if revokedBy.Valid {
		key.RevokedBy = &revokedBy.String
	}
	return key
}

// InsertApiKey creates a new identity_api_keys row (M11.9) — tokenHash is the caller's already-hashed
// token; permissionCodes is the owner's chosen allowlist, already validated by the caller.
func (r *Repository) InsertApiKey(ctx context.Context, personID, label, tokenHash string, permissionCodes []string) (domain.APIKey, error) {
	row, err := r.q.InsertApiKey(ctx, identitysql.InsertApiKeyParams{PersonID: personID, Label: label, TokenHash: tokenHash, PermissionCodes: permissionCodes})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.APIKey{}, domain.ErrAPIKeyNotFound
		}
		return domain.APIKey{}, err
	}
	return toAPIKey(row.ID, row.PersonID, row.Label, row.PermissionCodes, row.CreatedAt, row.LastUsedAt, row.RevokedAt, row.RevokedBy), nil
}

// GetApiKeyByTokenHash resolves a raw API key's hash to its row — the authenticator's ResolveByAPIKey
// hook. A revoked key resolves as ErrAPIKeyNotFound, same as an unknown hash.
func (r *Repository) GetApiKeyByTokenHash(ctx context.Context, tokenHash string) (domain.APIKey, error) {
	row, err := r.q.GetApiKeyByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.APIKey{}, domain.ErrAPIKeyNotFound
		}
		return domain.APIKey{}, err
	}
	return toAPIKey(row.ID, row.PersonID, row.Label, row.PermissionCodes, row.CreatedAt, row.LastUsedAt, row.RevokedAt, row.RevokedBy), nil
}

// TouchApiKeyLastUsed bumps last_used_at, throttled the same interval sessionTouchThrottle uses.
func (r *Repository) TouchApiKeyLastUsed(ctx context.Context, apiKeyID string) error {
	row, err := r.q.GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAPIKeyNotFound
		}
		return err
	}
	key := toAPIKey(row.ID, row.PersonID, row.Label, row.PermissionCodes, row.CreatedAt, row.LastUsedAt, row.RevokedAt, row.RevokedBy)
	if key.LastUsedAt != nil && time.Since(*key.LastUsedAt) <= sessionTouchThrottle {
		return nil
	}
	_, err = r.q.UpdateApiKeyLastUsed(ctx, apiKeyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAPIKeyNotFound
	}
	return err
}

// ListApiKeysByPerson returns personID's not-yet-revoked keys, most recently created first.
func (r *Repository) ListApiKeysByPerson(ctx context.Context, personID string) ([]domain.APIKey, error) {
	rows, err := r.q.ListApiKeysByPerson(ctx, personID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.APIKey, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAPIKey(row.ID, row.PersonID, row.Label, row.PermissionCodes, row.CreatedAt, row.LastUsedAt, row.RevokedAt, row.RevokedBy))
	}
	return out, nil
}

// ListApiKeysByPersonIncludingRevoked is the admin-oversight read — personID's full key history.
func (r *Repository) ListApiKeysByPersonIncludingRevoked(ctx context.Context, personID string) ([]domain.APIKey, error) {
	rows, err := r.q.ListApiKeysByPersonIncludingRevoked(ctx, personID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.APIKey, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAPIKey(row.ID, row.PersonID, row.Label, row.PermissionCodes, row.CreatedAt, row.LastUsedAt, row.RevokedAt, row.RevokedBy))
	}
	return out, nil
}

// RevokeApiKey sets revoked_at/revoked_by, scoped to personID so a caller can never revoke a key it
// doesn't own. Idempotent: revoking an already-revoked key is a no-op that still returns the
// current row.
func (r *Repository) RevokeApiKey(ctx context.Context, apiKeyID, personID, revokedBy string) (domain.APIKey, error) {
	row, err := r.q.RevokeApiKey(ctx, identitysql.RevokeApiKeyParams{ID: apiKeyID, PersonID: personID, RevokedBy: nullableText(revokedBy)})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.APIKey{}, domain.ErrAPIKeyNotFound
		}
		return domain.APIKey{}, err
	}
	return toAPIKey(row.ID, row.PersonID, row.Label, row.PermissionCodes, row.CreatedAt, row.LastUsedAt, row.RevokedAt, row.RevokedBy), nil
}

// ResolveByAPIKey maps a raw API key's hash to the account+person it belongs to. A revoked key, an
// unknown hash, or an owner whose account is disabled/deleted all resolve as ErrIdentityNotFound.
func (r *Repository) ResolveByAPIKey(ctx context.Context, tokenHash string) (domain.Resolution, []string, error) {
	row, err := r.q.ResolveByAPIKey(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Resolution{}, nil, domain.ErrIdentityNotFound
	}
	if err != nil {
		return domain.Resolution{}, nil, err
	}
	return domain.Resolution{PersonID: row.PersonID, AccountID: row.AccountID, Email: asString(row.Email)}, row.PermissionCodes, nil
}
