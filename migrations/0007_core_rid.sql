-- 0007_core_rid — M10.1 (D-OwnRIDs, D-CorePortScope). First of the "core absorption" migrations:
-- ports go-oikumenea's UUIDv8 resource-identifier scheme (../go-oikumenea/migrations/0001_platform_core.sql:22-84)
-- into the openfaithmap schema, under this repo's own service/kind/type numbering — not upstream's
-- (there is no compatibility requirement; this is a fresh mint, not a shim). Every later 00NN_core_*
-- migration's PRIMARY KEY DEFAULT and structural CHECK depends on the functions below existing first.
--
-- Byte layout (0-indexed, big-endian), identical to upstream's UUIDv8 (RFC 9562 §5.8):
--   0..5  unix-ms timestamp        (b-tree insert locality)
--   6     0x8<<4 version | kind    (kind: 1=object, 2=link)
--   7     app                      (openfaithmap = 1)
--   8     0b10<<6 variant | service
--   9     type low 8 bits
--   10    type high 4 bits<<4 | random low nibble
--   11..15 random
--
-- Service codes (this repo's own catalog, not go-oikumenea's — mirrored as Go constants when the
-- consuming module lands, M10.2+): 1=identity, 2=authz, 3=directory, 4=religion, 5=location,
-- 6=membership. Kind: 1=object (primary entity), 2=link (relationship/reified edge). Type is a
-- per-table sequential discriminator within (service, kind), assigned as each table is created below.
--
-- Reads no GUC (service/kind/type are caller-supplied literals), so seed migrations may call this
-- directly or use precomputed literals for values that must stay fixed across every deployment
-- (D-SeedBootstrap) — new_id() itself is only for tables whose PRIMARY KEY value need not be
-- deterministic (i.e. everything except the handful of seed rows 0015_core_seed.sql inserts by literal).

CREATE OR REPLACE FUNCTION openfaithmap.new_id(service int, kind int, type_code int) RETURNS uuid
  LANGUAGE plpgsql VOLATILE PARALLEL SAFE AS $$
DECLARE
  unix_ts_ms bigint;
  b bytea;
BEGIN
  IF service < 0 OR service > 63       THEN RAISE EXCEPTION 'rid service out of range (0..63): %', service; END IF;
  IF kind < 1 OR kind > 3               THEN RAISE EXCEPTION 'rid kind out of range (1..3): %', kind; END IF;
  IF type_code < 0 OR type_code > 4095  THEN RAISE EXCEPTION 'rid type out of range (0..4095): %', type_code; END IF;
  unix_ts_ms := floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint;
  b := gen_random_bytes(16);
  b := overlay(b PLACING substring(int8send(unix_ts_ms) FROM 3 FOR 6) FROM 1 FOR 6);  -- bytes 0..5
  b := set_byte(b, 6, 128 | (kind & 15));                                             -- version 8 | kind
  b := set_byte(b, 7, 1);                                                             -- app = openfaithmap
  b := set_byte(b, 8, 128 | (service & 63));                                          -- variant | service
  b := set_byte(b, 9, type_code & 255);                                               -- type low 8
  b := set_byte(b, 10, (((type_code >> 8) & 15) << 4) | (get_byte(b, 10) & 15));       -- type high 4 | rand
  RETURN encode(b, 'hex')::uuid;
END;
$$;

-- rid_* decoders: read the packed fields back out of a RID. IMMUTABLE so they can be used in the
-- per-table structural CHECKs below.
CREATE OR REPLACE FUNCTION openfaithmap.rid_app(id uuid) RETURNS int
  LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$ SELECT get_byte(uuid_send(id), 7) $$;
CREATE OR REPLACE FUNCTION openfaithmap.rid_service(id uuid) RETURNS int
  LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$ SELECT get_byte(uuid_send(id), 8) & 63 $$;
CREATE OR REPLACE FUNCTION openfaithmap.rid_kind(id uuid) RETURNS int
  LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$ SELECT get_byte(uuid_send(id), 6) & 15 $$;
CREATE OR REPLACE FUNCTION openfaithmap.rid_type(id uuid) RETURNS int
  LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT get_byte(uuid_send(id), 9) | (((get_byte(uuid_send(id), 10) >> 4) & 15) << 8) $$;

-- openfaithmap.reject_mutation() (the append-only-table guard) is NOT redefined here — it already
-- exists from 0004_moderation.sql, which always runs before this file in the squashed sequence.
