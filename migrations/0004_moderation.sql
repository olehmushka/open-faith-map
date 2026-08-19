-- 0004_moderation — M5/M7 (docs/milestones.md), docs/modules/moderation.md, D-Hardening. Squashed
-- from the original 0007/0009 into final shape directly — the keyset-pagination composite indexes
-- land as the real index set from the start, no "old index left in place" expand-only artifact.
--
-- Three tables: moderation_reports (mutable, soft-deletable), moderation_actions (append-only —
-- this platform's ledger of record for moderation decisions, per D-Moderation's Correction:
-- go-oikumenea's audit module has no write endpoint, so this is not a mirror of anything),
-- moderation_appeals (mutable). Every go-oikumenea RID referenced here (reporter_person_id,
-- actor_person_id, congregation_admin_person_id, assigned_moderator_person_id) is an opaque TEXT
-- foreign value — no cross-database FKs (architecture/conventions.md).
--
-- openfaithmap.reject_mutation() is this repo's own append-only guard (named in
-- conventions.md/moderation.md/decisions.md/vouching.md), first defined here — ported from
-- go-oikumenea's own migrations/0001_platform_core.sql pattern, schema-qualified to openfaithmap the
-- same way set_updated_at() already is (0001_registration.sql).

CREATE OR REPLACE FUNCTION openfaithmap.reject_mutation() RETURNS trigger
  LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'append-only table %.%: % is not permitted',
    TG_TABLE_SCHEMA, TG_TABLE_NAME, TG_OP USING ERRCODE = 'restrict_violation';
END;
$$;

CREATE TABLE openfaithmap.moderation_reports (
  id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  target_kind          text NOT NULL CHECK (target_kind IN ('SITE', 'DOCUMENT', 'CONGREGATION', 'VOUCHING_EDGE')),
  target_ref           text NOT NULL,
  reason_code          text NOT NULL CHECK (reason_code IN ('SPAM', 'INCORRECT_INFORMATION', 'INAPPROPRIATE_CONTENT', 'DUPLICATE', 'OTHER')),
  detail               text,
  reporter_person_id   text,
  queue_scope          text NOT NULL CHECK (queue_scope IN ('PLATFORM', 'CONGREGATION', 'JURISDICTION')),
  status               text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'ACTIONED', 'DISMISSED')),
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE INDEX moderation_reports_scope_status_created_id_idx
  ON openfaithmap.moderation_reports (queue_scope, status, created_at DESC, id DESC);
CREATE INDEX moderation_reports_target_idx
  ON openfaithmap.moderation_reports (target_kind, target_ref);

CREATE TRIGGER moderation_reports_set_updated_at
  BEFORE UPDATE ON openfaithmap.moderation_reports
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- Append-only: no updated_at/deleted_at at all — there is structurally nothing to timestamp a
-- mutation of, since reject_mutation() forbids UPDATE and DELETE outright.
--
-- reverses_action_id points BACKWARD, set only on a REVERSE row itself at insert time — never
-- forward onto the original row. moderation.md's API describes ModerationAction.reversedByActionId
-- as if it were a column set "on the original row once a reverse action targets it", but that would
-- require an UPDATE, which reject_mutation() forbids unconditionally. The application layer derives
-- that field at read time instead (does any REVERSE row exist with reverses_action_id = this id?) —
-- same fact, computed forward-scanning instead of stored backward-mutating, so the table stays
-- genuinely append-only with zero exceptions.
CREATE TABLE openfaithmap.moderation_actions (
  id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id            uuid REFERENCES openfaithmap.moderation_reports(id) ON DELETE SET NULL,
  action_kind          text NOT NULL CHECK (action_kind IN ('HIDE', 'SUSPEND', 'ARCHIVE', 'WARN_ADMIN', 'REVOKE_VOUCH', 'REVERSE')),
  target_kind          text NOT NULL CHECK (target_kind IN ('SITE', 'DOCUMENT', 'CONGREGATION', 'VOUCHING_EDGE')),
  target_ref           text NOT NULL,
  actor_person_id      text NOT NULL,
  reason               text NOT NULL,
  reverses_action_id   uuid REFERENCES openfaithmap.moderation_actions(id),
  created_at           timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT moderation_actions_reverses_shape CHECK (
    (action_kind = 'REVERSE' AND reverses_action_id IS NOT NULL) OR
    (action_kind <> 'REVERSE' AND reverses_action_id IS NULL)
  )
);

CREATE INDEX moderation_actions_target_idx
  ON openfaithmap.moderation_actions (target_kind, target_ref, created_at DESC);
CREATE INDEX moderation_actions_report_idx
  ON openfaithmap.moderation_actions (report_id);

-- At most one REVERSE row per reversed action — a second reversal attempt is rejected by the
-- application (domain.ErrActionNotReversible) before it would ever hit this, but the constraint is
-- the real guarantee, not the application check (same discipline as jurisdiction_reparenting_jobs'
-- live-unit unique index).
CREATE UNIQUE INDEX moderation_actions_reverses_idx
  ON openfaithmap.moderation_actions (reverses_action_id)
  WHERE reverses_action_id IS NOT NULL;

CREATE TRIGGER moderation_actions_reject_mutation
  BEFORE UPDATE OR DELETE ON openfaithmap.moderation_actions
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.reject_mutation();

CREATE TABLE openfaithmap.moderation_appeals (
  id                             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  action_id                      uuid NOT NULL REFERENCES openfaithmap.moderation_actions(id),
  congregation_admin_person_id   text NOT NULL,
  statement                      text NOT NULL,
  assigned_moderator_person_id   text,
  status                         text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'UPHELD', 'OVERTURNED')),
  created_at                     timestamptz NOT NULL DEFAULT now(),
  updated_at                     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX moderation_appeals_status_created_id_idx
  ON openfaithmap.moderation_appeals (status, created_at DESC, id DESC);
CREATE INDEX moderation_appeals_action_idx ON openfaithmap.moderation_appeals (action_id);

CREATE TRIGGER moderation_appeals_set_updated_at
  BEFORE UPDATE ON openfaithmap.moderation_appeals
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
