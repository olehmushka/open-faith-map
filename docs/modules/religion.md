# Module: religion

> Reads: [glossary](../glossary.md) · [architecture/conventions](../architecture/conventions.md) ·
> [architecture/decisions](../architecture/decisions.md)
> Table prefix: `openfaithmap.religion_*`

## Purpose

The faith taxonomy and a congregation's physical/online presence: the curated `religion_taxa` tree
(with a maintained closure table for ancestor/descendant queries), a unit's org profile and
tradition classifications, the `excludes_child_creation` policy check, and `religion_sites` — the
location(s)/visibility/precision/attributes a discovery search or a congregation's own admin form
reads. `internal/religion` (`internal/religion/{domain,application,adapters}`) has existed since
M10's core absorption, called in-process by `content`, `discovery`, `registration`, and
`congregationimport` — **M13.2 is its first transport layer**, `ReligionService`
(`api/religion.conjure.yml`, `internal/religion/transport`), a direct authenticated entrypoint
rather than always being reached through some other module's own application layer.

## Entities & aggregates

- **Taxon** — one node in the static, curated faith taxonomy (`religion_taxa`), with a rank
  (religion/tradition/denomination/etc.) and a maintained closure table
  (`religion_taxa_closure`) backing ancestor/descendant queries (e.g. discovery's `tradition=`
  filter). Seeded, not user-editable (`migrations/0014_core_religion.sql`).
- **OrgProfile** — the 1:1 faith attributes of a religious-body unit: org kind, short code.
- **OrgClassification** — a tradition tag on a unit (`religion_org_classifications`), pointing at a
  `Taxon`; a unit may hold several, at most one marked primary.
