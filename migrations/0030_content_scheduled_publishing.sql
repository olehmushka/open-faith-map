-- 0030_content_scheduled_publishing — M14.15 (docs/architecture/decisions.md#d-publishonread).
-- Scheduled publishing decided entirely in the read predicate, not by anything that fires: adds
-- publish_at and a new SCHEDULED state. No timer, no cron, no goroutine ever flips a row from
-- SCHEDULED to PUBLISHED — the public read predicate becomes
-- state = 'PUBLISHED' OR (state = 'SCHEDULED' AND publish_at <= now()), so a document's stored
-- state can legitimately lag reality forever; only an explicit later transition (Publish/Unlist/
-- Revert-to-Draft, taken against the document's *effective* state) ever settles the column.
--
-- published_at (existing, "when this document first actually went live") is deliberately left
-- alone by scheduling: it is stamped only by a real transition into PUBLISHED, never by SCHEDULE
-- itself, so a document that only ever becomes due without further action keeps published_at NULL
-- — the one real cost D-PublishOnRead names.

ALTER TABLE openfaithmap.content_documents
  ADD COLUMN publish_at timestamptz;

ALTER TABLE openfaithmap.content_documents
  DROP CONSTRAINT content_documents_state_check;

ALTER TABLE openfaithmap.content_documents
  ADD CONSTRAINT content_documents_state_check CHECK (state IN ('DRAFT', 'PUBLISHED', 'UNLISTED', 'SCHEDULED'));
