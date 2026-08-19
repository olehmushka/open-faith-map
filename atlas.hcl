// Atlas configuration for open-faith-map (M2, D-Stack — same toolchain go-oikumenea itself used).
// Versioned migrations in migrations/, applied to the one openfaithmap schema in this stack's one
// Postgres instance (docker-compose.yml's header comment — M10.8 dropped the second, oikumenea,
// schema this comment used to describe).
//
//   set -a; . ./.env; set +a
//   atlas migrate hash  --env local
//   atlas migrate apply --env local

locals {
  db_url = getenv("DATABASE_URL") != "" ? getenv("DATABASE_URL") : "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
}

env "local" {
  url = local.db_url

  migration {
    dir = "file://migrations"
    // Stored inside the openfaithmap schema itself (created by migration 0001), same rationale as
    // go-oikumenea's own atlas.hcl: avoids a standalone atlas_schema_revisions schema.
    revisions_schema = "openfaithmap"
  }
}
