-- name: IsActiveInstanceAdmin :one
SELECT EXISTS(
	SELECT 1 FROM openfaithmap.authz_instance_admins
	WHERE person_id = sqlc.arg('person_id') AND revoked_at IS NULL
);

-- name: HasActiveInstanceAdmin :one
SELECT EXISTS(SELECT 1 FROM openfaithmap.authz_instance_admins WHERE revoked_at IS NULL);

-- name: InsertInstanceAdmin :one
INSERT INTO openfaithmap.authz_instance_admins (person_id, granted_by)
VALUES (sqlc.arg('person_id'), sqlc.narg('granted_by'))
RETURNING id;

-- name: ListRoles :many
SELECT id, code, name, COALESCE(description, '') AS description, is_base
FROM openfaithmap.authz_roles
WHERE deleted_at IS NULL
ORDER BY name;

-- name: GetRoleByCode :one
SELECT id, code, name, COALESCE(description, '') AS description, is_base
FROM openfaithmap.authz_roles
WHERE code = sqlc.arg('code') AND deleted_at IS NULL;

-- name: ListRoleAssignmentsByUnit :many
SELECT a.id, a.subject_person_id, p.display_name, a.role_id, r.code AS role_code, a.target_unit_id, a.scope, a.granted_at, a.expires_at
FROM openfaithmap.authz_role_assignments a
JOIN openfaithmap.authz_roles r ON r.id = a.role_id
JOIN openfaithmap.identity_persons p ON p.id = a.subject_person_id
WHERE a.target_unit_id = sqlc.arg('target_unit_id') AND a.revoked_at IS NULL
ORDER BY a.granted_at DESC;

-- name: ListRoleAssignmentsByPerson :many
SELECT a.id, a.subject_person_id, p.display_name, a.role_id, r.code AS role_code, a.target_unit_id, a.scope, a.granted_at, a.expires_at
FROM openfaithmap.authz_role_assignments a
JOIN openfaithmap.authz_roles r ON r.id = a.role_id
JOIN openfaithmap.identity_persons p ON p.id = a.subject_person_id
WHERE a.subject_person_id = sqlc.arg('subject_person_id') AND a.revoked_at IS NULL
ORDER BY a.granted_at DESC;

-- name: RevokeRoleAssignment :one
UPDATE openfaithmap.authz_role_assignments
SET revoked_at = now(), revoked_by = sqlc.narg('revoked_by')
WHERE id = sqlc.arg('id') AND revoked_at IS NULL
RETURNING id, subject_person_id, role_id, target_unit_id, scope;

-- name: ClearRoleAssignmentExpiry :one
-- M12.3 — clears an active assignment's expires_at, leaving the grant itself untouched. RETURNING
-- shape matches RevokeRoleAssignment's own (id/subject/role/unit/scope) — identity for the audit
-- log, not the (now-cleared) expiry value itself.
UPDATE openfaithmap.authz_role_assignments
SET expires_at = NULL, updated_at = now()
WHERE id = sqlc.arg('id') AND revoked_at IS NULL
RETURNING id, subject_person_id, role_id, target_unit_id, scope;

-- name: ListInstanceAdmins :many
SELECT a.id, a.person_id, p.display_name, a.granted_at
FROM openfaithmap.authz_instance_admins a
JOIN openfaithmap.identity_persons p ON p.id = a.person_id
WHERE a.revoked_at IS NULL
ORDER BY a.granted_at DESC;

-- name: RevokeInstanceAdmin :one
UPDATE openfaithmap.authz_instance_admins
SET revoked_at = now(), revoked_by = sqlc.narg('revoked_by')
WHERE person_id = sqlc.arg('person_id') AND revoked_at IS NULL
RETURNING id, person_id;

-- name: ActiveGrantsForSubject :many
SELECT a.id, a.role_id, r.code AS role_code, a.target_unit_id, a.scope, COALESCE(a.graph_id::text, '') AS graph_id,
       COALESCE(g.code, '') AS graph_code, rp.permission_code
FROM openfaithmap.authz_role_assignments a
JOIN openfaithmap.authz_roles r ON r.id = a.role_id AND r.deleted_at IS NULL
JOIN openfaithmap.authz_role_permissions rp ON rp.role_id = a.role_id
LEFT JOIN openfaithmap.directory_graphs g ON g.id = a.graph_id
WHERE a.subject_person_id = sqlc.arg('subject_person_id')
  AND a.revoked_at IS NULL
  AND (a.expires_at IS NULL OR a.expires_at > now())
ORDER BY a.id;

