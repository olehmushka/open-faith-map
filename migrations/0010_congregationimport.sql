-- 0010_congregationimport — D-CongregationImport (docs/architecture/decisions.md), resolves
-- DS-OFM-10 (docs/open-questions.md). See docs/modules/congregationimport.md for the full design.
--
-- Five tables. Plain uuid PKs (architecture/conventions.md). go-oikumenea references
-- (congregation_unit_rid, taxon_id, country_id, *_person_rid) are opaque TEXT foreign values — no
-- cross-schema FK, a dangling reference is handled at read time as "no longer exists," never a
-- crash. Real in-schema FKs only between this module's own tables.
--
-- Pipeline: a connector run (congregationimport_runs) fetches raw records and stages them as
-- congregationimport_candidates. An operator reviews, edits, approves, or rejects. Approval
-- provisions a real go-oikumenea Unit under the OPERATOR's own forwarded token (never the service
-- principal — createChildOrg's real gate is a human-held permission) and writes a
-- congregationimport_congregation_status row. No congregation-admin grant happens at provisioning
-- time — there is no real submitter to grant it to (D-CongregationImport's "why this isn't an
-- on-behalf-of write" reasoning). claimed_by_person_rid/claimed_at are reserved, unused columns for
-- a future claim flow (vouching.md already names this gap; this migration just leaves room for it
-- so no later migration is needed).

-- One row per connector code. HTML connectors refuse to run without a row here (code-enforced in
-- adapters/connectors/html/base, not just documented) — the decision #4 "check robots.txt/ToS
-- before scraping" discipline made durable and queryable rather than left in a code comment.
CREATE TABLE openfaithmap.congregationimport_connector_citations (
  connector_code                text PRIMARY KEY,
  robots_txt_url                text,
  robots_checked_at             timestamptz,
  robots_checked_by_person_rid  text,
  terms_url                     text,
  terms_checked_at              timestamptz,
  user_agent                    text NOT NULL,
  rate_limit_notes              text,
  citation_notes                text NOT NULL,
  created_at                    timestamptz NOT NULL DEFAULT now(),
  updated_at                    timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER congregationimport_connector_citations_set_updated_at
  BEFORE UPDATE ON openfaithmap.congregationimport_connector_citations
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- One row per triggered connector execution — mirrors go-oikumenea's own hermenea import_runs
-- concept (docs/modules/import.md), scoped to one source per run.
CREATE TABLE openfaithmap.congregationimport_runs (
  id                        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_code               text NOT NULL,
  status                    text NOT NULL DEFAULT 'RUNNING'
                               CHECK (status IN ('RUNNING', 'SUCCEEDED', 'FAILED')),
  triggered_by_person_rid   text NOT NULL,
  cursor_at_start           text,
  cursor_at_end             text,
  records_fetched           integer NOT NULL DEFAULT 0,
  candidates_created        integer NOT NULL DEFAULT 0,
  candidates_updated        integer NOT NULL DEFAULT 0,
  candidates_auto_rejected  integer NOT NULL DEFAULT 0,
  error                     text,
  started_at                timestamptz NOT NULL DEFAULT now(),
  finished_at               timestamptz
);

CREATE INDEX congregationimport_runs_source_idx
  ON openfaithmap.congregationimport_runs (source_code, started_at DESC);

-- The staging table. source_code+source_record_id is the idempotency anchor — a re-run of the same
-- connector against the same upstream record upserts, never duplicates.
--
-- jurisdiction_hint / suggested_jurisdiction_unit_id: for denominations with a real institutional
-- hierarchy (Catholic dioceses/eparchies, Orthodox eparchies/exarchates, Lutheran synods,
-- Anglican/Episcopal dioceses, ...), a connector may carry a free-text hint naming the parish's
-- superior jurisdiction (e.g. a legal name that literally embeds the eparchy). suggested_jurisdiction_
-- unit_id is resolved the same way taxon_id is — substring match against
-- congregationimport_jurisdiction_aliases — and is ADVISORY ONLY: D-JurisdictionUnits already decided
-- jurisdiction is operator-assigned, never inferred, so this never gates status and is never applied
-- automatically at approval; ApproveCandidateRequest.jurisdictionUnitId still requires the operator's
-- own explicit choice. Most independent-polity congregations (Baptist, Pentecostal, many
-- non-denominational bodies) will simply have no hint and no suggestion — by design, not a gap.
CREATE TABLE openfaithmap.congregationimport_candidates (
  id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  import_run_id            uuid REFERENCES openfaithmap.congregationimport_runs(id),
  source_code              text NOT NULL,
  source_record_id         text NOT NULL,
  name                     text NOT NULL,
  taxon_hint               text,
  taxon_id                 text,
  jurisdiction_hint        text,
  suggested_jurisdiction_unit_id  text,
  country_id               text,
  admin_area1              text,
  locality                 text,
  street                   text,
  house_number             text,
  postal_code              text,
  latitude                 double precision,
  longitude                double precision,
  geocode_precision        text,
  raw_payload              jsonb NOT NULL,
  status                   text NOT NULL DEFAULT 'STAGED'
                              CHECK (status IN (
                                'STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE',
                                'APPROVED', 'PROVISIONING', 'PROVISIONED',
                                'REJECTED', 'REJECTED_EXCLUDED')),
  possible_duplicate_of_candidate_id  uuid REFERENCES openfaithmap.congregationimport_candidates(id),
  possible_duplicate_of_unit_id       text,
  rejection_reason         text,
  reviewed_by_person_rid   text,
  reviewed_at              timestamptz,
  created_unit_id          text,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT congregationimport_candidates_source_key UNIQUE (source_code, source_record_id),
  -- Mirrors registration_requests_decision_shape's own discipline (0001_registration.sql): the
  -- status column and its supporting fields must agree, checked at the database, not just trusted
  -- from application code.
  CONSTRAINT congregationimport_candidates_decision_shape CHECK (
    (status IN ('PROVISIONING', 'PROVISIONED') AND created_unit_id IS NOT NULL) OR
    (status IN ('REJECTED', 'REJECTED_EXCLUDED') AND rejection_reason IS NOT NULL) OR
    (status NOT IN ('PROVISIONING', 'PROVISIONED', 'REJECTED', 'REJECTED_EXCLUDED'))
  )
);

CREATE INDEX congregationimport_candidates_status_idx
  ON openfaithmap.congregationimport_candidates (status, created_at);
CREATE INDEX congregationimport_candidates_taxon_idx
  ON openfaithmap.congregationimport_candidates (taxon_id);
CREATE INDEX congregationimport_candidates_suggested_jurisdiction_idx
  ON openfaithmap.congregationimport_candidates (suggested_jurisdiction_unit_id);

CREATE TRIGGER congregationimport_candidates_set_updated_at
  BEFORE UPDATE ON openfaithmap.congregationimport_candidates
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- Operator-maintained free-text-hint -> religion_taxa RID mapping. source_code IS NULL means the
-- alias applies across every source (e.g. "баптист" -> the Baptist taxon, regardless of which
-- connector scraped it); a source-scoped alias overrides the global one for that source only.
CREATE TABLE openfaithmap.congregationimport_taxon_aliases (
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_code             text,
  alias_text              text NOT NULL,
  taxon_id                text NOT NULL,
  created_by_person_rid   text NOT NULL,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX congregationimport_taxon_aliases_scoped_key
  ON openfaithmap.congregationimport_taxon_aliases (source_code, alias_text) WHERE source_code IS NOT NULL;
CREATE UNIQUE INDEX congregationimport_taxon_aliases_global_key
  ON openfaithmap.congregationimport_taxon_aliases (alias_text) WHERE source_code IS NULL;

CREATE TRIGGER congregationimport_taxon_aliases_set_updated_at
  BEFORE UPDATE ON openfaithmap.congregationimport_taxon_aliases
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- Operator-maintained free-text-hint -> go-oikumenea jurisdiction Unit RID mapping. Same shape and
-- matching discipline as congregationimport_taxon_aliases, deliberately not merged with it: a taxon
-- alias resolves "what denomination is this" (religion_taxa), a jurisdiction alias resolves "which
-- specific diocese/eparchy/synod unit is this" (an ordinary go-oikumenea Unit, D-JurisdictionUnits) —
-- unrelated ID spaces, and a source may need one without the other (e.g. an independent Baptist
-- congregation has a taxon but no jurisdiction). jurisdiction_unit_id is never validated against
-- go-oikumenea at write time — same opaque-TEXT-reference discipline as every other cross-schema
-- pointer in this migration; a stale/renamed unit is handled at read/approve time, not here.
CREATE TABLE openfaithmap.congregationimport_jurisdiction_aliases (
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_code             text,
  alias_text              text NOT NULL,
  jurisdiction_unit_id    text NOT NULL,
  created_by_person_rid   text NOT NULL,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX congregationimport_jurisdiction_aliases_scoped_key
  ON openfaithmap.congregationimport_jurisdiction_aliases (source_code, alias_text) WHERE source_code IS NOT NULL;
CREATE UNIQUE INDEX congregationimport_jurisdiction_aliases_global_key
  ON openfaithmap.congregationimport_jurisdiction_aliases (alias_text) WHERE source_code IS NULL;

CREATE TRIGGER congregationimport_jurisdiction_aliases_set_updated_at
  BEFORE UPDATE ON openfaithmap.congregationimport_jurisdiction_aliases
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- The verified/claimed overlay — proposal, not a settled design (D-CongregationImport). Same shape
-- as vouching_guarantor_status: a mutable projection keyed by an immutable go-oikumenea entity.
-- claimed_by_person_rid/claimed_at stay NULL until a future claim flow exists.
CREATE TABLE openfaithmap.congregationimport_congregation_status (
  congregation_unit_rid    text PRIMARY KEY,
  source_code              text NOT NULL,
  import_candidate_id      uuid REFERENCES openfaithmap.congregationimport_candidates(id) ON DELETE SET NULL,
  verified_by_person_rid   text NOT NULL,
  verified_at              timestamptz NOT NULL,
  claimed_by_person_rid    text,
  claimed_at               timestamptz,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER congregationimport_congregation_status_set_updated_at
  BEFORE UPDATE ON openfaithmap.congregationimport_congregation_status
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
