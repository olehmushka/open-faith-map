-- name: UpsertCacheRow :one
INSERT INTO openfaithmap.discovery_site_cache
	(religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude,
	 name, address_line, tradition_taxon_id, tradition_taxon_code, tradition_taxon_name,
	 service_languages, service_days, attributes, refreshed_at)
VALUES (sqlc.arg('religion_site_rid'), sqlc.arg('congregation_unit_rid'), sqlc.narg('content_site_id'),
	sqlc.narg('latitude'), sqlc.narg('longitude'),
	sqlc.arg('name'), sqlc.narg('address_line'), sqlc.narg('tradition_taxon_id'),
	sqlc.narg('tradition_taxon_code'), sqlc.narg('tradition_taxon_name'),
	sqlc.arg('service_languages'), sqlc.arg('service_days'), sqlc.arg('attributes'), now())
ON CONFLICT (religion_site_rid) DO UPDATE SET
	congregation_unit_rid = EXCLUDED.congregation_unit_rid,
	content_site_id       = EXCLUDED.content_site_id,
	latitude              = EXCLUDED.latitude,
	longitude             = EXCLUDED.longitude,
	name                  = EXCLUDED.name,
	address_line          = EXCLUDED.address_line,
	tradition_taxon_id    = EXCLUDED.tradition_taxon_id,
	tradition_taxon_code  = EXCLUDED.tradition_taxon_code,
	tradition_taxon_name  = EXCLUDED.tradition_taxon_name,
	service_languages     = EXCLUDED.service_languages,
	service_days          = EXCLUDED.service_days,
	attributes            = EXCLUDED.attributes,
	refreshed_at          = now()
RETURNING id, religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude,
	name, address_line, tradition_taxon_id, tradition_taxon_code, tradition_taxon_name,
	service_languages, service_days, attributes, refreshed_at;

-- name: SearchAllCacheRows :many
SELECT id, religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude,
	name, address_line, tradition_taxon_id, tradition_taxon_code, tradition_taxon_name,
	service_languages, service_days, attributes, refreshed_at
FROM openfaithmap.discovery_site_cache;
