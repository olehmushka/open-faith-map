-- 0028_content_site_chrome — M14.11 (docs/modules/content.md). Adds the two content_sites-owned
-- fields the new site-chrome header/footer needs: a logo URL and a small fixed set of social
-- links. Everything else the header/footer shows (congregation name, address, service schedule) is
-- read live from religion_sites/religion_service_schedules at request time — never copied here,
-- the existing content.md invariant M14.11 restates.
--
-- social_links is a JSONB object with a small, named field set (facebook/instagram/youtube/
-- twitter/website — all optional strings), not a free-form map, so the frontend can render a known
-- icon per field deterministically. Shaped like content_sites.theme's own "NOT NULL DEFAULT '{}'"
-- precedent rather than a rigid per-provider column set, since it's edited wholesale by one admin
-- form the same way theme already is.

ALTER TABLE openfaithmap.content_sites
  ADD COLUMN logo_url     text,
  ADD COLUMN social_links jsonb NOT NULL DEFAULT '{}';
