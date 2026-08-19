-- 0003_discovery — M4 (docs/modules/discovery.md). discovery_site_cache: a disposable,
-- read-through cache of go-oikumenea's religion discovery search, refreshed lazily on a cache
-- miss (no scheduled job — DS-OFM-2 resolved for MVP). religion_site_rid/congregation_unit_rid are
-- opaque go-oikumenea RIDs (TEXT, no cross-schema FK — conventions.md). content_site_id IS a real
-- in-schema FK (DS-OFM-13 resolved: both tables live in one schema/deployable, matching M3's own
-- content_documents/content_blocks precedent) — ON DELETE SET NULL, since the cache row is still
-- valid go-oikumenea data even without a published site to link to.
--
-- No soft-delete and no set_updated_at trigger: this table is a cache, not a record — a stale row
-- is overwritten wholesale on refresh (refreshed_at tracks freshness), never partially updated or
-- audited.

CREATE TABLE openfaithmap.discovery_site_cache (
  id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  religion_site_rid      text NOT NULL,
  congregation_unit_rid  text NOT NULL,
  content_site_id        uuid REFERENCES openfaithmap.content_sites (id) ON DELETE SET NULL,
  latitude               numeric,
  longitude              numeric,
  tradition_taxon_id     text,
  service_languages      text[] NOT NULL DEFAULT '{}',
  service_days           smallint[] NOT NULL DEFAULT '{}',
  refreshed_at           timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX discovery_site_cache_religion_site_rid_idx
  ON openfaithmap.discovery_site_cache (religion_site_rid);
CREATE INDEX discovery_site_cache_tradition_idx
  ON openfaithmap.discovery_site_cache (tradition_taxon_id);
CREATE INDEX discovery_site_cache_content_site_idx
  ON openfaithmap.discovery_site_cache (content_site_id);
