-- Runs once, only when Postgres initializes a genuinely empty data directory
-- (docker-entrypoint-initdb.d semantics) — before openfaithmap-migrate ever connects. A no-op on the
-- existing shared dev volume, which already has both extensions from an earlier untracked step.
--
-- migrations/0015_core_identity.sql depends on citext (identity_accounts.email) and pg_trgm
-- (identity_persons' trigram search index); only pgcrypto is created by an Atlas migration
-- (migrations/0001_registration.sql). Atlas migrations are strictly ordered/expand-only, so a fresh
-- low-numbered migration file can't be inserted without breaking the already-applied version sequence
-- on any existing stack — this Postgres-native init step is the correct fix instead.
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
