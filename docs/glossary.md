# Glossary

Domain vocabulary for OpenFaithMap. Two kinds of terms live here: **new** vocabulary this project
introduces (content, moderation, vouching — nothing in go-oikumenea covers these), and **inherited**
vocabulary that keeps its original church-discovery meaning but is now backed by a go-oikumenea
primitive rather than an OpenFaithMap-owned table. The **term-mapping table** at the end is the
single most important reference in this glossary — read it before any module doc; every module doc
in [`modules/`](modules/) assumes it.

---

## Content (new — owned by OpenFaithMap)

**Site.** The public web presence of one congregation: its pages, posts, events, and the
church-admin-facing editor for them. One site per congregation `Unit`.

**Page.** A long-lived document on a site (home, about, beliefs, contact, staff). Composed of
ordered [blocks](#block). Supports a shallow page hierarchy (up to 3 levels).

**Post.** A time-stamped, reverse-chronological news item on a site's feed. Composed of ordered
blocks.

**Event.** A scheduled item on a site (a one-off gathering, not a recurring service — recurring
service times are `ServiceSchedule`, owned by go-oikumenea's [religion](#term-mapping-table)
module, see the mapping table). Composed of ordered blocks plus start/end/recurrence metadata.

**Block.** A typed, schema-validated unit of content. Pages, posts, and events are all ordered
lists of blocks — never HTML blobs. MVP block types: `heading`, `paragraph`, `image`,
`gallery`, `youtube_embed`, `social_embed`, `button`, `contact_info`, `map_embed`, `divider`,
`staff_card`, `quote`, `columns`. (`service_schedule` as a block type is **not** reintroduced —
service times are read live from go-oikumenea's `ServiceSchedule` via the [discovery](modules/discovery.md)
facade, never duplicated into content.)

**Translation group.** A UUID shared across documents that are translations of the same
conceptual page/post/event. Each document in a group has its own `locale`, edited independently.

**Draft / Published / Unlisted.** Content states. Draft is editor-visible only. Published is
public and menu-listed. Unlisted is public via direct URL but omitted from menus and sitemaps.

**Site theme.** A small set of presentation choices (accent color, font pairing, header layout)
a congregation picks for its site. Data, not code — never a per-tenant template fork.

---

## Moderation (new — owned by OpenFaithMap, reuses go-oikumenea's audit trail)

**Report.** A flag raised by anyone — including an anonymous visitor — against a site, a piece of
content, a congregation's claimed identity, or a vouching relationship. Queued for moderator
review.

**Moderation action.** An immutable record of a moderator decision (`hide`, `suspend`, `archive`,
`warn_admin`, `revoke_vouch`). Reversible within a grace window; the reversal is itself an action,
recorded the same way. Every action is written through go-oikumenea's [audit](#term-mapping-table)
module as its permission-sensitive-action ledger — OpenFaithMap does not keep a second audit log.

**Appeal.** A structured challenge by a congregation admin against a moderation action affecting
them, routed to a different moderator than the one who took the original action.

**Queue scope.** The visibility rule for which moderator sees which reports: platform-wide,
one congregation's own admins, or the congregation's jurisdictional ancestor chain (walking
go-oikumenea's `canonical` religion graph — see [core-integration](modules/core-integration.md)).

**Denomination exclusion.** OpenFaithMap's permanent, named policy that a given faith tradition
may never register a congregation on the platform (see [D-Exclusions](architecture/decisions.md)).
Enforced at the **taxon** level (facade-side, against go-oikumenea's `religion_taxa`), backed at
the **organization** level by go-oikumenea's `religion_org_policies` (`excludes_child_creation` /
`excluded_body`) once a body's root organization exists. This is the direct successor of the
original FaithMap `denomination_policy` mechanism.

---

## Vouching (new — owned by OpenFaithMap)

**Vouching edge.** An immutable, append-only record that congregation-admin A (guarantor) vouched
that congregation-admin B genuinely represents the congregation they claim. Used to raise trust on
a newly claimed/verified congregation without a manual moderator check for every claim.

**Guarantor status.** The mutable overlay on the immutable vouching graph, recording whether a
guarantor is currently trusted. A revoked guarantor triggers moderator review of every vouch they
made.

**Impersonation.** A moderator mechanism to log in as a congregation admin for debugging support
requests. Time-limited, banner-visible in the UI, double-logged in the audit trail (both the
moderator's action and the impersonated session).

---

## Discovery (facade — reads go-oikumenea's location + religion + search modules)

**Discovery search.** The public map/list search a visitor runs (by location, radius, service
time, service language). A thin facade over go-oikumenea's `GET /religion/discovery/sites`
(closure-aware, PostGIS-backed, `public_precision`-coarsened — see
[discovery.md](modules/discovery.md)). OpenFaithMap adds no search index of its own for site
location; it may add one for content full-text search (post/page bodies), which go-oikumenea has
no equivalent for.

**Precision.** How specifically a congregation's location is exposed publicly — inherited
unchanged from go-oikumenea's `religion_sites.public_precision`
(`exact`/`street`/`neighborhood`/`city`/`hidden`), coarsened app-side. This is the mechanism that
protects congregations in hostile or persecuted-church contexts; OpenFaithMap never overrides it
client-side.

---

## Architecture (shared vocabulary, inherited from go-oikumenea's own conventions)

**Facade.** An unprivileged consumer of go-oikumenea's API that owns its own session/UX concerns
but forwards the end user's IdP token on every call rather than asserting its own authority
(D-HeadlessTopology). OpenFaithMap's Next.js server tier is a facade in exactly this sense.

**Service principal.** A machine identity, authenticated via OAuth2 client-credentials against the
same external IdP humans use, that OpenFaithMap's backend registers as (D-ServiceIdentities) for
background jobs (moderation-queue sweeps, vouching-graph checks, exclusion-list sync) that are not
driven by a live user request.

**Core.** Shorthand throughout these docs for **go-oikumenea**, run as a headless internal service
(no public port) via its docker image — the sole source of truth for identity, tenant/organization
structure, role authority, location, and the religion taxonomy.

---

## Term-mapping table

The concept a church-discovery product needs, and what actually stores it now.

| Concept | Owned by | Detail |
|---|---|---|
| Tenant / organizational entity | **go-oikumenea `tenant`** | A `tenant_organizations` row (the `church` domain) + `tenant_units` graph nodes. See [core-integration.md](modules/core-integration.md). |
| Congregation | **go-oikumenea `tenant` + `religion`** | A `Unit` with `religion_org_kinds.code = congregation`, placed in the `canonical` graph under its jurisdiction. |
| Denomination / tradition | **go-oikumenea `religion`** | A node in `religion_taxa` (the faith taxonomy tree), not a tenant. A denomination's *governing body* (e.g. a national synod) is separately a `Unit`. |
| Jurisdiction | **go-oikumenea `tenant`** | A `Unit` one or more levels up the `canonical` graph from a congregation (diocese/synod/district), same graph, no separate entity. |
| Person (a pastor, staff member, visitor) | **go-oikumenea `person`/`personprofile`** | The instance-global person directory. A congregation's staff are people with a [membership](#term-mapping-table) position and/or a [clergy credential](#term-mapping-table) at that unit. |
| User account / login | **go-oikumenea `identity-federation`** | External IdP identity → go-oikumenea `account`, optionally attached to a `person`. OpenFaithMap issues no credentials of its own. |
| Role / permission | **go-oikumenea `authorization`** | Role assignments `(person, role, target_unit, scope)` over the `canonical` graph. OpenFaithMap defines no parallel RBAC. |
| Clergy office (pastor, elder, imam-equivalent) | **go-oikumenea `membership` + `religion`** | A `Position` (the billet) typed by `religion_office_types`, filled by a `membership`; standing recorded as a `religion_clergy_credentials` link. |
| Lay membership / adherence | **go-oikumenea `religion`** | A `religion_affiliations` link (`pii:special`, encrypted). OpenFaithMap never stores this itself. |
| Location / address / map coordinate | **go-oikumenea `location` + `religion`** | A shared `location_locations` row, exposed through a `religion_sites` link (`public_precision` coarsening). |
| Service schedule | **go-oikumenea `religion`** | `religion_service_schedules`, owned per `religion_sites` row. |
| Site content (pages/posts/events/blocks) | **OpenFaithMap `content`** | No go-oikumenea equivalent — genuinely new. See [content.md](modules/content.md). |
| Public discovery (map + search) | **OpenFaithMap `discovery`** (facade) | Reads go-oikumenea's `location`/`religion`/`search` modules; adds no location index of its own. |
| Moderation / reports / policy engine | **OpenFaithMap `moderation`** | New tables for reports/appeals; writes through go-oikumenea's `audit` module rather than a second ledger. |
| Denomination exclusion | **OpenFaithMap `moderation`, backed by go-oikumenea `religion`** | Facade-side taxon check + `religion_org_policies` at the org level. See [D-Exclusions](architecture/decisions.md). |
| Web-of-trust vouching | **OpenFaithMap `vouching`** | No go-oikumenea equivalent — genuinely new. |
| Audit trail | **go-oikumenea `audit`** | OpenFaithMap's moderation actions and admin operations are written here, not duplicated. |
