-- 0001_registration — OpenFaithMap's first schema (M2, docs/modules/registration.md).
--
-- Creates the openfaithmap schema and the registration_requests table: pending congregation
-- registrations awaiting a registration-operator's approval. Every go-oikumenea RID referenced here
-- (submitted_by_person_id, taxon_id, country_id, decided_by_person_id, created_unit_id) is stored as
-- an opaque TEXT foreign value — no cross-database FKs (architecture/conventions.md).

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
                           CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
  rejection_reason       text,
  decided_by_person_id   text,
  decided_at             timestamptz,
  created_unit_id        text,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT registration_requests_decision_shape CHECK (
    (status = 'PENDING'  AND decided_by_person_id IS NULL AND decided_at IS NULL) OR
    (status = 'APPROVED' AND decided_by_person_id IS NOT NULL AND decided_at IS NOT NULL AND created_unit_id IS NOT NULL) OR
    (status = 'REJECTED' AND decided_by_person_id IS NOT NULL AND decided_at IS NOT NULL AND rejection_reason IS NOT NULL)
  )
);

CREATE INDEX registration_requests_status_idx ON openfaithmap.registration_requests (status, created_at);
CREATE INDEX registration_requests_submitter_idx ON openfaithmap.registration_requests (submitted_by_person_id);

CREATE TRIGGER registration_requests_set_updated_at
  BEFORE UPDATE ON openfaithmap.registration_requests
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
