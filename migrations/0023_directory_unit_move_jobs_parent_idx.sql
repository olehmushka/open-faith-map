-- 0023_directory_unit_move_jobs_parent_idx — 0022 gave directory_unit_move_jobs FK columns
-- old_parent_unit_id/new_parent_unit_id (both REFERENCES directory_units(id) ON DELETE RESTRICT)
-- with no dedicated index on either: the only existing index covering them incidentally is the
-- composite (graph_id, unit_id) one, which doesn't help a lookup or FK-cascade-check keyed on a
-- parent column directly. The two columns are queried/constrained independently, not together, so
-- two single-column indexes rather than one composite.
CREATE INDEX directory_unit_move_jobs_old_parent_idx
  ON openfaithmap.directory_unit_move_jobs (old_parent_unit_id);
CREATE INDEX directory_unit_move_jobs_new_parent_idx
  ON openfaithmap.directory_unit_move_jobs (new_parent_unit_id);
