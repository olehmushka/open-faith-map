-- 0020_directory_unit_move_jobs — M12.2. Generalizes jurisdiction_reparenting_jobs
-- (migrations/0001_registration.sql) into internal/directory's own generic Move/GetMoveStatus: the
-- same add-before-remove, resumable state machine, no longer private to
-- internal/registration.Service.Reparent (which becomes a caller of it instead of the sole owner).
--
-- jurisdiction_reparenting_jobs itself is left in place, untouched, as a frozen historical log
-- (this repo's expand-only migration convention) — internal/registration no longer writes to it
-- after this milestone, but its existing rows stay as-is.
--
-- Real uuid FKs into directory_units/directory_graphs (unlike jurisdiction_reparenting_jobs' opaque
-- text columns, a cross-module concession that table needed and this one, living inside the module
-- that owns both tables, does not). No registration_request_id column — this module cannot depend on
-- internal/registration (D-InProcessAuthz); a caller that needs to correlate a move back to its own
-- domain object does so on its own side, keyed by (graph_id, unit_id).
--
-- old_parent_unit_id/new_parent_unit_id each get their own single-column index below (2026-08-25
-- squash pass, folded in from this migration's original follow-up patch): the two columns are
-- queried/constrained independently, not together, so two single-column indexes rather than one
-- composite — the only existing index covering them incidentally, (graph_id, unit_id), doesn't help a
-- lookup or FK-cascade-check keyed on a parent column directly.

CREATE TABLE openfaithmap.directory_unit_move_jobs (
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  graph_id                uuid NOT NULL REFERENCES openfaithmap.directory_graphs(id) ON DELETE RESTRICT,
  unit_id                 uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  old_parent_unit_id      uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  new_parent_unit_id      uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  status                  text NOT NULL DEFAULT 'PENDING'
                            CHECK (status IN ('PENDING', 'NEW_EDGE_ADDED', 'OLD_EDGE_REMOVED', 'VERIFIED', 'FAILED')),
  performed_by_person_id  text NOT NULL,
  error                   text,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT directory_unit_move_jobs_error_shape CHECK (
    (status = 'FAILED' AND error IS NOT NULL) OR
    (status <> 'FAILED' AND error IS NULL)
  )
);

-- At most one IN-PROGRESS job (PENDING/NEW_EDGE_ADDED/OLD_EDGE_REMOVED) per (graph, unit) at a time —
-- two simultaneous in-flight moves of the same unit in the same graph would race. Both FAILED and
-- VERIFIED are terminal and excluded, not just FAILED: unlike the single-use congregation-reparent
-- flow this generalizes, a generic Move is expected to be called on the same unit again after a
-- prior move already reached VERIFIED, and a terminal VERIFIED row must not permanently occupy the
-- one live slot and block every later move of that unit.
CREATE UNIQUE INDEX directory_unit_move_jobs_live_idx
  ON openfaithmap.directory_unit_move_jobs (graph_id, unit_id)
  WHERE status NOT IN ('FAILED', 'VERIFIED');

CREATE INDEX directory_unit_move_jobs_unit_idx
  ON openfaithmap.directory_unit_move_jobs (graph_id, unit_id, created_at DESC);

CREATE INDEX directory_unit_move_jobs_old_parent_idx
  ON openfaithmap.directory_unit_move_jobs (old_parent_unit_id);
CREATE INDEX directory_unit_move_jobs_new_parent_idx
  ON openfaithmap.directory_unit_move_jobs (new_parent_unit_id);

CREATE TRIGGER directory_unit_move_jobs_set_updated_at
  BEFORE UPDATE ON openfaithmap.directory_unit_move_jobs
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
