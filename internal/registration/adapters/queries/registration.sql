-- name: InsertRegistrationRequest :one
INSERT INTO openfaithmap.registration_requests
	(submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	 street, house_number, postal_code, latitude, longitude)
VALUES (sqlc.arg('submitted_by_person_id'), sqlc.arg('taxon_id'), sqlc.arg('congregation_name'), sqlc.arg('country_id'),
	sqlc.narg('admin_area1'), sqlc.narg('locality'), sqlc.narg('street'), sqlc.narg('house_number'), sqlc.narg('postal_code'),
	sqlc.arg('latitude'), sqlc.arg('longitude'))
RETURNING id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at;

-- name: GetRegistrationRequest :one
SELECT id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at
FROM openfaithmap.registration_requests WHERE id = sqlc.arg('id');

-- name: ListRegistrationRequests :many
SELECT id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at
FROM openfaithmap.registration_requests
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC LIMIT sqlc.arg('page_size');

-- name: ListRegistrationRequestsBySubmitter :many
SELECT id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at
FROM openfaithmap.registration_requests
WHERE submitted_by_person_id = sqlc.arg('submitted_by_person_id')
ORDER BY created_at DESC LIMIT sqlc.arg('page_size');

-- name: MarkRegistrationRequestProvisioning :one
UPDATE openfaithmap.registration_requests
SET status = 'PROVISIONING', decided_by_person_id = sqlc.arg('decided_by_person_id'),
	created_unit_id = sqlc.arg('created_unit_id'), jurisdiction_unit_id = sqlc.narg('jurisdiction_unit_id')
WHERE id = sqlc.arg('id') AND status = 'PENDING'
RETURNING id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at;

-- name: ApproveRegistrationRequest :one
UPDATE openfaithmap.registration_requests
SET status = 'APPROVED', decided_by_person_id = sqlc.arg('decided_by_person_id'), decided_at = sqlc.arg('decided_at'),
	created_unit_id = sqlc.arg('created_unit_id')
WHERE id = sqlc.arg('id') AND status IN ('PENDING', 'PROVISIONING')
RETURNING id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at;

-- name: RejectRegistrationRequest :one
UPDATE openfaithmap.registration_requests
SET status = 'REJECTED', decided_by_person_id = sqlc.arg('decided_by_person_id'), decided_at = sqlc.arg('decided_at'),
	rejection_reason = sqlc.arg('rejection_reason')
WHERE id = sqlc.arg('id') AND status = 'PENDING'
RETURNING id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at;
