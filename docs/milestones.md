# Milestones

The architecture sequenced into buildable, dependency-ordered milestones. A roadmap, not binding —
[`architecture/decisions.md`](architecture/decisions.md) governs *what*, this governs *in what
order*. Gate definitions are in [`development-process.md`](development-process.md).

## Status

**Design-complete for M0–M6 at the architecture level.** No application code exists yet — this
session produced the decision records, the integration mapping, and every module's design doc.
The build sequence below is dependency-ordered so each milestone leaves the system deployable.
Until code lands, "where's the code that does X" has the same answer go-oikumenea's own early docs
gave: "it does not exist yet — the design is here."

## Stage board

| # | Decided | Designed | Backend | Migrated | UI | Verified | Stage |
|---|---|---|---|---|---|---|---|
| M0 · Scope & core-dependency | ✅ | ✅ | ➖ | ➖ | ➖ | ✅ | **Verified.** Artifact: this doc set (`architecture/decisions.md`, `modules/core-integration.md`, `glossary.md`), coherence-checked (no dangling relative links, no `faithmap-app` references). A docs-only milestone; its exit criterion is the doc set existing and being internally consistent, met. |
| M1 · go-oikumenea integration wiring | ✅ | ✅ | ⬜ | ➖ | ⬜ | ⬜ | **Backend in progress (not ✅ — see prose).** `docker-compose.yml` runs a real go-oikumenea instance (published image, migrated, shared Postgres). Service-principal auth (D-ServiceIdentities) proven end-to-end — `internal/coreintegration`, `scripts/bootstrap-service-principal`. **Not done:** `openfaithmap-web`'s session layer / human login — M1's "login working" exit criterion is unmet. See M1.1 for doc corrections found while proving this out. |
| M1.1 · Core-integration doc corrections | ✅ | ✅ | ➖ | ➖ | ➖ | ⬜ | **Designed, not yet applied.** Three inaccuracies in `architecture/decisions.md` / `modules/core-integration.md`, found by testing M1 against a real go-oikumenea instance rather than assumed from its docs — see detail below. Docs-only; blocks nothing in code, but M2's own doc references `modules/core-integration.md` and should not compound an already-known-wrong doc. |
| M2 · Church-admin self-service facade | ✅ | ✅ | ⬜ | ➖ | ⬜ | ⬜ | **Designed.** `modules/core-integration.md` (the provisioning flow) + `modules/web-facade.md` (the registration/roster UI shape). No dedicated schema of its own — writes go through go-oikumenea directly. |
| M3 · Content / site-builder backend | ✅ | ✅ | ⬜ | ⬜ | ⬜ | ⬜ | **Designed.** `modules/content.md` — full entity model, Conjure sketch, first genuinely new schema in the project. |
| M4 · Public discovery site | ✅ | ✅ | ⬜ | ⬜ | ⬜ | ⬜ | **Designed.** `modules/discovery.md` — cache schema + facade contract over go-oikumenea's `religion` discovery search. |
| M5 · Moderation | ✅ | ✅ | ⬜ | ⬜ | ⬜ | ⬜ | **Designed.** `modules/moderation.md` — reports/actions/appeals + the D-Exclusions taxon check. |
| M6 · Vouching | ✅ | ✅ | ⬜ | ⬜ | ⬜ | ⬜ | **Designed.** `modules/vouching.md` — web-of-trust guarantor model. |
| M7 · Hardening / real-user feedback | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | **Idea.** Named and sequenced here; no `D-<Name>` block or module doc yet — first real milestone to pass through the full pipeline once M1–M6 have shipped code and real congregations are using the platform. |

## Per-milestone detail

### M0 · Scope & core-dependency

**Depends on:** nothing — the foundation. **Leaves deployable:** trivially yes (no code).

Establishes what OpenFaithMap is (Christian-only discovery + presence — D-Scope), what it will
never build (the named exclusion list — D-Exclusions), and how it relates to go-oikumenea
architecturally (thin facade — D-CoreDependency, D-Facade). Every later milestone's "designed" gate
depends on this one's decisions being settled first.

### M1 · go-oikumenea integration wiring

**Depends on:** M0. **Leaves deployable:** a running go-oikumenea instance behind
`openfaithmap-web`, with login working and zero OpenFaithMap-specific data yet — a valid,
demoable, if minimal, deployment.

Stand up go-oikumenea's docker image in OpenFaithMap's own `docker-compose.yml`
(D-CoreDependency), alongside its own Postgres/migrate/init-role sequence. Register OpenFaithMap's
backend as a service principal (D-ServiceIdentities pattern). Build `openfaithmap-web`'s session
layer and prove token passthrough works end-to-end against a real go-oikumenea call.

