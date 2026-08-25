-- name: GetUnit :one
SELECT id, code, name, level, state, metadata, created_at, updated_at
FROM openfaithmap.directory_units
WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: GetUnitByCode :one
-- Added for the seed-ID refactor (internal/platform/seed.Resolve): directory_units.code carries a
-- unique-while-active index (directory_units_code_active_idx) the same shape as
-- authz_roles_code_active_idx — mirrors GetGraphByCode's own shape.
SELECT id, code, name, level, state, metadata, created_at, updated_at
FROM openfaithmap.directory_units
WHERE code = sqlc.arg('code') AND deleted_at IS NULL;

-- name: MintUnitID :one
SELECT openfaithmap.new_id(3, 1, 1)::text AS id;

-- name: InsertUnit :one
INSERT INTO openfaithmap.directory_units (code, name, level, state, metadata)
VALUES (sqlc.narg('code'), sqlc.arg('name'), sqlc.narg('level'), sqlc.arg('state'), sqlc.arg('metadata'))
RETURNING id, code, name, level, state, metadata, created_at, updated_at;

-- name: InsertUnitWithID :one
INSERT INTO openfaithmap.directory_units (id, code, name, level, state, metadata)
VALUES (sqlc.arg('id'), sqlc.narg('code'), sqlc.arg('name'), sqlc.narg('level'), sqlc.arg('state'), sqlc.arg('metadata'))
RETURNING id, code, name, level, state, metadata, created_at, updated_at;

-- name: UpdateUnit :one
UPDATE openfaithmap.directory_units
SET name = sqlc.arg('name'), code = sqlc.narg('code'), level = sqlc.narg('level')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, code, name, level, state, metadata, created_at, updated_at;

-- name: SetUnitState :one
UPDATE openfaithmap.directory_units
SET state = sqlc.arg('state')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, code, name, level, state, metadata, created_at, updated_at;

-- name: HasChildren :one
SELECT EXISTS(
	SELECT 1 FROM openfaithmap.directory_unit_edges e
	JOIN openfaithmap.directory_units u ON u.id = e.child_id AND u.deleted_at IS NULL
	WHERE e.parent_id = sqlc.arg('id')
);

-- name: SoftDeleteUnit :one
UPDATE openfaithmap.directory_units
SET deleted_at = now()
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, code, name, level, state, metadata, created_at, updated_at;

-- name: SearchUnits :many
SELECT id, code, name, level, state, metadata, created_at, updated_at
FROM openfaithmap.directory_units
WHERE deleted_at IS NULL
  AND (sqlc.arg('query')::text = '' OR code ILIKE '%' || sqlc.arg('query')::text || '%' OR name ILIKE '%' || sqlc.arg('query')::text || '%')
ORDER BY name
LIMIT sqlc.arg('limit_count');

-- name: GetGraphByCode :one
SELECT id, code, name, is_default, is_authority_bearing
FROM openfaithmap.directory_graphs
WHERE code = sqlc.arg('code') AND deleted_at IS NULL;

-- name: GetGraphByID :one
SELECT id, code, name, is_default, is_authority_bearing
FROM openfaithmap.directory_graphs
WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: ListGraphs :many
SELECT id, code, name, is_default, is_authority_bearing
FROM openfaithmap.directory_graphs WHERE deleted_at IS NULL ORDER BY code;

-- name: LockGraphForClosure :one
SELECT id FROM openfaithmap.directory_graphs WHERE id = sqlc.arg('id') FOR NO KEY UPDATE;

-- name: ClosureHasPath :one
SELECT EXISTS(
	SELECT 1 FROM openfaithmap.directory_unit_closure
	WHERE graph_id = sqlc.arg('graph_id') AND ancestor_id = sqlc.arg('ancestor_id') AND descendant_id = sqlc.arg('descendant_id')
);

