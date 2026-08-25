-- name: GetActivePersonByCode :one
SELECT id, code, display_name, created_at, updated_at
FROM openfaithmap.identity_persons
WHERE code = sqlc.arg('code') AND deleted_at IS NULL;

-- name: InsertPerson :one
INSERT INTO openfaithmap.identity_persons (code, display_name)
VALUES (sqlc.narg('code'), sqlc.arg('display_name'))
RETURNING id, code, display_name, created_at, updated_at;

-- name: GetPerson :one
SELECT id, code, display_name, created_at, updated_at
FROM openfaithmap.identity_persons
WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: GetPersons :many
SELECT id, code, display_name, created_at, updated_at
FROM openfaithmap.identity_persons
WHERE id = ANY(sqlc.arg('ids')::uuid[]) AND deleted_at IS NULL;

-- name: UpdateDisplayName :one
UPDATE openfaithmap.identity_persons
SET display_name = sqlc.arg('display_name'), updated_at = now()
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, code, display_name, created_at, updated_at;

-- name: SearchPersons :many
SELECT p.id, p.code, p.display_name, p.created_at, p.updated_at, las.last_active
FROM openfaithmap.identity_persons p
LEFT JOIN openfaithmap.identity_accounts a ON a.person_id = p.id AND a.deleted_at IS NULL
LEFT JOIN (
	SELECT account_id, MAX(last_seen_at) AS last_active
	FROM openfaithmap.identity_sessions
	GROUP BY account_id
) las ON las.account_id = a.id
WHERE p.deleted_at IS NULL
  AND (sqlc.arg('query')::text = '' OR p.display_name ILIKE '%' || sqlc.arg('query')::text || '%' OR p.code ILIKE '%' || sqlc.arg('query')::text || '%')
ORDER BY p.display_name
LIMIT sqlc.arg('limit_count');

-- name: GetActiveAccountByPerson :one
SELECT id, person_id, COALESCE(email::text, '') AS email, status, created_at, updated_at
FROM openfaithmap.identity_accounts
WHERE person_id = sqlc.arg('person_id') AND deleted_at IS NULL;

-- name: GetActiveAccountByEmail :one
-- email is citext: this comparison is case-insensitive, and the partial unique active-index
-- (identity_accounts_email_active_idx) makes "the single account" true by construction.
SELECT id, person_id, COALESCE(email::text, '') AS email, status, created_at, updated_at
FROM openfaithmap.identity_accounts
WHERE email = sqlc.arg('email')::citext AND deleted_at IS NULL;

-- name: InsertAccount :one
INSERT INTO openfaithmap.identity_accounts (person_id, email)
VALUES (sqlc.arg('person_id'), sqlc.narg('email')::citext)
RETURNING id, person_id, COALESCE(email::text, '') AS email, status, created_at, updated_at;

-- name: SetAccountStatus :one
UPDATE openfaithmap.identity_accounts
SET status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, person_id, COALESCE(email::text, '') AS email, status, created_at, updated_at;

-- name: ResolveBySubject :one
SELECT a.person_id, a.id AS account_id, COALESCE(a.email::text, '') AS email
FROM openfaithmap.identity_external_identities x
JOIN openfaithmap.identity_accounts a ON a.id = x.account_id AND a.deleted_at IS NULL AND a.status = 'active'
WHERE x.issuer = sqlc.arg('issuer') AND x.subject = sqlc.arg('subject');

-- name: InsertIdentity :one
INSERT INTO openfaithmap.identity_external_identities (account_id, issuer, subject)
VALUES (sqlc.arg('account_id'), sqlc.arg('issuer'), sqlc.arg('subject'))
RETURNING id;

-- name: InsertSession :one
INSERT INTO openfaithmap.identity_sessions (account_id, issuer, device_label)
VALUES (sqlc.arg('account_id'), sqlc.arg('issuer'), sqlc.narg('device_label'))
RETURNING id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at;

-- name: GetSession :one
SELECT id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at
FROM openfaithmap.identity_sessions
WHERE id = sqlc.arg('id');

-- name: ListActiveSessionsByAccount :many
SELECT id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at
FROM openfaithmap.identity_sessions
WHERE account_id = sqlc.arg('account_id') AND revoked_at IS NULL
ORDER BY last_seen_at DESC;

-- name: UpdateSessionLastSeen :one
UPDATE openfaithmap.identity_sessions
SET last_seen_at = now()
WHERE id = sqlc.arg('id')
RETURNING id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at;

-- name: LastActiveAtByAccount :one
-- No cast on MAX(last_seen_at): an account with zero sessions must scan as NULL (nullable
-- pgtype.Timestamptz), not error — a ::timestamptz cast here made sqlc infer a non-nullable
-- time.Time instead, which would panic-scan on that empty case.
SELECT MAX(last_seen_at) AS last_active
FROM openfaithmap.identity_sessions
WHERE account_id = sqlc.arg('account_id');

