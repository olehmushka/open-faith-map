-- 0003_least_privilege_role — M2.4 item 3 (docs/milestones.md, D-SharedDatabase).
--
-- openfaithmap-api currently connects as the postgres superuser, so it can read and write
-- go-oikumenea's entire oikumenea schema — which D-CoreDependency calls "rejected outright."
-- openfaithmap_app is a NOLOGIN group role granted USAGE + DML on the openfaithmap schema only, and
-- no grant of any kind on oikumenea — mirroring go-oikumenea's own oikumenea_app role
-- (migrations/0005_document_order_rls.sql in that repo) so the same shape holds on both sides of the
-- shared instance. The LOGIN role the application actually connects as
-- (docker-compose.yml's openfaithmap-init-role, membership IN ROLE openfaithmap_app) is created
-- outside Atlas, the same way oikumenea-init-role creates go-oikumenea's login role — a role with a
-- password has no business being version-controlled DDL.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openfaithmap_app') THEN
    CREATE ROLE openfaithmap_app NOLOGIN;
  END IF;
END$$;

GRANT USAGE ON SCHEMA openfaithmap TO openfaithmap_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA openfaithmap TO openfaithmap_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA openfaithmap TO openfaithmap_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA openfaithmap TO openfaithmap_app;

-- Future objects (forward-compatible; keeps the grant correct as later migrations add tables).
ALTER DEFAULT PRIVILEGES IN SCHEMA openfaithmap GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO openfaithmap_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA openfaithmap GRANT USAGE, SELECT ON SEQUENCES TO openfaithmap_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA openfaithmap GRANT EXECUTE ON FUNCTIONS TO openfaithmap_app;
