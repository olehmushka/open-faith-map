-- 0020_core_api_keys — M11.9 (docs/milestones.md), the last M11.x sub-milestone. Additive; no
-- retrofitted table since none of this existed before.
--
-- identity_api_keys: a second credential for an existing person, not a new principal type —
-- authz.Subject's own doc comment ("no machine-subject arm, every subject is a person",
-- D-DirectTokenVerification) stays true. Resolution happens by token_hash instead of the
-- (issuer, subject) pair identity_external_identities uses (internal/identity/middleware's
-- authenticator branches on the raw bearer's shape before ever reaching JWT parsing).
--
-- person_id, not account_id: unlike identity_sessions (account-scoped, since a session belongs to
-- one login attachment), permissions evaluate per-person via authz.ActiveGrantsForSubject(personID),
-- matching authz_role_assignments.subject_person_id's own grain.
--
-- token_hash, not the raw secret: only a SHA-256 digest is ever persisted (Go's
-- internal/identity/application generates the random ofm_-prefixed token and hashes it before
-- InsertApiKey), the same shape identity_invites.token_hash already uses.
--
-- permission_codes: a fixed allowlist chosen by the owner at creation time, validated against the
-- closed Go catalog (internal/authz/domain/permissions.go) at insert time, not a DB constraint. This
-- is deliberately NOT a new authz_role_assignments row or a new authz migration: it's a snapshot
-- allowlist owned entirely by this table, intersected against the owning person's LIVE grants at
-- request time (internal/authz.Service.Require) rather than stored as a grant itself. No FK to any
-- authz table exists or is needed. Instance-scope codes are rejected at creation (CreateApiKey) since
-- RequireInstanceAdmin hard-denies every API-key-authenticated subject regardless of allowlist
-- contents (an API key can never ride the instance-admin plane's "allow everything" bypass).
--
-- Mutable, like identity_sessions: last_used_at is bumped on the request path (throttled in Go, same
-- pattern as adapters.sessionTouchThrottle) and revoked_at is set by RevokeApiKey. No expires_at: out
-- of scope for this milestone (docs/milestones.md's own field list is "created/revoked timestamps"
-- only) — an optional nullable column could be added additively later without breaking this design.
CREATE TABLE openfaithmap.identity_api_keys (
  id                uuid PRIMARY KEY DEFAULT openfaithmap.new_id(1,1,7),  -- identity / object / api_key
  person_id         uuid NOT NULL REFERENCES openfaithmap.identity_persons(id) ON DELETE RESTRICT,
  label             text NOT NULL,
  token_hash        text NOT NULL,
  permission_codes  text[] NOT NULL,
  created_at        timestamptz NOT NULL DEFAULT now(),
  last_used_at      timestamptz,
  revoked_at        timestamptz,
  revoked_by        uuid REFERENCES openfaithmap.identity_persons(id),

  CONSTRAINT identity_api_keys_rid_shape
    CHECK (openfaithmap.rid_service(id)=1 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=7)
);

-- Backs ResolveByAPIKey's token lookup — the only way this table is ever read by hash.
CREATE UNIQUE INDEX identity_api_keys_token_hash_idx
  ON openfaithmap.identity_api_keys (token_hash);

-- Backs both ListMyApiKeys/ListApiKeysByPerson (an owner's still-active keys) and the admin-oversight
-- list, which additionally reads revoked rows via a plain person_id scan (no index needed — that path
-- is low-volume, admin-only).
CREATE INDEX identity_api_keys_person_active_idx
  ON openfaithmap.identity_api_keys (person_id) WHERE revoked_at IS NULL;
