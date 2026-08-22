-- 0018_core_invites — M11.6 (docs/milestones.md), D-InviteLinkMVP
-- (docs/architecture/decisions.md). Additive; no retrofitted table since none of this existed
-- before.
--
-- identity_invites: one row per invite-a-teammate link. Mutable, like identity_sessions
-- (0017_core_sessions.sql), not append-only like identity_audit_log — status flips pending ->
-- accepted in place, no updated_at trigger needed since accepted_at itself records the one mutation
-- this table ever sees.
--
-- No "expired"/"revoked" stored status: expiry is checked live against expires_at (no sweeper job
-- needed), and revocation deliberately reuses M11.1's existing account deactivate/reactivate rather
-- than inventing a second status column — ResolveInvite also checks the linked account's own status,
-- so deactivating the pre-provisioned account already invalidates a bad invite.
--
-- token_hash, not the raw token: only a SHA-256 digest is ever persisted (Go's
-- internal/identity/application generates the random token and hashes it before InsertInvite), the
-- same defensive shape password-reset tokens use elsewhere — a stolen row can't be replayed as a
-- credential.
CREATE TABLE openfaithmap.identity_invites (
  id           uuid PRIMARY KEY DEFAULT openfaithmap.new_id(1,1,6),  -- identity / object / invite
  person_id    uuid NOT NULL REFERENCES openfaithmap.identity_persons(id) ON DELETE RESTRICT,
  account_id   uuid NOT NULL REFERENCES openfaithmap.identity_accounts(id) ON DELETE RESTRICT,
  email        citext NOT NULL,
  token_hash   text NOT NULL,
  status       text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted')),
  invited_by   uuid NOT NULL REFERENCES openfaithmap.identity_persons(id),
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  accepted_at  timestamptz,

  CONSTRAINT identity_invites_rid_shape
    CHECK (openfaithmap.rid_service(id)=1 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=6)
);

-- Backs ResolveInvite's token lookup — the only way this table is ever read by hash.
CREATE UNIQUE INDEX identity_invites_token_hash_idx
  ON openfaithmap.identity_invites (token_hash);

-- Backs LinkOnMatch's post-link "does this account have a pending invite to accept" lookup.
CREATE INDEX identity_invites_account_pending_idx
  ON openfaithmap.identity_invites (account_id) WHERE status = 'pending';