- **Site** (`religion_sites`) — a religious body's physical/online presence: a location, a site
  type (church/mission/etc.), `visibility`/`public_precision` (the privacy-tier gate discovery's own
  search predicate enforces), and `attributes` (M13.0's JSONB column — accessibility criteria +
  online-stream flag, `SiteAttributes`). A congregation has at most one **primary** site in
  practice (`religion_sites_one_primary`, a unique partial index on `org_unit_id WHERE is_primary`)
  — every write path in this codebase (`registration`'s `ensureSite`, `congregationimport`'s
  provisioning, M13.2's own `GetSiteByUnit`/`UpdateSiteAttributes`) resolves "the unit's site" by
  preferring the primary row.

## Data model

Conventions per [conventions.md](../architecture/conventions.md): plain `uuid` PKs, `TIMESTAMPTZ`,
`set_updated_at`, soft-delete.

**`religion_sites`**
- `id` PK · `org_unit_id`, `location_id`, `site_type_id` FKs
- `visibility TEXT`, `public_precision TEXT` — the coarsening tier `religiondomain.Coarsen`/
  `CoarsenAddress` (see [discovery.md](discovery.md), D-DiscoveryAddressPrecision) key off
- `is_primary BOOLEAN` — at most one `true` per `org_unit_id` (partial unique index)
- `attributes JSONB NOT NULL DEFAULT '{}'` (M13.0) — `SiteAttributes`'s shape: seven named
  accessibility criteria plus `onlineStream`, GIN-indexed (`religion_sites_attributes_gin`,
  `jsonb_path_ops`) for M13.1's containment filtering
- timestamps + soft-delete

`religion_taxa`/`religion_taxa_closure`/`religion_org_profiles`/`religion_org_classifications`/
`religion_org_policies`/`religion_policy_kinds`/`religion_site_types`/`religion_service_schedules`
round out the schema — all seeded or written only through `content`/`registration`/
`congregationimport`'s own flows; see `migrations/0014_core_religion.sql`.

## Conjure API surface

**`ReligionService`** (`api/religion.conjure.yml`, `/religion/v1`, `default-auth: header` —
`openfaithmap-admin` only; M13.2, religion's first transport layer):

| Op | Intent | Perm |
|---|---|---|
| `GET /units/{unitId}/site` | The unit's primary site, exact/uncoarsened — an owner's own private view, unlike discovery's public-precision-filtered `DiscoverySite` | `site.manage` (on the unit) |
| `PUT /units/{unitId}/site/attributes` | Overwrite the unit's primary site's attributes wholesale (the admin form always submits the complete `SiteAttributes` shape, never a partial patch) | `site.manage` (on the unit) |

Both throw `Religion:SiteNotFound` if the unit has no `religion_sites` row at all — site creation
stays `registration`'s/`congregationimport`'s own job (`ensureSite`), never this service's; a
`religion_sites` row with `attributes` still permanently empty until an admin visits and saves is
the M13.0→M13.2 gap this ticket closes.

Every other religion capability (taxon lookups, org profile/classification management, site
creation, the public discovery search) is still reached in-process only — via `content`'s
`ContentService`, `core`'s `CoreService` (`createChildOrg`/`getOrgProfile`/`listTaxa`/etc.),
`registration`'s approval flow, or `discovery`'s `DiscoveryPublicService` — never through
`ReligionService` itself. This is deliberate, not an oversight: those write paths already have an
established, working caller-side authorization gate (`content.manage`'s `requireManage`, `core`'s
`requireReligionOrgManage`, registration's `requireOperator`), and moving them onto a new service
mid-milestone would be scope creep this ticket didn't need.

## Dependencies

- **Calls:** nothing outside its own schema — `internal/religion` reads/writes only
  `religion_*` tables (plus a `directory_units`/`location_locations` join inside `SearchSites`' own
  hand-written SQL for the discovery-card enrichment, M13.0).
- **Called by:** `content` (`content.manage`'s `requireManage` target-unit resolution),
  `discovery` (`SearchSites`/`SearchFacets`, the public search substrate — see
  [discovery.md](discovery.md)), `registration` (`ensureSite`, `ensureGrant`'s target-unit
  authority), `congregationimport` (site/classification provisioning), `core` (`createChildOrg`/
  `getOrgProfile`/taxonomy reads) — all in-process, all pre-M13.2. `web/apps/admin`'s site editor
  (`app/[locale]/admin/sites/[unitId]/page.tsx`) is `ReligionService`'s one new consumer (M13.2),
  via `lib/religion.ts`.

## Authorization touchpoints

**`site.manage`** — a target-scoped capability check (`internal/authz.Require`, `PermSiteManage`),
the identical mechanism [D-PlatformModerator](../architecture/decisions.md) already established for
`content.manage` (see [content.md](content.md)), just a different permission code: `site.manage`
existed in the closed catalog and was seeded to `congregation-admin`/`registration-operator`
(`migrations/0015_core_seed.sql`) well before M13.2, but no code path checked it until now.
`internal/religion/application/authorize.go`'s `requireManage` mirrors
`internal/content/application/authorize.go`'s own almost exactly.

**Religion previously carried zero authorization logic by design** (`D-InProcessAuthz`) — every
prior write went through some *caller's* own application layer, which already knew which
`authz.Service`/permission to check before calling into `religionapplication.Service`. M13.2's
`GetSiteByUnit`/`UpdateSiteAttributes` are religion's first *direct* authenticated entrypoints, so
`religionapplication.Service` itself now holds an `authz.Service` and checks `site.manage` before
doing anything — a deliberate departure from that old framing, joining every other module's own
established "authz lives in the application layer" convention (content, discovery, core,
registration all do it this way).

**Live-verified (2026-08-26):** a `congregation-admin` grant on their own unit passes both `getSite`
and `updateSiteAttributes`; the same grant on an *unrelated* unit is denied (`Religion:Forbidden`,
`403`); a `registration-operator` grant also passes (both roles hold `site.manage` per the same
seed row); no bearer at all is `401`; a unit with no `religion_sites` row throws
`Religion:SiteNotFound`. A write persists into `religion_sites.attributes` and is visible on the
very next `discovery`/`GET /search` hit (the disposable cache upserts from a live `SearchSites`
call, same as every other attributes-visible field since M13.0).

## Invariants

- **`religion_sites.attributes` is only ever written wholesale.** `UpdateSiteAttributes` replaces
  the entire `SiteAttributes` document — there is no partial-patch endpoint, matching how the admin
  form itself always submits every checkbox's current state, checked or not.
- **A congregation's "site" is always resolved by preferring `is_primary`.** Every read/write path
  that scopes by unit (`GetSiteByUnit`, `SearchSites(UnitID)`, `ensureSite`'s idempotency check)
  uses the same `is_primary DESC, id` ordering — a second, non-primary site (schema-legal, never
  exercised in practice) never silently wins over the primary one.
- **No public read ever bypasses precision coarsening.** `Site` (this module's own, exact/
  uncoarsened type) is never returned to an anonymous caller — only through `ReligionService`
  (`site.manage`-gated) or `SearchSitesExact` (`authz.SystemContext`-only, panics otherwise). The
  public-safe projection is always `DiscoverySite`, coarsened at the `discovery` boundary (see
  [discovery.md](discovery.md)).

## Open seams

- **`ReligionService` covers only site attributes today.** Taxon/org-profile/classification
  management and site *creation* (visibility, precision, location, site type) have no direct
  authenticated endpoint of their own — still reached only through `content`/`core`/`registration`'s
  own in-process calls. A real "manage my congregation's site" admin surface beyond attributes
  (editing visibility/precision/address directly, not just accessibility criteria) is real future
  scope, not attempted here.
- **A genuine cross-graph `site.manage` SUBTREE grant (`ScopeSubtree` + an authority-bearing
  graph) is untested by this module's own integration coverage** — M13.2's own test proves
  `PermSiteManage` is wired correctly using the same exact-unit (`ScopeUnit`) shape
  `content.manage` already exercises; real subtree-resolution correctness is covered elsewhere
  (`internal/authz/domain`'s own PDP tests, `core_unit_move_integration_test.go`), not duplicated
  here.
