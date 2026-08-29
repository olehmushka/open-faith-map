-- 0027_content_nav_items — M14.10 (docs/modules/content.md). A hand-built site navigation menu,
-- deliberately independent of content_documents.parent_document_id (M14.0 replaced the original
-- page-tree-derived-nav assumption with a curated menu — parent_document_id still governs page
-- nesting/breadcrumbs, just not the nav menu itself).
--
-- No soft-delete, no updated_at/set_updated_at trigger: mirrors content_document_revisions'
-- (migrations/0025) own precedent for a table whose rows are never mutated in place —
-- ContentService.putNavItems is a full replace (delete-then-insert in one transaction), so a
-- soft-deleted row would never be resurrected and would only accumulate as unpruned garbage on
-- every save.
--
-- target_document_id/target_url: exactly one is ever set (a nav item points at an internal page OR
-- an external URL, never both, never neither) — enforceable as a same-table CHECK, unlike the
-- "target_document_id must be a PAGE in this same site" rule, which needs a cross-table read and
-- stays application-only (internal/content/application/service.go's PutNavItems, same shape as
-- checkParentDepth's own cross-table, app-only 3-level-cap check).
--
-- ON DELETE CASCADE on target_document_id (not SET NULL): SET NULL would leave both target columns
-- null, which violates the CHECK below and would abort the delete outright rather than silently
-- leave a broken link — CASCADE removes the now-pointless nav item row instead. No document-delete
-- endpoint exists yet, so this is dormant today; it's the correct shape for when one lands.

CREATE TABLE openfaithmap.content_site_nav_items (
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  site_id             uuid NOT NULL REFERENCES openfaithmap.content_sites (id) ON DELETE CASCADE,
  label               text NOT NULL,
  target_document_id  uuid REFERENCES openfaithmap.content_documents (id) ON DELETE CASCADE,
  target_url          text,
  sort_order          integer NOT NULL,
  created_at          timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT content_site_nav_items_target_xor CHECK (
    (target_document_id IS NOT NULL) <> (target_url IS NOT NULL)
  )
);

CREATE UNIQUE INDEX content_site_nav_items_sort_order_idx
  ON openfaithmap.content_site_nav_items (site_id, sort_order);
CREATE INDEX content_site_nav_items_target_document_idx
  ON openfaithmap.content_site_nav_items (target_document_id);
