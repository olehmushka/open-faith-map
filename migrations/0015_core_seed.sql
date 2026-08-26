-- 0015_core_seed — M10.1 (D-SeedBootstrap). Deterministic seed data with fixed RIDs, replacing four
-- manual script runs (scripts/bootstrap-{registration-org,exclusion-backstop}; the other two,
-- bootstrap-service-principal and bootstrap-admin-person, are deleted outright at M10.2/M10.6 — the
-- service principal concept goes away with D-DirectTokenVerification, and the first admin becomes a
-- boot-time seed from install config, D-SeedBootstrap's own amendment — NOT this migration, since
-- identity is the one deliberate exception to determinism).
--
-- Fixed RID literals below were minted once (a throwaway script replicating openfaithmap.new_id()'s
-- bit-packing with a fixed base timestamp instead of clock_timestamp()) and hardcoded — calling
-- new_id() itself here would produce a different value on every fresh deployment, exactly what
-- D-SeedBootstrap exists to avoid for these specific rows. Every other seed row in this migration
-- (org kinds, site types — already seeded by 0014_core_religion.sql) is looked up by `code`, not by
-- RID, so it did not need a fixed literal.
--
-- **Deduplicated 2026-08-25** (this migration-squash pass): a literal is written exactly once, on the
-- INSERT that defines that row's id (the canonical graph, the root unit, the three base roles, the
-- exclusion-backstop unit). Every other reference to one of those ids — permission grants, edges,
-- closure rows, the policy row — looks it up by `code` via a subquery instead of repeating the
-- literal. Same end state as before (every table's final rows are byte-identical), just without ~25
-- repeated copies of the same six UUIDs to keep in sync by hand. All the `code` columns used below
-- (`authz_roles.code`, `directory_units.code`, `directory_graphs.code`) already carry a
-- unique-while-active index, so these lookups are index-backed, not sequential scans.
--
-- **Trimmed 2026-08-19** (docs/milestones.md's migration-collapse session, same pass that trimmed
-- 0014_core_religion.sql to Christianity-only): the exclusion backstop below originally seeded three
-- placeholder units, one per D-Exclusions denomination. Since `lds_church`/`jehovahs_witnesses` are
-- now hard-deleted from the taxonomy entirely (no religion_taxa row exists for either), their
-- placeholder org-level backstop units are dead weight — nothing can ever reference a denomination
-- that no longer exists in the taxon picker or anywhere else. Only the `russian_orthodox_church`
-- placeholder remains (its taxon row, and its app-code exclusion in
-- internal/registration/domain.ExcludedTaxonCodes, both stay — political, not doctrinal, exclusion).

-- ===================================================================================================
-- Root unit + canonical graph.
-- ===================================================================================================

INSERT INTO openfaithmap.directory_graphs (id, code, name, is_default, is_authority_bearing) VALUES
  ('01989e26-ce01-8101-8302-0196a3b0bdca', 'canonical', 'Canonical', true, true);

INSERT INTO openfaithmap.directory_units (id, code, name, state) VALUES
  ('01989e26-ce01-8101-8301-0196a3b0bdca', 'root', 'OpenFaithMap', 'active');

-- ===================================================================================================
-- Base roles. Permission sets are the trimmed successors of
-- scripts/bootstrap-registration-org/main.go's three roles — assignment.read is dropped from all
-- three (it existed solely to satisfy go-oikumenea's cross-service Authorize meta-check, which
-- D-OwnCore's own text records as "meaningless" once Authorize is a pure in-process function of the
-- subject — see D-OwnCore's "One deliberate behaviour change" and M10.6's punch list).
--
-- platform-moderator is seeded directly with `moderation.standing` (M12.0, D-PlatformModerator's
-- addendum) — its own dedicated identity-marker permission, not the overloaded `unit.lifecycle` an
-- earlier pass here used as a stand-in before the Go permission catalog could mint a new code.
-- ===================================================================================================

INSERT INTO openfaithmap.authz_roles (id, code, name, description, is_base) VALUES
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'registration-operator', 'Registration operator',
   'Approves congregation self-service registrations on the shared root unit (D-Registration).', true),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'congregation-admin', 'Congregation admin',
   'Granted per-congregation at registration-approval time, not seeded onto any unit here.', true),
  ('01989e26-ce03-8101-8201-03a4b1becbd8', 'platform-moderator', 'Platform moderator',
   'Adjudicates reports and appeals on the shared root unit (D-PlatformModerator).', true);

