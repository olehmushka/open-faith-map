-- 0001_registration — OpenFaithMap's first schema (M2/M2.3/M2.4/M4.1, docs/modules/registration.md,
-- D-JurisdictionUnits). Squashed from the original 0001/0002/0003/0006 (docs/development-process.md's
-- migration-collapse pass) into final shape directly — no need to replay the PROVISIONING-status
-- widening or the jurisdiction_unit_id column-add as separate steps.
--
-- Creates the openfaithmap schema, the least-privilege openfaithmap_app role (D-SharedDatabase — the
-- application connects as a LOGIN role in that group, created outside Atlas by
-- docker-compose.yml's openfaithmap-init-role; a role with a password has no business being
-- version-controlled DDL), and registration_requests: pending congregation registrations awaiting a
-- registration-operator's approval. Every go-oikumenea RID referenced here (submitted_by_person_id,
-- taxon_id, country_id, decided_by_person_id, created_unit_id, jurisdiction_unit_id) is stored as an
-- opaque TEXT foreign value — no cross-database FKs (architecture/conventions.md).
--
-- jurisdiction_reparenting_jobs (D-JurisdictionUnits) tracks re-parenting an already-APPROVED
-- request's congregation unit onto a different jurisdiction unit — a resumable state machine, since
-- go-oikumenea's addEdge+removeEdge (canonical graph) is two non-transactional calls, not one atomic
-- move.

CREATE SCHEMA IF NOT EXISTS openfaithmap;

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

CREATE OR REPLACE FUNCTION openfaithmap.set_updated_at() RETURNS trigger
  LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$;

CREATE TABLE openfaithmap.registration_requests (
  id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  submitted_by_person_id text NOT NULL,
  taxon_id               text NOT NULL,
  congregation_name      text NOT NULL,
  country_id             text NOT NULL,
  admin_area1            text,
  locality               text,
  street                 text,
  house_number           text,
  postal_code            text,
  latitude               double precision NOT NULL,
  longitude              double precision NOT NULL,
  status                 text NOT NULL DEFAULT 'PENDING'
                           CHECK (status IN ('PENDING', 'PROVISIONING', 'APPROVED', 'REJECTED')),
  rejection_reason       text,
  decided_by_person_id   text,
  decided_at             timestamptz,
  created_unit_id        text,
  jurisdiction_unit_id   text,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT registration_requests_decision_shape CHECK (
    (status = 'PENDING'      AND decided_by_person_id IS NULL     AND decided_at IS NULL) OR
    (status = 'PROVISIONING' AND decided_by_person_id IS NOT NULL AND created_unit_id IS NOT NULL) OR
    (status = 'APPROVED'     AND decided_by_person_id IS NOT NULL AND decided_at IS NOT NULL AND created_unit_id IS NOT NULL) OR
    (status = 'REJECTED'     AND decided_by_person_id IS NOT NULL AND decided_at IS NOT NULL AND rejection_reason IS NOT NULL)
  )
);

CREATE INDEX registration_requests_status_idx ON openfaithmap.registration_requests (status, created_at);
CREATE INDEX registration_requests_submitter_idx ON openfaithmap.registration_requests (submitted_by_person_id);

CREATE TRIGGER registration_requests_set_updated_at
  BEFORE UPDATE ON openfaithmap.registration_requests
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- Least-privilege role (D-SharedDatabase): openfaithmap-api connects as a LOGIN role in this group,
-- never as the postgres superuser.
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

CREATE TABLE openfaithmap.jurisdiction_reparenting_jobs (
  id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  registration_request_id  uuid REFERENCES openfaithmap.registration_requests(id) ON DELETE SET NULL,
  congregation_unit_id     text NOT NULL,
  old_parent_unit_id       text NOT NULL,
  new_parent_unit_id       text NOT NULL,
  status                   text NOT NULL DEFAULT 'PENDING'
                             CHECK (status IN ('PENDING', 'NEW_EDGE_ADDED', 'OLD_EDGE_REMOVED', 'VERIFIED', 'FAILED')),
  performed_by_person_id   text NOT NULL,
  error                    text,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT jurisdiction_reparenting_jobs_error_shape CHECK (
    (status = 'FAILED' AND error IS NOT NULL) OR
    (status <> 'FAILED' AND error IS NULL)
  )
);

-- At most one live (non-FAILED) job per congregation unit at a time — a FAILED job doesn't block a
-- fresh attempt, but two simultaneous in-flight moves of the same unit would race.
CREATE UNIQUE INDEX jurisdiction_reparenting_jobs_live_unit_idx
  ON openfaithmap.jurisdiction_reparenting_jobs (congregation_unit_id)
  WHERE status <> 'FAILED';

CREATE INDEX jurisdiction_reparenting_jobs_request_idx
  ON openfaithmap.jurisdiction_reparenting_jobs (registration_request_id, created_at DESC);

CREATE TRIGGER jurisdiction_reparenting_jobs_set_updated_at
  BEFORE UPDATE ON openfaithmap.jurisdiction_reparenting_jobs
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