-- name: InsertEdge :one
INSERT INTO openfaithmap.directory_unit_edges (graph_id, parent_id, child_id)
VALUES (sqlc.arg('graph_id'), sqlc.arg('parent_id'), sqlc.arg('child_id'))
RETURNING id, parent_id, child_id, created_at;

-- name: DeleteEdge :execrows
DELETE FROM openfaithmap.directory_unit_edges
WHERE graph_id = sqlc.arg('graph_id') AND parent_id = sqlc.arg('parent_id') AND child_id = sqlc.arg('child_id');

-- name: SeedClosureSelfRows :exec
INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
VALUES (sqlc.arg('graph_id'), sqlc.arg('parent_id'), sqlc.arg('parent_id'), 0), (sqlc.arg('graph_id'), sqlc.arg('child_id'), sqlc.arg('child_id'), 0)
ON CONFLICT DO NOTHING;

-- name: ExtendClosureForEdge :exec
INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
SELECT sqlc.arg('graph_id')::uuid, anc.ancestor_id, dsc.descendant_id, anc.depth + dsc.depth + 1
FROM openfaithmap.directory_unit_closure anc
JOIN openfaithmap.directory_unit_closure dsc
  ON dsc.graph_id = sqlc.arg('graph_id') AND dsc.ancestor_id = sqlc.arg('child_id')
WHERE anc.graph_id = sqlc.arg('graph_id') AND anc.descendant_id = sqlc.arg('parent_id')
ON CONFLICT (graph_id, ancestor_id, descendant_id)
DO UPDATE SET depth = LEAST(openfaithmap.directory_unit_closure.depth, EXCLUDED.depth);

-- name: DeleteClosureSlice :exec
WITH anc AS (
  SELECT tc.ancestor_id AS u FROM openfaithmap.directory_unit_closure tc
  WHERE tc.graph_id = sqlc.arg('graph_id') AND tc.descendant_id = sqlc.arg('parent_id')
  UNION SELECT sqlc.arg('parent_id')::uuid
),
dsc AS (
  SELECT tc.descendant_id AS u FROM openfaithmap.directory_unit_closure tc
  WHERE tc.graph_id = sqlc.arg('graph_id') AND tc.ancestor_id = sqlc.arg('child_id')
  UNION SELECT sqlc.arg('child_id')::uuid
)
DELETE FROM openfaithmap.directory_unit_closure tc
WHERE tc.graph_id = sqlc.arg('graph_id')
  AND tc.ancestor_id   IN (SELECT u FROM anc)
  AND tc.descendant_id IN (SELECT u FROM dsc);

-- name: RederiveClosureSlice :exec
WITH RECURSIVE
anc AS (
  SELECT tc.ancestor_id AS u FROM openfaithmap.directory_unit_closure tc
  WHERE tc.graph_id = sqlc.arg('graph_id') AND tc.descendant_id = sqlc.arg('parent_id')
  UNION SELECT sqlc.arg('parent_id')::uuid
),
dsc AS (
  SELECT tc.descendant_id AS u FROM openfaithmap.directory_unit_closure tc
  WHERE tc.graph_id = sqlc.arg('graph_id') AND tc.ancestor_id = sqlc.arg('child_id')
  UNION SELECT sqlc.arg('child_id')::uuid
),
walk AS (
  SELECT a.u AS ancestor_id, a.u AS node, 0 AS depth FROM anc a
  UNION ALL
  SELECT w.ancestor_id, e.child_id, w.depth + 1
  FROM walk w
  JOIN openfaithmap.directory_unit_edges e ON e.graph_id = sqlc.arg('graph_id') AND e.parent_id = w.node
  WHERE w.node IN (SELECT u FROM anc)
),
pairs AS (
  SELECT w.ancestor_id, tc.descendant_id, w.depth + tc.depth AS depth
  FROM walk w
  JOIN openfaithmap.directory_unit_closure tc ON tc.graph_id = sqlc.arg('graph_id') AND tc.ancestor_id = w.node
  WHERE w.node NOT IN (SELECT u FROM anc)
    AND tc.descendant_id IN (SELECT u FROM dsc)
)
INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
SELECT sqlc.arg('graph_id')::uuid, ancestor_id, descendant_id, min(depth)::int
FROM pairs GROUP BY ancestor_id, descendant_id;

