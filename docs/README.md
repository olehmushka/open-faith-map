# OpenFaithMap — architecture documentation

> **Audience: Claude Code (and any future contributor).** These docs describe a project that is
> **design-complete, pre-code** (see [Status](#status)). They are the source of truth code will be
> held to once it lands. Every module doc is self-contained: read one without reading the others.
> Treat [`architecture/decisions.md`](architecture/decisions.md) as binding, and follow the feature
> pipeline in [`development-process.md`](development-process.md) — the
> [stage board](milestones.md#stage-board) shows where each milestone sits.

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
Geographic rollout: Ukraine + USA first, then Poland/UK, then the rest of EU/LATAM/Africa/Asia.

## The modules

| Module | Owns | Kind |
|---|---|---|
| [core-integration](modules/core-integration.md) | The mapping from a church-discovery product's needs onto go-oikumenea's modules; the congregation-provisioning flow | Bridge doc — no schema |
| [registration](modules/registration.md) | Congregation self-service registration: submission, D-Exclusions check, operator approval | New backend module — OpenFaithMap's first schema (M2) |
| [content](modules/content.md) | Site content: pages, posts, events, typed blocks, translation groups | New backend module |
| [discovery](modules/discovery.md) | Public map/search facade + a disposable read-through cache over go-oikumenea's religion discovery search | New backend module (cache only) |
| [moderation](modules/moderation.md) | Reports, actions, appeals, the denomination-exclusion taxon check | New backend module |
| [vouching](modules/vouching.md) | Web-of-trust guarantor verification for congregation-admin claims | New backend module |
| [web-facade](modules/web-facade.md) | The anonymous public Next.js app — map, search, congregation pages, public report filing | Consumer — no schema |
| [web-admin](modules/web-admin.md) | The verified Next.js app — registration wizard, operator/congregation-admin console, moderator console (D-AdminSurface) | Consumer — no schema |

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
   (`openfaithmap-web` + `openfaithmap-admin` + `openfaithmap-api`), request paths, deployment
   topology.
4. [`architecture/decisions.md`](architecture/decisions.md) — the binding decisions: scope,
   exclusions, the core dependency, the facade split, the content model, moderation, vouching, the
   shared toolchain. If code and a decision disagree, the code is wrong.
5. [`architecture/conventions.md`](architecture/conventions.md) — what's inherited unchanged from
   go-oikumenea's own conventions vs. what's OpenFaithMap-specific (schema name, RID handling
   across two databases, no cross-database FKs).
6. [`modules/core-integration.md`](modules/core-integration.md) — read this **before** any other
   module doc; it's the bridge every one of them assumes.
7. [`modules/web-facade.md`](modules/web-facade.md) and [`modules/web-admin.md`](modules/web-admin.md)
   — the two UI surfaces (D-AdminSurface): which one holds a session, which one never does.
8. The relevant [`modules/*.md`](modules/) for the work at hand.
9. [`open-questions.md`](open-questions.md) — the live backlog for the next planning session.
10. [`milestones.md`](milestones.md) — the implementation roadmap, sequenced M0…M7, and its
    [stage board](milestones.md#stage-board), the scannable index of where each milestone sits.
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

**Design-complete at the architecture level for M0–M6** (see [milestones.md](milestones.md)).
**No application code exists yet.** When asked to "find the code that does X," the answer is "it
does not exist yet — the design is here." Build order: M0 (done, this doc set) → M1 (go-oikumenea
integration wiring) → M2 (congregation self-service) → M3 (content backend) → M4 (public discovery)
→ M5 (moderation) → M6 (vouching) → M7 (hardening, not yet designed).
