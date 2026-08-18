-- 0019_core_location — M10.1 (D-CorePortScope). Ports the 2 kept location tables from
-- ../go-oikumenea/migrations/0007_reference_verticals.sql:494-622 — the WOF gazetteer (geo_places)
-- is dropped, nothing here queries it. PostGIS is already enabled on this instance (confirmed:
-- postgis/postgis:16-3.4 image, `SELECT extname FROM pg_extension` includes postgis).
--
-- Closes the location_id FK religion_sites (0018_core_religion.sql) deferred, since
-- location_locations didn't exist yet when that file ran.

CREATE TABLE openfaithmap.location_location_types (
  id         uuid PRIMARY KEY DEFAULT openfaithmap.new_id(5,1,2),  -- location / object / location_type
  code       text NOT NULL,
  name       text NOT NULL,
  status     text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT location_location_types_rid_shape
    CHECK (openfaithmap.rid_service(id)=5 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=2)
);
CREATE UNIQUE INDEX location_location_types_code_active
  ON openfaithmap.location_location_types (code) WHERE deleted_at IS NULL;
CREATE TRIGGER location_location_types_set_updated_at
  BEFORE UPDATE ON openfaithmap.location_location_types
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.location_location_types (code, name) VALUES
  ('building', 'Building'),
  ('address',  'Address'),
  ('online',   'Online');

-- location_locations: the shared place object. `geom` GEOGRAPHY(POINT,4326) is the authoritative
-- coordinate; `country_id` sits over refdata_countries (0021_core_refdata.sql, applied after this
-- file — FK added there, same deferred-FK pattern as religion_sites.location_id above).
CREATE TABLE openfaithmap.location_locations (
  id                uuid PRIMARY KEY DEFAULT openfaithmap.new_id(5,1,1),  -- location / object / location
  geom              geography(Point, 4326) NOT NULL,
  mgrs              text,
  source_coordinate jsonb NOT NULL DEFAULT '{}',
  country_id        uuid NOT NULL,  -- REFERENCES openfaithmap.refdata_countries(id); FK added in 0021_core_refdata.sql
  admin_area_1      text,
  admin_area_2      text,
  locality          text,
  street            text,
  house_number      text,
  postal_code       text,
  raw_address       text,
  type_id           uuid REFERENCES openfaithmap.location_location_types(id) ON DELETE RESTRICT,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  deleted_at        timestamptz,
  CONSTRAINT location_locations_rid_shape
    CHECK (openfaithmap.rid_service(id)=5 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=1)
);
CREATE TRIGGER location_locations_set_updated_at
  BEFORE UPDATE ON openfaithmap.location_locations
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
CREATE INDEX location_locations_geom_gist ON openfaithmap.location_locations USING gist (geom);
CREATE INDEX location_locations_country ON openfaithmap.location_locations (country_id) WHERE deleted_at IS NULL;

-- Now that location_locations exists, close the FK religion_sites.location_id deferred.
ALTER TABLE openfaithmap.religion_sites
  ADD CONSTRAINT religion_sites_location_fk
    FOREIGN KEY (location_id) REFERENCES openfaithmap.location_locations(id) ON DELETE RESTRICT;
CREATE INDEX religion_sites_location_idx ON openfaithmap.religion_sites (location_id) WHERE deleted_at IS NULL;
