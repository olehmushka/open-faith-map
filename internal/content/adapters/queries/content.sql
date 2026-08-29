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

-- name: GetSiteBySlug :one
SELECT id, congregation_unit_rid, slug, theme, created_at, updated_at
FROM openfaithmap.content_sites WHERE slug = sqlc.arg('slug') AND deleted_at IS NULL;

-- name: UpdateSiteTheme :one
UPDATE openfaithmap.content_sites SET theme = sqlc.arg('theme')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, congregation_unit_rid, slug, theme, created_at, updated_at;

-- name: GetBlockTypeByCode :one
SELECT id, code, name, json_schema, ui_schema, status, sort_order
FROM openfaithmap.content_block_types WHERE code = sqlc.arg('code') AND deleted_at IS NULL;

-- name: ListActiveBlockTypes :many
SELECT id, code, name, json_schema, ui_schema, status, sort_order
FROM openfaithmap.content_block_types
WHERE status = 'ACTIVE' AND deleted_at IS NULL
ORDER BY sort_order ASC;

-- name: InsertDocument :one
INSERT INTO openfaithmap.content_documents
	(site_id, kind, translation_group_id, locale, parent_document_id, slug,
	 event_starts_at, event_ends_at, event_recurrence_rrule)
VALUES (sqlc.arg('site_id'), sqlc.arg('kind'), COALESCE(sqlc.narg('translation_group_id')::uuid, gen_random_uuid()),
	sqlc.arg('locale'), sqlc.narg('parent_document_id'), sqlc.arg('slug'),
	sqlc.narg('event_starts_at'), sqlc.narg('event_ends_at'), sqlc.narg('event_recurrence_rrule'))
RETURNING id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id;

-- name: GetDocument :one
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id
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
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id;

-- name: UpdateDocumentState :one
UPDATE openfaithmap.content_documents
SET state = sqlc.arg('state'), published_at = CASE WHEN sqlc.arg('first_publish')::bool THEN now() ELSE published_at END
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id;

-- name: ListDocuments :many
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND deleted_at IS NULL
	AND (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind')::text)
	AND (sqlc.narg('locale')::text IS NULL OR locale = sqlc.narg('locale')::text)
	AND (sqlc.narg('state')::text IS NULL OR state = sqlc.narg('state')::text)
ORDER BY created_at DESC;

-- name: ListPublicDocuments :many
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND deleted_at IS NULL AND state IN ('PUBLISHED', 'UNLISTED')
	AND (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind')::text)
	AND (sqlc.narg('locale')::text IS NULL OR locale = sqlc.narg('locale')::text)
ORDER BY CASE WHEN kind = 'EVENT' THEN event_starts_at END ASC NULLS LAST, created_at DESC;

-- ---- revisions (M14.6) ----

-- name: InsertRevision :one
INSERT INTO openfaithmap.content_document_revisions (document_id, revision_no, data, author_person_id, label)
VALUES (sqlc.arg('document_id'), sqlc.arg('revision_no'), sqlc.arg('data'), sqlc.narg('author_person_id'), sqlc.narg('label'))
RETURNING id, document_id, revision_no, data, author_person_id, created_at, label;

-- name: NextRevisionNo :one
SELECT COALESCE(MAX(revision_no), 0) + 1 FROM openfaithmap.content_document_revisions WHERE document_id = sqlc.arg('document_id');

-- name: GetRevision :one
SELECT id, document_id, revision_no, data, author_person_id, created_at, label
FROM openfaithmap.content_document_revisions WHERE id = sqlc.arg('id');

-- name: UpdateRevisionData :one
UPDATE openfaithmap.content_document_revisions SET data = sqlc.arg('data')
WHERE id = sqlc.arg('id')
RETURNING id, document_id, revision_no, data, author_person_id, created_at, label;

-- name: SetDraftRevision :exec
UPDATE openfaithmap.content_documents SET draft_revision_id = sqlc.arg('draft_revision_id') WHERE id = sqlc.arg('id');

-- name: SetPublishedRevision :exec
UPDATE openfaithmap.content_documents SET published_revision_id = sqlc.arg('published_revision_id') WHERE id = sqlc.arg('id');

-- name: ListCheckpointRevisions :many
SELECT id, document_id, revision_no, data, author_person_id, created_at, label
FROM openfaithmap.content_document_revisions
WHERE document_id = sqlc.arg('document_id') AND id != sqlc.arg('exclude_id')
ORDER BY revision_no DESC;

-- name: PruneCheckpointRevisions :exec
DELETE FROM openfaithmap.content_document_revisions cdr
WHERE cdr.document_id = sqlc.arg('document_id')
  AND cdr.id != sqlc.arg('keep_draft_id')
  AND cdr.id != sqlc.arg('keep_published_id')
  AND cdr.revision_no <= (
    SELECT r2.revision_no FROM openfaithmap.content_document_revisions r2
    WHERE r2.document_id = sqlc.arg('document_id')
    ORDER BY r2.revision_no DESC
    OFFSET sqlc.arg('keep_count') LIMIT 1
  );

-- name: GetDocumentBySlug :one
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND kind = sqlc.arg('kind') AND locale = sqlc.arg('locale')
	AND slug = sqlc.arg('slug') AND deleted_at IS NULL;

-- ---- nav items (M14.10) ----

-- name: DeleteNavItems :exec
DELETE FROM openfaithmap.content_site_nav_items WHERE site_id = sqlc.arg('site_id');

-- name: InsertNavItem :one
INSERT INTO openfaithmap.content_site_nav_items (site_id, label, target_document_id, target_url, sort_order)
VALUES (sqlc.arg('site_id'), sqlc.arg('label'), sqlc.narg('target_document_id'), sqlc.narg('target_url'), sqlc.arg('sort_order'))
RETURNING id, site_id, label, target_document_id, target_url, sort_order;

-- name: ListNavItems :many
SELECT id, site_id, label, target_document_id, target_url, sort_order
FROM openfaithmap.content_site_nav_items
WHERE site_id = sqlc.arg('site_id')
ORDER BY sort_order ASC;
