-- 0017_core_sessions — M11.3 (docs/milestones.md), D-SessionTracking
-- (docs/architecture/decisions.md). Additive; no retrofitted table since none of this existed
-- before.
--
-- identity_sessions: one row per NextAuth sign-in on the admin app. Unlike identity_audit_log
-- (0016_core_audit.sql), this table is MUTABLE, not append-only — last_seen_at is bumped on the
-- request path (throttled in Go, internal/identity/adapters) and revoked_at is set by
-- RevokeSession — so it follows identity_accounts' shape (0008_core_identity.sql) instead of
-- moderation_actions' reject_mutation() pattern.
--
-- account_id, not person_id: a session belongs to the login attachment (identity_accounts), same
-- level identity_external_identities links at. ON DELETE CASCADE matches
-- identity_external_identities' own FK to identity_accounts — an account's sessions have no
-- meaning once the account itself is gone.
--
-- issuer is recorded per-session (not just implied by "the only IdP is Google") so a future second
-- IdP, or the reserved local/dev issuer used by tests, is visible per-row without a join.
--
-- No secret/token column distinct from id: the RID itself (openfaithmap.new_id, UUIDv8 with ~44
-- bits of random tail) is the session identifier handed to the client as X-Session-Id. It is a
-- revocation handle, not a standalone credential — reaching the per-request session check at all
-- already requires a separately-valid bearer (Google ID token or, in dev, the reserved local HS256
-- issuer) that resolves to the SAME account_id (internal/identity/middleware.Authenticator.Handle
-- cross-checks this), so a guessed session id alone grants nothing.
CREATE TABLE openfaithmap.identity_sessions (
  id            uuid PRIMARY KEY DEFAULT openfaithmap.new_id(1,1,5),  -- identity / object / session
  account_id    uuid NOT NULL REFERENCES openfaithmap.identity_accounts(id) ON DELETE CASCADE,
  issuer        text NOT NULL,          -- the IdP `iss` claim active when this session was created
  device_label  text,                   -- best-effort User-Agent captured at sign-in; optional
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_seen_at  timestamptz NOT NULL DEFAULT now(),
  revoked_at    timestamptz,

  CONSTRAINT identity_sessions_rid_shape
    CHECK (openfaithmap.rid_service(id)=1 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=5)
);

-- Backs both ListSessions (an account's still-active sessions) and the per-request Touch lookup's
-- "is this session revoked" check.
CREATE INDEX identity_sessions_account_active_idx
  ON openfaithmap.identity_sessions (account_id) WHERE revoked_at IS NULL;
