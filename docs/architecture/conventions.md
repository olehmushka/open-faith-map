# Conventions

OpenFaithMap's own backend (`openfaithmap-api`, owning `content`/`moderation`/`vouching`) follows
go-oikumenea's schema, Go, Conjure, and API conventions **by reference** — restating them here
would just drift out of sync with the upstream source of truth. This doc records only the handful
of choices specific to OpenFaithMap.

## Inherited unchanged from go-oikumenea

- `TIMESTAMPTZ` UTC everywhere, soft-delete (`deleted_at`), a `set_updated_at()` trigger, `TEXT` +
  `CHECK` for fixed enums (never native Postgres enums), a `reject_mutation()` guard on
  append-only tables (moderation actions, vouching edges). See go-oikumenea's own
  `docs/architecture/conventions.md` for the exact SQL patterns.
  *(Corrected at M3, 2026-08-10 — see the RID entry below: the composed-URN-`TEXT` "RID" primary
  key this bullet used to describe is a documented-but-unshipped go-oikumenea redesign, not what
  the actually-deployed schema does. Plain `uuid` PKs are inherited unchanged; the URN scheme is
  not.)*
- Hexagonal layering per module: `transport → application → domain → adapters`, domain owns its
  interfaces, imports no framework.
- Cross-module queries inside `openfaithmap-api` are direct interface calls; cross-module
  mutations are domain events — same rule go-oikumenea applies inside its own monolith.
- Conjure-first: `api/<module>.conjure.yml` is the source of truth, generated Go/TypeScript code is
  never hand-edited. *(~~Generated **Go** is real today. Generated **TypeScript** does not exist —
  there is no codegen pipeline for `openfaithmap-api` yet, so `web/apps/admin/lib/registration.ts`
  is hand-written.~~ **Corrected 2026-08-18:** stale since M2.6 shipped the pipeline. Both exist —
  `scripts/gen-ts-client.sh` generates into `web/apps/{admin,web}/lib/openfaithmap/generated/`, and
  `make sdk-verify` fails on drift. [milestones.md](../milestones.md)'s M2.6.)*
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
- > **Superseded (2026-08-18) by [D-OwnRIDs](decisions.md#d-ownrids--uuidv8-resource-identifiers-owned-by-openfaithmap)
  > and its amendment.** M10 ports go-oikumenea's `new_id(service, kind, type)` function and its
  > per-table structural CHECKs into the `openfaithmap` schema, so new core tables get bit-packed
  > UUIDv8 primary keys rather than `gen_random_uuid()`. The bullet below is still right about the
  > thing it was correcting — the composed-URN-`TEXT` scheme was never go-oikumenea's real
  > convention and is not adopted now either. **Values on the wire stay bare uuids**; D-OwnRIDs'
  > original `ofm:` prefix rendering was dropped precisely because existing `*_rid TEXT` columns and
  > the public congregation URL path segment round-trip bare uuids today. Existing
  > `registration_*`/`content_*`/`moderation_*` tables keep their plain `uuid` PKs unchanged.

  **Primary keys are plain `uuid` (`gen_random_uuid()`), not composed URN RIDs — decided at M3,
  2026-08-10, correcting a premise this bullet held until then.** The prior text described
  OpenFaithMap minting "RIDs" in go-oikumenea's composed-URN-`TEXT` style
  (`urn:oikumenea:<service>:<env>:<entity_type>:<uuid>`, via a `new_rid()` function) and flagged
  `registration_requests`'s plain `uuid` PK as an undocumented deviation from it. Checked directly
  against go-oikumenea's actually-deployed migrations (`migrations/0003_person_membership.sql` and
  every other table in that repo's checked-out `main` branch): every PK is a plain `uuid PRIMARY
  KEY DEFAULT oikumenea.new_id(app,service,kind)` — a bit-packed UUID, not a URN-`TEXT` string; no
  `new_rid()` function exists anywhere in those migrations. The composed-URN scheme is a
  documented-but-unshipped future go-oikumenea redesign (referenced in that repo's own longer-term
  planning docs), not its current convention. **`registration_requests` was never a deviation** —
  it already matched go-oikumenea's real schema. `content_*` (M3) and every module after it use the
  same plain `uuid` PK, for consistency with what go-oikumenea actually does today, not with a
  redesign it hasn't shipped. Revisit if/when go-oikumenea itself ships composed-URN RIDs.

  Separately, **opaque foreign values are unchanged**: where OpenFaithMap's tables reference a
  go-oikumenea entity (a congregation `Unit`, a `Person`), the column stores go-oikumenea's own PK
  as an opaque `TEXT` foreign value — never a local foreign key, even though the referenced row now
  lives one schema away rather than one database away. There is no cross-schema FK integrity;
  OpenFaithMap treats a dangling reference as "the unit/person no longer exists" and handles it at
  read time (soft 404, not a crash).
- **Authorization for OpenFaithMap-owned tables is a target-scoped capability check** against
  ~~go-oikumenea's PDP~~ **OpenFaithMap's own in-process PDP from M10 onward
  ([D-InProcessAuthz](decisions.md#d-inprocessauthz--the-pdp-runs-in-process-app-layer-only)) —
  same rule, same shape, no network hop** — "does this caller hold *this authority* over *this
  unit*" — never an untargeted "holds P anywhere" check, and never a successful read treated as
  proof of write standing.

  **Added 2026-08-18:** the check is `authz.Require(ctx, action, unitID)`, which takes its subject
  from the request context. A subject-parameter form exists (`authz.DecideFor`) for the super-admin
  "what can this person do" screen only, and is itself gated on the instance-admin plane. This is
  not a style preference — a subject parameter makes the PDP an oracle over arbitrary subjects,
  safe only by call-site convention, which is the same defect class as the two worked examples
  below. Both anti-patterns have already occurred in this repo; see
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
  unit can this connection see" the way go-oikumenea's `app.readable_units` GUC does. ~~This is a
  **known, accepted gap** relative to go-oikumenea's defense-in-depth posture~~ — see
  [open-questions.md](../open-questions.md).

  **Settled 2026-08-18 (`DS-OFM-1` closed): no longer a gap relative to anything, because there is
  no longer a second posture to be relative to.** From M10, OpenFaithMap owns every table, and
  [D-InProcessAuthz](decisions.md#d-inprocessauthz--the-pdp-runs-in-process-app-layer-only) chooses
  app-layer-only deliberately. The choice is only sound because that decision's amendment **also
  drops the grant cache** and reads grants per request: upstream's cache is documented as safe
  precisely because an exact/live RLS backstop sits underneath it, so keeping the cache while
  dropping RLS would have left a 2-second stale-ALLOW window with no floor. App-layer-only plus
  no cache is coherent; app-layer-only plus a cache would not have been.
- **i18n.** Content translation groups (see [content.md](../modules/content.md)) are OpenFaithMap's
  own mechanism, not go-oikumenea's `locale → text` translation store — a content translation is a
  *separate document*, not a label on one row, since a page's blocks differ per locale, not just
  its text.
- **Block schema validation.** Each block type's shape is validated against a JSON Schema stored
  alongside the block-type catalog (the same `attr_schema JSONB` pattern go-oikumenea uses for
  `tenant_unit_kinds`/`document_types`), not a Go struct per type — new block types are a catalog
  row + schema, not a code change, matching go-oikumenea's own "catalog data, not a code branch"
  philosophy wherever the vocabulary might grow.
