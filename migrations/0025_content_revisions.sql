-- 0025_content_revisions — M14.6 (docs/modules/content.md, D-ContentRevisions). Forward revisions:
-- editing a live page must never touch what visitors see. content_document_revisions.data is a
-- full blocks snapshot (an ordered JSON array of {blockTypeCode,position,data}, the same shape
-- Service.PutBlocks already validates), not a second copy of content_blocks rows.
--
-- Two roles for a row in this table, distinguished only by which pointer on content_documents
-- references it — no separate "kind" column:
--   * the DRAFT row (content_documents.draft_revision_id) is exactly one per document, created
--     alongside it, and mutated in place (UPDATE ... SET data) by every autosave/manual save —
--     this is what GetBlocks/PutBlocks operate on from here on, replacing content_blocks as their
--     backing store.
--   * CHECKPOINT rows are immutable snapshots created only on Publish, copying the draft's data at
--     that moment; content_documents.published_revision_id points at the most recent one.
--     ContentPublicService reads through this pointer, completely decoupled from the draft, so
--     further edits/autosaves never change what's live until the next explicit publish.
-- History listing excludes the draft row (id != draft_revision_id) and shows checkpoints only,
-- newest first. Pruning keeps the 50 most recent checkpoints per document (owner decision,
-- 2026-08-28 — a config change away from "keep all" once storage cost is no longer a concern).
--
-- content_blocks becomes unused as of this migration (superseded, not dropped — matches this
-- module's additive/expand-only migration history at 0022/0023/0024; nothing else in the codebase
-- reads it, per a repo-wide grep during planning, so leaving it costs nothing but a little disk).
-- Its data is migrated forward into each existing document's initial draft/published revision
-- below. Per 0022's/0023's own note, there is no live congregation content in any environment this
-- migration runs against yet, so the backfill below is a correctness guarantee, not a real-data
-- concern.
--
-- draft_revision_id/published_revision_id are deliberately NOT NOT NULL at the DB level: the two
-- tables reference each other (a revision needs its document's id; a document's draft pointer
-- needs the revision's id), so a new document is created in three steps inside one transaction
-- (insert document, insert its initial revision, point draft_revision_id at it) — the same
-- app-enforced-not-DB-constraint shape this module already uses for the 3-level parent-depth cap
-- (content.md's own invariants section). No caller ever observes a document with a null
-- draft_revision_id in practice; application.Service guarantees it's set before the creating
-- transaction commits.

CREATE TABLE openfaithmap.content_document_revisions (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id       uuid NOT NULL REFERENCES openfaithmap.content_documents (id) ON DELETE CASCADE,
  revision_no       integer NOT NULL,
  data              jsonb NOT NULL,
  author_person_id  text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  label             text
);

CREATE UNIQUE INDEX content_document_revisions_doc_no_idx
  ON openfaithmap.content_document_revisions (document_id, revision_no);

ALTER TABLE openfaithmap.content_documents
  ADD COLUMN draft_revision_id     uuid REFERENCES openfaithmap.content_document_revisions (id),
  ADD COLUMN published_revision_id uuid REFERENCES openfaithmap.content_document_revisions (id);

-- Data migration: one revision per existing document, aggregating its current content_blocks
-- (ordered by position) into the new snapshot shape. draft_revision_id is always set;
-- published_revision_id only for documents already PUBLISHED, so GetPublicBlocks's new
-- published-revision read path sees exactly the same documents it could read before this
-- migration (DRAFT/UNLISTED documents keep no published_revision_id, matching the pre-M14.6
-- "state != DRAFT" gate for PUBLISHED and the still-forbidden-to-editors-only UNLISTED case
-- identically).
WITH snapshot AS (
  SELECT
    d.id AS document_id,
    COALESCE(
      (SELECT jsonb_agg(jsonb_build_object('blockTypeCode', bt.code, 'position', b.position, 'data', b.data) ORDER BY b.position)
       FROM openfaithmap.content_blocks b
       JOIN openfaithmap.content_block_types bt ON bt.id = b.block_type_id
       WHERE b.document_id = d.id AND b.deleted_at IS NULL),
      '[]'::jsonb
    ) AS blocks,
    d.state
  FROM openfaithmap.content_documents d
  WHERE d.deleted_at IS NULL
), inserted AS (
  INSERT INTO openfaithmap.content_document_revisions (document_id, revision_no, data)
  SELECT document_id, 1, blocks FROM snapshot
  RETURNING id, document_id
)
UPDATE openfaithmap.content_documents doc
SET draft_revision_id = inserted.id,
    published_revision_id = CASE WHEN snapshot.state = 'PUBLISHED' THEN inserted.id ELSE NULL END
FROM inserted
JOIN snapshot ON snapshot.document_id = inserted.document_id
WHERE doc.id = inserted.document_id;

COMMENT ON TABLE openfaithmap.content_blocks IS
  'Superseded by content_document_revisions as of M14.6 — no longer read or written. Left in place rather than dropped (data already migrated forward); a candidate for a future cleanup migration.';
