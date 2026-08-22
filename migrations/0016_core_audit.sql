-- 0016_core_audit — M11.2 (docs/milestones.md). Additive; no retrofitted table since none of this
-- existed before. See docs/architecture/decisions.md for the two concrete choices this migration
-- makes: the RID placement (folded into the identity service rather than a new service number) and
-- the before/after payload shape (curated jsonb per call site, not a full-row diff).
--
-- identity_audit_log: append-only ledger of every mutating super-admin action — one row per action,
-- written by internal/auditlog. Shape mirrors moderation_actions' (0004_moderation.sql) proven
-- append-only pattern, the only existing precedent for an actor+target ledger in this schema:
-- no updated_at/deleted_at (nothing to timestamp a mutation of — reject_mutation() forbids UPDATE
-- and DELETE outright), actor+target+created_at as the core shape.
--
-- target_kind/target_id is an opaque discriminator+ref pair, not a single FK, since targets already
-- span role assignments, instance-admin grants, and accounts, and will span sessions (M11.3) and
-- persons (M11.8 merge) later — same reasoning moderation_reports.target_kind/target_ref already
-- established for a target set that can't be pinned to one table.
--
-- actor_person_id is nullable with ON DELETE SET NULL, same convention as
-- authz_role_assignments.granted_by/revoked_by (0009_core_authz.sql) — a later person deletion must
-- never cascade away the log entry documenting what that person did.
--
-- action is free text, validated against a closed catalog in Go (internal/auditlog/domain), not a DB
-- CHECK — this milestone's own stated goal is that later milestones (M11.3 session revocation, M11.7
-- bulk role assignment, M11.8 person merge) log against this table from day one, which means the
-- action catalog grows across migrations that never touch this table again.
CREATE TABLE openfaithmap.identity_audit_log (
  id               uuid PRIMARY KEY DEFAULT openfaithmap.new_id(1,1,4),  -- identity / object / audit_log
  actor_person_id  uuid REFERENCES openfaithmap.identity_persons(id) ON DELETE SET NULL,
  action           text NOT NULL,
  target_kind      text NOT NULL,
  target_id        text NOT NULL,
  before           jsonb,
  after            jsonb,
  created_at       timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT identity_audit_log_rid_shape
    CHECK (openfaithmap.rid_service(id)=1 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=4)
);

CREATE INDEX identity_audit_log_created_id_idx
  ON openfaithmap.identity_audit_log (created_at DESC, id DESC);
CREATE INDEX identity_audit_log_actor_idx
  ON openfaithmap.identity_audit_log (actor_person_id, created_at DESC);
CREATE INDEX identity_audit_log_target_idx
  ON openfaithmap.identity_audit_log (target_kind, target_id, created_at DESC);

-- openfaithmap.reject_mutation() is not redefined here — it already exists, first defined in
-- 0004_moderation.sql and reused by every append-only table since.
CREATE TRIGGER identity_audit_log_reject_mutation
  BEFORE UPDATE OR DELETE ON openfaithmap.identity_audit_log
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.reject_mutation();
