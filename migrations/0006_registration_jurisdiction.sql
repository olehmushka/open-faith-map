-- 0006_registration_jurisdiction — M4.1 (docs/milestones.md), D-JurisdictionUnits
-- (architecture/decisions.md).
--
-- Two additive pieces, both expand-only:
--
-- 1. registration_requests.jurisdiction_unit_id — the operator's jurisdiction choice at approval
--    time. Nullable: jurisdiction stays genuinely optional (D-JurisdictionUnits — some congregations
--    have no real denomination-specific jurisdiction and remain direct children of root, exactly as
--    today).
--
-- 2. jurisdiction_reparenting_jobs — tracks re-parenting an already-APPROVED request's congregation
--    unit onto a different jurisdiction unit. go-oikumenea's addEdge+removeEdge (canonical graph) is
--    two non-transactional calls, not one atomic move (live-verified: no dedicated reparent endpoint
--    exists for religion units), so this is a resumable state machine mirroring the
--    PROVISIONING/created_unit_id pattern 0002 already established: persist the durable fact as soon
--    as the irreversible-ish call returns.

ALTER TABLE openfaithmap.registration_requests
  ADD COLUMN jurisdiction_unit_id text;

CREATE TABLE openfaithmap.jurisdiction_reparenting_jobs (
  id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  registration_request_id  uuid REFERENCES openfaithmap.registration_requests(id) ON DELETE SET NULL,
  congregation_unit_id     text NOT NULL,
  old_parent_unit_id       text NOT NULL,
  new_parent_unit_id       text NOT NULL,
  status                   text NOT NULL DEFAULT 'PENDING'
                             CHECK (status IN ('PENDING', 'NEW_EDGE_ADDED', 'OLD_EDGE_REMOVED', 'VERIFIED', 'FAILED')),
  performed_by_person_id   text NOT NULL,
  error                    text,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT jurisdiction_reparenting_jobs_error_shape CHECK (
    (status = 'FAILED' AND error IS NOT NULL) OR
    (status <> 'FAILED' AND error IS NULL)
  )
);

-- At most one live (non-FAILED) job per congregation unit at a time — a FAILED job doesn't block a
-- fresh attempt, but two simultaneous in-flight moves of the same unit would race.
CREATE UNIQUE INDEX jurisdiction_reparenting_jobs_live_unit_idx
  ON openfaithmap.jurisdiction_reparenting_jobs (congregation_unit_id)
  WHERE status <> 'FAILED';

CREATE INDEX jurisdiction_reparenting_jobs_request_idx
  ON openfaithmap.jurisdiction_reparenting_jobs (registration_request_id, created_at DESC);

CREATE TRIGGER jurisdiction_reparenting_jobs_set_updated_at
  BEFORE UPDATE ON openfaithmap.jurisdiction_reparenting_jobs
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