**As built (partial):** `docker-compose.yml` runs go-oikumenea from its published image
(`docker.io/olegamysk/oikumenea`) against one shared Postgres instance with OpenFaithMap
(`oikumenea` / `openfaithmap` schemas — a simplification from two separate database instances,
decided over chat, not yet its own `D-<Name>`). **No Keycloak** — go-oikumenea is configured to
trust Google directly (`deploy/oikumenea-install.yml`); a real deviation from this decision's
original "shared Keycloak realm" premise (see `architecture/decisions.md`'s D-CoreDependency),
also not yet its own `D-<Name>`. `internal/coreintegration` + `scripts/bootstrap-service-principal`
prove the service-principal path for real: a GCP service account mints its own Google ID token per
call, go-oikumenea resolves it by `(issuer, subject)`, and the PDP enforces its grant — verified
against `connector.read` (see M1.1, item 2, for why not `religion.read`). **Not built:**
`openfaithmap-web`'s session layer — no human ever logs in yet, so "login working" is unmet.

### M1.1 · Core-integration doc corrections

**Depends on:** M1 (these were found while proving M1's service-principal path against a real
go-oikumenea instance, not while writing the original design). **Leaves deployable:** no code
change — this is `architecture/decisions.md` / `modules/core-integration.md` prose catching up to
verified reality.

Three inaccuracies to fix before M2 (so M2 doesn't cite an already-known-wrong doc) or M4
(discovery-cache refresh depends directly on item 2):

1. **`modules/core-integration.md`'s authorization-touchpoints table lists `audit.write`** as a
   grant OpenFaithMap's service principal holds. That permission does not exist: go-oikumenea's
   `audit` module has no write endpoint — "there is no write endpoint; writes happen in-process"
   (go-oikumenea's own `docs/modules/audit.md`), and its permission catalog defines only
   `audit.read`. `scripts/bootstrap-service-principal` rejected the request live
   (`PrincipalGrantInvalid: unknown permission code`). Fix: remove `audit.write` from the table; if
   D-Moderation (M5) still needs OpenFaithMap's own moderation actions written into go-oikumenea's
   audit trail, that needs a real design — the current text assumes a mechanism that isn't there.
2. **The same table implies the service principal can call go-oikumenea's `religion` read
   endpoints** (discovery-cache refresh, D-Exclusions taxon-ancestor resolution) using its
   `religion.read` grant. Verified false against a real instance: every `religion` module read
   endpoint is gated with `RequireAnywhere`, a person-shaped PEP path that unconditionally denies a
   machine subject, regardless of grants (`internal/authorization/pep` — "every person-shaped PEP
   path denies it at its empty-subject guard"). Only the `connector`/`wiring` modules are
   machine-reachable (`RequireService`/`RequireServiceOrPerson`) in the current go-oikumenea
   version. This is an upstream go-oikumenea gap, not an OpenFaithMap misconfiguration — worth a
   go-oikumenea feature request (make the relevant religion reads `RequireServiceOrPerson`) before
   M4 is built on an assumption that doesn't hold today.
3. **D-CoreDependency's "shared Keycloak realm" premise** no longer matches what M1 actually built
   (Google directly, no Keycloak — see M1's as-built note above). Needs its own decision update:
   either amend D-CoreDependency or add a new `D-<Name>` recording the Google-direct choice and why
   (client-credentials has no Google equivalent for arbitrary resource servers; a GCP service
   account's self-minted ID token substitutes for the service-principal leg).

### M2 · Church-admin self-service facade

**Depends on:** M1. **Leaves deployable:** a congregation admin can register a congregation (a
real `church`-domain `Organization`/`Unit` in go-oikumenea) and see their own roster — still no
public-facing content or discovery.

Implements the provisioning flow in
[core-integration.md](modules/core-integration.md#provisioning-a-congregation-the-core-end-to-end-flow)
end-to-end, steps 1–5 (the D-Exclusions taxon check through the role-assignment grant). Step 6
(creating the local `content` site) is stubbed until M3.

### M3 · Content / site-builder backend

**Depends on:** M2 (needs a real congregation `Unit` RID to attach a site to). **Leaves
deployable:** a congregation admin can build and publish a real page; nothing public discovers it
yet (no map).

Builds `openfaithmap-api`'s `content` module per [content.md](modules/content.md): sites, pages
(post/event deferred to this milestone's own later iteration per that doc's open seams), blocks,
translation groups, the block-type catalog seeded with the MVP set.

### M4 · Public discovery site

**Depends on:** M2 (site data exists) and go-oikumenea's `religion` sites/schedules being
populated as part of M2's provisioning flow. **Leaves deployable:** the first real end-to-end
product demo — a visitor can find a congregation on a map and read its published page.

Builds `discovery`'s cache + search facade per [discovery.md](modules/discovery.md) and the public
site rendering in [web-facade.md](modules/web-facade.md).

### M5 · Moderation

**Depends on:** M3, M4 (needs real content and real discoverable congregations to moderate).
**Leaves deployable:** yes — a functioning platform with a safety net, ready for real (if limited)
users.

Builds `moderation` per [moderation.md](modules/moderation.md): reports, actions, appeals, and
wires the D-Exclusions check (already used at M2 registration time) into the moderator console.

### M6 · Vouching

**Depends on:** M2 (needs real admins to vouch), M5 (a revoked guarantor's vouches route into the
moderation queue). **Leaves deployable:** yes — reduces manual verification load without changing
what's already live.

Builds `vouching` per [vouching.md](modules/vouching.md).

### M7 · Hardening / real-user feedback (idea stage)

**Depends on:** M1–M6 all live with real congregations using the platform daily. Not yet decided
or designed — named here as the expected next milestone once real usage surfaces real problems
(rate limiting, moderation-queue UX, observability), matching the "real-user hardening" spirit of
the original FaithMap roadmap's own post-MVP stage, expressed here as a normal numbered milestone
rather than a separately-tracked stage.
