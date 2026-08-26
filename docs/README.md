# OpenFaithMap — architecture documentation

> **Audience: Claude Code (and any future contributor).** These docs are the source of truth code is
> held to. Part of the system is built and part is still design-only — the
> [stage board](milestones.md#stage-board) is authoritative for which is which, and
> [Status](#status) summarizes it. Every module doc is self-contained: read one without reading the
> others. Treat [`architecture/decisions.md`](architecture/decisions.md) as binding, and follow the
> feature pipeline in [`development-process.md`](development-process.md).

## What OpenFaithMap is

A free, open-source, **Christian** church-discovery-and-presence platform — a **map** (discovery)
and a per-congregation **site builder** (presence) — built as a facade on top of
**[go-oikumenea](https://github.com/olehmushka/go-oikumenea)**, consumed as a headless internal
core via its docker image. go-oikumenea supplies identity, authorization, the organizational
graph, location, and the multi-faith religion taxonomy; OpenFaithMap supplies everything a general
directory/authz core has no reason to own: site content, public discovery UX, moderation, and a
web-of-trust vouching layer. See [D-Scope and D-Facade](architecture/decisions.md) for the full
rationale.

Two audiences: **visitors** (anonymous, use the map and read congregation sites) and
**congregation admins** (verified, manage one or more congregations' presence and roster). A small
platform-wide **moderator** roster handles reports, appeals, and the denomination-exclusion policy.
A further, smaller role sits outside OpenFaithMap's own product surface entirely: **super admins**
manage go-oikumenea itself (taxonomy, tenants, service principals) through go-oikumenea's own
`oikumenea-console`, reused unmodified (D-InstanceAdminConsole). go-oikumenea's own `hermenea`
companion service, deployed by OpenFaithMap for reference-data seeding (D-BulkImport), has no
OpenFaithMap-specific role of its own — it runs on cron with its own credential, not on behalf of
any OpenFaithMap user. Geographic rollout: Ukraine + USA first, then Poland/UK, then the rest of
EU/LATAM/Africa/Asia.

## The modules

| Module | Owns | Kind |
|---|---|---|
| [core-integration](modules/core-integration.md) | The mapping from a church-discovery product's needs onto go-oikumenea's modules; the congregation-provisioning flow | Bridge doc — no schema |
| [religion](modules/religion.md) | The faith taxonomy, a unit's org profile/classifications, and `religion_sites` (location, visibility/precision, accessibility/online-stream attributes) | In-process module (M10 absorption); own transport (`ReligionService`) since M13.2 |
| [registration](modules/registration.md) | Congregation self-service registration: submission, D-Exclusions check, operator approval | New backend module — OpenFaithMap's first schema (M2) |
| [content](modules/content.md) | Site content: pages, posts, events, typed blocks, translation groups | New backend module |
| [discovery](modules/discovery.md) | Public map/search facade + a disposable read-through cache over go-oikumenea's religion discovery search | New backend module (cache only) |
| [moderation](modules/moderation.md) | Reports, actions, appeals, the denomination-exclusion taxon check | New backend module |
| [vouching](modules/vouching.md) | Web-of-trust guarantor verification for congregation-admin claims | New backend module |
| [web-facade](modules/web-facade.md) | The anonymous public Next.js app — map, search, congregation pages, public report filing | Consumer — no schema |
| [web-admin](modules/web-admin.md) | The verified Next.js app — registration wizard, operator/congregation-admin console, moderator console (D-AdminSurface) | Consumer — no schema |
| [import](modules/import.md) | Deploy wiring for `hermenea` — go-oikumenea's own reference-data companion service (D-BulkImport) | Not an OpenFaithMap module — deploy doc only, no schema, no code |

**Not an OpenFaithMap module at all:** `oikumenea-console`, go-oikumenea's own published console
image, reused unmodified as the third UI surface — super admins only (D-InstanceAdminConsole, see
[architecture/decisions.md](architecture/decisions.md)). OpenFaithMap deploys it; it has no module
doc here because OpenFaithMap builds none of it.

**Delegated entirely to go-oikumenea** (no OpenFaithMap module — see the
[term-mapping table](glossary.md#term-mapping-table) for the full list): identity/login, the
congregation/jurisdiction organizational graph, the people directory, roles and authority, physical
addresses, the faith taxonomy, clergy credentials, lay affiliation, service schedules, cross-type
search, and the audit trail.

## Reading order

1. **This file** — what OpenFaithMap is, the module map.
2. [`glossary.md`](glossary.md) — domain vocabulary, **including the term-mapping table**
   (church-discovery concept → what actually stores it now). Read this before any module doc.
3. [`architecture/overview.md`](architecture/overview.md) — the three-service shape
   (`openfaithmap-web` + `openfaithmap-admin` + `openfaithmap-api`), plus the third UI surface
   OpenFaithMap doesn't build (`oikumenea-console`) and the reference-data companion service
   OpenFaithMap deploys but doesn't build (`hermenea`); request paths; deployment topology.
4. [`architecture/decisions.md`](architecture/decisions.md) — the binding decisions: scope,
   exclusions, the core dependency, the facade split, the content model, moderation, vouching, the
   shared toolchain. If code and a decision disagree, the code is wrong.
5. [`architecture/conventions.md`](architecture/conventions.md) — what's inherited unchanged from
   go-oikumenea's own conventions vs. what's OpenFaithMap-specific (schema name, RID handling
   across two databases, no cross-database FKs).
6. [`modules/core-integration.md`](modules/core-integration.md) — read this **before** any other
   module doc; it's the bridge every one of them assumes.
7. [`modules/web-facade.md`](modules/web-facade.md) and [`modules/web-admin.md`](modules/web-admin.md)
   — the two OpenFaithMap-built UI surfaces (D-AdminSurface): which one holds a session, which one
   never does. [`modules/import.md`](modules/import.md) — the deploy wiring for `hermenea`,
   go-oikumenea's own reference-data companion service (D-BulkImport). `oikumenea-console` (the
   third, super-admin-only surface, D-InstanceAdminConsole) has no module doc — it's documented in
   `architecture/decisions.md` and `architecture/overview.md` only, since OpenFaithMap builds none
   of it.
8. The relevant [`modules/*.md`](modules/) for the work at hand.
9. [`open-questions.md`](open-questions.md) — the live backlog for the next planning session.
10. [`milestones.md`](milestones.md) — the implementation roadmap, sequenced M0…M7, and its
    [stage board](milestones.md#stage-board), the scannable index of where each milestone sits.
    It opens with the
    [unresolved-unknowns index](milestones.md#unresolved-unknowns--read-this-before-building-anything)
    — read that before starting any build work, whichever milestone you're on.
11. [`development-process.md`](development-process.md) — the feature pipeline (idea → decided →
    designed → backend → migrated → ui → verified), the runbook to advance a milestone, and how the
    stage board is kept honest. Read it before starting or reporting on any feature.

## Provenance

OpenFaithMap's product vision — free, cross-denominational church discovery and presence for small,
often volunteer-run congregations — carries forward a church-discovery concept explored earlier.
What's different this time is the foundation: rather than building tenant/identity/RBAC/location
modeling from scratch, OpenFaithMap sits on go-oikumenea's already-built, general-purpose directory
and authorization core, using its `religion` vertical as the identity/organizational backbone and
adding only what a general directory core has no reason to own (content, discovery UX, moderation,
vouching). See [D-Facade](architecture/decisions.md) for the full rationale and
[core-integration.md](modules/core-integration.md) for exactly what's delegated vs. owned.

## Status

The [stage board](milestones.md#stage-board) is authoritative; this is the summary.

> **Corrected 2026-08-17.** This section listed M3–M6 as "designed, not built" and told readers that
> "find the code that does X" would have no answer for them. That was true around M2.2 and went
> stale; all four have been built and Verified since. Corrected rather than left standing, because
> the stale version actively misdirected anyone searching the codebase.

**Built and Verified:** M0 (this doc set) · M1 + M1.1 (go-oikumenea integration, service principal,
session layer) · M2.2 (`hermenea` deploy wiring) · M2.4 (CI repair) · M2.5 (discovery reachability
spike) · M2.6 (TypeScript codegen) · M3 (content / site builder) · M4 (public discovery) · M4.1
(jurisdiction units) · M5 (moderation) · M6 (vouching) · M9 (production-deployment *design* only —
nothing is provisioned).

**Built, not yet Verified:** M7 (hardening — rate limiting, metrics) · M8 (congregation import —
four connectors and a jurisdiction sync; blocked on a browser click-through and a green CI run at
the merge commit).

**Built, blocked on an external action nobody in this repo can perform** (a Google OAuth redirect
URI): M1.2 (`oikumenea-console`) · M2 (congregation self-service) · M2.1 (the
`openfaithmap-web` / `openfaithmap-admin` split) · M2.3 (registration hardening).

**Decided, not started:** M10 — absorbing the go-oikumenea core into this repo
([D-OwnCore](architecture/decisions.md#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed)).
Note that this supersedes [D-Facade](architecture/decisions.md), cited a few lines above: the
delegation described there becomes an implementation. The *domains* D-Facade assigns to
OpenFaithMap — content, discovery glue, moderation, vouching — are unchanged.

**Build order:** M0 → M1 → M2 → M2.3–M2.6 → M3 → M4 → M4.1 → M5 → M6 → M7 → M8 → M9 → **M10.1–M10.9**.

> **Before building anything, read
> [milestones.md's unresolved-unknowns index](milestones.md#unresolved-unknowns--read-this-before-building-anything).**
> It lists every place this doc set says "we don't actually know" — three assumptions that must be
> measured against a live go-oikumenea instance before the plans resting on them are trustworthy,
> five deferred decisions, and five contradictions or orphans. Nothing is parked silently; if a
> design has a hole, it is in that table.

Two habits this doc set has, worth knowing before reading further. First, a decision that turned
out to be wrong is **corrected in place with its history intact**, not deleted — see D-BulkImport
and D-Moderation, each of which carries a `Correction` block above superseded text. Second, several
things were found false only by testing against a real go-oikumenea instance rather than reading its
docs; where that happened, the doc says so explicitly.
