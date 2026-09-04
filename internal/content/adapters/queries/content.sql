-- name: InsertSite :one
INSERT INTO openfaithmap.content_sites (congregation_unit_rid, slug)
VALUES (sqlc.arg('congregation_unit_rid'), sqlc.arg('slug'))
RETURNING id, congregation_unit_rid, slug, theme, logo_url, social_links, created_at, updated_at;

-- name: GetSiteByID :one
SELECT id, congregation_unit_rid, slug, theme, logo_url, social_links, created_at, updated_at
FROM openfaithmap.content_sites WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: GetSiteByUnit :one
SELECT id, congregation_unit_rid, slug, theme, logo_url, social_links, created_at, updated_at
FROM openfaithmap.content_sites WHERE congregation_unit_rid = sqlc.arg('congregation_unit_rid') AND deleted_at IS NULL;

-- name: GetSiteBySlug :one
SELECT id, congregation_unit_rid, slug, theme, logo_url, social_links, created_at, updated_at
FROM openfaithmap.content_sites WHERE slug = sqlc.arg('slug') AND deleted_at IS NULL;

-- name: UpdateSiteTheme :one
UPDATE openfaithmap.content_sites SET theme = sqlc.arg('theme')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, congregation_unit_rid, slug, theme, logo_url, social_links, created_at, updated_at;

-- name: UpdateSiteChrome :one
-- M14.11: logo_url/social_links are content_sites' own site-level settings (never a content
-- document) — everything else the header/footer needs (congregation name, address, service
-- schedule) is composed at read time from religion_sites/religion_service_schedules, never stored
-- here.
UPDATE openfaithmap.content_sites SET logo_url = sqlc.narg('logo_url'), social_links = sqlc.arg('social_links')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, congregation_unit_rid, slug, theme, logo_url, social_links, created_at, updated_at;

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
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description;

