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
-- (org kinds, site types — already seeded by 0011_core_religion.sql) is looked up by `code`, not by
-- RID, so it did not need a fixed literal; only rows a future Go constant or the M10.9 authorization
-- matrix must name by ID do.
--
-- **Trimmed 2026-08-19** (docs/milestones.md's migration-collapse session, same pass that trimmed
-- 0011_core_religion.sql to Christianity-only): the exclusion backstop below originally seeded three
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
-- ===================================================================================================

INSERT INTO openfaithmap.authz_roles (id, code, name, description, is_base) VALUES
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'registration-operator', 'Registration operator',
   'Approves congregation self-service registrations on the shared root unit (D-Registration).', true),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'congregation-admin', 'Congregation admin',
   'Granted per-congregation at registration-approval time, not seeded onto any unit here.', true),
  ('01989e26-ce03-8101-8201-03a4b1becbd8', 'platform-moderator', 'Platform moderator',
   'Adjudicates reports and appeals on the shared root unit (D-PlatformModerator).', true);

INSERT INTO openfaithmap.authz_role_permissions (role_id, permission_code) VALUES
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'religionorg.manage'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'site.manage'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'schedule.manage'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'assignment.grant'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'assignment.revoke'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'person.create'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'person.update'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'membership.create'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'membership.update'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'position.create'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'position.update'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'unit.read'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'unit.edges.manage'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'religion.read'),
  ('01989e26-ce01-8101-8201-0196a3b0bdca', 'location.create'),

  ('01989e26-ce02-8101-8201-029daab7c4d1', 'unit.read'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'person.read'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'membership.read'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'position.read'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'role.read'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'person.create'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'person.update'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'membership.create'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'membership.update'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'position.create'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'position.update'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'religionorg.manage'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'site.manage'),
  ('01989e26-ce02-8101-8201-029daab7c4d1', 'schedule.manage'),

  ('01989e26-ce03-8101-8201-03a4b1becbd8', 'unit.lifecycle');

-- ===================================================================================================
-- Exclusion backstop (D-Exclusions' org-level second layer, behind checkNotExcluded's taxon-ancestor
-- walk). One placeholder unit under root, carrying an excludes_child_creation policy — kept in sync
-- with internal/registration/domain.ExcludedTaxonCodes (russian_orthodox_church only, see this file's
-- header for why the other two are gone).
-- ===================================================================================================

INSERT INTO openfaithmap.directory_units (id, code, name, state) VALUES
  ('01989e26-ce02-8101-8301-029daab7c4d1', 'excluded-russian_orthodox_church',
   'Russian Orthodox Church (excluded — D-Exclusions)', 'active');

INSERT INTO openfaithmap.directory_unit_edges (graph_id, parent_id, child_id) VALUES
  ('01989e26-ce01-8101-8302-0196a3b0bdca', '01989e26-ce01-8101-8301-0196a3b0bdca', '01989e26-ce02-8101-8301-029daab7c4d1');

-- Closure over the seed edge above (root -> the placeholder, plus its own reflexive row).
INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth) VALUES
  ('01989e26-ce01-8101-8302-0196a3b0bdca', '01989e26-ce01-8101-8301-0196a3b0bdca', '01989e26-ce01-8101-8301-0196a3b0bdca', 0),
  ('01989e26-ce01-8101-8302-0196a3b0bdca', '01989e26-ce02-8101-8301-029daab7c4d1', '01989e26-ce02-8101-8301-029daab7c4d1', 0),
  ('01989e26-ce01-8101-8302-0196a3b0bdca', '01989e26-ce01-8101-8301-0196a3b0bdca', '01989e26-ce02-8101-8301-029daab7c4d1', 1);

INSERT INTO openfaithmap.directory_closure_status (graph_id) VALUES
  ('01989e26-ce01-8101-8302-0196a3b0bdca');

INSERT INTO openfaithmap.religion_org_policies (unit_id, policy_kind_id, reason)
SELECT v.id_val, pk.id, 'D-Exclusions permanent exclusion — see internal/registration/domain.ExcludedTaxonCodes'
FROM (VALUES
  ('01989e26-ce02-8101-8301-029daab7c4d1'::uuid)
) AS v(id_val)
CROSS JOIN openfaithmap.religion_policy_kinds pk
WHERE pk.code = 'excludes_child_creation';
