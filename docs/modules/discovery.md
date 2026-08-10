# Module: discovery

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [content](content.md)
> Table prefix: `openfaithmap.discovery_*` (cache only — see below)

## Purpose

The public map + search experience: "find a church near me," filterable by service time, service
language, and tradition. Almost entirely a **read facade** over go-oikumenea's `religion`,
`location`, and `search` modules (D-Facade) — OpenFaithMap adds no location index of its own and
never becomes a second source of truth for where a congregation is. The one thing it owns is a
**read-through cache** of go-oikumenea's discovery results, needed because the public map is
OpenFaithMap's single highest-traffic, mostly-anonymous surface and re-querying go-oikumenea's
PostGIS discovery search on every map pan/zoom would be wasteful for both services.

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

## Entities & aggregates

- **Discovery cache row** — a denormalized, short-TTL projection of one go-oikumenea
  `religion_sites` row (already `public_precision`-coarsened by go-oikumenea itself) plus a link to
  the matching `content_sites` row if the congregation has published pages. **Not authoritative** —
  rebuildable from go-oikumenea + OpenFaithMap's own `content` schema at any time; safe to truncate.

## Data model

**`discovery_site_cache`**
- `id` PK (RID)
- `religion_site_rid TEXT NOT NULL UNIQUE` — the go-oikumenea `religion_sites` RID this row
  projects (opaque foreign value)
- `congregation_unit_rid TEXT NOT NULL`
- `content_site_id` FK → `content_sites` (nullable — a discoverable congregation need not have
  published a site yet)
- `latitude NUMERIC`, `longitude NUMERIC` — **already coarsened** by go-oikumenea per
  `public_precision`; never a finer-grained value than go-oikumenea itself would return publicly
- `tradition_taxon_id TEXT` — denormalized from `religion_org_classifications` (primary tag) for
  fast filter-by-tradition
- `service_languages TEXT[]`, `service_days SMALLINT[]` — denormalized from
  `religion_service_schedules` for fast filtering without a join fan-out per search
- `refreshed_at TIMESTAMPTZ NOT NULL`
- No soft-delete — a stale row is simply overwritten or pruned on refresh; this table carries no
  history and is never audited (it is a cache, not a record).

## Conjure API surface

`api/discovery.conjure.yml`, split the same way `content` split at M3 (Conjure's auth is a fixed
per-service choice, not a per-endpoint runtime one — see [content.md](content.md)):

| Service | Op | Intent | Auth |
|---|---|---|---|
| `DiscoveryPublicService` (`/discovery/v1`) | `GET /search?lat=&lng=&radiusM=&tradition=&language=&dayOfWeek=&query=` | Public map/list search. Reads `discovery_site_cache` first; on a miss, calls go-oikumenea live via the service principal, upserts the cache, returns the result in the same request. | none — no `bearertoken.Token` param at all |
| `DiscoveryService` (`/discovery/v1`) | `POST /refresh` | Force a cache rebuild for a region (operator tool, not public) | header — target-scoped check, see below |

`GET /search` is the only endpoint most traffic ever hits. It never widens what go-oikumenea would
already return publicly — the cache is a performance layer, not a privacy boundary of its own; the
`public_precision` coarsening it stores came from go-oikumenea already coarsened.

## Dependencies

- **Calls:** go-oikumenea's `religion` module (`GET /religion/v1/discovery/sites`,
  `GET /religion/v1/taxa/{id}` for tradition resolution — both via
  `coreintegration.NewServiceClient`, the service principal, never a forwarded user token) and
  OpenFaithMap's own [content](content.md) module (to know which cached sites have a published
  page worth linking to, via the `content_site_id` FK).
- **Called by:** `openfaithmap-web`'s public map/search page and per-congregation page
  ([web-facade.md](web-facade.md)) — its only calls into `openfaithmap-api`, and always through the
  generated TypeScript client, never a raw `fetch`.

## Authorization touchpoints

`POST /refresh` reuses `content.manage`'s target-scoped-capability pattern
([content.md](content.md), D-PlatformModerator) checked against `Config.RootUnitID` — the same
`registration-operator` subtree grant M2.3 already established, not a new go-oikumenea permission.
Everything else is intentionally **unauthenticated** — discovery is the public product surface;
the privacy boundary (`public_precision`) is enforced by go-oikumenea before data ever reaches this
cache, not re-applied here.

## Invariants

- **Never store finer-grained coordinates than go-oikumenea returned publicly.** The cache
  persists exactly what `GET /religion/discovery/sites` returned — coarsening is go-oikumenea's
  job, done once, upstream; this module never has access to (and so cannot leak) an
  un-coarsened coordinate for a `hidden`/`city`-precision site.
- **The cache is disposable.** No migration ever needs to "fix" cached data — the correct
  operation for any inconsistency is a refresh, never a manual row edit.
- **A cache miss never blocks the user.** On miss, `GET /search` falls through to a live
  go-oikumenea call rather than returning an error or a stale empty result.

## Open seams

- **Refresh cadence — resolved (M4, 2026-08-10): lazy-only, no scheduled job.** See the redesign
  section above. A future webhook-driven or timer-driven proactive refresh remains a real option —
  go-oikumenea still has no outbound webhook for religion-site changes today — but is deliberately
  not designed here; revisit only once real traffic patterns show the lazy cache-miss path leaves
  results stale in practice.
- **`discovery_site_cache.content_site_id` is a real FK to `content_sites` — resolved (M4,
  2026-08-10): the FK is fine.** Both tables live in one schema and one deployable;
  [conventions.md](../architecture/conventions.md)'s cross-module rule is scoped to cross-*service*
  boundaries (interface calls, domain events across a network hop), not schema-level foreign keys
  inside one owned database. M3 already set this precedent (`content_documents.site_id →
  content_sites`, `content_blocks.document_id → content_documents`). The FK is `ON DELETE SET
  NULL` — deleting a `content_sites` row (rare; sites aren't currently deletable) unlinks the cache
  row rather than cascading, since the cache row itself is still valid go-oikumenea data even
  without a published site to link to. `DS-OFM-13` closed.
- **Content full-text search** (searching page bodies, not just location) is explicitly out of
  scope here — see [content.md](content.md#open-seams).
- **A dedicated map tile/vector layer** for high-density regions (many congregations in one city)
  is deferred until real usage data shows the naive point-list approach doesn't scale.
- **Bounding-box search, `onlineOnly`, and server-side pagination** are all real params on
  go-oikumenea's own `searchSites` that this module's `GET /search` does not yet expose — MVP
  mirrors only `lat`/`lng`/`radiusM`/`tradition`/`language`/`dayOfWeek`/`query`, matching what the
  public map's first cut actually needs. Add the rest if/when the UI needs them.
