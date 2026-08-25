-- 0011_core_membership — M10.1 (D-CorePortScope). Ports the 2 kept membership tables from
-- ../go-oikumenea/migrations/0003_person_membership.sql:300-393. Drops required_rank_id (all of
-- internal/rank is dropped) and order_item_id (the order/наказ module is never ported).

CREATE TABLE openfaithmap.membership_positions (
  id          uuid PRIMARY KEY DEFAULT openfaithmap.new_id(6,1,1),  -- membership / object / position
  unit_id     uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  code        text NOT NULL,
  title       text NOT NULL,
  status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active','abolished')),
  sort_order  integer,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,
  CONSTRAINT membership_positions_rid_shape
    CHECK (openfaithmap.rid_service(id)=6 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=1)
);
CREATE UNIQUE INDEX membership_positions_unit_code_active_idx
  ON openfaithmap.membership_positions (unit_id, code) WHERE deleted_at IS NULL;
CREATE INDEX membership_positions_unit_active_idx
  ON openfaithmap.membership_positions (unit_id) WHERE status = 'active' AND deleted_at IS NULL;
CREATE TRIGGER membership_positions_set_updated_at
  BEFORE UPDATE ON openfaithmap.membership_positions
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.membership_memberships (
  id             uuid PRIMARY KEY DEFAULT openfaithmap.new_id(6,2,1),  -- membership / link / member_of
  person_id      uuid NOT NULL REFERENCES openfaithmap.identity_persons(id) ON DELETE RESTRICT,
  unit_id        uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  position_id    uuid REFERENCES openfaithmap.membership_positions(id) ON DELETE RESTRICT,  -- NULL = plain belonging
  status         text NOT NULL DEFAULT 'active' CHECK (status IN ('active','ended')),
  effective_from timestamptz NOT NULL DEFAULT now(),
  effective_to   timestamptz,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  deleted_at     timestamptz,
  CONSTRAINT membership_memberships_rid_shape
    CHECK (openfaithmap.rid_service(id)=6 AND openfaithmap.rid_kind(id)=2 AND openfaithmap.rid_type(id)=1)
);
-- One billet, one holder: a position has at most one ACTIVE filling.
CREATE UNIQUE INDEX membership_memberships_one_holder_idx
  ON openfaithmap.membership_memberships (position_id)
  WHERE position_id IS NOT NULL AND status = 'active' AND deleted_at IS NULL;
-- Plain belonging is unique per (person, unit) among active position-less memberships.
CREATE UNIQUE INDEX membership_memberships_belonging_idx
  ON openfaithmap.membership_memberships (person_id, unit_id)
  WHERE position_id IS NULL AND status = 'active' AND deleted_at IS NULL;
CREATE INDEX membership_memberships_person_idx
  ON openfaithmap.membership_memberships (person_id) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX membership_memberships_unit_idx
  ON openfaithmap.membership_memberships (unit_id) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX membership_memberships_position_idx
  ON openfaithmap.membership_memberships (position_id) WHERE status = 'active' AND deleted_at IS NULL;
CREATE TRIGGER membership_memberships_set_updated_at
  BEFORE UPDATE ON openfaithmap.membership_memberships
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
