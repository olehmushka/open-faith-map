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

`DiscoveryService` (`/discovery/v1`), `api/discovery.conjure.yml`:

| Op | Intent | Perm |
|---|---|---|
| `GET /search?lat=&lng=&radiusM=&tradition=&language=&dayOfWeek=&query=` | Public map/list search. Reads `discovery_site_cache` first; falls back live to go-oikumenea's `GET /religion/discovery/sites` on a cache miss or explicit `fresh=true` | none (public) |
| `POST /refresh` | Force a cache rebuild for a region (operator/admin tool, not public) | `discovery.refresh` (platform) |

`GET /search` is the only endpoint most traffic ever hits. It never widens what go-oikumenea would
already return publicly — the cache is a performance layer, not a privacy boundary of its own; the
`public_precision` coarsening it stores came from go-oikumenea already coarsened.

## Dependencies

- **Calls:** go-oikumenea's `religion` module (`GET /religion/discovery/sites`, the primary data
  source), `location` (indirectly, through the religion endpoint — never called directly), and
  OpenFaithMap's own [content](content.md) module (to know which cached sites have a published
  page worth linking to).
- **Called by:** the [web-facade](web-facade.md) public map/search pages.

## Authorization touchpoints

`discovery.refresh` (platform-scoped, operator tooling only — forcing a manual cache rebuild).
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

- **Refresh cadence** (background job on a timer vs. webhook-driven from go-oikumenea) is not
  decided — go-oikumenea has no outbound webhook for religion-site changes today; scheduled
  polling via the service principal is the interim default (see
  [core-integration.md](core-integration.md)).
- **Content full-text search** (searching page bodies, not just location) is explicitly out of
  scope here — see [content.md](content.md#open-seams).
- **A dedicated map tile/vector layer** for high-density regions (many congregations in one city)
  is deferred until real usage data shows the naive point-list approach doesn't scale.
