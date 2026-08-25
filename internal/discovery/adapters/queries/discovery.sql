-- name: UpsertCacheRow :one
INSERT INTO openfaithmap.discovery_site_cache
	(religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude,
	 tradition_taxon_id, service_languages, service_days, refreshed_at)
VALUES (sqlc.arg('religion_site_rid'), sqlc.arg('congregation_unit_rid'), sqlc.narg('content_site_id'),
	sqlc.narg('latitude'), sqlc.narg('longitude'), sqlc.narg('tradition_taxon_id'),
	sqlc.arg('service_languages'), sqlc.arg('service_days'), now())
ON CONFLICT (religion_site_rid) DO UPDATE SET
	congregation_unit_rid = EXCLUDED.congregation_unit_rid,
	content_site_id       = EXCLUDED.content_site_id,
	latitude              = EXCLUDED.latitude,
	longitude             = EXCLUDED.longitude,
	tradition_taxon_id    = EXCLUDED.tradition_taxon_id,
	service_languages     = EXCLUDED.service_languages,
	service_days          = EXCLUDED.service_days,
	refreshed_at          = now()
RETURNING id, religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude,
	tradition_taxon_id, service_languages, service_days, refreshed_at;

-- name: SearchAllCacheRows :many
SELECT id, religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude,
	tradition_taxon_id, service_languages, service_days, refreshed_at
FROM openfaithmap.discovery_site_cache;
