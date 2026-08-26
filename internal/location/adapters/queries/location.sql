-- name: InsertLocation :one
INSERT INTO openfaithmap.location_locations
	(geom, mgrs, source_coordinate, country_id, admin_area_1, admin_area_2, locality, street,
	 house_number, postal_code, raw_address, type_id)
VALUES (
	ST_SetSRID(ST_MakePoint(sqlc.arg('longitude')::double precision, sqlc.arg('latitude')::double precision), 4326)::geography,
	NULL, sqlc.arg('source_coordinate'), sqlc.arg('country_id'), sqlc.narg('admin_area_1'), sqlc.narg('admin_area_2'),
	sqlc.narg('locality'), sqlc.narg('street'), sqlc.narg('house_number'), sqlc.narg('postal_code'),
	sqlc.narg('raw_address'), sqlc.narg('type_id'))
RETURNING id, ST_Y(geom::geometry)::double precision AS latitude, ST_X(geom::geometry)::double precision AS longitude,
	country_id, coalesce(admin_area_1,'')::text AS admin_area_1, coalesce(admin_area_2,'')::text AS admin_area_2,
	coalesce(locality,'')::text AS locality, coalesce(street,'')::text AS street,
	coalesce(house_number,'')::text AS house_number, coalesce(postal_code,'')::text AS postal_code,
	coalesce(raw_address,'')::text AS raw_address, coalesce(type_id::text,'')::text AS type_id, created_at, updated_at;

-- name: GetLocation :one
SELECT id, ST_Y(geom::geometry)::double precision AS latitude, ST_X(geom::geometry)::double precision AS longitude,
	country_id, coalesce(admin_area_1,'')::text AS admin_area_1, coalesce(admin_area_2,'')::text AS admin_area_2,
	coalesce(locality,'')::text AS locality, coalesce(street,'')::text AS street,
	coalesce(house_number,'')::text AS house_number, coalesce(postal_code,'')::text AS postal_code,
	coalesce(raw_address,'')::text AS raw_address, coalesce(type_id::text,'')::text AS type_id, created_at, updated_at
FROM openfaithmap.location_locations WHERE id = sqlc.arg('id') AND deleted_at IS NULL;
