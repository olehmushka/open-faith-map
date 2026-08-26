-- 0021_discovery_site_enrichment — M13.0. The public discovery search (internal/religion's
-- SearchSites, exposed via DiscoveryPublicService) has always returned only a coordinate and a
-- site-type label — no congregation name, no address, no tradition/service tags. Both a name
-- (directory_units.name) and a structured address (location_locations' own columns) already exist
-- and are already joined into this same query for other purposes (docs/modules/discovery.md,
-- internal/religion/adapters/repository.go's siteFrom); this migration adds the one genuinely new
-- column, religion_sites.attributes, and the discovery_site_cache mirror needed to keep serving
-- enriched results from the cache without a live call on every hit.
--
-- attributes is deliberately a single JSONB column, not individual boolean columns per criterion
-- (scoped with the user): accessibility is modeled as a set of specific, named criteria rather than
-- one flag, and the exact criteria list is expected to grow — a JSONB document with a GIN index
-- avoids a migration per new criterion. Shape (enforced in Go, internal/religion/domain.SiteAttributes,
-- not a DB CHECK — matches this repo's existing convention of validating structured JSONB in the
-- application layer, e.g. directory_units.metadata):
--   {"accessibility": {"stepFreeEntrance": bool, "accessibleRestroom": bool, "hearingLoop": bool,
--     "signLanguageInterpretation": bool, "accessibleParking": bool, "wheelchairSeating": bool,
--     "brailleOrLargePrint": bool}, "onlineStream": bool}

ALTER TABLE openfaithmap.religion_sites
  ADD COLUMN attributes jsonb NOT NULL DEFAULT '{}';

CREATE INDEX religion_sites_attributes_gin
  ON openfaithmap.religion_sites USING gin (attributes jsonb_path_ops)
  WHERE deleted_at IS NULL;

-- discovery_site_cache mirrors the enriched projection so a cache hit never needs a live re-fetch
-- just to render a name. Expand-only, no backfill: the cache is disposable (docs/modules/
-- discovery.md's own invariant — "a stale row is simply overwritten... never audited") and every
-- existing row already gets overwritten wholesale by UpsertRow on its next refresh, lazy or
-- operator-forced.
ALTER TABLE openfaithmap.discovery_site_cache
  ADD COLUMN name                 text NOT NULL DEFAULT '',
  ADD COLUMN address_line         text,
  ADD COLUMN tradition_taxon_code text,
  ADD COLUMN tradition_taxon_name text,
  ADD COLUMN attributes           jsonb NOT NULL DEFAULT '{}';