-- name: PruneClosureSelfRows :exec
DELETE FROM openfaithmap.directory_unit_closure tc
WHERE tc.graph_id = sqlc.arg('graph_id')
  AND tc.ancestor_id = tc.descendant_id
  AND tc.ancestor_id IN (sqlc.arg('parent_id')::uuid, sqlc.arg('child_id')::uuid)
  AND NOT EXISTS (
    SELECT 1 FROM openfaithmap.directory_unit_edges e
    WHERE e.graph_id = sqlc.arg('graph_id') AND (e.parent_id = tc.ancestor_id OR e.child_id = tc.ancestor_id)
  );

-- name: DeleteClosureForGraph :exec
DELETE FROM openfaithmap.directory_unit_closure WHERE graph_id = sqlc.arg('graph_id');

-- name: RebuildClosureForGraph :exec
WITH RECURSIVE
nodes AS (
  SELECT parent_id AS u FROM openfaithmap.directory_unit_edges e0 WHERE e0.graph_id = sqlc.arg('graph_id')
  UNION SELECT child_id FROM openfaithmap.directory_unit_edges e1 WHERE e1.graph_id = sqlc.arg('graph_id')
),
reach AS (
  SELECT u AS ancestor_id, u AS descendant_id, 0 AS depth FROM nodes
  UNION ALL
  SELECT r.ancestor_id, e.child_id, r.depth + 1
  FROM reach r
  JOIN openfaithmap.directory_unit_edges e ON e.graph_id = sqlc.arg('graph_id') AND e.parent_id = r.descendant_id
)
INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
SELECT sqlc.arg('graph_id')::uuid, ancestor_id, descendant_id, min(depth)::int
FROM reach GROUP BY ancestor_id, descendant_id;

-- name: VerifyClosureForGraph :one
WITH RECURSIVE
nodes AS (
  SELECT parent_id AS u FROM openfaithmap.directory_unit_edges e0 WHERE e0.graph_id = sqlc.arg('graph_id')
  UNION SELECT child_id FROM openfaithmap.directory_unit_edges e1 WHERE e1.graph_id = sqlc.arg('graph_id')
),
reach AS (
  SELECT u AS ancestor_id, u AS descendant_id, 0 AS depth FROM nodes
  UNION ALL
  SELECT r.ancestor_id, e.child_id, r.depth + 1
  FROM reach r
  JOIN openfaithmap.directory_unit_edges e ON e.graph_id = sqlc.arg('graph_id') AND e.parent_id = r.descendant_id
),
expected AS (
  SELECT ancestor_id, descendant_id, min(depth)::int AS depth FROM reach GROUP BY ancestor_id, descendant_id
),
stored AS (
  SELECT tc.ancestor_id, tc.descendant_id, tc.depth FROM openfaithmap.directory_unit_closure tc WHERE tc.graph_id = sqlc.arg('graph_id')
),
missing AS (SELECT ancestor_id, descendant_id, depth FROM expected EXCEPT SELECT ancestor_id, descendant_id, depth FROM stored),
extra   AS (SELECT ancestor_id, descendant_id, depth FROM stored   EXCEPT SELECT ancestor_id, descendant_id, depth FROM expected)
SELECT
  (SELECT count(*) FROM missing)::int AS missing_count,
  (SELECT count(*) FROM extra)::int AS extra_count,
  (SELECT coalesce(jsonb_agg(s), '[]'::jsonb) FROM (
     (SELECT 'missing'::text AS kind, ancestor_id, descendant_id FROM missing LIMIT 5)
     UNION ALL
     (SELECT 'extra'::text AS kind, ancestor_id, descendant_id FROM extra LIMIT 5)
   ) s)::jsonb AS sample;

