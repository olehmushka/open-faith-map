-- name: InsertSite :one
INSERT INTO openfaithmap.content_sites (congregation_unit_rid, slug)
VALUES (sqlc.arg('congregation_unit_rid'), sqlc.arg('slug'))
RETURNING id, congregation_unit_rid, slug, theme, created_at, updated_at;

-- name: GetSiteByID :one
SELECT id, congregation_unit_rid, slug, theme, created_at, updated_at
FROM openfaithmap.content_sites WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: GetSiteByUnit :one
SELECT id, congregation_unit_rid, slug, theme, created_at, updated_at
FROM openfaithmap.content_sites WHERE congregation_unit_rid = sqlc.arg('congregation_unit_rid') AND deleted_at IS NULL;

-- name: UpdateSiteTheme :one
UPDATE openfaithmap.content_sites SET theme = sqlc.arg('theme')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, congregation_unit_rid, slug, theme, created_at, updated_at;

-- name: GetBlockTypeByCode :one
SELECT id, code, name, json_schema, status, sort_order
FROM openfaithmap.content_block_types WHERE code = sqlc.arg('code') AND deleted_at IS NULL;

-- name: ListActiveBlockTypes :many
SELECT id, code, name, json_schema, status, sort_order
FROM openfaithmap.content_block_types
WHERE status = 'ACTIVE' AND deleted_at IS NULL
ORDER BY sort_order ASC;

-- name: ListBlocks :many
SELECT b.id, b.document_id, b.block_type_id, t.code AS block_type_code, b.position, b.data, b.created_at, b.updated_at
FROM openfaithmap.content_blocks b
JOIN openfaithmap.content_block_types t ON t.id = b.block_type_id
WHERE b.document_id = sqlc.arg('document_id') AND b.deleted_at IS NULL
ORDER BY b.position ASC;

-- name: DeleteBlocksForDocument :exec
DELETE FROM openfaithmap.content_blocks WHERE document_id = sqlc.arg('document_id');

-- name: InsertBlockByTypeCode :exec
INSERT INTO openfaithmap.content_blocks (document_id, block_type_id, position, data)
SELECT sqlc.arg('document_id'), t.id, sqlc.arg('position'), sqlc.arg('data')
FROM openfaithmap.content_block_types t
WHERE t.code = sqlc.arg('block_type_code') AND t.deleted_at IS NULL;

-- name: InsertDocument :one
INSERT INTO openfaithmap.content_documents
	(site_id, kind, translation_group_id, locale, parent_document_id, slug,
	 event_starts_at, event_ends_at, event_recurrence_rrule)
VALUES (sqlc.arg('site_id'), sqlc.arg('kind'), COALESCE(sqlc.narg('translation_group_id')::uuid, gen_random_uuid()),
	sqlc.arg('locale'), sqlc.narg('parent_document_id'), sqlc.arg('slug'),
	sqlc.narg('event_starts_at'), sqlc.narg('event_ends_at'), sqlc.narg('event_recurrence_rrule'))
RETURNING id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at;

-- name: GetDocument :one
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at
FROM openfaithmap.content_documents WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: UpdateDocument :one
UPDATE openfaithmap.content_documents
SET slug = COALESCE(sqlc.narg('slug'), slug),
    parent_document_id = CASE
      WHEN sqlc.arg('clear_parent')::bool THEN NULL
      WHEN sqlc.narg('parent_document_id')::uuid IS NOT NULL THEN sqlc.narg('parent_document_id')::uuid
      ELSE parent_document_id
    END
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at;

-- name: UpdateDocumentState :one
UPDATE openfaithmap.content_documents
SET state = sqlc.arg('state'), published_at = CASE WHEN sqlc.arg('first_publish')::bool THEN now() ELSE published_at END
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at;

-- name: ListDocuments :many
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND deleted_at IS NULL
	AND (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind')::text)
	AND (sqlc.narg('locale')::text IS NULL OR locale = sqlc.narg('locale')::text)
	AND (sqlc.narg('state')::text IS NULL OR state = sqlc.narg('state')::text)
ORDER BY created_at DESC;

-- name: ListPublicDocuments :many
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND deleted_at IS NULL AND state IN ('PUBLISHED', 'UNLISTED')
	AND (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind')::text)
	AND (sqlc.narg('locale')::text IS NULL OR locale = sqlc.narg('locale')::text)
ORDER BY CASE WHEN kind = 'EVENT' THEN event_starts_at END ASC NULLS LAST, created_at DESC;
