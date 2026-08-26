-- name: InsertPosition :one
INSERT INTO openfaithmap.membership_positions (unit_id, code, title)
VALUES (sqlc.arg('unit_id'), sqlc.arg('code'), sqlc.arg('title'))
RETURNING id, unit_id, code, title, status, created_at, updated_at;

-- name: GetPosition :one
SELECT id, unit_id, code, title, status, created_at, updated_at
FROM openfaithmap.membership_positions
WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: ListPositionsByUnit :many
SELECT id, unit_id, code, title, status, created_at, updated_at
FROM openfaithmap.membership_positions
WHERE unit_id = sqlc.arg('unit_id') AND deleted_at IS NULL
ORDER BY sort_order NULLS LAST, code;

-- name: ListMembershipsByUnit :many
SELECT id, person_id, unit_id, position_id, status, effective_from
FROM openfaithmap.membership_memberships
WHERE unit_id = sqlc.arg('unit_id') AND status = 'active' AND deleted_at IS NULL
ORDER BY effective_from;

-- name: InsertMembershipFillingPosition :one
INSERT INTO openfaithmap.membership_memberships (person_id, unit_id, position_id)
VALUES (sqlc.arg('person_id'), sqlc.arg('unit_id'), sqlc.arg('position_id'))
RETURNING id, person_id, unit_id, position_id, status, effective_from;

-- name: CountRepointableMemberships :one
-- The EXISTS predicate below is deliberately duplicated across this query and
-- RepointMemberships*: sqlc has no macro/include system, so the shared Go const the hand-written
-- store used (membershipRepointCollisionPredicate) becomes literal duplication here. Keep the two
-- copies identical if either changes — they must never disagree (see the store's original comment).
SELECT
	count(*) FILTER (WHERE NOT (
		EXISTS (
			SELECT 1 FROM openfaithmap.membership_memberships s
			WHERE s.person_id = sqlc.arg('survivor_id') AND s.status = 'active' AND s.deleted_at IS NULL
			  AND ((m.position_id IS NOT NULL AND s.position_id = m.position_id)
			    OR (m.position_id IS NULL AND s.position_id IS NULL AND s.unit_id = m.unit_id))
		)
	))::bigint AS to_move,
	count(*) FILTER (WHERE (
		EXISTS (
			SELECT 1 FROM openfaithmap.membership_memberships s
			WHERE s.person_id = sqlc.arg('survivor_id') AND s.status = 'active' AND s.deleted_at IS NULL
			  AND ((m.position_id IS NOT NULL AND s.position_id = m.position_id)
			    OR (m.position_id IS NULL AND s.position_id IS NULL AND s.unit_id = m.unit_id))
		)
	))::bigint AS to_end
FROM openfaithmap.membership_memberships m
WHERE m.person_id = sqlc.arg('duplicate_id') AND m.status = 'active' AND m.deleted_at IS NULL;

-- name: RepointMoveMemberships :many
UPDATE openfaithmap.membership_memberships m
SET person_id = sqlc.arg('survivor_id'), updated_at = now()
WHERE m.person_id = sqlc.arg('duplicate_id') AND m.status = 'active' AND m.deleted_at IS NULL
  AND NOT (
	EXISTS (
		SELECT 1 FROM openfaithmap.membership_memberships s
		WHERE s.person_id = sqlc.arg('survivor_id') AND s.status = 'active' AND s.deleted_at IS NULL
		  AND ((m.position_id IS NOT NULL AND s.position_id = m.position_id)
		    OR (m.position_id IS NULL AND s.position_id IS NULL AND s.unit_id = m.unit_id))
	)
  )
RETURNING m.id;

-- name: RepointEndRedundantMemberships :many
UPDATE openfaithmap.membership_memberships
SET status = 'ended', effective_to = now(), updated_at = now()
WHERE person_id = sqlc.arg('duplicate_id') AND status = 'active' AND deleted_at IS NULL
RETURNING id;