-- name: UpsertClosureStatus :exec
INSERT INTO openfaithmap.directory_closure_status (graph_id, missing_count, extra_count, in_drift, sample, last_checked_at)
VALUES (sqlc.arg('graph_id'), sqlc.arg('missing_count'), sqlc.arg('extra_count'), sqlc.arg('in_drift'), sqlc.arg('sample'), now())
ON CONFLICT (graph_id) DO UPDATE SET
  missing_count = EXCLUDED.missing_count, extra_count = EXCLUDED.extra_count,
  in_drift = EXCLUDED.in_drift, sample = EXCLUDED.sample, last_checked_at = now();

-- name: ListAncestors :many
SELECT u.id, COALESCE(u.code, '') AS code, u.name, c.depth
FROM openfaithmap.directory_unit_closure c
JOIN openfaithmap.directory_units u ON u.id = c.ancestor_id AND u.deleted_at IS NULL
WHERE c.graph_id = sqlc.arg('graph_id') AND c.descendant_id = sqlc.arg('unit_id') AND c.depth > 0
ORDER BY c.depth, u.code;

-- name: ListDescendants :many
SELECT u.id, COALESCE(u.code, '') AS code, u.name, c.depth
FROM openfaithmap.directory_unit_closure c
JOIN openfaithmap.directory_units u ON u.id = c.descendant_id AND u.deleted_at IS NULL
WHERE c.graph_id = sqlc.arg('graph_id') AND c.ancestor_id = sqlc.arg('unit_id') AND c.depth > 0
ORDER BY c.descendant_id
LIMIT sqlc.arg('limit_count');

-- name: CreateMoveJob :one
INSERT INTO openfaithmap.directory_unit_move_jobs
	(graph_id, unit_id, old_parent_unit_id, new_parent_unit_id, performed_by_person_id)
VALUES (sqlc.arg('graph_id'), sqlc.arg('unit_id'), sqlc.arg('old_parent_unit_id'), sqlc.arg('new_parent_unit_id'), sqlc.arg('performed_by_person_id'))
RETURNING id, graph_id, unit_id, old_parent_unit_id, new_parent_unit_id, status, performed_by_person_id, error, created_at, updated_at;

-- name: GetLiveMoveJob :one
SELECT id, graph_id, unit_id, old_parent_unit_id, new_parent_unit_id, status, performed_by_person_id, error, created_at, updated_at
FROM openfaithmap.directory_unit_move_jobs
WHERE graph_id = sqlc.arg('graph_id') AND unit_id = sqlc.arg('unit_id') AND status NOT IN ('FAILED', 'VERIFIED');

-- name: GetLatestMoveJob :one
SELECT id, graph_id, unit_id, old_parent_unit_id, new_parent_unit_id, status, performed_by_person_id, error, created_at, updated_at
FROM openfaithmap.directory_unit_move_jobs
WHERE graph_id = sqlc.arg('graph_id') AND unit_id = sqlc.arg('unit_id')
ORDER BY created_at DESC LIMIT 1;

-- name: AdvanceMoveJob :one
UPDATE openfaithmap.directory_unit_move_jobs
SET status = sqlc.arg('status'), error = NULL
WHERE id = sqlc.arg('id')
RETURNING id, graph_id, unit_id, old_parent_unit_id, new_parent_unit_id, status, performed_by_person_id, error, created_at, updated_at;

-- name: FailMoveJob :one
UPDATE openfaithmap.directory_unit_move_jobs
SET status = 'FAILED', error = sqlc.arg('error')
WHERE id = sqlc.arg('id')
RETURNING id, graph_id, unit_id, old_parent_unit_id, new_parent_unit_id, status, performed_by_person_id, error, created_at, updated_at;
