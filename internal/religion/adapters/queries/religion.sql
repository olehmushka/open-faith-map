-- name: GetTaxon :one
SELECT t.id, t.parent_id, t.rank_id, rk.code AS rank_code, t.code, t.name, t.sort_order
FROM openfaithmap.religion_taxa t
JOIN openfaithmap.religion_taxon_ranks rk ON rk.id = t.rank_id
WHERE t.id = sqlc.arg('id') AND t.deleted_at IS NULL;

-- name: ListTaxa :many
SELECT t.id, t.parent_id, t.rank_id, rk.code AS rank_code, t.code, t.name, t.sort_order
FROM openfaithmap.religion_taxa t
JOIN openfaithmap.religion_taxon_ranks rk ON rk.id = t.rank_id
WHERE t.deleted_at IS NULL
	AND (sqlc.arg('query')::text = '' OR t.code ILIKE '%' || sqlc.arg('query')::text || '%' OR t.name ILIKE '%' || sqlc.arg('query')::text || '%')
ORDER BY t.sort_order NULLS LAST, t.code
LIMIT sqlc.arg('limit_count');

-- name: GetOrgProfileRow :one
SELECT unit_id, org_kind_id, short_code, created_at, updated_at
FROM openfaithmap.religion_org_profiles WHERE unit_id = sqlc.arg('unit_id') AND deleted_at IS NULL;

-- name: UpsertOrgProfile :exec
INSERT INTO openfaithmap.religion_org_profiles (unit_id, org_kind_id, short_code)
VALUES (sqlc.arg('unit_id'), sqlc.narg('org_kind_id'), sqlc.narg('short_code'))
ON CONFLICT (unit_id) DO UPDATE SET
	org_kind_id = EXCLUDED.org_kind_id, short_code = EXCLUDED.short_code, deleted_at = NULL;

-- name: ListOrgClassifications :many
SELECT oc.id, oc.unit_id, oc.taxon_id, t.code AS taxon_code, t.name AS taxon_name, oc.is_primary, oc.created_at
FROM openfaithmap.religion_org_classifications oc
JOIN openfaithmap.religion_taxa t ON t.id = oc.taxon_id
WHERE oc.unit_id = sqlc.arg('unit_id') AND oc.deleted_at IS NULL
ORDER BY oc.is_primary DESC, t.code;

-- name: ClearPrimaryClassification :exec
UPDATE openfaithmap.religion_org_classifications
SET is_primary = false WHERE unit_id = sqlc.arg('unit_id') AND is_primary AND deleted_at IS NULL;

-- name: InsertOrgClassification :one
INSERT INTO openfaithmap.religion_org_classifications (unit_id, taxon_id, is_primary)
VALUES (sqlc.arg('unit_id'), sqlc.arg('taxon_id'), sqlc.arg('is_primary')) RETURNING id;

-- name: HasActivePolicy :one
SELECT count(*) FROM openfaithmap.religion_org_policies p
JOIN openfaithmap.religion_policy_kinds k ON k.id = p.policy_kind_id
WHERE p.unit_id = sqlc.arg('unit_id') AND k.code = sqlc.arg('policy_kind_code') AND p.deleted_at IS NULL;

-- name: ListSiteTypes :many
SELECT id, code, name FROM openfaithmap.religion_site_types
WHERE deleted_at IS NULL ORDER BY sort_order NULLS LAST, code;

-- name: ListOrgKinds :many
SELECT id, code, name FROM openfaithmap.religion_org_kinds
WHERE deleted_at IS NULL ORDER BY sort_order NULLS LAST, code;

-- name: ListSitesByUnit :many
SELECT s.id, s.org_unit_id, s.location_id, s.site_type_id, st.code AS site_type_code, st.name AS site_type_name,
	s.visibility, s.public_precision, s.is_primary, s.attributes,
	ST_Y(l.geom::geometry)::double precision AS latitude, ST_X(l.geom::geometry)::double precision AS longitude
FROM openfaithmap.religion_sites s
JOIN openfaithmap.religion_site_types st ON st.id = s.site_type_id
JOIN openfaithmap.location_locations l ON l.id = s.location_id
WHERE s.org_unit_id = sqlc.arg('unit_id') AND s.deleted_at IS NULL
ORDER BY s.is_primary DESC, s.id;

-- name: InsertSite :one
INSERT INTO openfaithmap.religion_sites (org_unit_id, location_id, site_type_id, is_primary)
VALUES (sqlc.arg('org_unit_id'), sqlc.arg('location_id'), sqlc.arg('site_type_id'), sqlc.arg('is_primary'))
RETURNING id;

-- name: UpdateSiteAttributes :exec
-- M13.2: the caller already holds the pre-fetched row (ListSitesByUnit resolved which site by
-- unit+is_primary), so this only needs to persist the new value, not RETURNING a full re-read.
UPDATE openfaithmap.religion_sites SET attributes = sqlc.arg('attributes')
WHERE id = sqlc.arg('id') AND deleted_at IS NULL;

-- name: GetSiteRow :one
SELECT s.id, s.org_unit_id, s.location_id, s.site_type_id, st.code AS site_type_code, st.name AS site_type_name,
	s.visibility, s.public_precision, s.is_primary, s.attributes,
	ST_Y(l.geom::geometry)::double precision AS latitude, ST_X(l.geom::geometry)::double precision AS longitude
FROM openfaithmap.religion_sites s
JOIN openfaithmap.religion_site_types st ON st.id = s.site_type_id
JOIN openfaithmap.location_locations l ON l.id = s.location_id
WHERE s.id = sqlc.arg('id');

-- SearchSites is deliberately NOT ported here: its WHERE/ORDER BY shape is genuinely dynamic
-- (radius search XOR bbox search XOR neither; four independently-optional EXISTS predicates), not a
-- fixed set of "narg IS NULL OR" branches — forcing it into one static statement would either lose
-- the mutually-exclusive ORDER BY behavior or require duplicating the query per filter combination.
-- It stays hand-written in repository.go against the same db.DBTX this package's Queries wraps,
-- reusing GetSiteRow's/ListSitesByUnit's siteCols/siteFrom shape for its own row-scan.
