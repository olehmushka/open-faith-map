-- 0005_vouching — M6 (docs/milestones.md), docs/modules/vouching.md, D-Vouching.
--
-- Two tables: vouching_edges (append-only event log — a vouch, once filed, is never edited or
-- deleted; reject_mutation()-guarded, reusing the function 0004_moderation.sql already created,
-- not redefined here) and vouching_guarantor_status (a mutable one-row-per-guarantor overlay
-- recording whether a guarantor is currently trusted). guarantor_person_rid, claimant_person_rid,
-- and congregation_unit_rid are opaque TEXT foreign values referencing go-oikumenea Person/Unit
-- entities — no cross-schema FK (architecture/conventions.md); a dangling reference is handled at
-- read time as "no longer exists," not a crash.
--
-- The absence of a vouching_guarantor_status row for a person means TRUSTED (the column DEFAULT) —
-- application code synthesizes that value rather than requiring a row to exist up front.

CREATE TABLE openfaithmap.vouching_edges (
  id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  guarantor_person_rid   text NOT NULL,
  claimant_person_rid    text NOT NULL,
  congregation_unit_rid  text NOT NULL,
  statement              text,
  created_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX vouching_edges_claimant_congregation_idx
  ON openfaithmap.vouching_edges (claimant_person_rid, congregation_unit_rid, created_at DESC);
CREATE INDEX vouching_edges_guarantor_idx
  ON openfaithmap.vouching_edges (guarantor_person_rid);

CREATE TRIGGER vouching_edges_reject_mutation
  BEFORE UPDATE OR DELETE ON openfaithmap.vouching_edges
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.reject_mutation();

CREATE TABLE openfaithmap.vouching_guarantor_status (
  guarantor_person_rid   text PRIMARY KEY,
  status                 text NOT NULL DEFAULT 'trusted' CHECK (status IN ('trusted', 'revoked')),
  revoked_at             timestamptz,
  revoked_reason         text,
  revoked_by_person_rid  text,
  updated_at             timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER vouching_guarantor_status_set_updated_at
  BEFORE UPDATE ON openfaithmap.vouching_guarantor_status
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
