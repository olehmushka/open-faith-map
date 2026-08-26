# Module: discovery

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [content](content.md)
> Table prefix: `openfaithmap.discovery_*` (cache only — see below)

## Purpose

The public map + search experience: "find a church near me," filterable by service time, service
language, and tradition — showing a real name, address, and (M13.0) accessibility/online-stream
attributes per result, not just a coordinate. Almost entirely a **read facade** over
`internal/religion`'s (and `internal/location`'s) closure-aware discovery search — this module adds
no location index of its own and never becomes a second source of truth for where a congregation
is. The one thing it owns is a **read-through cache** of that search's results, needed because the
public map is OpenFaithMap's single highest-traffic, mostly-anonymous surface and re-running the
underlying PostGIS query on every map pan/zoom would be wasteful. (D-Facade, the original framing
for this when the search lived in a separate go-oikumenea service, is superseded by D-OwnCore — see
the M10.6 note below; the facade *shape* is unchanged, only the fact that it's now an in-process
call rather than a network one.)

## Redesign (M4, 2026-08-10) — cache-only public reads, resolving M2.5's finding

M2.5 measured two suspected failure modes live; one is fixed upstream, one is permanent by design,
and this section is the redesign both required, closing M4's previously reopened `designed` gate.

1. **The cache refresh path — fixed upstream, now buildable.** A service principal holding the
   `religion.read` grant used to get `403 Authorization:PermissionDenied` on both
   `GET /religion/v1/discovery/sites` and `GET /religion/v1/taxa/{id}` (`RequireAnywhere` denied
   machine subjects regardless of grants). Filed as
   [go-oikumenea#33](https://github.com/olehmushka/go-oikumenea/issues/33); fixed same-day
   (`fedc094`, released as `docker.io/olegamysk/oikumenea:0.0.2`) — reads now dispatch through
   `RequireServiceOrPerson`. Re-verified live against `0.0.2`: the same service-principal token now
   gets `200`/correct `404`. `internal/discovery`'s service-principal client
   (`coreintegration.NewServiceClient`, built at M1, unused in production code until now) genuinely
   works against both endpoints.
2. **The public read path is permanently denied — by design, not a gap.** An unauthenticated
   `GET /religion/v1/discovery/sites` still returns `401 IdentityFederation:Unauthorized` — denied
   at **authentication**, before any authorization check runs — and #33 explicitly scoped genuine
   anonymous access out as "a separate, larger design question" that this module does not wait on.
   `openfaithmap-web` has no token to forward by design (D-AdminSurface): **it must never call
   go-oikumenea directly for discovery reads, full stop** — it only ever calls
   `openfaithmap-api`'s own `DiscoveryPublicService` (`GET /discovery/v1/search`, below), which is
   backed entirely by `discovery_site_cache`.

**The resolved design.** `GET /discovery/v1/search` reads `discovery_site_cache` first. On a cache
miss (or an explicitly stale row), `internal/discovery`'s application layer calls go-oikumenea
*itself*, server-side, via the service principal — not on behalf of the anonymous caller, who never
had a token to begin with — upserts the result into `discovery_site_cache`, and returns it in the
same request. **Refresh cadence: lazy-only, no scheduled job (`DS-OFM-2` resolved for MVP).** There
is no cron/timer component refreshing regions proactively; the cache populates itself purely as a
side effect of real traffic. This is deliberately simpler than the original "background job on a
timer" sketch — a proactive scheduled refresh is deferred until real usage shows staleness is a
problem worth the extra moving part, not designed speculatively here. A separate, authenticated
`POST /discovery/v1/refresh` remains as an operator tool for forcing a rebuild (e.g. after a bulk
content change), gated by the same target-scoped `registration-operator` check
`content.manage` already reuses (no new go-oikumenea permission).

> **Update (2026-08-10): live verification found a third layer, one level deeper than #33 —
> filed as [go-oikumenea#34](https://github.com/olehmushka/go-oikumenea/issues/34).** Brought up a
> real `docker compose` stack and drove `GET /discovery/v1/search` end-to-end against a real,
> `visibility='public'`, non-deleted `religion_sites` row (confirmed present by direct SQL). The
> service-principal call **succeeded at the API layer** (#33's fix genuinely works — no `403`,
> no error) but returned `200 {"sites":[]}` regardless of query params, including a radius search
> centered exactly on the known site. Root cause, confirmed by inspecting the RLS policy and
> `authz_principal_grants` directly: `religion_sites` has RLS force-enabled, and for a
> service-principal subject it falls through to `authz_principal_org_in_reach`, which requires
> `pg.org_id = org` — an exact match. Our principal's `religion.read` grant has `org_id IS NULL`
> (correctly, per `religion.md:336`'s own "instance-wide" description of that permission) — and
> `NULL = <uuid>` is never `true` in SQL, so an instance-wide grant is **structurally invisible**
> to this RLS function. Unlike the person-shaped path (`authz_unit_in_reach` has an explicit
> `app.is_instance_admin` bypass), there is no equivalent "instance-wide grant" bypass for a
> service principal. Net effect: the PEP (API-layer) check now correctly says yes, but RLS
> silently returns nothing underneath it, for any org, no matter how much real data exists.
>
> **This did not reopen the redesign above or require any code change in this repo** — the
> cache-only architecture was already correct, and `internal/discovery` was already making the
> right calls the right way (confirmed via a temporary diagnostic that propagated the upstream
> error instead of swallowing it: the service-principal call itself returned no error, only an
> empty result).
>
> **Update: fixed upstream the same day, released as `oikumenea:0.0.3`.** Re-ran the identical
> live repro against `0.0.3` (`docker-compose.yml`'s pin bumped to match): the same request that
> returned `{"sites":[]}` on `0.0.2` now returns the real, previously-invisible site, with the
> correct `contentSiteId` resolved via `internal/discovery`'s `ContentResolver` interface-call to
> `content` — proving the cross-module FK wiring end-to-end, not just the go-oikumenea half.
> `discovery_site_cache` confirmed populated with real data in Postgres directly. One more real
> bug surfaced by this same live pass, unrelated to go-oikumenea: `adapters.UpsertRow` returned the
> pre-insert struct (empty `id`, zero `refreshedAt`) instead of the persisted row — harmless until a
> caller uses `id` as a list key (the map UI does, for React), fixed by having `UpsertRow` return
> the `RETURNING` row instead of its input.
>
> **Fully live-verified end-to-end**, not just at the API boundary: `openfaithmap-web`'s actual
> home page (`GET /`) was fetched over HTTP and confirmed to embed the real site's data
> (`latitude`/`longitude`/`contentSiteId`) in its server-rendered payload, and the per-congregation
> page (`GET /congregations/{unitId}`) rendered the real published page's blocks
> (`<h1>Welcome</h1><p>This is the home page.</p>`) sourced from a real `content_sites`/
> `content_documents`/`content_blocks` row created in an earlier session. A second real bug
> surfaced only by loading the actual page, not `next build`'s static-page check: Leaflet touches
> `window` at module-evaluation time and crashed SSR outright
> (`ReferenceError: window is not defined`, HTTP 500) — fixed by loading `DiscoveryMap` through
> `next/dynamic({ ssr: false })` via a small client-component wrapper
> (`app/discovery-map-loader.tsx`), since `ssr: false` isn't usable directly from `page.tsx`'s
> Server Component. `Verified` flips in the stage board above.

> **Note (M10.6, 2026-08-18): go-oikumenea is gone — everything above is historical.** M10 absorbed
> go-oikumenea in-process (D-OwnCore): `internal/religion`, `internal/location`, and
> `internal/directory` are now this repo's own modules, no longer a separate service reached over
> HTTP. `internal/discovery`'s cache-miss/refresh path now calls `internal/religion.SearchSites`
> directly in-process (`authz.SystemContext`, D-InProcessAuthz amendment #5) — no service
> principal, no `coreintegration.NewServiceClient`, no network hop. The redesign's *shape*
> (cache-first, lazy-refresh-on-miss, no scheduled job) is unchanged and still accurate; only the
> mechanism behind "calls go-oikumenea" changed. Frozen above, not edited, per this repo's
> append-only-correction convention (see `docs/milestones.md`'s own audit-pass note); read every
> "go-oikumenea" reference below this point as "`internal/religion`/`internal/location`" unless a
> later note says otherwise.

## Entities & aggregates

- **Discovery cache row** — a denormalized, short-TTL projection of one `internal/religion`
  discovery-search hit (already `public_precision`-coarsened by `religiondomain.Coarsen`/
  `CoarsenAddress`, D-DiscoveryAddressPrecision) plus a link to
  the matching `content_sites` row if the congregation has published pages. **Not authoritative** —
  rebuildable from go-oikumenea + OpenFaithMap's own `content` schema at any time; safe to truncate.

## Data model

**`discovery_site_cache`**
- `id` PK (RID)
- `religion_site_rid TEXT NOT NULL UNIQUE` — `internal/religion`'s `religion_sites.id` this row
  projects (opaque foreign value, still named `_rid` for historical reasons predating M10.6's
  in-process cutover)
- `congregation_unit_rid TEXT NOT NULL`
- `content_site_id` FK → `content_sites` (nullable — a discoverable congregation need not have
  published a site yet)
- `latitude NUMERIC`, `longitude NUMERIC` — **already coarsened** by `religiondomain.Coarsen` per
  `public_precision`; never a finer-grained value than `internal/religion` itself would return
  publicly
- `name TEXT NOT NULL` (M13.0) — the congregation's display name (`directory_units.name`), shown
  regardless of precision (D-DiscoveryAddressPrecision)
- `address_line TEXT` (M13.0) — precision-coarsened address text, `religiondomain.CoarsenAddress`'s
  output; nullable (`hidden`-precision sites, or a site whose location has no address fields set,
  produce no line)
- `tradition_taxon_id TEXT`, `tradition_taxon_code TEXT`, `tradition_taxon_name TEXT` (code/name
  added M13.0) — denormalized from `religion_org_classifications`' primary tag, for fast
  filter-by-tradition and display without a second lookup
- `service_languages TEXT[]`, `service_days SMALLINT[]` — denormalized from
  `religion_service_schedules` for fast filtering without a join fan-out per search
- `attributes JSONB NOT NULL DEFAULT '{}'` (M13.0) — mirrors `religion_sites.attributes` (see
  below); accessibility criteria + online-stream flag, passed through unfiltered by precision
- `refreshed_at TIMESTAMPTZ NOT NULL`
- No soft-delete — a stale row is simply overwritten or pruned on refresh; this table carries no
  history and is never audited (it is a cache, not a record).

**`religion_sites.attributes JSONB NOT NULL DEFAULT '{}'`** (M13.0, `internal/religion`'s own
table, not this module's) — `religiondomain.SiteAttributes`'s shape: named accessibility criteria
(step-free entrance, accessible restroom, hearing loop, sign-language interpretation, accessible
parking, wheelchair seating, braille/large-print) plus an `onlineStream` boolean. Validated/shaped
in Go, not a DB CHECK (matches this repo's existing convention for structured JSONB, e.g.
`directory_units.metadata`). A `GIN (attributes jsonb_path_ops)` index backs the containment
(`@>`) filtering M13.1 adds. Every criterion defaults to unset (not a false claim of
inaccessibility) until an operator sets it via the admin UI (M13.2).

## Conjure API surface

`api/discovery.conjure.yml`, split the same way `content` split at M3 (Conjure's auth is a fixed
per-service choice, not a per-endpoint runtime one — see [content.md](content.md)):

| Service | Op | Intent | Auth |
|---|---|---|---|
| `DiscoveryPublicService` (`/discovery/v1`) | `GET /search?lat=&lng=&radiusM=&tradition=&language=&dayOfWeek=&query=` | Public map/list search. Reads `discovery_site_cache` first; on a miss, calls `internal/religion.SearchSites` live in-process, upserts the cache, returns the result in the same request. | none — no `bearertoken.Token` param at all |
| `DiscoveryPublicService` (`/discovery/v1`) | `GET /sites/{unitId}` (M13.0) | A single congregation's discoverable site, always live (never the cache) — the per-congregation detail page's server-rendered fetch. Throws `SiteNotFound` if the unit has no public, non-hidden site. | none |
| `DiscoveryService` (`/discovery/v1`) | `POST /refresh` | Force a cache rebuild for a region (operator tool, not public) | header — target-scoped check, see below |

`GET /search` is the only endpoint most traffic ever hits. It never widens what
`internal/religion.SearchSites` would already return publicly — the cache is a performance layer,
not a privacy boundary of its own; the `public_precision` coarsening it stores came from
`internal/religion` already coarsened.

## Dependencies

- **Calls:** `internal/religion.SearchSites` in-process (no network hop since M10.6 — see the note
  above), and OpenFaithMap's own [content](content.md) module (to know which cached sites have a
  published page worth linking to, via the `content_site_id` FK).
- **Called by:** `openfaithmap-web`'s public map/search page and per-congregation page
  ([web-facade.md](web-facade.md)) — its only calls into `openfaithmap-api`, and always through the
  generated TypeScript client, never a raw `fetch`.

## Authorization touchpoints

`POST /refresh` reuses `content.manage`'s target-scoped-capability pattern
([content.md](content.md), D-PlatformModerator) checked against `Config.RootUnitID` — the same
`registration-operator` subtree grant M2.3 already established. Everything else is intentionally
**unauthenticated** — discovery is the public product surface; the privacy boundary
(`public_precision`) is enforced inside `internal/religion.SearchSites` before data ever reaches
this cache, not re-applied here.

## Invariants

- **Never store finer-grained coordinates or address text than `internal/religion` returned
  publicly.** The cache persists exactly what `SearchSites` returned — coarsening
  (`religiondomain.Coarsen`/`CoarsenAddress`) is done once, at the source; this module never has
  access to (and so cannot leak) an un-coarsened coordinate or address for a `hidden`/`city`-
  precision site.
- **The cache is disposable.** No migration ever needs to "fix" cached data — the correct
  operation for any inconsistency is a refresh, never a manual row edit.
- **A cache miss never blocks the user.** On miss, `GET /search` falls through to a live
  `SearchSites` call rather than returning an error or a stale empty result.

## Open seams

- **Refresh cadence — resolved (M4, 2026-08-10): lazy-only, no scheduled job.** See the redesign
  section above. A future webhook-driven or timer-driven proactive refresh remains a real option,
  but is deliberately not designed here; revisit only once real traffic patterns show the lazy
  cache-miss path leaves results stale in practice.
- **`discovery_site_cache.content_site_id` is a real FK to `content_sites` — resolved (M4,
  2026-08-10): the FK is fine.** Both tables live in one schema and one deployable;
  [conventions.md](../architecture/conventions.md)'s cross-module rule is scoped to cross-*service*
  boundaries (interface calls, domain events across a network hop), not schema-level foreign keys
  inside one owned database. M3 already set this precedent (`content_documents.site_id →
  content_sites`, `content_blocks.document_id → content_documents`). The FK is `ON DELETE SET
  NULL` — deleting a `content_sites` row (rare; sites aren't currently deletable) unlinks the cache
  row rather than cascading, since the cache row itself is still valid data even without a
  published site to link to. `DS-OFM-13` closed.
- **Content full-text search** (searching page bodies, not just location) is explicitly out of
  scope here — see [content.md](content.md#open-seams).
- **A dedicated map tile/vector layer** for high-density regions (many congregations in one city)
  is deferred until real usage data shows the naive point-list approach doesn't scale.
- **Congregation name/address were entirely absent from the public projection — resolved (M13.0,
  2026-08-26).** `DiscoverySite`/`CacheRow` now carry `name`, precision-coarsened `address`, a
  primary tradition tag (code/name, not just an id), and `attributes` (accessibility criteria +
  online-stream flag) — closing a gap this doc never actually named (found by M13's discovery
  pass, not tracked here beforehand). See D-DiscoveryAddressPrecision
  (`architecture/decisions.md`) for the address/name precision-gating rule.
- **Bounding-box search, `onlineOnly`, server-side pagination, accessibility filtering, and
  facet/filter-value discovery** are all real gaps `GET /search` still doesn't cover — M13.0 added
  the *data* (`attributes`) but not yet the *filter*; M13.1 is scoped to add
  `DiscoveryQuery.Accessibility`/`OnlineOnly` (JSONB containment against the new GIN index), a
  facets endpoint, and a real fix for the pre-existing `tradition` filter bug (the query param is
  documented as a taxon *code* but the adapter binds it straight into a `uuid` column with no
  code→id resolution — likely silently matches nothing today; zero test coverage of this filter
  existed before M13.0's own SearchSites integration-test additions started exercising the search
  path more thoroughly). Also still real: cache-side filtering by tradition/language/dayOfWeek —
  `SearchQuery.BypassesCache` still routes all of these live even though `CacheRow` now reliably
  carries the data, since no matching predicate exists yet in `filterByRadius`
  (`internal/discovery/application/service.go`) and an older cached row from before M13.0 shipped
  still has them empty until its next refresh.
