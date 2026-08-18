-- 0016_core_authz — M10.1 (D-InProcessAuthz, D-CorePortScope's amendment). Ports the authz portion
-- of ../go-oikumenea/migrations/0004_authz_identity.sql (the authz_* section). No Postgres RLS
-- anywhere (D-InProcessAuthz) — the in-process PDP (M10.3) is the sole authority. No grant cache
-- table either (the amendment drops it; grants are read per request), so authz_role_assignments and
-- authz_instance_admins are the only tables the PDP reads at decision time.
--
-- authz_instance_admins is confirmed must-port (not upstream's original plan): PDP.Decide branches
-- on it first — without it every instance-scope action is permanently denied to everyone.

CREATE TABLE openfaithmap.authz_roles (
  id          uuid PRIMARY KEY DEFAULT openfaithmap.new_id(2,1,1),  -- authz / object / role
  code        text NOT NULL,                 -- stable, locale-agnostic; unique among active
  name        text NOT NULL,
  description text,
  is_base     boolean NOT NULL DEFAULT false, -- seeded base roles (0022_core_seed.sql); not instance-editable
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,

  CONSTRAINT authz_roles_rid_shape
    CHECK (openfaithmap.rid_service(id)=2 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=1)
);
CREATE UNIQUE INDEX authz_roles_code_active_idx
  ON openfaithmap.authz_roles (code) WHERE deleted_at IS NULL;
CREATE TRIGGER authz_roles_set_updated_at
  BEFORE UPDATE ON openfaithmap.authz_roles
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- authz_role_permissions: a role's membership of code-defined permissions. No RID (a plain grants
-- link — a composite-PK row, not a reified entity). permission_code is validated against the closed
-- Go permission catalog (internal/authz, M10.3) at write time, not by a DB constraint.
CREATE TABLE openfaithmap.authz_role_permissions (
  role_id         uuid NOT NULL REFERENCES openfaithmap.authz_roles(id) ON DELETE CASCADE,
  permission_code text NOT NULL,
  PRIMARY KEY (role_id, permission_code)
);

-- authz_role_assignments: the unit of granted authority and the PDP's core input. graph_id names the
-- hierarchy a `subtree` grant cascades over and is NULL iff scope='unit'.
CREATE TABLE openfaithmap.authz_role_assignments (
  id                uuid PRIMARY KEY DEFAULT openfaithmap.new_id(2,2,1),  -- authz / link / has_role
  subject_person_id uuid NOT NULL REFERENCES openfaithmap.identity_persons(id) ON DELETE RESTRICT,
  role_id           uuid NOT NULL REFERENCES openfaithmap.authz_roles(id) ON DELETE RESTRICT,
  target_unit_id    uuid NOT NULL,  -- REFERENCES openfaithmap.directory_units(id); FK added in 0017_core_directory.sql
  scope             text NOT NULL CHECK (scope IN ('unit','subtree')),
  graph_id          uuid,           -- REFERENCES openfaithmap.directory_graphs(id); FK added in 0017_core_directory.sql
  granted_by        uuid REFERENCES openfaithmap.identity_persons(id) ON DELETE SET NULL,  -- NULL for bootstrap
  granted_at        timestamptz NOT NULL DEFAULT now(),
  revoked_at        timestamptz,
  revoked_by        uuid REFERENCES openfaithmap.identity_persons(id) ON DELETE SET NULL,
  expires_at        timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT authz_role_assignments_rid_shape
    CHECK (openfaithmap.rid_service(id)=2 AND openfaithmap.rid_kind(id)=2 AND openfaithmap.rid_type(id)=1),
  CONSTRAINT authz_role_assignments_graph_scope CHECK ((scope = 'subtree') = (graph_id IS NOT NULL))
);
CREATE UNIQUE INDEX authz_role_assignments_active_idx
  ON openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope, graph_id)
  WHERE revoked_at IS NULL;
CREATE INDEX authz_role_assignments_subject_idx
  ON openfaithmap.authz_role_assignments (subject_person_id) WHERE revoked_at IS NULL;
CREATE INDEX authz_role_assignments_target_idx
  ON openfaithmap.authz_role_assignments (target_unit_id) WHERE revoked_at IS NULL;
CREATE TRIGGER authz_role_assignments_set_updated_at
  BEFORE UPDATE ON openfaithmap.authz_role_assignments
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- authz_instance_admins: the instance-wide authority plane (D-InProcessAuthz's amendment — a plane,
-- not a role: IsInstanceScope makes instance-scope permissions unsatisfiable by any unit-scoped
-- role). granted_by is NULL for the boot-time first-admin seed (M10.2, D-SeedBootstrap's amendment).
CREATE TABLE openfaithmap.authz_instance_admins (
  id         uuid PRIMARY KEY DEFAULT openfaithmap.new_id(2,2,2),  -- authz / link / instance_admin
  person_id  uuid NOT NULL REFERENCES openfaithmap.identity_persons(id) ON DELETE RESTRICT,
  granted_by uuid REFERENCES openfaithmap.identity_persons(id) ON DELETE SET NULL,
  granted_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz,
  revoked_by uuid REFERENCES openfaithmap.identity_persons(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT authz_instance_admins_rid_shape
    CHECK (openfaithmap.rid_service(id)=2 AND openfaithmap.rid_kind(id)=2 AND openfaithmap.rid_type(id)=2)
);
CREATE UNIQUE INDEX authz_instance_admins_person_active_idx
  ON openfaithmap.authz_instance_admins (person_id) WHERE revoked_at IS NULL;
CREATE TRIGGER authz_instance_admins_set_updated_at
  BEFORE UPDATE ON openfaithmap.authz_instance_admins
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- authz_epoch: NOT ported (D-InProcessAuthz's amendment drops the grant cache it existed to
-- invalidate — grants are read per request, one indexed join, no cache to bump an epoch for).
