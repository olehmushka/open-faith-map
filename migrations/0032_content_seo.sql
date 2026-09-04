-- 0032_content_seo — M14.17 (docs/modules/content.md). Adds the two optional per-document SEO
-- overrides the public renderer's generateMetadata needs: an explicit meta title and meta
-- description. Both are nullable — when unset, the renderer derives a fallback from the
-- document's own blocks (first heading / first text-bearing block) at read time, exactly like
-- content_sites.logo_url is an optional override rather than a required field. No backfill: a
-- null value here is a real, meaningful "use the derived fallback" state, not missing data.

ALTER TABLE openfaithmap.content_documents
  ADD COLUMN meta_title       text,
  ADD COLUMN meta_description text;
