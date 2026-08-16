-- 0013_congregationimport_jurisdiction_units — D-CatholicJurisdictionSync
-- (docs/architecture/decisions.md), a narrow, deliberate exception to D-JurisdictionUnits: automated,
-- unattended creation of JURISDICTION-TIER go-oikumenea Units (jurisdiction/diocese/eparchy/deanery
-- org-kinds) from a specific, high-confidence structured source (Wikidata, cross-referenced to
-- Catholic-Hierarchy.org via P1866) — never congregation-level Units, which keep the existing
-- operator-approval flow (congregationimport_candidates) completely unchanged.
--
-- (source_code, external_id) is the idempotency anchor — the source's own stable natural key (a
-- Wikidata QID). A re-run of the same JurisdictionSource recognizes an already-CREATED node by this
-- key and skips it rather than calling createChildOrg a second time, mirroring
-- congregationimport_candidates_source_key's own role for congregation candidates.
--
-- created_unit_id is an opaque TEXT cross-schema reference, never validated against go-oikumenea at
-- write time — same discipline congregationimport_jurisdiction_aliases.jurisdiction_unit_id already
-- uses; a stale/renamed unit is a read-time concern, not a write-time gate.
--
-- parent_external_id is a self-reference BY NATURAL KEY, not a real FK to this table's own id — a
-- node's parent may not have a row here yet at insert time (the sync processes nodes in the order a
-- source's Fetch emits them, parent-before-child by JurisdictionSource's own contract, but a batch
-- boundary can still see a child before its parent's row is committed within the same transaction
-- boundary the application layer chooses).
CREATE TABLE openfaithmap.congregationimport_jurisdiction_units (
  id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_code          text NOT NULL,
  external_id          text NOT NULL,
  parent_external_id   text,
  name                 text NOT NULL,
  org_kind_id          text NOT NULL,
  status               text NOT NULL DEFAULT 'PENDING'
                          CHECK (status IN ('PENDING', 'CREATED', 'FAILED')),
  created_unit_id      text,
  failure_reason       text,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT congregationimport_jurisdiction_units_natural_key UNIQUE (source_code, external_id),
  -- Mirrors congregationimport_candidates_decision_shape's own discipline: the status column and its
  -- supporting fields must agree, checked at the database, not just trusted from application code.
  CONSTRAINT congregationimport_jurisdiction_units_status_shape CHECK (
    (status = 'CREATED' AND created_unit_id IS NOT NULL) OR
    (status = 'FAILED' AND failure_reason IS NOT NULL) OR
    (status = 'PENDING' AND created_unit_id IS NULL)
  )
);

CREATE INDEX congregationimport_jurisdiction_units_status_idx
  ON openfaithmap.congregationimport_jurisdiction_units (source_code, status);
CREATE INDEX congregationimport_jurisdiction_units_parent_idx
  ON openfaithmap.congregationimport_jurisdiction_units (source_code, parent_external_id);

CREATE TRIGGER congregationimport_jurisdiction_units_set_updated_at
  BEFORE UPDATE ON openfaithmap.congregationimport_jurisdiction_units
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