-- name: RevokeSession :one
UPDATE openfaithmap.identity_sessions
SET revoked_at = COALESCE(revoked_at, now())
WHERE id = sqlc.arg('id')
RETURNING id, account_id, issuer, device_label, created_at, last_seen_at, revoked_at;

-- name: InsertInvite :one
INSERT INTO openfaithmap.identity_invites (person_id, account_id, email, token_hash, invited_by, expires_at)
VALUES (sqlc.arg('person_id'), sqlc.arg('account_id'), sqlc.arg('email')::citext, sqlc.arg('token_hash'), sqlc.arg('invited_by'), sqlc.arg('expires_at'))
RETURNING id, person_id, account_id, email::text AS email, status, invited_by, expires_at, created_at, accepted_at;

-- name: GetInviteByTokenHash :one
SELECT id, person_id, account_id, email::text AS email, status, invited_by, expires_at, created_at, accepted_at
FROM openfaithmap.identity_invites
WHERE token_hash = sqlc.arg('token_hash');

-- name: MarkInviteAcceptedByAccount :one
UPDATE openfaithmap.identity_invites
SET status = 'accepted', accepted_at = now()
WHERE account_id = sqlc.arg('account_id') AND status = 'pending'
RETURNING id;

-- name: MoveAccountToPerson :one
UPDATE openfaithmap.identity_accounts
SET person_id = sqlc.arg('person_id'), updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING id;

-- name: DisableAccount :one
UPDATE openfaithmap.identity_accounts
SET status = sqlc.arg('status'), deleted_at = now(), updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING id;

-- name: RevokeAccountSessions :many
UPDATE openfaithmap.identity_sessions
SET revoked_at = now()
WHERE account_id = sqlc.arg('account_id') AND revoked_at IS NULL
RETURNING id;

-- name: DeactivatePerson :one
UPDATE openfaithmap.identity_persons
SET status = sqlc.arg('status'), deleted_at = now(), updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING id;

-- name: InsertApiKey :one
INSERT INTO openfaithmap.identity_api_keys (person_id, label, token_hash, permission_codes)
VALUES (sqlc.arg('person_id'), sqlc.arg('label'), sqlc.arg('token_hash'), sqlc.arg('permission_codes'))
RETURNING id, person_id, label, permission_codes, created_at, last_used_at, revoked_at, revoked_by;

-- name: GetApiKeyByTokenHash :one
SELECT id, person_id, label, permission_codes, created_at, last_used_at, revoked_at, revoked_by
FROM openfaithmap.identity_api_keys
WHERE token_hash = sqlc.arg('token_hash') AND revoked_at IS NULL;

-- name: GetApiKeyByID :one
SELECT id, person_id, label, permission_codes, created_at, last_used_at, revoked_at, revoked_by
FROM openfaithmap.identity_api_keys
WHERE id = sqlc.arg('id');

-- name: UpdateApiKeyLastUsed :one
UPDATE openfaithmap.identity_api_keys
SET last_used_at = now()
WHERE id = sqlc.arg('id')
RETURNING id, person_id, label, permission_codes, created_at, last_used_at, revoked_at, revoked_by;

-- name: ListApiKeysByPerson :many
SELECT id, person_id, label, permission_codes, created_at, last_used_at, revoked_at, revoked_by
FROM openfaithmap.identity_api_keys
WHERE person_id = sqlc.arg('person_id') AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: ListApiKeysByPersonIncludingRevoked :many
SELECT id, person_id, label, permission_codes, created_at, last_used_at, revoked_at, revoked_by
FROM openfaithmap.identity_api_keys
WHERE person_id = sqlc.arg('person_id')
ORDER BY created_at DESC;

-- name: RevokeApiKey :one
UPDATE openfaithmap.identity_api_keys
SET revoked_at = COALESCE(revoked_at, now()), revoked_by = COALESCE(revoked_by, sqlc.arg('revoked_by'))
WHERE id = sqlc.arg('id') AND person_id = sqlc.arg('person_id')
RETURNING id, person_id, label, permission_codes, created_at, last_used_at, revoked_at, revoked_by;

-- name: ResolveByAPIKey :one
SELECT a.person_id, a.id AS account_id, COALESCE(a.email::text, '') AS email, k.permission_codes
FROM openfaithmap.identity_api_keys k
JOIN openfaithmap.identity_accounts a ON a.person_id = k.person_id AND a.deleted_at IS NULL AND a.status = 'active'
WHERE k.token_hash = sqlc.arg('token_hash') AND k.revoked_at IS NULL;
