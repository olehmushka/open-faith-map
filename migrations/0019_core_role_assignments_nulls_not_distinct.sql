-- 0019_core_role_assignments_nulls_not_distinct — found live during M11.7 (docs/milestones.md)
-- testing, not anticipated by any prior plan.
--
-- authz_role_assignments_active_idx (0009_core_authz.sql) is a bare unique index on
-- (subject_person_id, role_id, target_unit_id, scope, graph_id) WHERE revoked_at IS NULL, with no
-- NULLS NOT DISTINCT clause. Postgres unique indexes treat NULL as distinct from NULL by default, and
-- graph_id is always NULL for scope='unit' (authz_role_assignments_graph_scope's own CHECK requires
-- it) — every scope='unit' grant, the vast majority of grants in this app, was therefore never
-- actually covered by this index's uniqueness guarantee. InsertRoleAssignment's own idempotent-
-- conflict fallback (catching a 23505 on this index, adapters/store.go) was consequently dead code:
-- repeated identical grants silently accumulated as duplicate active rows instead of being
-- deduplicated, with no functional PDP-decision impact (duplicate rows for the same person/role/unit
-- grant the same permissions either way) but a real, visible defect on the role-grants screen (each
-- duplicate rendering as its own separately-revocable row).
--
-- M11.7's own BulkGrantUnitRole needed real idempotent-conflict handling inside a shared transaction
-- (a per-row 23505-catch-then-select, safe on the old single-autocommitted-statement InsertRoleAssignment
-- path, would abort the whole transaction once shared across a batch — see
-- internal/authz/adapters/store.go's BulkInsertRoleAssignments doc comment) and its own integration
-- test caught this exact defect live: a pre-existing grant re-submitted inside a batch produced a
-- second active row instead of being upserted, confirming the index was never enforcing what its own
-- name and every prior milestone's comments claimed.
--
-- NULLS NOT DISTINCT (Postgres 15+; this stack runs 16) makes two NULL graph_id values compare equal
-- for this index's own uniqueness purposes, closing the gap for scope='unit' without touching
-- scope='subtree' rows at all — graph_id is never NULL there, so their existing behavior (and
-- RevokeRoleAssignment/subtree-scope handling) is unaffected. Safe to apply: confirmed zero existing
-- duplicate active scope='unit' rows in this deployment before applying (a NULLS NOT DISTINCT unique
-- index creation fails outright if any already exist).
DROP INDEX openfaithmap.authz_role_assignments_active_idx;
CREATE UNIQUE INDEX authz_role_assignments_active_idx
  ON openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope, graph_id) NULLS NOT DISTINCT
  WHERE revoked_at IS NULL;
