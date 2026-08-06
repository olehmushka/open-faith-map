# Module: core-integration

> Reads: [glossary](../glossary.md) · [architecture/overview](../architecture/overview.md) ·
> [architecture/decisions](../architecture/decisions.md)
> Owns no schema — this is the bridge doc, not a backend module.

## Purpose

Documents exactly how OpenFaithMap reaches go-oikumenea (D-CoreDependency, D-Facade): which
go-oikumenea modules it calls, for what, under which identity, and what it deliberately does
**not** build because go-oikumenea already owns it. Every other module doc in this repo assumes
this mapping — read it first.

Governing property: **OpenFaithMap makes zero authorization decisions of its own** over anything
go-oikumenea owns. It never caches a permission check across requests, never widens what a token
can do, and never asserts "act as person X." Every read/write against tenant/person/membership/
authorization/location/religion data is a live, token-forwarded call into go-oikumenea; go-oikumenea's
PDP is authoritative every time.

## What OpenFaithMap delegates entirely

| go-oikumenea module | What OpenFaithMap uses it for | Calling identity |
|---|---|---|
| `identity-federation` | Login (OIDC via the shared Keycloak realm), account lookup | User token (facade passthrough) |
| `tenant` | The congregation/jurisdiction organizational graph: `church`-domain `Organization` + `Unit` nodes in the `canonical` graph | User token for admin actions; service principal for read-heavy discovery caching |
| `person` / `personprofile` | The people directory: pastors, staff, congregation admins as `Person` records | User token |
| `membership` | Clergy/staff **positions** at a congregation `Unit` | User token |
| `authorization` | Role assignments granting a congregation admin authority over their `Unit` (and, via `subtree` scope, its descendants) | User token; go-oikumenea's PDP decides, never OpenFaithMap |
| `location` | The physical address + coordinate behind a congregation's site | User token (write) / unauthenticated public read (via `religion` discovery) |
| `religion` | The faith taxonomy (`religion_taxa`), organization profiles, clergy credentials, lay affiliation, sites, service schedules, discovery search | Mixed — see [discovery.md](discovery.md) |
| `search` | Cross-type object search inside the congregation-admin console (finding a person/unit by name) | User token |
| `audit` | The single append-only ledger for both go-oikumenea's own actions and OpenFaithMap's moderation actions | Service principal (writes from OpenFaithMap's backend) |

## What OpenFaithMap owns instead

| Domain | Why go-oikumenea doesn't cover it | Module doc |
|---|---|---|
| Site content (pages/posts/events/blocks, themes) | go-oikumenea's `religion` module doc explicitly scopes this out: *"Content / CMS (pages, blocks, themes, slugs, content-i18n groups) stays in the consuming app — out of scope for this identity/directory service."* | [content.md](content.md) |
| Public discovery UX (map rendering, search-result shaping, SEO) | go-oikumenea exposes the data (`/religion/discovery/sites`); rendering it as a map/list product is facade work | [discovery.md](discovery.md) |
| Moderation (reports, appeals, the denomination-exclusion taxon check) | go-oikumenea has no report/appeal concept, and its `religion_org_policies` exclusion mechanism is org-scoped, not taxon-scoped — see [D-Exclusions](../architecture/decisions.md) | [moderation.md](moderation.md) |
| Web-of-trust vouching | No equivalent concept in go-oikumenea (role assignments grant *authority*; vouching only raises *trust*) | [vouching.md](vouching.md) |

## Provisioning a congregation (the core end-to-end flow)

This is the flow every other module doc in this repo references, so it's spelled out once here:

1. A prospective admin picks their congregation's tradition — a `religion_taxa` node, read via
   `GET /religion/taxa?query=…` (unauthenticated, `religion.read`). OpenFaithMap checks the picked
   taxon (and its ancestors, via the taxonomy closure) against the [D-Exclusions](../architecture/decisions.md)
   list **before** proceeding — a match stops the flow with a clear message, no go-oikumenea call
   made.
2. If eligible, OpenFaithMap calls go-oikumenea's `POST /religion-orgs` (if this is a new top-level
   body) or `POST /units/{parentId}/child-orgs` (a congregation under an existing jurisdiction) to
   create the `church`-domain `Organization`/`Unit` — user-token-authenticated, so go-oikumenea's
   PDP decides whether this caller may create it.