-- name: GetDocument :one
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description
FROM openfaithmap.content_documents WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: UpdateDocument :one
-- meta_title/meta_description: nil (narg unset) leaves the column unchanged; an empty string is a
-- real value (clears the override back to the renderer's derived fallback), so this is COALESCE on
-- the arg itself, same as slug — never on the empty string.
UPDATE openfaithmap.content_documents
SET slug = COALESCE(sqlc.narg('slug'), slug),
    parent_document_id = CASE
      WHEN sqlc.arg('clear_parent')::bool THEN NULL
      WHEN sqlc.narg('parent_document_id')::uuid IS NOT NULL THEN sqlc.narg('parent_document_id')::uuid
      ELSE parent_document_id
    END,
    meta_title = COALESCE(sqlc.narg('meta_title'), meta_title),
    meta_description = COALESCE(sqlc.narg('meta_description'), meta_description)
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description;

-- name: UpdateDocumentState :one
-- publish_at is set unconditionally (not COALESCE): every transition other than SCHEDULE passes
-- NULL, so leaving SCHEDULED by any path clears a stale future date (M14.15).
UPDATE openfaithmap.content_documents
SET state = sqlc.arg('state'), publish_at = sqlc.narg('publish_at'),
    published_at = CASE WHEN sqlc.arg('first_publish')::bool THEN now() ELSE published_at END
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description;

-- name: ListDocuments :many
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND deleted_at IS NULL
	AND (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind')::text)
	AND (sqlc.narg('locale')::text IS NULL OR locale = sqlc.narg('locale')::text)
	AND (sqlc.narg('state')::text IS NULL OR state = sqlc.narg('state')::text)
ORDER BY created_at DESC;

-- name: ListPublicDocuments :many
-- M14.15/D-PublishOnRead: a SCHEDULED document whose publish_at has passed is publicly listed with
-- no job ever having flipped its state — correctness lives entirely in this WHERE clause.
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND deleted_at IS NULL
	AND (state IN ('PUBLISHED', 'UNLISTED') OR (state = 'SCHEDULED' AND publish_at <= now()))
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
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description
FROM openfaithmap.content_documents
WHERE site_id = sqlc.arg('site_id') AND kind = sqlc.arg('kind') AND locale = sqlc.arg('locale')
	AND slug = sqlc.arg('slug') AND deleted_at IS NULL;

-- name: ListDocumentsByTranslationGroup :many
-- M14.14: every document sharing one translation_group_id, any state, any site — deliberately not
-- scoped by site_id, so CreateDocument's cross-site guard (application.Service) can tell "brand new
-- group" apart from "this group belongs to a different site" by inspecting the returned rows'
-- own site_id, rather than the query silently filtering a cross-site collision down to zero rows.
SELECT id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at, draft_revision_id, published_revision_id, publish_at, meta_title, meta_description
FROM openfaithmap.content_documents
WHERE translation_group_id = sqlc.arg('translation_group_id') AND deleted_at IS NULL
ORDER BY created_at ASC;

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

-- ---- block-type catalog admin (M14.13, content.catalog.manage) ----

-- name: GetBlockTypeByID :one
SELECT id, code, name, json_schema, ui_schema, status, sort_order
FROM openfaithmap.content_block_types WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: ListAllBlockTypes :many
-- Every status, unlike ListActiveBlockTypes — the moderator catalog page needs to see/edit
-- RETIRED types too.
SELECT id, code, name, json_schema, ui_schema, status, sort_order
FROM openfaithmap.content_block_types
WHERE deleted_at IS NULL
ORDER BY sort_order ASC;

-- name: InsertBlockType :one
INSERT INTO openfaithmap.content_block_types (code, name, json_schema, ui_schema, sort_order)
VALUES (sqlc.arg('code'), sqlc.arg('name'), sqlc.arg('json_schema'), sqlc.arg('ui_schema'), sqlc.arg('sort_order'))
RETURNING id, code, name, json_schema, ui_schema, status, sort_order;

-- name: UpdateBlockType :one
-- json_schema/ui_schema are deliberately not settable here (owner decision, M14.13): locked after
-- creation, so a runtime catalog edit can never silently break already-saved blocks or the admin
-- form for an existing type.
UPDATE openfaithmap.content_block_types
SET name = COALESCE(sqlc.narg('name'), name),
    status = COALESCE(sqlc.narg('status'), status),
    sort_order = COALESCE(sqlc.narg('sort_order'), sort_order)
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, code, name, json_schema, ui_schema, status, sort_order;

-- ---- patterns (M14.13, D-SitePatterns) ----

-- name: InsertPattern :one
INSERT INTO openfaithmap.content_patterns (name, description, blocks, sort_order)
VALUES (sqlc.arg('name'), sqlc.arg('description'), sqlc.arg('blocks'), sqlc.arg('sort_order'))
RETURNING id, name, description, blocks, sort_order, created_at, updated_at;

-- name: UpdatePattern :one
UPDATE openfaithmap.content_patterns
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    blocks = COALESCE(sqlc.narg('blocks')::jsonb, blocks),
    sort_order = COALESCE(sqlc.narg('sort_order'), sort_order)
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, name, description, blocks, sort_order, created_at, updated_at;

-- name: DeletePattern :execrows
UPDATE openfaithmap.content_patterns SET deleted_at = now()
WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: ListPatterns :many
SELECT id, name, description, blocks, sort_order, created_at, updated_at
FROM openfaithmap.content_patterns
WHERE deleted_at IS NULL
ORDER BY sort_order ASC;

-- ---- form submissions (M14.16, D-InAppInbox) ----

-- name: InsertFormSubmission :one
INSERT INTO openfaithmap.content_form_submissions (site_id, name, email, message)
VALUES (sqlc.arg('site_id'), sqlc.narg('name'), sqlc.narg('email'), sqlc.arg('message'))
RETURNING id, site_id, name, email, message, created_at;

-- name: ListFormSubmissionsBySite :many
SELECT id, site_id, name, email, message, created_at
FROM openfaithmap.content_form_submissions
WHERE site_id = sqlc.arg('site_id')
ORDER BY created_at DESC;
