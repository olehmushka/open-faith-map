-- 0008_core_identity — M10.1 (D-CorePortScope, D-OwnCore). Ports the one kept table of
-- go-oikumenea's ~50-table person module (../go-oikumenea/migrations/0003_person_membership.sql,
-- the person_persons definition) plus its identity-federation login/account tables
-- (../go-oikumenea/migrations/0004_authz_identity.sql, the account_* section, renamed identity_*).
--
-- Trimmed relative to upstream (D-CorePortScope: "internal/person is rewritten, not lifted"):
-- birthdate/sex/country_of_birth_id/attributes/provisional-person-stub machinery are dropped — none
-- of it is used anywhere in this repo. The CLDR name field set and the trigram search_text column
-- are kept verbatim; they are the part upstream itself calls "worth copying".
--
-- No RLS (D-InProcessAuthz — no Postgres RLS anywhere in the ported core). No seed rows: per
-- D-SeedBootstrap's amendment, identity is the one deliberate exception to deterministic seeding —
-- the first admin is seeded at boot from install config (M10.2), never by a migration.

CREATE TABLE openfaithmap.identity_persons (
  id             uuid PRIMARY KEY DEFAULT openfaithmap.new_id(1,1,1),  -- identity / object / person
  code           text,                          -- optional stable external id; unique among active
  display_name   text NOT NULL,                 -- canonical full-name form; authoritative for search/display

  -- Unicode CLDR Person Names field set (all optional; advisory — display_name is authoritative).
  title          text,
  given          text,
  given2         text,
  surname        text,
  surname_prefix text,
  surname2       text,
  generation     text,
  credentials    text,
  preferred      text,

  status         text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'deactivated')),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  deleted_at     timestamptz,

  -- Denormalized lowercased search haystack, STORED + trigram-indexed (pg_trgm, already enabled).
  search_text    text GENERATED ALWAYS AS (
                   lower(coalesce(display_name,'') || ' ' || coalesce(code,'') || ' ' ||
                         coalesce(given,'') || ' ' || coalesce(surname,''))) STORED,

  CONSTRAINT identity_persons_rid_shape
    CHECK (openfaithmap.rid_service(id)=1 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=1)
);
CREATE UNIQUE INDEX identity_persons_code_active_idx
  ON openfaithmap.identity_persons (code) WHERE deleted_at IS NULL AND code IS NOT NULL;
CREATE INDEX identity_persons_search_trgm_idx
  ON openfaithmap.identity_persons USING gin (search_text gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE TRIGGER identity_persons_set_updated_at
  BEFORE UPDATE ON openfaithmap.identity_persons
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- identity_accounts: an optional login attachment to exactly one person (<=1 active per person).
-- Tokens/passwords are never stored — auth is delegated to Google (D-DirectTokenVerification); the
-- dormant credential columns stay CHECK-enforced NULL, same discipline upstream uses.
CREATE TABLE openfaithmap.identity_accounts (
  id              uuid PRIMARY KEY DEFAULT openfaithmap.new_id(1,1,2),  -- identity / object / account
  person_id       uuid NOT NULL REFERENCES openfaithmap.identity_persons(id) ON DELETE RESTRICT,
  email           citext,                       -- IdP-asserted; unique among active when set
  status          text NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
  password_hash   text,
  mfa_enrolled_at timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  deleted_at      timestamptz,

  CONSTRAINT identity_accounts_rid_shape
    CHECK (openfaithmap.rid_service(id)=1 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=2),
  CONSTRAINT identity_accounts_dormant_credentials CHECK (password_hash IS NULL AND mfa_enrolled_at IS NULL)
);
CREATE UNIQUE INDEX identity_accounts_person_active_idx
  ON openfaithmap.identity_accounts (person_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX identity_accounts_email_active_idx
  ON openfaithmap.identity_accounts (email) WHERE email IS NOT NULL AND deleted_at IS NULL;
CREATE TRIGGER identity_accounts_set_updated_at
  BEFORE UPDATE ON openfaithmap.identity_accounts
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- identity_external_identities: a verified (issuer, subject) login point federating to one account.
-- Immutable once created (no UPDATE); an unlink is a hard DELETE.
CREATE TABLE openfaithmap.identity_external_identities (
  id         uuid PRIMARY KEY DEFAULT openfaithmap.new_id(1,1,3),  -- identity / object / external_identity
  account_id uuid NOT NULL REFERENCES openfaithmap.identity_accounts(id) ON DELETE CASCADE,
  issuer     text NOT NULL,   -- the IdP `iss` (https://accounts.google.com — D-GoogleDirect)
  subject    text NOT NULL,   -- the IdP `sub` (pseudonymous identifier)
  created_at timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT identity_external_identities_rid_shape
    CHECK (openfaithmap.rid_service(id)=1 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=3)
);
CREATE TRIGGER identity_external_identities_no_update
  BEFORE UPDATE ON openfaithmap.identity_external_identities
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.reject_mutation();
CREATE UNIQUE INDEX identity_external_identities_issuer_subject_idx
  ON openfaithmap.identity_external_identities (issuer, subject);
CREATE INDEX identity_external_identities_account_idx
  ON openfaithmap.identity_external_identities (account_id);