3. OpenFaithMap sets the unit's `religion_org_classifications` (tradition tag) and
   `religion_org_profiles` via `PUT /units/{unitId}/religion-profile`.
4. The admin attaches a location (`location` module) and a `religion_sites` row
   (`POST /units/{unitId}/sites`) — this is what makes the congregation appear in discovery.
5. The admin's own `person` record gets a `membership` `Position` at the unit and an
   `authorization` role assignment scoped to that unit — granting them authority over exactly this
   congregation (and its descendants only if scope is `subtree`, e.g. a diocese admin).
6. OpenFaithMap creates the congregation's `content` site (its own table, step 3 in
   [content.md](content.md)) — a purely local operation, no go-oikumenea call.

Nothing in this flow gives OpenFaithMap's backend elevated authority: every write in steps 2–5 is
made with the *admin's own token*, so an admin can only provision what go-oikumenea's PDP already
lets them provision (e.g. a brand-new top-level body requires whatever permission
`POST /religion-orgs` requires today; a congregation under an existing jurisdiction requires
authority over that parent unit).

## Dependencies

- **Calls:** every go-oikumenea module listed above, via the generated Go/TypeScript SDK
  (D-ClientSDK) — never a raw HTTP client, never a direct database connection.
- **Called by:** every other OpenFaithMap module doc (`content`, `discovery`, `moderation`,
  `vouching`, `web-facade`) references this doc rather than restating the delegation.

## Authorization touchpoints

OpenFaithMap defines **no permission codes of its own** for anything in the delegation table above
— it forwards the caller's token and lets go-oikumenea's PDP answer. It *does* define permission
codes for its own domains (content/moderation/vouching), documented in their own module docs.

For background work, OpenFaithMap's service principal (D-ServiceIdentities) holds exactly these
grants, nothing more:
- `religion.read` (instance-wide) — refresh the discovery cache, resolve taxon ancestors for the
  exclusion check.
- `audit.write` (instance-wide) — write moderation actions into go-oikumenea's audit ledger.

No role assignment, no unit reach, no write access to tenant/person/religion data as the
principal — every write a real congregation admin makes is made with *their* token, never the
principal's.

## Invariants

- **No shadow authorization state.** OpenFaithMap never stores "person X can edit congregation Y" —
  it asks go-oikumenea's PDP (indirectly, by making the call with the user's token and letting it
  fail or succeed) every time. A cache of who-can-do-what would be a second, driftable source of
  truth; not built.
- **No on-behalf-of.** OpenFaithMap's backend never presents its service-principal token to act as
  a specific person, even for automation — matching D-HeadlessTopology's "no confused deputy"
  guarantee on the go-oikumenea side.
- **Delegated data has one home.** Anything in the "delegated entirely" table is never duplicated
  into an OpenFaithMap table, even for caching — the discovery facade may cache *read* results
  (see [discovery.md](discovery.md)) but never treats a cache as authoritative for a write decision.

## Open seams

- **Taxon-level exclusion has no go-oikumenea-native home.** If a second consuming app ever needed
  the same "block this whole tradition" behavior, this logic (currently facade-side only) would be
  a candidate for a genuine go-oikumenea feature request — not attempted here since OpenFaithMap
  is currently the only consumer.
- **Location-scoped role assignments** (e.g., a "campus admin" scoped below a congregation unit)
  are called out as reserved (`DS-50`) in go-oikumenea's own religion module doc — OpenFaithMap has
  no workaround today; a multi-site congregation's sub-locations share one admin set until that
  seam is picked up upstream.
