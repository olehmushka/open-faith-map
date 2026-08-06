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
  never hand-edited.
- Atlas versioned migrations, one repo-root `migrations/` directory, expand/contract releases, a
  boot-time schema-version check.

## OpenFaithMap-specific

- **Schema name: `openfaithmap`.** Table prefixes per module: `openfaithmap.content_*`,
  `openfaithmap.moderation_*`, `openfaithmap.vouching_*` — mirroring go-oikumenea's
  `oikumenea.<module>_*` pattern one level down, in OpenFaithMap's own database.
- **RID slot allocation.** OpenFaithMap mints its own RIDs for its own entities
  (`content_pages`, `moderation_reports`, `vouching_edges`, …) — never in go-oikumenea's RID
  space. Where OpenFaithMap's tables reference a go-oikumenea entity (a congregation `Unit`, a
  `Person`), the column stores go-oikumenea's **RID as an opaque `TEXT` foreign value** — never a
  local foreign key, since the referenced row lives in a different database. There is no
  cross-database FK integrity; OpenFaithMap treats a dangling reference as "the unit/person no
  longer exists" and handles it at read time (soft 404, not a crash).
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
