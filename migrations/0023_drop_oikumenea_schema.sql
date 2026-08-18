-- 0023_drop_oikumenea_schema — M10.8 teardown (D-OwnCore). go-oikumenea's own schema and roles,
-- created externally by the docker-compose services this milestone deletes (oikumenea-migrate,
-- oikumenea-init-role) — every object inside the schema is owned by `postgres` (confirmed live,
-- not by the `oikumenea` login role itself), so DROP SCHEMA ... CASCADE alone removes everything;
-- no DROP OWNED BY step is needed (confirmed live: pg_default_acl has zero entries for either role
-- below, so there is nothing for it to do even as a no-op, and it has no IF EXISTS form — it would
-- error outright on a fresh volume where the role was never created).
--
-- IF EXISTS-safe throughout: a fresh volume that never ran the pre-M10.8 stack has neither the
-- schema nor the roles, and this migration must apply clean on it too, same as every prior
-- migration in this repo (expand-only, never assumes prior out-of-band state). Contract-phase, not
-- expand-phase — the one deliberate exception to that convention, since this drops rather than adds.
DROP SCHEMA IF EXISTS oikumenea CASCADE;
DROP ROLE IF EXISTS oikumenea;
DROP ROLE IF EXISTS oikumenea_app;
