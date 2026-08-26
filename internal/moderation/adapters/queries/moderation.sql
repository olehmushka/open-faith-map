-- name: InsertReport :one
INSERT INTO openfaithmap.moderation_reports (target_kind, target_ref, reason_code, detail, queue_scope)
VALUES (sqlc.arg('target_kind'), sqlc.arg('target_ref'), sqlc.arg('reason_code'), sqlc.narg('detail'), sqlc.arg('queue_scope'))
RETURNING id, target_kind, target_ref, reason_code, detail, reporter_person_id, queue_scope, status, created_at, updated_at;

-- name: GetReportByID :one
SELECT id, target_kind, target_ref, reason_code, detail, reporter_person_id, queue_scope, status, created_at, updated_at
FROM openfaithmap.moderation_reports WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: ListReports :many
SELECT id, target_kind, target_ref, reason_code, detail, reporter_person_id, queue_scope, status, created_at, updated_at
FROM openfaithmap.moderation_reports
WHERE deleted_at IS NULL
	AND (sqlc.narg('queue_scope')::text IS NULL OR queue_scope = sqlc.narg('queue_scope')::text)
	AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
	AND (
		sqlc.narg('after_created_at')::timestamptz IS NULL
		OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
	)
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg('page_size');

-- name: MarkReportStatus :one
UPDATE openfaithmap.moderation_reports SET status = sqlc.arg('status')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, target_kind, target_ref, reason_code, detail, reporter_person_id, queue_scope, status, created_at, updated_at;

-- name: InsertAction :one
INSERT INTO openfaithmap.moderation_actions (report_id, action_kind, target_kind, target_ref, actor_person_id, reason, reverses_action_id)
VALUES (sqlc.narg('report_id'), sqlc.arg('action_kind'), sqlc.arg('target_kind'), sqlc.arg('target_ref'), sqlc.arg('actor_person_id'),
	sqlc.arg('reason'), sqlc.narg('reverses_action_id'))
RETURNING id, report_id, action_kind, target_kind, target_ref, actor_person_id, reason, reverses_action_id, created_at;

-- name: GetActionByID :one
SELECT id, report_id, action_kind, target_kind, target_ref, actor_person_id, reason, reverses_action_id, created_at
FROM openfaithmap.moderation_actions WHERE id = sqlc.arg('id');

-- name: GetReverserActionID :one
-- Read-time counterpart to reverses_action_id's backward-only, insert-time-only write (see
-- domain.Action's doc comment) — returns pgx.ErrNoRows when no REVERSE action points at this one.
SELECT id FROM openfaithmap.moderation_actions WHERE reverses_action_id = sqlc.arg('action_id');

-- name: InsertAppeal :one
INSERT INTO openfaithmap.moderation_appeals (action_id, congregation_admin_person_id, statement)
VALUES (sqlc.arg('action_id'), sqlc.arg('congregation_admin_person_id'), sqlc.arg('statement'))
RETURNING id, action_id, congregation_admin_person_id, statement, assigned_moderator_person_id, status, created_at, updated_at;

-- name: GetAppealByID :one
SELECT id, action_id, congregation_admin_person_id, statement, assigned_moderator_person_id, status, created_at, updated_at
FROM openfaithmap.moderation_appeals WHERE id = sqlc.arg('id');

-- name: ListAppeals :many
SELECT id, action_id, congregation_admin_person_id, statement, assigned_moderator_person_id, status, created_at, updated_at
FROM openfaithmap.moderation_appeals
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
	AND (
		sqlc.narg('after_created_at')::timestamptz IS NULL
		OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
	)
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg('page_size');

-- name: DecideAppeal :one
UPDATE openfaithmap.moderation_appeals SET assigned_moderator_person_id = sqlc.arg('assigned_moderator_person_id'), status = sqlc.arg('status')
WHERE id = sqlc.arg('id')
RETURNING id, action_id, congregation_admin_person_id, statement, assigned_moderator_person_id, status, created_at, updated_at;
