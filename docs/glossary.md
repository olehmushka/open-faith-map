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
ordered **blocks** (defined below). Supports a shallow page hierarchy (up to 3 levels).

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

## Moderation (new — owned by OpenFaithMap, including its own ledger)

**Report.** A flag raised by anyone — including an anonymous visitor — against a site, a piece of
content, a congregation's claimed identity, or a vouching relationship. Queued for moderator
review.

**Moderation action.** An immutable record of a moderator decision (`hide`, `suspend`, `archive`,
`warn_admin`, `revoke_vouch`). Reversible within a grace window; the reversal is itself an action,
recorded the same way. `openfaithmap.moderation_actions` is the **ledger of record** — append-only,
`reject_mutation()`-guarded. *(Corrected 2026-08-09: this previously said every action is written
through go-oikumenea's audit module and that OpenFaithMap "does not keep a second audit log."
go-oikumenea's audit module has no write endpoint, so that was never possible — see D-Moderation's
Correction.)*

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

## Roles (who does what — see [D-Scope](architecture/decisions.md)'s audience table)

**Visitor.** Anonymous. Uses `openfaithmap-web`; never authenticates, never has a session.

**Congregation admin.** Holds a `unit`-scoped `congregation-admin` role assignment over their own
congregation's `Unit`, granted at registration approval. Never `subtree` — approving one
registration never gives its submitter reach over any other congregation.

**Registration operator.** A real person holding the `registration-operator` role `subtree`-scoped
on the shared root unit, who reviews and approves congregation-registration requests; approval
performs the real go-oikumenea writes with *their own* token
([registration.md](modules/registration.md)). Bootstrapped once via
`scripts/bootstrap-registration-org` plus one out-of-band SQL insert, the same "operator-owned DB
access" trust level go-oikumenea's own `D-Bootstrap` uses for the first instance admin.

**Platform moderator.** A real person holding the `platform-moderator` role `subtree`-scoped on the
shared root unit ([D-PlatformModerator](architecture/decisions.md)). Handles reports, actions,
appeals, and guarantor management. Platform-wide authority expressed as subtree scope on the root —
not a separate OpenFaithMap roster table.

**Super admin (instance admin).** Holds go-oikumenea's own instance-wide authority: the
`religion_taxa` catalog, tenant structure, service-principal issuance, other instance admins. Works
through `oikumenea-console`, never through an OpenFaithMap-built surface
([D-InstanceAdminConsole](architecture/decisions.md)). Strictly larger blast radius than any
OpenFaithMap role.

*(Registration operator and super admin were missing from this glossary and from D-Scope's original
three-audience framing until the 2026-08-09 audit; both are real, separately-bootstrapped roles.)*

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

**`oikumenea-console`.** go-oikumenea's own published admin-console image, reused unmodified as
OpenFaithMap's third UI surface (D-InstanceAdminConsole) — for **super admins** (go-oikumenea
instance admins) only. Manages instance-wide go-oikumenea concerns (the `religion_taxa` catalog,
tenant structure, service-principal issuance, other instance admins), never anything
OpenFaithMap-owned. Not built by OpenFaithMap; see [architecture/decisions.md](architecture/decisions.md).

**`hermenea`.** **go-oikumenea's own** pre-existing reference-data companion service (not built by
OpenFaithMap) — seeds and enriches go-oikumenea's catalog data (countries, languages, external
organizations, geo places) via declarative source connectors, coupled to go-oikumenea's core purely
over HTTP. OpenFaithMap deploys it (compose wiring + an install config) — no code, no new write
path, no new credential of OpenFaithMap's own (D-BulkImport). See
[modules/import.md](modules/import.md).

---

## Term-mapping table

The concept a church-discovery product needs, and what actually stores it now.

| Concept | Owned by | Detail |
|---|---|---|
| Tenant / organizational entity | **go-oikumenea `tenant`** | A `tenant_organizations` row (the `church` domain) + `tenant_units` graph nodes. See [core-integration.md](modules/core-integration.md). |
| Congregation | **go-oikumenea `tenant` + `religion`** | A `Unit` with `religion_org_kinds.code = congregation`, placed in the `canonical` graph under its jurisdiction. |
| Denomination / tradition | **go-oikumenea `religion`** | A node in `religion_taxa` (the faith taxonomy tree), not a tenant. A denomination's *governing body* (e.g. a national synod) is separately a `Unit`. |
| Jurisdiction | **go-oikumenea `tenant`** | A `Unit` zero or more levels up the `canonical` graph from a congregation (diocese/synod/district), same graph, no separate entity. Operator-assigned at approval or re-parent time, never inferred from the congregation's tradition — [D-JurisdictionUnits](architecture/decisions.md) deliberately rejects one canonical hierarchy per denomination (Orthodox jurisdiction is often multiple and parallel even within one tradition; many Protestant congregations have none). "Zero levels" is legal: a congregation with no real jurisdiction remains a direct child of root, same as under the pre-M4.1 flat-root design. |
| Person (a pastor, staff member, visitor) | **go-oikumenea `person`/`personprofile`** | The instance-global person directory. A congregation's staff are people with a [membership](#term-mapping-table) position and/or a [clergy credential](#term-mapping-table) at that unit. |
| User account / login | **go-oikumenea `identity-federation`** | External IdP identity → go-oikumenea `account`, optionally attached to a `person`. OpenFaithMap issues no credentials of its own. |
| Role / permission | **go-oikumenea `authorization`** | Role assignments `(person, role, target_unit, scope)` over the `canonical` graph. OpenFaithMap defines no parallel RBAC. |
| Clergy office (pastor, elder, imam-equivalent) | **go-oikumenea `membership` + `religion`** | A `Position` (the billet) typed by `religion_office_types`, filled by a `membership`; standing recorded as a `religion_clergy_credentials` link. |
| Lay membership / adherence | **go-oikumenea `religion`** | A `religion_affiliations` link (`pii:special`, encrypted). OpenFaithMap never stores this itself. |
| Location / address / map coordinate | **go-oikumenea `location` + `religion`** | A shared `location_locations` row, exposed through a `religion_sites` link (`public_precision` coarsening). |
| Service schedule | **go-oikumenea `religion`** | `religion_service_schedules`, owned per `religion_sites` row. |
| Site content (pages/posts/events/blocks) | **OpenFaithMap `content`** | No go-oikumenea equivalent — genuinely new. See [content.md](modules/content.md). |
| Public discovery (map + search) | **OpenFaithMap `discovery`** (facade) | Reads go-oikumenea's `location`/`religion`/`search` modules; adds no location index of its own. |
| Moderation / reports / policy engine | **OpenFaithMap `moderation`** | New tables for reports/actions/appeals. `moderation_actions` is the append-only ledger of record — *not* mirrored into go-oikumenea's audit module, which has no write endpoint (D-Moderation's Correction). |
| Denomination exclusion | **OpenFaithMap `moderation`, backed by go-oikumenea `religion`** | Facade-side taxon check + `religion_org_policies` at the org level. See [D-Exclusions](architecture/decisions.md). |
| Web-of-trust vouching | **OpenFaithMap `vouching`** | No go-oikumenea equivalent — genuinely new. |
| Audit trail | **both, separately** | go-oikumenea's `audit` covers every write a forwarded token makes through go-oikumenea. OpenFaithMap's own `moderation_actions` (append-only) covers its moderation decisions. ⚠️ **Corrected 2026-08-09:** this row previously said OpenFaithMap's actions "are written here, not duplicated" — go-oikumenea's audit module has **no write endpoint**, so that was never possible. See D-Moderation's Correction. |
