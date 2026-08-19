-- 0010_core_directory — M10.1 (D-CorePortScope, amendment). Ports the kept slice of
-- ../go-oikumenea/migrations/0002_tenant_rank.sql's tenant module: tenant_units/graphs/unit_edges/
-- unit_closure/closure_status, renamed directory_*. Dropped per D-CorePortScope: tenant_domains,
-- tenant_unit_kinds, tenant_organizations, tenant_org_lifecycle_events, tenant_unit_lifecycle_events,
-- tenant_unit_code_events (all org/domain/lifecycle-event machinery — OpenFaithMap is one product,
-- one tenant), and all rank_* tables (not used anywhere in this repo).
--
-- Also dropped (the amendment's explicit "named so the port does not copy a column without its
-- logic"): tenant_units.org_id/domain_id/kind_id (organizations are gone), .visibility + ShadowGate
-- (OpenFaithMap has no shadow-unit concept — site-level privacy is religion_sites.public_precision).
--
-- Retroactively adds the FKs 0009_core_authz.sql's authz_role_assignments left as bare uuid columns
-- (directory_units/directory_graphs didn't exist yet when that file ran).
--
-- The closure lock (D-CorePortScope's amendment): FOR NO KEY UPDATE on directory_graphs is a ROW
-- lock, not advisory — taken by application code (M10.4/M10.6), not this migration. Binding
-- invariant carried forward here as a comment since there is no other home for it in schema DDL: no
-- network call, geocode, or external fetch may occur while that lock is held.

CREATE TABLE openfaithmap.directory_units (
  id          uuid PRIMARY KEY DEFAULT openfaithmap.new_id(3,1,1),  -- directory / object / unit
  code        text,                          -- optional, mutable, unique among active coded units
  name        text NOT NULL,
  level       smallint,                      -- optional ordinal for sort/filter; never a PDP input
  state       text NOT NULL DEFAULT 'active' CHECK (state IN ('active','suspended','archived')),
  metadata    jsonb NOT NULL DEFAULT '{}',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,

  CONSTRAINT directory_units_rid_shape
    CHECK (openfaithmap.rid_service(id)=3 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=1)
);
CREATE UNIQUE INDEX directory_units_code_active_idx
  ON openfaithmap.directory_units (code) WHERE deleted_at IS NULL AND code IS NOT NULL;
CREATE INDEX directory_units_level_idx ON openfaithmap.directory_units (level) WHERE deleted_at IS NULL;
CREATE TRIGGER directory_units_set_updated_at
  BEFORE UPDATE ON openfaithmap.directory_units
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- directory_graphs: the named-hierarchy registry. `canonical` is the default + authority-bearing
-- graph (the PDP cascades subtree grants over it); a single-tenant product needs no org_id column.
CREATE TABLE openfaithmap.directory_graphs (
  id                   uuid PRIMARY KEY DEFAULT openfaithmap.new_id(3,1,2),  -- directory / object / graph
  code                 text NOT NULL,           -- stable, locale-agnostic (e.g. 'canonical')
  name                 text NOT NULL,
  is_default           boolean NOT NULL DEFAULT false,
  is_authority_bearing  boolean NOT NULL DEFAULT true,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz,

  CONSTRAINT directory_graphs_rid_shape
    CHECK (openfaithmap.rid_service(id)=3 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=2),
  CONSTRAINT directory_graphs_canonical_authority CHECK (code <> 'canonical' OR is_authority_bearing)
);
CREATE UNIQUE INDEX directory_graphs_code_active_idx
  ON openfaithmap.directory_graphs (code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX directory_graphs_one_default_idx
  ON openfaithmap.directory_graphs ((true)) WHERE is_default AND deleted_at IS NULL;
CREATE TRIGGER directory_graphs_set_updated_at
  BEFORE UPDATE ON openfaithmap.directory_graphs
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- Now that both tables exist, close the FKs 0009_core_authz.sql's authz_role_assignments deferred.
ALTER TABLE openfaithmap.authz_role_assignments
  ADD CONSTRAINT authz_role_assignments_target_unit_fk
    FOREIGN KEY (target_unit_id) REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  ADD CONSTRAINT authz_role_assignments_graph_fk
    FOREIGN KEY (graph_id) REFERENCES openfaithmap.directory_graphs(id) ON DELETE RESTRICT;

-- directory_unit_edges: the reified parent->child edge, per graph. Cycle prevention is enforced in
-- the application on insert (via the closure); the closure is recomputed in the same transaction.
CREATE TABLE openfaithmap.directory_unit_edges (
  id         uuid PRIMARY KEY DEFAULT openfaithmap.new_id(3,2,1),  -- directory / link / parent_of
  graph_id   uuid NOT NULL REFERENCES openfaithmap.directory_graphs(id) ON DELETE RESTRICT,
  parent_id  uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  child_id   uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  valid_from timestamptz NOT NULL DEFAULT now(),
  valid_to   timestamptz CHECK (valid_to IS NULL OR valid_to >= valid_from),
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by uuid REFERENCES openfaithmap.identity_persons(id) ON DELETE SET NULL,

  CONSTRAINT directory_unit_edges_rid_shape
    CHECK (openfaithmap.rid_service(id)=3 AND openfaithmap.rid_kind(id)=2 AND openfaithmap.rid_type(id)=1),
  CONSTRAINT directory_unit_edges_no_self_loop CHECK (parent_id <> child_id),
  CONSTRAINT directory_unit_edges_unique UNIQUE (graph_id, parent_id, child_id)
);
CREATE INDEX directory_unit_edges_parent_idx ON openfaithmap.directory_unit_edges (graph_id, parent_id);
CREATE INDEX directory_unit_edges_child_idx  ON openfaithmap.directory_unit_edges (graph_id, child_id);

-- directory_unit_closure: derived, maintained per graph on every edge change. Composite key, no RID
-- — a materialized derived relation, not a source of truth. Includes the reflexive (g,u,u,0) row.
CREATE TABLE openfaithmap.directory_unit_closure (
  graph_id      uuid NOT NULL REFERENCES openfaithmap.directory_graphs(id) ON DELETE RESTRICT,
  ancestor_id   uuid NOT NULL,
  descendant_id uuid NOT NULL,
  depth         integer NOT NULL,

  PRIMARY KEY (graph_id, ancestor_id, descendant_id)
);
CREATE INDEX directory_unit_closure_descendant_idx
  ON openfaithmap.directory_unit_closure (graph_id, descendant_id);

-- directory_closure_status: derived diagnostic overlay, one row per graph. Upserted by a verify
-- endpoint (M10.4); not append-only, not audited.
CREATE TABLE openfaithmap.directory_closure_status (
  graph_id        uuid PRIMARY KEY REFERENCES openfaithmap.directory_graphs(id) ON DELETE CASCADE,
  last_checked_at timestamptz NOT NULL DEFAULT now(),
  missing_count   integer NOT NULL DEFAULT 0,
  extra_count     integer NOT NULL DEFAULT 0,
  in_drift        boolean NOT NULL DEFAULT false,
  sample          jsonb,
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER directory_closure_status_set_updated_at
  BEFORE UPDATE ON openfaithmap.directory_closure_status
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
