-- name: InsertAuditEntry :exec
INSERT INTO openfaithmap.identity_audit_log (actor_person_id, action, target_kind, target_id, before, after)
VALUES (sqlc.narg('actor_person_id'), sqlc.arg('action'), sqlc.arg('target_kind'), sqlc.arg('target_id'),
	sqlc.narg('before'), sqlc.narg('after'));

-- name: ListAuditEntries :many
-- Dynamic filter via "narg IS NULL OR column = narg", the same idiom go-oikumenea's own
-- filter queries use, so this stays one static statement sqlc can analyze — the runtime
-- WHERE-clause assembly the hand-written store did is no longer needed.
SELECT
	l.id,
	COALESCE(l.actor_person_id::text, '')::text AS actor_person_id,
	COALESCE(p.display_name, '')::text AS actor_person_name,
	l.action,
	l.target_kind,
	l.target_id,
	l.before,
	l.after,
	l.created_at
FROM openfaithmap.identity_audit_log l
LEFT JOIN openfaithmap.identity_persons p ON p.id = l.actor_person_id
WHERE (sqlc.narg('actor_person_id')::uuid IS NULL OR l.actor_person_id = sqlc.narg('actor_person_id')::uuid)
	AND (sqlc.narg('target_kind')::text IS NULL OR l.target_kind = sqlc.narg('target_kind')::text)
	AND (sqlc.narg('target_id')::text IS NULL OR l.target_id = sqlc.narg('target_id')::text)
	AND (sqlc.narg('created_from')::timestamptz IS NULL OR l.created_at >= sqlc.narg('created_from')::timestamptz)
	AND (sqlc.narg('created_to')::timestamptz IS NULL OR l.created_at <= sqlc.narg('created_to')::timestamptz)
	AND (
		sqlc.narg('after_created_at')::timestamptz IS NULL
		OR (l.created_at, l.id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
	)
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg('page_size');
