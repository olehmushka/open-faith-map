-- 0023_content_media_urls — M14.3. A normalizer for known share-link hosts (Google Drive,
-- Dropbox, the long-form OneDrive URL) runs at write time in
-- internal/content/application/medianormalize.go, called from Service.PutBlocks before
-- validateBlockData. A Drive/Dropbox share link is an HTML viewer page, not an image — pasted
-- into an image block it renders nothing, with no feedback anywhere (D-ExternalMediaOnly, U15).
-- The pre-normalization URL is preserved in a new optional `originalUrl` field alongside the
-- normalized `url`, per D-ExternalMediaOnly's own consequence line ("mitigated, not eliminated,
-- by M14.3 preserving the original URL alongside a normalized one") and DS-OFM-17 — a future
-- normalizer fix is a re-derivation, not a data-loss event.
--
-- OneDrive's short "1drv.ms" links are not normalized: resolving them requires following a
-- redirect, i.e. a server-side fetch of an admin-supplied URL — the SSRF surface this arc
-- otherwise never introduces. Only the already-long-form onedrive.live.com/redir?... URL is
-- rewritten by pure string substitution.
--
-- `alt` becomes schema-required on image and each gallery image, structurally enforced rather
-- than requested — the only version of alt text that survives contact with real editors.
--
-- Expand-and-data migration in one file, per this repo's convention: the schema loosens (adds
-- originalUrl) and tightens (alt required) in the same UPDATE, and a data migration backfills
-- alt='' for any existing image/gallery rows missing it, so the new required constraint never
-- rejects a row already in the table. No such rows exist today — the product has no live
-- congregation content yet (see migrations/0022_content_richtext.sql's own note) — so this is a
-- no-op in practice.
--
-- Known, deliberately-accepted gap, mirroring 0022's own: nested image/gallery blocks under a
-- "columns" block's data.columns[].blocks[] bypass content_block_types.json_schema entirely and
-- are neither normalized nor alt-required here.

UPDATE openfaithmap.content_block_types SET json_schema = '{"type":"object","required":["url","alt"],"additionalProperties":false,"properties":{"url":{"type":"string","format":"uri"},"originalUrl":{"type":"string","format":"uri"},"alt":{"type":"string","minLength":1},"caption":{"type":"string"}}}'
WHERE code = 'image';

UPDATE openfaithmap.content_block_types SET json_schema = '{"type":"object","required":["images"],"additionalProperties":false,"properties":{"images":{"type":"array","minItems":1,"items":{"type":"object","required":["url","alt"],"additionalProperties":false,"properties":{"url":{"type":"string","format":"uri"},"originalUrl":{"type":"string","format":"uri"},"alt":{"type":"string","minLength":1}}}}}}'
WHERE code = 'gallery';

-- Data migration: backfill alt='' for any existing image rows missing it (defensive; no known
-- rows exist).
UPDATE openfaithmap.content_blocks cb
SET data = jsonb_set(cb.data, '{alt}', '""'::jsonb)
FROM openfaithmap.content_block_types bt
WHERE cb.block_type_id = bt.id
  AND bt.code = 'image'
  AND NOT (cb.data ? 'alt');

-- Same backfill for each gallery image missing alt.
UPDATE openfaithmap.content_blocks cb
SET data = jsonb_set(
  cb.data, '{images}',
  (SELECT jsonb_agg(CASE WHEN img ? 'alt' THEN img ELSE img || '{"alt":""}'::jsonb END)
   FROM jsonb_array_elements(cb.data->'images') AS img)
)
FROM openfaithmap.content_block_types bt
WHERE cb.block_type_id = bt.id
  AND bt.code = 'gallery';