INSERT INTO openfaithmap.authz_role_permissions (role_id, permission_code)
SELECT (SELECT id FROM openfaithmap.authz_roles WHERE code = 'registration-operator'), p.code
FROM (VALUES
  ('religionorg.manage'), ('site.manage'), ('schedule.manage'), ('assignment.grant'),
  ('assignment.revoke'), ('person.create'), ('person.update'), ('membership.create'),
  ('membership.update'), ('position.create'), ('position.update'), ('unit.read'),
  ('unit.edges.manage'), ('religion.read'), ('location.create')
) AS p(code);

INSERT INTO openfaithmap.authz_role_permissions (role_id, permission_code)
SELECT (SELECT id FROM openfaithmap.authz_roles WHERE code = 'congregation-admin'), p.code
FROM (VALUES
  ('unit.read'), ('person.read'), ('membership.read'), ('position.read'), ('role.read'),
  ('person.create'), ('person.update'), ('membership.create'), ('membership.update'),
  ('position.create'), ('position.update'), ('religionorg.manage'), ('site.manage'),
  ('schedule.manage')
) AS p(code);

INSERT INTO openfaithmap.authz_role_permissions (role_id, permission_code) VALUES
  ((SELECT id FROM openfaithmap.authz_roles WHERE code = 'platform-moderator'), 'moderation.standing');

-- ===================================================================================================
-- Exclusion backstop (D-Exclusions' org-level second layer, behind checkNotExcluded's taxon-ancestor
-- walk). One placeholder unit under root, carrying an excludes_child_creation policy — kept in sync
-- with internal/registration/domain.ExcludedTaxonCodes (russian_orthodox_church only, see this file's
-- header for why the other two are gone).
-- ===================================================================================================

INSERT INTO openfaithmap.directory_units (id, code, name, state) VALUES
  ('01989e26-ce02-8101-8301-029daab7c4d1', 'excluded-russian_orthodox_church',
   'Russian Orthodox Church (excluded — D-Exclusions)', 'active');

INSERT INTO openfaithmap.directory_unit_edges (graph_id, parent_id, child_id)
SELECT (SELECT id FROM openfaithmap.directory_graphs WHERE code = 'canonical'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'root'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'excluded-russian_orthodox_church');

-- Closure over the seed edge above (root -> the placeholder, plus its own reflexive row).
INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
SELECT (SELECT id FROM openfaithmap.directory_graphs WHERE code = 'canonical'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'root'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'root'), 0
UNION ALL
SELECT (SELECT id FROM openfaithmap.directory_graphs WHERE code = 'canonical'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'excluded-russian_orthodox_church'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'excluded-russian_orthodox_church'), 0
UNION ALL
SELECT (SELECT id FROM openfaithmap.directory_graphs WHERE code = 'canonical'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'root'),
       (SELECT id FROM openfaithmap.directory_units WHERE code = 'excluded-russian_orthodox_church'), 1;

INSERT INTO openfaithmap.directory_closure_status (graph_id)
SELECT (SELECT id FROM openfaithmap.directory_graphs WHERE code = 'canonical');

INSERT INTO openfaithmap.religion_org_policies (unit_id, policy_kind_id, reason)
SELECT (SELECT id FROM openfaithmap.directory_units WHERE code = 'excluded-russian_orthodox_church'),
       pk.id, 'D-Exclusions permanent exclusion — see internal/registration/domain.ExcludedTaxonCodes'
FROM openfaithmap.religion_policy_kinds pk
WHERE pk.code = 'excludes_child_creation';
