-- name: InsertVouch :one
INSERT INTO openfaithmap.vouching_edges (guarantor_person_rid, claimant_person_rid, congregation_unit_rid, statement)
VALUES (sqlc.arg('guarantor_person_rid'), sqlc.arg('claimant_person_rid'), sqlc.arg('congregation_unit_rid'), sqlc.narg('statement'))
RETURNING id, guarantor_person_rid, claimant_person_rid, congregation_unit_rid, statement, created_at;

-- name: ListVouches :many
-- Consolidates the hand-written store's four claimant/congregation filter-combination branches into
-- one static "narg IS NULL OR column = narg" statement — same idiom as auditlog's ListAuditEntries.
SELECT id, guarantor_person_rid, claimant_person_rid, congregation_unit_rid, statement, created_at
FROM openfaithmap.vouching_edges
WHERE (sqlc.narg('claimant_person_rid')::text IS NULL OR claimant_person_rid = sqlc.narg('claimant_person_rid')::text)
	AND (sqlc.narg('congregation_unit_rid')::text IS NULL OR congregation_unit_rid = sqlc.narg('congregation_unit_rid')::text)
ORDER BY created_at DESC
LIMIT sqlc.arg('page_size');

-- name: ListVouchesByGuarantor :many
SELECT id, guarantor_person_rid, claimant_person_rid, congregation_unit_rid, statement, created_at
FROM openfaithmap.vouching_edges
WHERE guarantor_person_rid = sqlc.arg('guarantor_person_rid')
ORDER BY created_at ASC;

-- name: GetGuarantorStatus :one
SELECT guarantor_person_rid, status, revoked_at, revoked_reason, revoked_by_person_rid, updated_at
FROM openfaithmap.vouching_guarantor_status WHERE guarantor_person_rid = sqlc.arg('guarantor_person_rid');

-- name: UpsertRevokedGuarantor :one
INSERT INTO openfaithmap.vouching_guarantor_status (guarantor_person_rid, status, revoked_at, revoked_reason, revoked_by_person_rid)
VALUES (sqlc.arg('guarantor_person_rid'), 'revoked', now(), sqlc.arg('revoked_reason'), sqlc.arg('revoked_by_person_rid'))
ON CONFLICT (guarantor_person_rid) DO UPDATE SET
	status = 'revoked', revoked_at = now(), revoked_reason = sqlc.arg('revoked_reason'), revoked_by_person_rid = sqlc.arg('revoked_by_person_rid')
RETURNING guarantor_person_rid, status, revoked_at, revoked_reason, revoked_by_person_rid, updated_at;
