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
| `identity-federation` | Login (OIDC via Google directly — no Keycloak, no shared realm), account lookup | User token (facade passthrough — the Google ID token) |
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

**Superseded by M2's real build — see [registration.md](registration.md) for the authoritative
flow.** This section originally assumed a prospective admin's own token could call
`POST /religion-orgs` directly and self-grant authority over the resulting unit. Verified false
while building M2: creating a top-level body needs `religion.catalog.manage` (instance-wide,
instance-admin-only in practice), and granting *anyone* authority over a brand-new unit needs
`assignment.grant` **on that unit** — which not even an instance admin holds automatically
(instance-admin only auto-passes instance-scope PDP checks; a unit-scoped one still needs a real
assignment row). go-oikumenea has no self-service path for an ungranted user to create an org or
gain authority over one — full detail, including the actual bootstrap mechanism used, is in
registration.md's "authority-bootstrapping finding".

The real flow, kept here at a glance (registration.md is authoritative):

1. A prospective admin picks their congregation's tradition — a `religion_taxa` node — and submits
   a request (name, tradition, address/coordinates) to OpenFaithMap's own `registration` module.
   The [D-Exclusions](../architecture/decisions.md) taxon-ancestor check runs at submission time,
   against a live `religion.read` call — a match rejects the submission, no go-oikumenea write
   attempted.
2. A **registration operator** — a real person holding authority over OpenFaithMap's one shared
   root unit (registration.md's "single shared root organization" simplification) — reviews and
   approves. Approval uses the **operator's own forwarded token**, never OpenFaithMap's backend
   acting on the submitter's behalf, to call go-oikumenea's `POST /units/{rootUnitId}/child-orgs`
   (the submitter's congregation, as a child of the shared root), set its tradition classification,
   attach a location + site, create and fill a `membership` `Position`, and grant the submitter a
   `unit`-scoped role over their own new unit.
3. OpenFaithMap creates the congregation's `content` site (its own table, step 3 in
   [content.md](content.md)) — still stubbed until M3, unchanged from the original plan.

Nothing in this flow gives OpenFaithMap's backend elevated authority: every go-oikumenea write is
made with a *real person's own token* (the submitter's, for reads; the operator's, for the actual
provisioning writes) — go-oikumenea's PDP decides every one for real.

## Dependencies

- **Calls:** every go-oikumenea module listed above, via the generated Go/TypeScript SDK
  (D-ClientSDK) — never a raw HTTP client, never a direct database connection.
- **Called by:** every other OpenFaithMap module doc (`content`, `discovery`, `moderation`,
  `vouching`, `web-facade`, `web-admin`) references this doc rather than restating the delegation.

## Authorization touchpoints

OpenFaithMap defines **no permission codes of its own** for anything in the delegation table above
— it forwards the caller's token and lets go-oikumenea's PDP answer. It *does* define permission
codes for its own domains (content/moderation/vouching), documented in their own module docs.

For background work, OpenFaithMap's service principal (D-ServiceIdentities) holds exactly this
grant today:
- `connector.read` (instance-wide) — the only machine-reachable grant proven end-to-end so far
  (`internal/coreintegration`'s integration test).

Two grants this doc previously listed turned out not to work against a real instance (found while
proving M1, see [milestones.md](../milestones.md) M1.1):

- **`religion.read` is not usable by a service principal today.** Every `religion` module read
  endpoint is `RequireAnywhere`-gated — a person-shaped PEP path that unconditionally denies a
  machine subject, regardless of grants. Only the `connector`/`wiring` modules are
  machine-reachable in the current go-oikumenea version. Discovery-cache refresh and taxon-ancestor
  resolution for the D-Exclusions check therefore have **no machine-callable path yet** — an
  upstream go-oikumenea feature request (make the relevant `religion` reads
  `RequireServiceOrPerson`) is needed before M4 is built on an assumption that doesn't hold today.
- **`audit.write` does not exist.** go-oikumenea's `audit` module has no write endpoint — writes
  happen in-process — confirmed live (`scripts/bootstrap-service-principal` rejects the grant with
  `PrincipalGrantInvalid: unknown permission code`). If D-Moderation (M5) still needs OpenFaithMap's
  own moderation actions written into go-oikumenea's audit trail, that needs a real design — the
  mechanism this doc assumed isn't there.

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

- **No machine-callable `religion` reads.** See the authorization-touchpoints correction above —
  discovery-cache refresh and taxon-ancestor resolution have no service-principal path today. A
  go-oikumenea feature request (`RequireServiceOrPerson` on the relevant reads) is a prerequisite
  for M4, not yet filed.
- **Taxon-level exclusion has no go-oikumenea-native home.** If a second consuming app ever needed
  the same "block this whole tradition" behavior, this logic (currently facade-side only) would be
  a candidate for a genuine go-oikumenea feature request — not attempted here since OpenFaithMap
  is currently the only consumer.
- **Location-scoped role assignments** (e.g., a "campus admin" scoped below a congregation unit)
  are called out as reserved (`DS-50`) in go-oikumenea's own religion module doc — OpenFaithMap has
  no workaround today; a multi-site congregation's sub-locations share one admin set until that
  seam is picked up upstream.
