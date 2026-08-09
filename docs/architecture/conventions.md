# Conventions

OpenFaithMap's own backend (`openfaithmap-api`, owning `content`/`moderation`/`vouching`) follows
go-oikumenea's schema, Go, Conjure, and API conventions **by reference** — restating them here
would just drift out of sync with the upstream source of truth. This doc records only the handful
of choices specific to OpenFaithMap.

## Inherited unchanged from go-oikumenea

- Composed URN **RID** primary keys via `new_rid()` (or its OpenFaithMap-local equivalent —
  see below), `TIMESTAMPTZ` UTC everywhere, soft-delete (`deleted_at`), a `set_updated_at()`
  trigger, `TEXT` + `CHECK` for fixed enums (never native Postgres enums), a `reject_mutation()`
  guard on append-only tables (moderation actions, vouching edges).
  See go-oikumenea's own `docs/architecture/conventions.md` for the exact SQL patterns.
- Hexagonal layering per module: `transport → application → domain → adapters`, domain owns its
  interfaces, imports no framework.
- Cross-module queries inside `openfaithmap-api` are direct interface calls; cross-module
  mutations are domain events — same rule go-oikumenea applies inside its own monolith.
- Conjure-first: `api/<module>.conjure.yml` is the source of truth, generated Go/TypeScript code is
  never hand-edited. *(Generated **Go** is real today. Generated **TypeScript** does not exist —
  there is no codegen pipeline for `openfaithmap-api` yet, so `web/apps/admin/lib/registration.ts`
  is hand-written. [milestones.md](../milestones.md)'s M2.6.)*
- Atlas versioned migrations, one repo-root `migrations/` directory, expand/contract releases, a
  boot-time schema-version check. *(The boot-time check is not implemented —
  `cmd/openfaithmap-api` reads `DATABASE_URL` and opens a pool with no schema-version assertion.
  Migrations are applied out-of-band by `docker-compose.yml`'s `openfaithmap-migrate`.)*
- pgx **+ sqlc** for queries. *(`internal/registration/adapters/store.go` is hand-written pgx — a
  documented, deliberate simplification for a single-table module, not a new convention. See
  [registration.md](../modules/registration.md)'s open seams.)*

## OpenFaithMap-specific

- **Service naming: `openfaithmap-<noun>`.** One deployable per noun —
  `openfaithmap-web` (public UI), `openfaithmap-admin` (verified UI, D-AdminSurface),
  `openfaithmap-api` (backend). Precedent-only until D-AdminSurface; codified here now that a
  second UI service exists.
- **Schema name: `openfaithmap`**, in the **same Postgres instance** as `oikumenea`
  ([D-SharedDatabase](decisions.md)) — not a separate database instance, as earlier drafts of these
  docs implied. Table prefixes per module: `openfaithmap.content_*`, `openfaithmap.moderation_*`,
  `openfaithmap.vouching_*`, `openfaithmap.registration_*` — mirroring go-oikumenea's
  `oikumenea.<module>_*` pattern one level down. Everything below about "a different database"
  still holds as written: the boundary is real, it is just a schema boundary rather than an
  instance boundary. `openfaithmap-api` connects as a least-privilege role scoped to the
  `openfaithmap` schema (M2.4, D-SharedDatabase) — the boundary is now enforced at the database
  level, not just by convention.
- **RID slot allocation.** OpenFaithMap mints its own RIDs for its own entities
  (`content_pages`, `moderation_reports`, `vouching_edges`, …) — never in go-oikumenea's RID
  space. Where OpenFaithMap's tables reference a go-oikumenea entity (a congregation `Unit`, a
  `Person`), the column stores go-oikumenea's **RID as an opaque `TEXT` foreign value** — never a
  local foreign key, even though the referenced row now lives one schema away rather than one
  database away. There is no cross-schema FK integrity; OpenFaithMap treats a dangling reference as
  "the unit/person no longer exists" and handles it at read time (soft 404, not a crash).
  *(Note: `registration_requests` uses a `uuid` PK, not a composed URN RID — a deviation from the
  inherited convention above, undocumented until the 2026-08-09 audit. Decide at M3 whether new
  modules follow `registration`'s uuid precedent or go-oikumenea's RID convention, and make them
  consistent.)*
- **Authorization for OpenFaithMap-owned tables is a target-scoped capability check** against
  go-oikumenea's PDP — "does this caller hold *this authority* over *this unit*" — never an
  untargeted "holds P anywhere" check, and never a successful read treated as proof of write
  standing. Both anti-patterns have already occurred in this repo; see
  [D-PlatformModerator](decisions.md) and
  [core-integration.md](../modules/core-integration.md#authorization-touchpoints) for the pattern
  and the two worked examples.
- **Cross-module foreign keys inside `openfaithmap-api` are undecided.**
  `discovery_site_cache.content_site_id` is a real FK into `content_sites` — a schema-level
  coupling between two modules, where the rules above only cover cross-*service* references and the
  layering rules only cover Go-level calls. Settle before M3/M4 add more; see `DS-OFM-13`.
- **No RLS-based tenant isolation.** OpenFaithMap's own tables are scoped by a go-oikumenea unit
  RID column, but access control for that column is enforced at the application layer against
  go-oikumenea's PDP response (see [core-integration.md](../modules/core-integration.md)) — there
  is no Postgres RLS policy keyed on it, because OpenFaithMap's database has no notion of "which
  unit can this connection see" the way go-oikumenea's `app.readable_units` GUC does. This is a
  **known, accepted gap** relative to go-oikumenea's defense-in-depth posture — see
  [open-questions.md](../open-questions.md).
- **i18n.** Content translation groups (see [content.md](../modules/content.md)) are OpenFaithMap's
  own mechanism, not go-oikumenea's `locale → text` translation store — a content translation is a
  *separate document*, not a label on one row, since a page's blocks differ per locale, not just
  its text.
- **Block schema validation.** Each block type's shape is validated against a JSON Schema stored
  alongside the block-type catalog (the same `attr_schema JSONB` pattern go-oikumenea uses for
  `tenant_unit_kinds`/`document_types`), not a Go struct per type — new block types are a catalog
  row + schema, not a code change, matching go-oikumenea's own "catalog data, not a code branch"
  philosophy wherever the vocabulary might grow.