-- name: InsertRoleAssignment :one
INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope, graph_id, granted_by, expires_at)
VALUES (sqlc.arg('subject_person_id'), sqlc.arg('role_id'), sqlc.arg('target_unit_id'), sqlc.arg('scope'), sqlc.narg('graph_id'), sqlc.narg('granted_by'), sqlc.narg('expires_at'))
RETURNING id;

-- name: GetActiveRoleAssignmentID :one
-- Idempotent-conflict fallback for InsertRoleAssignment: looked up after a 23505 on
-- authz_role_assignments_active_idx, whose partial-unique shape this predicate mirrors exactly.
SELECT id FROM openfaithmap.authz_role_assignments
WHERE subject_person_id = sqlc.arg('subject_person_id') AND role_id = sqlc.arg('role_id')
  AND target_unit_id = sqlc.arg('target_unit_id') AND scope = sqlc.arg('scope')
  AND graph_id IS NOT DISTINCT FROM sqlc.narg('graph_id') AND revoked_at IS NULL;

-- name: UpsertRoleAssignment :one
-- BulkInsertRoleAssignments' per-row statement, run in a loop inside the caller's own tx (see
-- repository.go/service.go: a caught 23505 would abort the whole tx, so this uses a real upsert
-- instead of InsertRoleAssignment's catch-then-select). ON CONFLICT target matches
-- authz_role_assignments_active_idx's partial-index predicate exactly (migrations/0009_core_authz.sql).
INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope, graph_id, granted_by, expires_at)
VALUES (sqlc.arg('subject_person_id'), sqlc.arg('role_id'), sqlc.arg('target_unit_id'), sqlc.arg('scope'), sqlc.narg('graph_id'), sqlc.narg('granted_by'), sqlc.narg('expires_at'))
ON CONFLICT (subject_person_id, role_id, target_unit_id, scope, graph_id) WHERE revoked_at IS NULL
DO UPDATE SET updated_at = now(), expires_at = EXCLUDED.expires_at
RETURNING id;

-- name: CountRepointableRoleAssignments :one
SELECT
	count(*) FILTER (WHERE NOT EXISTS (
		SELECT 1 FROM openfaithmap.authz_role_assignments s
		WHERE s.subject_person_id = sqlc.arg('survivor_id') AND s.role_id = ra.role_id AND s.target_unit_id = ra.target_unit_id
		  AND s.scope = ra.scope AND s.graph_id IS NOT DISTINCT FROM ra.graph_id AND s.revoked_at IS NULL
	))::bigint AS to_move,
	count(*) FILTER (WHERE EXISTS (
		SELECT 1 FROM openfaithmap.authz_role_assignments s
		WHERE s.subject_person_id = sqlc.arg('survivor_id') AND s.role_id = ra.role_id AND s.target_unit_id = ra.target_unit_id
		  AND s.scope = ra.scope AND s.graph_id IS NOT DISTINCT FROM ra.graph_id AND s.revoked_at IS NULL
	))::bigint AS to_revoke
FROM openfaithmap.authz_role_assignments ra
WHERE ra.subject_person_id = sqlc.arg('duplicate_id') AND ra.revoked_at IS NULL;

-- name: RepointMoveRoleAssignments :many
UPDATE openfaithmap.authz_role_assignments ra
SET subject_person_id = sqlc.arg('survivor_id'), updated_at = now()
WHERE ra.subject_person_id = sqlc.arg('duplicate_id') AND ra.revoked_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM openfaithmap.authz_role_assignments s
    WHERE s.subject_person_id = sqlc.arg('survivor_id') AND s.role_id = ra.role_id AND s.target_unit_id = ra.target_unit_id
      AND s.scope = ra.scope AND s.graph_id IS NOT DISTINCT FROM ra.graph_id AND s.revoked_at IS NULL
  )
RETURNING ra.id;

-- name: RepointRevokeRoleAssignments :many
UPDATE openfaithmap.authz_role_assignments
SET revoked_at = now(), revoked_by = sqlc.narg('revoked_by')
WHERE subject_person_id = sqlc.arg('duplicate_id') AND revoked_at IS NULL
RETURNING id;

-- name: RepointMoveInstanceAdmin :one
UPDATE openfaithmap.authz_instance_admins a
SET person_id = sqlc.arg('survivor_id'), updated_at = now()
WHERE a.person_id = sqlc.arg('duplicate_id') AND a.revoked_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM openfaithmap.authz_instance_admins s WHERE s.person_id = sqlc.arg('survivor_id') AND s.revoked_at IS NULL
  )
RETURNING a.id;

-- name: RepointRevokeInstanceAdmin :one
UPDATE openfaithmap.authz_instance_admins
SET revoked_at = now(), revoked_by = sqlc.narg('revoked_by')
WHERE person_id = sqlc.arg('duplicate_id') AND revoked_at IS NULL
RETURNING id;
