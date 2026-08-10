# Milestones

The architecture sequenced into buildable, dependency-ordered milestones. A roadmap, not binding —
[`architecture/decisions.md`](architecture/decisions.md) governs *what*, this governs *in what
order*. Gate definitions are in [`development-process.md`](development-process.md).

## Status

**M0–M2.2 are built; M3–M7 are designed or later.** Real, running code exists: `openfaithmap-api`
with its first module (`registration`), OpenFaithMap's first migration, two Next.js apps, and a
`docker-compose.yml` that stands up go-oikumenea, `oikumenea-console`, and `hermenea` alongside
them. M0, M1, M1.1, and M2.2 are Verified; M1.2, M2, and M2.1 are built but blocked on one shared
external step (see the stage board's 🔶 rows). M3–M6 remain design-complete-not-built; M7 is still
at idea stage.

**Audit pass, 2026-08-09.** A full docs↔code consistency and architecture review added six
milestones (M2.3–M2.6, M4.1) and five decision blocks (D-SharedDatabase, D-GoogleDirect,
D-OAuthClients, D-FlatRoot, D-PlatformModerator), plus a Correction to D-Moderation. Verified rows
below are frozen; where their prose cites a path that later moved, an append-only
`> **Note (audit 2026-08-09):**` line records the correction without editing the original text.

## Unresolved unknowns — read this before building anything

Every place this doc set currently says "we don't actually know." Kept here, at the top of the file
every session reads, so none of it can be lost in a module doc's open-seams section. Detail lives
where the third column points; this table exists so nothing is hidden, not to duplicate it.

### Group 1 — must be measured against a real instance

These are **assumptions, not facts.** Each one has a design built on top of it, and each was written
from reading go-oikumenea's docs rather than calling it. M1.1 and M2 both discovered the hard way
that this repo's assumptions about go-oikumenea's permission model are wrong more often than not —
so measure first, then build.

| # | The unknown | Blocks | Detail |
|---|---|---|---|
| U2 | **Can an anonymous caller read `GET /religion/discovery/sites`?** Every `religion` read is `RequireAnywhere`-gated, which denies subjects with no person behind them — and an anonymous caller is one. If the answer is no, the public map cannot read go-oikumenea at all and M4 needs a different design, not an adjustment. | **M4's `designed` gate** | [M2.5](#m25--discovery-reachability-spike) · [discovery.md](modules/discovery.md) |
| U3 | **Can go-oikumenea re-parent a live `Unit` in the `canonical` graph?** M4.1 has to move every existing congregation under a new jurisdiction unit while preserving its admin's grant. If there is no re-parenting path, that milestone blocks on an upstream request — exactly how M2's own org-creation assumption failed. Check this before scoping the rest of M4.1. | **M4.1**, and therefore **M5** | [M4.1](#m41--jurisdiction-units) |

### Group 2 — decisions deliberately deferred to their milestone

Real choices with no obvious default, parked on purpose. Each needs someone to pick, not measure.

| # | The decision | Owner |
|---|---|---|
| U7 | Cross-module foreign keys inside `openfaithmap-api` (`discovery_site_cache.content_site_id` → `content_sites`) are neither permitted nor forbidden by conventions.md. | `DS-OFM-13` |

### Group 3 — contradictions and orphans needing a call

Places where the doc set disagrees with itself or specifies something with no home. These do not
block a milestone; they mislead whoever reads them next, which is worse.

| # | The problem | Where |
|---|---|---|
| U9 | **`Impersonation` is an orphan and contradicts a binding invariant.** It is defined in the glossary — a moderator logging in as a congregation admin — and appears **nowhere else**: no `D-` block, no endpoint in moderation.md's API surface, no milestone. It also directly contradicts [core-integration.md](modules/core-integration.md)'s **no-on-behalf-of** invariant, which forbids OpenFaithMap ever acting as a specific person. Either write a decision explaining how it can exist (go-oikumenea would have to mint the impersonated session, not OpenFaithMap), or delete the term. **Do not build it from the glossary entry alone.** | `DS-OFM-15` · [glossary.md](glossary.md) |
| U10 | **M5 ships public unauthenticated write endpoints one milestone before M7 hardens them.** `POST /reports` and `POST /exclusion-check` are both anonymous by design; rate limiting is parked at M7. Decide at M5 scoping whether basic limiting moves forward. | `DS-OFM-9` · [moderation.md](modules/moderation.md) |
| U11 | **`churchSiteTypeID` fails silently.** If go-oikumenea's seeded `church` site type is ever renamed, `approveRequest` attaches every congregation to whatever the first site type happens to be, with no error. Prefer failing loudly. | [registration.md](modules/registration.md) |
| U12 | **Config bypasses the install-config convention.** `internal/platform/config` exists to hold openfaithmap-api's settings and is empty; `cmd/openfaithmap-api` reads five real settings straight from the environment via `requireEnv` — no schema, no validation, no ECV path for the secrets among them. | [conventions.md](architecture/conventions.md) |
| U13 | **Per-surface OAuth clients and WireGuard have no milestone**, because there is no deployment milestone at all. Both are recorded as prerequisites for any non-local-dev deployment; whoever creates that milestone inherits them. | `DS-OFM-14` |

## Stage board

**Gate legend.** ✅ passed · ⬜ not started · ➖ not applicable · 🔶 **passed once, now blocked on a
named dependency** — either built and waiting on an external action nobody in this repo can perform
(M1.2, M2, M2.1: an OAuth redirect URI), or designed and reopened because a load-bearing assumption
turned out to be unverified (M4, M5). Always named in that milestone's prose; 🔶 without a named
blocker is just ⬜. `Verified` additionally requires CI green on `main` — see
[development-process.md](development-process.md).

| # | Decided | Designed | Backend | Migrated | UI | Verified | Stage |
|---|---|---|---|---|---|---|---|
| M0 · Scope & core-dependency | ✅ | ✅ | ➖ | ➖ | ➖ | ✅ | **Verified.** Artifact: this doc set (`architecture/decisions.md`, `modules/core-integration.md`, `glossary.md`), coherence-checked (no dangling relative links, no `faithmap-app` references). A docs-only milestone; its exit criterion is the doc set existing and being internally consistent, met. |
| M1 · go-oikumenea integration wiring | ✅ | ✅ | ✅ | ➖ | ✅ | ✅ | **Verified.** `docker-compose.yml` runs a real go-oikumenea instance (published image, migrated, shared Postgres). Service-principal auth (D-ServiceIdentities) proven end-to-end — `internal/coreintegration`, `scripts/bootstrap-service-principal`. `openfaithmap-web`'s session layer (Auth.js v5, Google as sole OIDC provider, ID-token forwarding — `web/auth.ts`, `web/lib/oikumenea.ts`, `/login`, `/whoami`) proven end-to-end with a real browser OAuth round-trip: Google login → `/whoami` resolves a real `personId`/`email` through go-oikumenea's PDP. Required `scripts/bootstrap-admin-person` (go-oikumenea's JIT is link-on-match only — a fresh instance has no person for a new Google identity to link onto) and a restart of `oikumenea-app` after an install-config edit (install config is read once at boot, not hot-reloaded from the bind-mounted file — worth remembering for future config changes). |
| M1.1 · Core-integration doc corrections | ✅ | ✅ | ➖ | ➖ | ➖ | ✅ | **Applied.** Three inaccuracies in `architecture/decisions.md` / `modules/core-integration.md` / `modules/web-facade.md` / `architecture/overview.md`, found by testing M1 against a real go-oikumenea instance rather than assumed from its docs. Items 1 (`audit.write` doesn't exist) and 3 (Keycloak → Google-direct) corrected in the docs themselves. Item 2 (`religion.read` unusable by a service principal) recorded as an upstream go-oikumenea gap — a feature request, not a doc-only fix, needed before M4. |
| M1.2 · Instance-admin console (`oikumenea-console`) | ✅ | ✅ | ➖ | ➖ | ✅ | 🔶 | **Built, blocked on the shared Google-redirect-URI step — see prose.** D-InstanceAdminConsole (`architecture/decisions.md`). `docker-compose.yml` runs go-oikumenea's own published console image as OpenFaithMap's third UI surface, super-admin-only. |
| M2 · Church-admin self-service facade | ✅ | ✅ | ✅ | ✅ | ✅ | 🔶 | **Built, blocked — see prose.** `modules/registration.md` (new — corrects the original "no schema of its own" framing). Backend, migration, and UI all built and proven end-to-end via curl against the live stack (submit, D-Exclusions check, list, approve's real go-oikumenea writes, reject, double-approve guard). **Verified blocked on two things:** the real browser round-trip (submit → operator approves → roster renders), and M2.3's security fixes — the 2026-08-09 audit found the operator gate discloses every submitter's PII to any congregation admin, so this milestone must not be marked Verified as built. |
| M2.1 · Split the UI into public and admin surfaces | ✅ | ✅ | ➖ | ➖ | ✅ | 🔶 | **Built, blocked — see prose.** `architecture/decisions.md`'s D-AdminSurface, `modules/web-facade.md` (narrowed to the public surface) + `modules/web-admin.md`. `web/` split into two independent Next.js apps, `web/apps/web` (no session, ever) and `web/apps/admin` (the only surface that ever holds a credential) — no application logic changed, only moved. |
| M2.2 · Reference-data seeding (`hermenea`) | ✅ | ✅ | ➖ | ➖ | ➖ | ✅ | **Verified.** D-BulkImport (`architecture/decisions.md`, corrected), `modules/import.md` (corrected). Deploys go-oikumenea's own `hermenea` companion service for reference-data seeding — no OpenFaithMap code, corrects the original congregation-bulk-import-CLI premise. Real `docker compose up` proof: `geo-countries-iso3166` synced successfully, 250 rows confirmed in `oikumenea.geo_countries` — see prose below. |
| M2.3 · Registration hardening (security + correctness) | ✅ | ✅ | ✅ | ✅ | ➖ | 🔶 | **Built, blocked on the live two-real-token proof — see prose.** D-PlatformModerator's target-scoped-capability pattern, implemented via go-oikumenea's `Authorize` + a new `assignment.read` grant. All three defects the 2026-08-09 audit found in M2's shipped code are fixed: the operator gate is now target-scoped (no longer discloses every submitter's PII to any congregation admin), `getRequest` is authorized (submitter or operator), and `approveRequest` is idempotent and resumable. **Blocks M2's Verified.** |
| M2.4 · CI repair + deployment hygiene | ✅ | ✅ | ➖ | ⬜ | ➖ | ✅ | **Verified (2026-08-10).** All five items landed (PR #9) and CI has run green on `main` repeatedly since — [run 31331332615](https://github.com/olehmushka/open-faith-map/actions/runs/31331332615) (the merge commit itself) through at least [run 31335009849](https://github.com/olehmushka/open-faith-map/actions/runs/31335009849) (M2.5's merge). Item 1's own acceptance criterion ("a green CI run on `main`, with both apps' lint and build visible as separate matrix legs") is met for real, not just reasoned about — the thing this milestone existed to fix is confirmed fixed. |
| M2.5 · Discovery reachability spike | ➖ | ✅ | ➖ | ➖ | ➖ | ✅ | **Verified (2026-08-10).** A measurement, not a decision — hence `Decided ➖`. Both the machine-subject and anonymous-subject denials were live-verified true; [go-oikumenea#33](https://github.com/olehmushka/go-oikumenea/issues/33) fixed the machine-subject half (`0.0.2`, re-verified live), leaving anonymous access as the one still-open gap, by design (#33's own scope). Its own exit criterion (measure + file the upstream issue) is fully met, and the inherited CI-on-`main` blocker is resolved — [run 31335009849](https://github.com/olehmushka/open-faith-map/actions/runs/31335009849), this milestone's own merge, is green. **Still blocks M4's `designed` gate** — that's a separate, un-inherited block: the redesign this enables (cache-only public reads) is buildable now but still needs to be written, not just enabled. |
| M2.6 · TypeScript SDK for `openfaithmap-api` | ✅ | ✅ | ➖ | ➖ | ✅ | ✅ | **Verified (2026-08-10).** D-Stack's Conjure-first rule and both consumer module docs required a generated typed client; `web/apps/admin/lib/registration.ts` was hand-written and had already drifted (missing the `PROVISIONING` status M2.3 added). `U4` resolved: generated in place into `web/apps/admin/lib/openfaithmap/generated`, not a separate package — see D-Stack. Proven live: removing a field from `api/registration.conjure.yml` and regenerating breaks `web/apps/admin`'s build with a real compile error, the milestone's own acceptance criterion. Merged as PR #12; [run 31341045908](https://github.com/olehmushka/open-faith-map/actions/runs/31341045908) confirms CI green on `main` at that merge commit. |
| M3 · Content / site-builder backend | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **Verified (2026-08-10).** `modules/content.md` — sites/documents(pages)/blocks/block-type catalog, `internal/content`, `migrations/0004_content.sql`, a site-editor UI in `web/apps/admin`. U5 (slug collisions) and U6 (RID vs uuid) both resolved. Proven live end-to-end: create→write→schema-validate→publish, public-read filtering (draft hidden, published visible, no auth), against a real stack. Found and fixed two real bugs no static check caught: a `httprouter` startup panic from two routes sharing a wildcard slot under different parameter names, and `congregation-admin`'s role missing `assignment.read` (same defect class M2.3 already fixed once, for a different role). CI-green acceptance criterion now met — see prose. |
| M4 · Public discovery site | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **Verified (2026-08-10).** `modules/discovery.md` — cache schema + facade over go-oikumenea's `religion` discovery search, redesigned per M2.5's finding: lazy cache-only public reads, no scheduled refresh job, `DS-OFM-13`'s FK resolved. Also ships `content`'s `POST`/`EVENT` kinds (deferred from M3) and the public map/congregation-page UI. Found and got fixed same-day a real upstream RLS bug ([go-oikumenea#34](https://github.com/olehmushka/go-oikumenea/issues/34)); live end-to-end proof against `oikumenea:0.0.3` with real data. |
| M4.1 · Jurisdiction units | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-10), not yet Verified.** D-JurisdictionUnits (supersedes D-FlatRoot's simplification). Real, operator-assigned jurisdiction units; existing congregations can be re-parented onto one. Proven live end-to-end against a real `docker compose` stack — see prose. **`Verified` needs a green CI run on `main` at the merge commit** (M2.4's gate), not yet attempted here. |
| M5 · Moderation | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-10), not yet Verified.** `modules/moderation.md` — reports/actions/appeals + a standalone D-Exclusions taxon-check dry-run. All three dependencies the 2026-08-09 audit found are now resolved (D-PlatformModerator, D-Moderation's Correction, and M4.1 landing cleared the third). **`Verified` needs a green CI run on `main` at the merge commit and a live two-real-token proof**, not yet attempted here. |
| M6 · Vouching | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-10), not yet Verified.** `modules/vouching.md` — web-of-trust guarantor model. Its `moderation.read`/`moderation.act` gates and its `content.manage`-equivalent guarantor-standing check both resolved through D-PlatformModerator, the same mechanism moderation already uses. **`Verified` needs a green CI run on `main` at the merge commit and a real two-browser-session proof (a guarantor-with-standing vs. a guarantor-with-none, and a moderator vs. a non-moderator)**, not yet attempted here — see prose. |
| M7 · Hardening / real-user feedback | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | **Idea.** Named and sequenced here; no `D-<Name>` block or module doc yet — first real milestone to pass through the full pipeline once M1–M6 have shipped code and real congregations are using the platform. Note that the audit moved three items people might expect here (CI, least-privilege DB role, API port exposure) forward into M2.4, because they gate every intervening milestone's Verified rather than being end-state polish. |

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

**As built:** `docker-compose.yml` runs go-oikumenea from its published image
(`docker.io/olegamysk/oikumenea`) against one shared Postgres instance with OpenFaithMap
(`oikumenea` / `openfaithmap` schemas — a simplification from two separate database instances,
decided over chat, not yet its own `D-<Name>`). **No Keycloak** — go-oikumenea is configured to
trust Google directly (`deploy/oikumenea-install.yml`); a real deviation from this decision's
original "shared Keycloak realm" premise, now corrected in `architecture/decisions.md`'s
D-CoreDependency and `architecture/overview.md` (M1.1 item 3), though still not its own `D-<Name>`.
`internal/coreintegration` + `scripts/bootstrap-service-principal` prove the service-principal path
for real: a GCP service account mints its own Google ID token per call, go-oikumenea resolves it by
`(issuer, subject)`, and the PDP enforces its grant — verified against `connector.read` (see M1.1,
item 2, for why not `religion.read`).

`openfaithmap-web`'s session layer is now built: Auth.js v5 with Google as the sole OIDC provider
(`web/auth.ts`), forwarding the Google **ID token** (not the access token) as the bearer on every
go-oikumenea call (`web/lib/oikumenea.ts`, via the published `oikumenea-client` npm package). A
`/login` page starts the flow; `/whoami` is the end-to-end proof artifact — it calls
`identityFederation.whoami()` through the forwarded token and renders the result, relying on
go-oikumenea's JIT provisioning (`deploy/oikumenea-install.yml`'s `idp.jit`) to resolve a first-time
Google login with no manual account setup. `docker-compose.yml` gained an `openfaithmap-web`
service (port `3002`) so the login flow can actually reach `oikumenea-app` (which publishes no host
port).

**Still open:** the actual browser OAuth round-trip has not been run — it needs the project owner's
real Google OAuth client secret (`AUTH_GOOGLE_SECRET` in the root `.env`) and a real browser. Until
that's done and `/whoami` is confirmed rendering a real resolved identity, this milestone's
`Verified` column stays `⬜`.

> **Note (audit 2026-08-09).** This milestone is Verified and its text above is frozen. Three
> corrections for readers, none of which change what it proved:
> 1. **Paths moved at M2.1.** `web/auth.ts` → `web/apps/admin/auth.ts`; `web/lib/oikumenea.ts` →
>    `web/apps/admin/lib/oikumenea.ts`. `/login` and `/whoami` now live in `openfaithmap-admin`,
>    not `openfaithmap-web` — which, post-split, holds no session at all (D-AdminSurface).
> 2. **"Still open" is resolved.** The stage-board row above already records the real browser
>    round-trip as done; this prose paragraph predates that and was never revised.
> 3. **Two as-built deviations named here as "not yet its own `D-<Name>`" now have one:** the
>    shared-Postgres simplification is [D-SharedDatabase](architecture/decisions.md), and
>    Google-direct-instead-of-Keycloak is
>    [D-GoogleDirect](architecture/decisions.md). D-SharedDatabase records a gap this milestone
>    introduced without noticing: `openfaithmap-api` connects as the `postgres` superuser, so it can
>    reach go-oikumenea's schema directly — see M2.4.

### M1.1 · Core-integration doc corrections

**Depends on:** M1 (these were found while proving M1's service-principal path against a real
go-oikumenea instance, not while writing the original design). **Leaves deployable:** no code
change — this is `architecture/decisions.md` / `modules/core-integration.md` /
`modules/web-facade.md` / `architecture/overview.md` prose catching up to verified reality.

Three inaccuracies, applied:

1. **`modules/core-integration.md`'s authorization-touchpoints table listed `audit.write`** as a
   grant OpenFaithMap's service principal holds. That permission does not exist: go-oikumenea's
   `audit` module has no write endpoint — "there is no write endpoint; writes happen in-process"
   (go-oikumenea's own `docs/modules/audit.md`), and its permission catalog defines only
   `audit.read`. `scripts/bootstrap-service-principal` rejected the request live
   (`PrincipalGrantInvalid: unknown permission code`). **Fixed:** `audit.write` removed from the
   table; if D-Moderation (M5) still needs OpenFaithMap's own moderation actions written into
   go-oikumenea's audit trail, that needs a real design — the mechanism the old text assumed isn't
   there.
2. **The same table implied the service principal can call go-oikumenea's `religion` read
   endpoints** (discovery-cache refresh, D-Exclusions taxon-ancestor resolution) using its
   `religion.read` grant. Verified false against a real instance: every `religion` module read
   endpoint is gated with `RequireAnywhere`, a person-shaped PEP path that unconditionally denies a
   machine subject, regardless of grants (`internal/authorization/pep` — "every person-shaped PEP
   path denies it at its empty-subject guard"). Only the `connector`/`wiring` modules are
   machine-reachable (`RequireService`/`RequireServiceOrPerson`) in the current go-oikumenea
   version. **Recorded, not fixed in code** — this is an upstream go-oikumenea gap, not an
   OpenFaithMap misconfiguration. `core-integration.md`'s open seams now cross-reference it; a
   go-oikumenea feature request (make the relevant religion reads `RequireServiceOrPerson`) is
   needed before M4 is built on an assumption that doesn't hold today.
3. **D-CoreDependency's "shared Keycloak realm" premise** no longer matched what M1 actually built
   (Google directly, no Keycloak). **Fixed:** `architecture/decisions.md`'s D-CoreDependency,
   `architecture/overview.md`'s diagram and deployment-topology section, and
   `modules/web-facade.md`'s session/identity section all now describe Google-direct, ID-token
   forwarding — not their own `D-<Name>` yet, still folded into D-CoreDependency's existing text.

> **Note (audit 2026-08-09).** Frozen text above; two follow-ups landed since.
> - Item 3's "not their own `D-<Name>` yet" is resolved:
>   [D-GoogleDirect](architecture/decisions.md) now records it, including two consequences this
>   milestone did not draw out — every human user needs a Google account (a real adoption
>   constraint for the Ukraine-first rollout), and there is no second issuer to fail over to.
> - Item 1's open question ("if D-Moderation still needs moderation actions in go-oikumenea's audit
>   trail, that needs a real design") is answered: see D-Moderation's **Correction**.
>   `openfaithmap.moderation_actions` becomes the ledger of record and the one-ledger goal is
>   dropped. Item 2 (`religion.read` unusable by a service principal) is still open and is now
>   owned by **M2.5**, which also asks the larger question this item did not: whether those reads
>   deny *anonymous* callers too.

### M1.2 · Instance-admin console (`oikumenea-console`)

**Depends on:** M1 (needs `oikumenea-app` running). **Leaves deployable:** yes — adds a third UI
surface with no dependency on anything OpenFaithMap-specific, so it can land any time after M1,
independent of M2's registration work.

**As built.** `docker-compose.yml` gained an `oikumenea-console` service, pinned to
`docker.io/olegamysk/oikumenea-console:0.0.1` (matching how `oikumenea-app` is pinned rather than
using `latest`), reaching `oikumenea-app` the same way `openfaithmap-api` does (compose-internal
network at `https://oikumenea-app:8443`, dev-only `NODE_TLS_REJECT_UNAUTHORIZED=0`). It reuses
`openfaithmap-web`'s own Google OAuth client (same `AUTH_GOOGLE_ID`/`AUTH_GOOGLE_SECRET`) rather
than provisioning a new one or reintroducing Keycloak — `deploy/oikumenea-install.yml`'s Google
issuer entry already trusted this audience, so no install-config value changed, only its comment
(doc-accuracy). Its own Auth.js session secret (`OIKUMENEA_CONSOLE_AUTH_SECRET` in `.env.example`)
is distinct from `openfaithmap-web`'s `AUTH_SECRET` — two independent sessions. Published on host
port `3003` (continuing the `300x` sequence `openfaithmap-api`/`openfaithmap-web` already use) for
local dev — not profile-gated the way go-oikumenea's own `docker-compose.yml` treats its
`console-bff`, a deliberate choice for this repo.

**Network exposure — decided, not yet implemented.** A real (non-local-dev) deployment puts
`oikumenea-console` behind a WireGuard VPN rather than a bare public port, given its instance-wide
blast radius (D-InstanceAdminConsole). There is no real deployment target in this project yet, so
there is nothing to implement beyond recording the decision — see D-InstanceAdminConsole's
consequences in `architecture/decisions.md`.

**Still open:** the real login round trip (Google OAuth → `oikumenea-console` rendering the
instance-admin UI) has not been run — same caveat M1's own browser round-trip has: it needs the
project owner's real `AUTH_GOOGLE_SECRET` and a browser, plus one manual, out-of-repo step: adding
`http://localhost:3003/api/auth/callback/google` as an authorized redirect URI on the same Google
Cloud OAuth client `openfaithmap-web` already uses. Until that's done, `Verified` stays 🔶
(built, blocked on a named external action — see the stage board's gate legend).
`scripts/bootstrap-service-principal`/`scripts/bootstrap-admin-person` are untouched — they remain
the CI/reproducible-bootstrap path; `oikumenea-console` is the human-facing alternative for the same
actions, not a replacement. No OpenFaithMap backend or schema work — `oikumenea-console` has no
OpenFaithMap module doc of its own since OpenFaithMap builds none of it (D-Facade extended to the
UI layer — see D-InstanceAdminConsole in `architecture/decisions.md`).

### M2 · Church-admin self-service facade

**Depends on:** M1. **Leaves deployable:** a congregation admin can register a congregation (a
real `church`-domain `Organization`/`Unit` in go-oikumenea) and see their own roster — still no
public-facing content or discovery.

**As built — deviates from the original provisioning-flow design in one load-bearing way.**
`core-integration.md`'s original flow assumed a prospective admin's own token could call
`POST /religion-orgs`/self-grant authority. Verified false against a real instance: creating a
top-level org needs `religion.catalog.manage` (instance-wide), and granting anyone authority over a
brand-new unit needs `assignment.grant` **on that unit**, which not even an instance admin holds
automatically — go-oikumenea has no self-service org-creation path for an ungranted user. Real
shape built instead: submit (any authenticated person, D-Exclusions-checked) → a registration
operator (a real person with authority over one shared root unit — bootstrapped once via
`scripts/bootstrap-registration-org`, the same "operator-owned DB access" trust level
go-oikumenea's own `D-Bootstrap` uses for the first instance admin) approves, and *their* token
performs the real writes. Full detail: [modules/registration.md](modules/registration.md).

`openfaithmap-api` gained its first real module doing this: `internal/registration`
(domain/adapters/application/transport), `api/registration.conjure.yml` (the first Conjure IDL in
this repo — `godel-conjure-plugin` wired for real), and `migrations/0001_registration.sql`
(OpenFaithMap's first schema, applied by a new `openfaithmap-migrate` compose service).
`openfaithmap-web` gained `/register` (the wizard), `/admin/registrations` (operator
approve/reject), and `/my-congregation` (the roster — M2's "see their own roster" exit criterion),
via a hand-written fetch client (`web/lib/registration.ts` — no generated TS SDK for
openfaithmap-api yet, see registration.md's open seams).

Proven end-to-end via curl against the live stack: submit, the D-Exclusions check (rejects
`jehovahs_witnesses`), list, approve (real `createChildOrg`/location/site/position/grant all
landed and were confirmed in go-oikumenea), reject, and the double-approve guard. **Not yet run:**
the real browser round-trip — submitting through `/register`, approving through
`/admin/registrations`, and confirming `/my-congregation` renders the result — needed before
`Verified` flips to ✅.

**Audit 2026-08-09 — three defects found in the code above; `Verified` must not flip until M2.3
lands.** The curl proof exercised the happy path of each endpoint, which is why none of these
surfaced: they are authorization and failure-mode defects, not functional ones.

1. **The operator gate discloses every submitter's PII to every congregation admin.**
   `application.IsOperator` asks `MyCapabilities()` whether the caller holds `religionorg.manage`
   *anywhere* — there is no target unit in the check. `scripts/bootstrap-registration-org` puts
   `religionorg.manage` in the **`congregation-admin`** role. So the moment anyone's registration is
   approved, they can call `GET /registration/v1/requests` and read every pending submission
   platform-wide: congregation names, street addresses, coordinates, and submitter person RIDs.
   `registration.md` describes this gate as "cosmetic only," which is true for *writes* — approve
   and reject are still re-decided by go-oikumenea's PDP — and false for *reads*, where this check
   is the entire access-control decision.
2. **`getRequest` has no authorization at all.** `transport.GetRequest` calls the application
   service directly: no `whoami`, no operator check, no submitter comparison. Any authenticated
   person can read any request by id. The Conjure contract states "The submitter or an operator
   (verified live) may read it"; the code implements neither half.
3. **`approveRequest` is a non-atomic 7-call distributed write.** `createChildOrg` →
   `listSiteTypes` → `createLocation` → `createSite` → `createPosition` → `fillPosition` →
   `grantAssignment` → local `UPDATE`, with no compensation and no idempotency key. A failure at
   any step after the first leaves an orphaned go-oikumenea unit (possibly with a location, a site,
   and a position) while the local request stays `PENDING`. Retrying then creates a *second* child
   org — `slugCode` appends a random suffix, so there is no unique-code collision to stop it, and
   nothing reconciles the orphan.

The as-built text above is otherwise accurate and stays as written.

### M2.1 · Split the UI into public and admin surfaces

**Depends on:** M1 (the session layer this revises), M2 (the UI routes this revises). **Leaves
deployable:** no code change yet — this is a decision-and-design pass, same shape as M1.1.

`modules/web-facade.md`'s original design put the entire UI — anonymous public site,
congregation-admin console, moderator console — in one Next.js app, on the explicit reasoning that
splitting would "duplicate session handling for no isolation benefit." That premise turned out to
be false: the public site never authenticates anyone at all (no session, no tracking — analytics is
deferred, not built), so there was never a session to share. Discussed and decided: the UI splits
into `openfaithmap-web` (anonymous, no session) and a new `openfaithmap-admin` (the only surface
that ever holds a credential) — full rationale in `architecture/decisions.md`'s D-AdminSurface.

**As built.** `web/auth.ts`, `web/lib/oikumenea.ts`, `web/lib/registration.ts`, `/login`,
`/whoami`, `/register`, `/register/submitted`, `/admin/registrations`, and `/my-congregation` all
moved unchanged into the new `web/apps/admin` app; `web/app/page.tsx`/`layout.tsx` moved into
`web/apps/web`, which now has no Auth.js dependency at all — confirmed by dropping `next-auth`/
`oikumenea-client` from its `package.json` entirely. `docker-compose.yml` gained an
`openfaithmap-admin` service (port `3004`); `openfaithmap-web`'s service lost its entire `AUTH_*`
environment and its dependency on `oikumenea-app`/`openfaithmap-api`. No application logic
changed — this was a pure restructure plus config/doc sync. Two independent apps, no npm workspace,
no `web/packages/*` — see D-AdminSurface's "as implemented" note in `architecture/decisions.md`.

**Still open:** the new `openfaithmap-admin` origin needs
`http://localhost:3004/api/auth/callback/google` added as an authorized redirect URI on the shared
Google OAuth client (external, Google Cloud Console) before a real login round-trip can be tested —
same open item M1/M1.2 already carry. `Verified` stays 🔶 until that's done and a browser confirms
both `openfaithmap-admin` login and `openfaithmap-web` shipping zero Auth.js code.

**Audit 2026-08-09 — this milestone broke CI and nobody noticed.** Deleting `web/package.json` and
`web/package-lock.json` left `.github/workflows/ci.yml`'s `web` job pointing at files that no longer
exist; it fails at `actions/setup-node` before `npm ci` even runs
(`Some specified paths were not resolved, unable to cache dependencies`). Every CI run on `main`
since PR #5 has been red, including M2.2's. `CONTRIBUTING.md` promises "`./godelw verify` — the same
gate CI runs" and `development-process.md` defines Verified as "tests pass"; neither has been true
since this landed. Fixed by **M2.4**, which also makes CI-green an explicit part of the Verified
gate so this cannot recur silently.

Also noted: the shared-Google-OAuth-client arrangement this milestone inherited from M1.2 is now
recorded as [D-OAuthClients](architecture/decisions.md) — `openfaithmap-admin` and
`oikumenea-console` mint tokens with an identical `aud`, so the "ascending blast radius" tiering
D-InstanceAdminConsole describes has no enforcement at the token layer. Not an escalation (the PDP
still gates on the caller's own assignments), but per-surface clients are now a documented
prerequisite for any real deployment.

### M2.2 · Reference-data seeding (`hermenea`)

**Depends on:** M0 only (D-CoreDependency — go-oikumenea must be running). **Leaves deployable:**
yes — go-oikumenea's core gets enriched reference data (country geometries, a fuller language
catalog, external orgs); nothing else about the platform changes.

**Correction (found while scoping this milestone for real, before any code landed).** The original
design — a Go CLI at `cmd/hermenea` bulk-onboarding *congregations* by replaying `registration`'s
`POST /requests`/`POST /requests/{id}/approve` in a loop — didn't survive contact with
`registration`'s actual contract. `SubmitRegistrationRequest` has no contact-person field, and
`submittedByPersonId` is always resolved server-side from the caller's own token, never
client-supplied ([registration.md](modules/registration.md)) — so a CLI holding only the operator's
token would have made the operator, not each congregation's real contact, the submitter and
resulting `congregation-admin` grantee for every row. Worse, the name was already taken:
go-oikumenea ships its own service also named `hermenea` (sibling repo, `cmd/hermenea`) — a
persistent reference-data companion (countries, languages, external orgs, geo places) with its own
database and credential, entirely unrelated to congregation registration. Full detail in
D-BulkImport's Correction ([architecture/decisions.md](architecture/decisions.md)).

**As built.** M2.2 is now deploy wiring only: `docker-compose.yml` gained `init-hermenea-db`,
`migrate-hermenea`, and a `hermenea` service (built from a sibling go-oikumenea checkout via
`Dockerfile.hermenea` — no published image exists — matching the `OIKUMENEA_SRC` sibling-checkout
pattern `oikumenea-migrate` already uses), plus the two shared-secret env vars
(`HERMENEA_OIKUMENEA_TOKEN`/`OIKUMENEA_HERMENEA_TOKEN`) added to `oikumenea-app`. A new
`deploy/hermenea-install.docker.yml` (adapted from go-oikumenea's own reference config, retargeting
`oikumenea.base-url` at this repo's `oikumenea-app` service name) declares the source list —
including the bundled, network-free `geo-countries-iso3166` source that covers the "countries"
case directly. No OpenFaithMap backend code, schema, or UI — see
[modules/import.md](modules/import.md)'s "Operating hermenea" section for the runbook. The original
congregation-bulk-import scenario is retargeted to a future, unscoped, separately-named tool — see
[open-questions.md](open-questions.md)'s `DS-OFM-10`.

**Verified.** Brought the full stack up (`OIKUMENEA_SRC=../go-oikumenea docker compose up --build`)
against real containers: `init-hermenea-db`/`migrate-hermenea` completed, `hermenea` built from the
sibling checkout and started, reached readiness (`GET /status/readiness` → 200). The bundled
`geo-countries-iso3166` source ran automatically on first boot and reached
`hermenea.import_runs.status = succeeded` (0 created / 2 updated / 30 skipped against go-oikumenea's
own pinax-seeded baseline); `oikumenea.geo_countries` held 250 real rows afterward. Manually
re-triggering via `POST /hermenea/v1/sync/geo-countries-iso3166` (bearer =
`OIKUMENEA_HERMENEA_TOKEN`) also returned `200 {"jobId":...,"status":"queued"}` — confirming the
corrected route (`docs/modules/import.md`'s runbook fixes the `/sync/{source}` path go-oikumenea's
own compose comment shows, which 404s against a real instance; the real base-path is
`/hermenea/v1`). Stack torn down after verification (`docker compose down`).

> **Note (audit 2026-08-09).** Verified and frozen; two deployment-hygiene items this milestone
> introduced are picked up by **M2.4**, neither affecting what it proved:
> - `HERMENEA_OIKUMENEA_TOKEN` / `OIKUMENEA_HERMENEA_TOKEN` are **hardcoded literals** in the
>   committed `docker-compose.yml` (both `oikumenea-app` and `hermenea`), unlike every other secret
>   in this repo, which goes through `${...}` + `.env.example`. A real deployment has to remember to
>   edit the compose file itself.
> - `hermenea` publishes host ports `9443`/`9444` with that same hardcoded trigger token as the only
>   gate on `POST /hermenea/v1/sync/{source}`. Fine for local dev, and the compose comment explains
>   the reasoning, but it is the one service in this stack whose exposure was decided without a
>   corresponding note in `architecture/overview.md`'s deployment topology.
>
> Also, the "exact verified table name" open seam in `modules/import.md` is resolved by this
> milestone's own verification (`oikumenea.geo_countries`, 250 rows) — that doc still lists it as
> unconfirmed and has been corrected.

### M2.3 · Registration hardening (security + correctness)

**Depends on:** M2 (this fixes M2's own code). **Blocks:** M2's `Verified`. **Leaves deployable:**
yes — no schema-breaking change; one expand-only migration adds a status value and an index.

Three defects the 2026-08-09 audit found in M2's shipped `registration` module (full description in
M2's detail section above). Each item below is independently landable; they are grouped because
they touch the same three files.

**1 · Scope the operator gate to the root unit.** `application.IsOperator` currently asks
`MyCapabilities()` for a bare `religionorg.manage` with no target, and `congregation-admin` holds
that permission on its own unit — so every approved congregation admin reads as an operator and
`listRequests` returns every submitter's name, address, and coordinates to them.

- Replace the untargeted check with a **target-scoped** capability check against
  `Config.RootUnitID` specifically, following D-PlatformModerator's pattern.
- **`U1`, settled.** `MyCapabilities()` is flat and self-only by deliberate go-oikumenea design — it
  cannot be scoped to a unit (confirmed against its Go SDK signature, response struct, Conjure spec,
  and go-oikumenea's own docs). The real target-scoped primitive is `Authorize` (`POST /authorize`,
  `{subjectPersonId, action, unitId}` → `{allow}`) — but it requires the *caller* to already hold
  `assignment.read` reaching the target unit, no self-exemption (go-oikumenea's own "OQ-5",
  deliberate). Neither `registration-operator` nor `congregation-admin` held it, so
  `scripts/bootstrap-registration-org` now grants `registration-operator` that permission (and
  reconciles it onto an already-bootstrapped instance's existing role via `UpdateRole`, not just a
  fresh one). An "authority probe" alternative (attempt a real operation, infer authority from
  success) was investigated and ruled out: no read anywhere in go-oikumenea is gated on
  `religionorg.manage`, and the one plausible no-op write, `SetOrgProfile` with omitted fields, is
  actively destructive (its SQL upsert uses `EXCLUDED.*` without `COALESCE`, silently nulling an
  existing profile's fields on repeat calls).
- Acceptance: a person holding only a `unit`-scoped `congregation-admin` grant on their own
  congregation calls `GET /registration/v1/requests` and sees **only their own** submissions. A
  person holding the `subtree`-scoped `registration-operator` grant on the root unit sees all.
  Prove both with two real tokens against the live stack, not with a unit test alone.

**2 · Authorize `getRequest`.** It currently has no check of any kind. Resolve the caller via
`whoami`, then permit iff the caller is the request's `submittedByPersonId` **or** passes the same
root-unit operator check as item 1. Return `Registration:RequestNotFound` (not a distinct
"forbidden") for a non-permitted read, so the endpoint does not confirm the existence of requests
the caller may not see.

- Acceptance: person A submits, person B (authenticated, unrelated) gets `RequestNotFound` for A's
  id; A and an operator both get the row.

**3 · Make `approveRequest` idempotent and resumable.** Today a failure after `createChildOrg`
orphans a real go-oikumenea unit and leaves the request `PENDING`, and a retry creates a second org.

- Add a `PROVISIONING` status to `registration_requests.status`'s `CHECK` constraint and to the
  Conjure `RegistrationStatus` enum, plus a nullable `created_unit_id` write **as soon as
  `createChildOrg` returns** — that is the only step that cannot be re-derived.
- `approveRequest` on a `PROVISIONING` row resumes from the persisted `created_unit_id` rather than
  re-creating the org. The remaining steps (location, site, position, grant) are re-runnable; where
  go-oikumenea rejects a duplicate, treat the conflict as success.
- The `registration_requests_decision_shape` CHECK gains a `PROVISIONING` arm
  (`created_unit_id IS NOT NULL AND decided_by_person_id IS NOT NULL`).
- Acceptance: kill `openfaithmap-api` between `createSite` and `grantAssignment`, restart, re-issue
  the same approve call, and confirm exactly one child org exists in go-oikumenea and the request
  reaches `APPROVED`.

> **As implemented (2026-08-09).** Item 3 landed first, on its own; items 1 and 2 landed together in a
> follow-up pass, once `U1` was settled (see item 1's text above). One addition beyond item 3's
> original text: `createSite` turns out to have no uniqueness key at all (checked against
> go-oikumenea's own `religion.conjure.yml`), so "where go-oikumenea rejects a duplicate, treat the
> conflict as success" doesn't apply to it the way it does to
> `createPosition`/`fillPosition`/`grantAssignment`. `ensureSite` instead lists the unit's sites first
> and reuses an existing primary one on resume. See `modules/registration.md`'s defect 3 for the full
> design.
>
> For items 1 and 2: live-verified against a real `docker compose` stack (not just review) that
> `scripts/bootstrap-registration-org`'s new `reconcilePermissions` actually calls go-oikumenea's
> `UpdateRole` and adds `assignment.read` to an already-existing `registration-operator` role (and
> makes zero calls, correctly, on a third run); and that `Authorize` itself returns `Allow=false`
> before the corresponding role-assignment grant exists and `Allow=true` after, using the same call
> shape `IsOperator` now makes. Not yet run against a live instance: the milestone's own two-real-token
> acceptance test (a `congregation-admin`-only account, a `registration-operator` account, both via
> real browser Google OAuth) — that's the one piece not achievable headlessly. This milestone's gates
> stay as they are until that's done.

**Also in scope, because there is nothing to regress against today:** the first real unit tests in
this repo. `checkNotExcluded`'s ancestor walk (including the cycle cap), `slugCode`, and the
status-transition guards currently have **zero** coverage — `go test ./...` passes vacuously, since
the only test in the repo is the skipped `coreintegration` integration test.

### M2.4 · CI repair + deployment hygiene

**Depends on:** M2.1 (which broke CI), M2.2 (which added the hermenea secrets). **Blocks:** every
later milestone's `Verified`. **Leaves deployable:** yes — CI and configuration only, no product
change.

**1 · Repair the `web` CI job.** `.github/workflows/ci.yml`'s `web` job still runs
`working-directory: web` with `cache-dependency-path: web/package-lock.json`; M2.1 deleted both
`web/package.json` and `web/package-lock.json` when it split the app in two. The job has failed on
every run since — it never reaches `npm ci`. Replace it with a matrix over `web/apps/web` and
`web/apps/admin`, each running its own `npm ci && npm run lint && npm run build`.

- Acceptance: a green CI run on `main`, with both apps' lint and build visible as separate matrix
  legs.

**2 · Make CI-green part of the Verified gate.** `development-process.md`'s Verified row says "tests
pass" without saying whose. Amend it to require a green CI run on `main` at the milestone's merge
commit, and note it in the stage-board honesty section — the failure mode this milestone exists to
fix is precisely that M2.2 was marked `✅` Verified on top of a `main` that had already been red
since M2.1, and M2.1 itself sat at `🔶` rather than being caught as the actual cause.

**3 · Least-privilege database role (D-SharedDatabase).** `openfaithmap-api` connects as the
`postgres` superuser, so it can read and write go-oikumenea's entire schema — which D-CoreDependency
calls "rejected outright." Add an `openfaithmap_app` role with `USAGE` + DML on the `openfaithmap`
schema only and **no** grant on `oikumenea`, mirroring `oikumenea-init-role`'s existing shape; point
`DATABASE_URL` at it.

- Acceptance: from inside `openfaithmap-api`'s container, `SELECT` against any `oikumenea.*` table
  fails with a permission error, while the registration flow still works end-to-end.
- While here, check whether `--allow-dirty` is still needed on both migrate services now that each
  has its own revisions schema; it currently suppresses Atlas's drift detection permanently.

**4 · Stop publishing `openfaithmap-api`'s host ports.** `docker-compose.yml` maps `3000`/`3001` to
the host, contradicting `architecture/overview.md`'s "the two UI surfaces are the only public
ingress" target — and this service accepts and forwards bearer tokens. `openfaithmap-admin` already
reaches it internally at `https://openfaithmap-api:3000`, so removing the mapping should be nearly
free; keep `3001` (management/health) if local debugging needs it, and say so in the compose
comment.

**5 · Move hermenea's shared secrets into `.env`.** `HERMENEA_OIKUMENEA_TOKEN` and
`OIKUMENEA_HERMENEA_TOKEN` are hardcoded literals in `docker-compose.yml` in two places each.
Convert to `${...}` with entries in `.env.example`, matching every other secret in this repo.

> **As implemented (2026-08-09).** All five items landed. Verified against a real stack, not just
> code review: `web/apps/{web,admin}`'s `npm ci && npm run lint && npm run build` both pass clean
> with no env vars set (item 1); a fresh `docker compose up` through `openfaithmap-api` boots on the
> new `openfaithmap` role, `SELECT` against `oikumenea.persons` from that role returns `permission
> denied for schema oikumenea`, and the exact INSERT/SELECT/UPDATE/DELETE `adapters/store.go` issues
> against `openfaithmap.registration_requests` — including the `set_updated_at` trigger — all succeed
> (item 3); `localhost:3000` refuses the connection while `localhost:3001/status/liveness` returns
> `200` (item 4); both `oikumenea-app` and `hermenea` resolve the token env vars from `.env` via
> `docker inspect`/`docker compose config` (item 5). Item 3's "while here" sub-question is settled,
> not just checked off: `--allow-dirty` stays on both migrate services permanently, but not for the
> reason originally guessed — see `D-SharedDatabase`'s updated consequences and the compose comment
> above `oikumenea-migrate`. **Update (2026-08-10): item 1's CI-green acceptance criterion is now
> met.** PR #9's merge to `main` (commit `c4d2111`) produced a green CI run —
> [31331332615](https://github.com/olehmushka/open-faith-map/actions/runs/31331332615) — and every
> merge since has stayed green through at least
> [31335009849](https://github.com/olehmushka/open-faith-map/actions/runs/31335009849). `Verified`
> flips in the stage board above.

### M2.5 · Discovery reachability spike

**Depends on:** M1 (a running instance to measure against). **Blocks:** M4's `designed` gate.
**Leaves deployable:** yes — a measurement and an upstream issue; no code in this repo.

M1.1 item 2 established that `religion` module reads are `RequireAnywhere`-gated, a person-shaped
PEP path that denies machine subjects, which kills `discovery`'s service-principal cache refresh.
The audit found a larger, **unverified** question hiding behind it:
`architecture/overview.md`'s anonymous-discovery request path claims
`GET /religion/discovery/sites` is an "unauthenticated public read." If `RequireAnywhere` denies
callers with an empty subject, an anonymous caller is *also* an empty subject — which would mean the
public map cannot read go-oikumenea at all, and M4's entire premise fails, not just its cache.

Nobody has measured this. It is the single highest-risk unknown on the roadmap and it costs one
afternoon to settle.

**Do:**

1. Bring up the stack and call `GET /religion/discovery/sites` three ways — with no `Authorization`
   header, with a real person's ID token, and with the service principal's token — recording the
   exact status and error body for each.
2. Repeat for the taxon reads `registration`'s D-Exclusions check depends on (`GetTaxon` and its
   ancestor walk), which today only ever run under a real person's token.
3. Write the result into `modules/discovery.md` and `modules/core-integration.md` as verified fact,
   replacing the current inference.
4. File the upstream go-oikumenea feature request (`RequireServiceOrPerson`, or genuinely public,
   on the relevant `religion` reads) and link the issue from `core-integration.md`'s open seams —
   M1.1 named this as needed "before M4" and it has been unowned since.

**Exit criterion:** a documented answer to "can an anonymous caller and can a service principal each
read go-oikumenea's discovery and taxon endpoints today," plus a filed upstream issue for whichever
answers are no. If anonymous reads turn out to be denied, M4's design needs reopening before it is
built — that is the whole point of doing this first.

> **As measured (2026-08-09).** Brought up a real `docker compose` stack
> (`OIKUMENEA_SRC=../go-oikumenea docker compose up --build`), registered OpenFaithMap's real GCP
> service principal via `scripts/bootstrap-service-principal` (fresh instance, so it had never been
> registered — no prior M2.5 measurement could have skipped this step), and called
> `GET /religion/v1/discovery/sites` and `GET /religion/v1/taxa/{id}` three ways. **Both unverified
> assumptions confirmed, and the anonymous case is the worse of the two:**
>
> | Caller | Result |
> |---|---|
> | No `Authorization` header | `401 IdentityFederation:Unauthorized` — denied at authentication, before `RequireAnywhere` is even evaluated |
> | Service principal, holding `religion.read` (instance-wide, freshly granted) | `403 Authorization:PermissionDenied {action: religion.read}` — `RequireAnywhere` denies the machine subject despite the grant, exactly as M1.1 inferred |
> | Real person token (go-oikumenea's own local-dev bootstrap-admin identity — a genuine person-shaped subject, holding **no** `religion.read` grant of its own) | `200 {"sites":[]}` on `discovery/sites`; `404 Religion:TaxonNotFound` on a probed nonexistent taxon id — both correct, both reached real application logic |
>
> The person-token row is a useful control: it succeeded with **no explicit grant**, confirming
> `RequireAnywhere` really does mean "any person passes, any machine fails," not "any grant passes."
> That pins the service-principal row's 403 on subject-shape alone. **`checkNotExcluded`'s ancestor
> walk needed no separate endpoint** — it's repeated `GetTaxon` calls
> (`internal/registration/application/service.go`), so the `taxa/{id}` measurement covers it
> directly. One correction to the method as originally scoped: item 1's "no `Authorization` header"
> case had to be tested against the **correct** base path, `/religion/v1/...` — go-oikumenea's own
> `religion.conjure.yml` declares `base-path: /religion/v1`, not `/religion` as
> `architecture/overview.md`'s prose (now corrected) had it; the wrong path also returns 401 (a
> global auth filter runs before route matching), which would have produced the right answer for
> the wrong reason had the base path not been checked against the conjure source directly.
>
> **Exit criterion fully met.** Three-way measurement done and written into `discovery.md`,
> `core-integration.md`, and `architecture/overview.md` (this session). Upstream feature request
> filed: [go-oikumenea#33](https://github.com/olehmushka/go-oikumenea/issues/33) (requests
> `RequireServiceOrPerson` on the read-only `religion` discovery/taxon endpoints; deliberately does
> not ask upstream to solve genuine anonymous access, a separate and larger question).
>
> **Update: the upstream fix landed the same day, before this milestone's own row closed.**
> `fedc094` ("fix(religion): gate instance-wide reads with RequireServiceOrPerson") replaced
> `RequireAnywhere` with `RequireServiceOrPerson` on every `religion` read touchpoint, released as
> `docker.io/olegamysk/oikumenea:0.0.2` — `docker-compose.yml`'s pin bumped from `0.0.1` to match.
> Re-ran the identical three-way measurement against `0.0.2` rather than trusting the diff or the
> closed-issue label:
>
> | Caller | `discovery/sites` | `taxa/{id}` |
> |---|---|---|
> | No `Authorization` header | `401` — unchanged, by design | `401` — unchanged |
> | Service principal, `religion.read` | `200` (was `403`) | correct `404` (was `403`) |
> | Real person token | `200` (unchanged) | correct `404` (unchanged) |
>
> The machine-subject row flips exactly as the issue requested; the anonymous row is untouched
> exactly as the issue scoped it to be — both consistent with intent, not just "some status code
> changed."
>
> **Update (2026-08-10): `Verified`, now that M2.4's inherited CI-on-`main` block has cleared.**
> This milestone's own merge produced a green run —
> [31335009849](https://github.com/olehmushka/open-faith-map/actions/runs/31335009849) — so the
> stage board above now reads `✅`. `designed` gate for M4 stays reopened regardless of the CI
> question, and that block is *not* inherited — it's this milestone's own finding: anonymous reads
> are still denied, so `openfaithmap-web` must still never call go-oikumenea directly for discovery
> — only `discovery_site_cache`. What changed is that the service principal that would populate that
> cache can now actually read go-oikumenea, so the redesign is buildable, not just aspirational.
> Writing it into `discovery.md` and `web-facade.md` is real M4 work, not attempted here.

### M2.6 · TypeScript SDK for `openfaithmap-api`

**Depends on:** M2 (the first Conjure contract to generate from). **Blocks:** nothing hard, but it
should land **before M3's UI** so a second hand-written client is never written. **Leaves
deployable:** yes — a build-tooling addition plus one client swap.

D-Stack makes Conjure the contract source of truth and says generated code is never hand-edited;
`web-admin.md` and `web-facade.md` both state as a dependency that every call goes through the
generated typed client, "never a raw `fetch`." `web/apps/admin/lib/registration.ts` is a
hand-written fetch client — a documented, deliberate scope cut at M2, but one that multiplies with
every module M3/M5/M6 add.

Stand up the pipeline go-oikumenea already has (its `scripts/gen-ts-client.sh` + `tools/ir2openapi`
+ `conjure-typescript` chain is the reference implementation), generate a client from
`api/registration.conjure.yml`, and replace `lib/registration.ts` with it.

- Decide and record: does the generated client ship as a `file:` dependency into
  `web/apps/admin`, or as a published package? D-Stack notes go-oikumenea uses `file:` in dev and a
  real npm package once published; with two independent apps and no workspace (D-AdminSurface's
  as-implemented note), `file:` across directories needs checking against each app's own Dockerfile
  build context, which currently only copies its own directory.
- Acceptance: `web/apps/admin` has no hand-written HTTP client for `openfaithmap-api`, the
  registration flow still works end-to-end, and regenerating from an edited `.conjure.yml` produces
  a compile error in the app when a field is removed.

> **As implemented (2026-08-10).** Pipeline ported from go-oikumenea's reference implementation
> almost line-for-line: `tools/conjure-ir-dump` (a trimmed `ir2openapi`, IR extraction only — this
> repo has no OpenAPI doc generation to preserve), `scripts/rewrite-ir-packages.mjs` (same 2-seg →
> 3-seg fix go-oikumenea needed — `api/registration.conjure.yml`'s `default-package:
> openfaithmap.registration` hit the identical `conjure-typescript` "at least 3 segments" error),
> `scripts/gen-ts-client.sh` (`--verify` drift mode included), `make sdk`.
>
> **U4 resolved, not deferred:** generated directly into
> `web/apps/admin/lib/openfaithmap/generated`, no separate package — see D-Stack
> (`architecture/decisions.md`) for the full reasoning (single in-repo consumer, isolated Docker
> build context, no workspace). `web/apps/admin/lib/openfaithmap/index.ts` is the one hand-written
> file, wiring the generated `RegistrationService` onto a `conjure-client` `DefaultHttpApiBridge` —
> the same façade shape as go-oikumenea's own `clients/typescript/src/index.ts`, and as
> `lib/oikumenea.ts` already used for go-oikumenea itself.
>
> `lib/registration.ts` keeps its five exported function names and error shape
> (`RegistrationApiError`) unchanged — none of its four call sites needed to change — but now
> delegates to the generated client and re-exports its types instead of hand-copying them. That
> swap fixes a real, live bug: the hand-written `RegistrationStatus` type was `"PENDING" |
> "APPROVED" | "REJECTED"`, missing `"PROVISIONING"` (added in M2.3 item 3's migration) — silently
> wrong until now, exactly the drift class a generated client rules out.
>
> **Acceptance criterion proven directly, not just reasoned about:** removed `congregationName`
> from `RegistrationRequest` in `api/registration.conjure.yml`, regenerated, and confirmed
> `web/apps/admin`'s `next build` fails with three real `TS2339` errors (one per call site reading
> that field) — then reverted. `npm run lint && npm run build` pass clean on the real swap;
> `./godelw verify` passes clean on the new Go tool.
>
> **Real end-to-end proof against a live stack**, same shape as M2's own original curl proof but
> now through the generated client: brought up a fresh `docker compose` stack, bootstrapped the
> registration org (`scripts/bootstrap-registration-org`) against it, and drove
> `submitRequest → listRequests → getRequest → approveRequest → submitRequest → rejectRequest`
> through `createOpenFaithMapClient` from a plain Node process (not through Next.js — the
> generated client and its façade have no framework dependency). `approveRequest` performed a real
> go-oikumenea `createChildOrg` and returned a real `createdUnitId`; the reject path set a real
> `rejectionReason`; a lookup on a nonexistent id returned the correct
> `404 Registration:RequestNotFound`. **Update (2026-08-10): merged as PR #12** — [run
> 31341045908](https://github.com/olehmushka/open-faith-map/actions/runs/31341045908) confirms CI
> green on `main` at that merge commit, so `Verified` flips in the stage board above.

### M3 · Content / site-builder backend

**Depends on:** M2 (needs a real congregation `Unit` RID to attach a site to), plus **M2.3**
(don't build a second module on top of an authorization pattern that is known-broken) and
**M2.6** (so its UI consumes a generated client rather than adding a second hand-written one).
**Leaves deployable:** a congregation admin can build and publish a real page; nothing public
discovers it yet (no map).

Builds `openfaithmap-api`'s `content` module per [content.md](modules/content.md): sites, pages,
blocks, translation groups, the block-type catalog seeded with the MVP set. **`post` and `event`
are out of scope for M3** and land at M4 — see that doc's open seams, where the audit resolved a
contradiction between this milestone's old "this milestone's own later iteration" phrasing and
content.md's "land at the discovery milestone." M4 is the answer: a post or event with no public
site to surface it has nowhere to be read.

Two audit corrections to [content.md](modules/content.md) apply to this milestone's scope:

- **`content.manage` is a target-scoped capability check**, not "call `GET /units/{unitId}` and
  treat a successful read as proof of standing" (D-PlatformModerator). Read authority is not write
  authority; the original definition would let anyone who can see a congregation edit its site.
- **`content_sites.slug` collision has no design.** The column is globally `UNIQUE`, but under
  D-FlatRoot self-service registration two congregations named "St. Mary's" collide, and unlike
  `slugCode`'s unit codes there is no disambiguation story. Pick one (suffix, per-country
  namespacing, or admin-chosen with a uniqueness probe) when writing the migration.

> **As implemented (2026-08-10).** Full stack: `api/content.conjure.yml`, `internal/content/{domain,
> application,adapters,transport}` (mirroring `internal/registration`'s exact hexagonal shape),
> `migrations/0004_content.sql`, and a site-editor UI (`web/apps/admin/app/admin/sites/[unitId]/...`)
> using the M2.6 generated-client pattern (`lib/content.ts` mirrors `lib/registration.ts`).
>
> **U5 and U6 both resolved, not deferred.** U5 (slug collisions): admin-chosen slug, probed
> race-safely at write time (insert first, catch the unique-violation, translate to
> `Content:SlugTaken` — never check-then-insert), content.md's own recommended option. U6 (RID vs
> uuid): checked go-oikumenea's *actually-deployed* migrations directly — every table uses a plain
> `uuid` PK, not the composed-URN scheme `conventions.md` described; that scheme is a
> documented-but-unshipped go-oikumenea redesign. `registration_requests`'s uuid PK was never a
> deviation, and `content_*` follows the same real precedent. `conventions.md` corrected
> accordingly.
>
> **`content.manage` resolved as a reuse, not a new permission:** go-oikumenea's existing
> `religionorg.manage`, already granted to `congregation-admin` on their own unit. One real
> consequence, live-verified: a registration operator's subtree grant on the shared root also
> satisfies `content.manage` for every congregation, not just ones tied to their own approvals —
> see content.md's authorization-touchpoints section. Acceptable for now (operators are a small,
> trusted set), flagged as an open seam for later.
>
> **Conjure has no per-endpoint anonymous auth — verified against the compiler's own `AuthType`
> definition (Header/Cookie only, no None), not just inferred.** `openfaithmap-web` holds no session
> at all, so content.md's original "content.manage for draft / none for public" single-endpoint
> shape doesn't compile to anything buildable. Split into two services in one `.conjure.yml`:
> `ContentService` (`default-auth: header`) and `ContentPublicService` (no auth declared at all —
> confirmed the generated Go interface has no `bearertoken.Token` parameter whatsoever, not just an
> optional one).
>
> **Two real bugs found only by actually booting the server and calling it live, neither caught by
> `go build`, `go vet`, or `./godelw verify`:**
> 1. A `httprouter` startup panic: `ContentPublicService`'s original `GET /sites/{congregationUnitId}`
>    shared a wildcard tree position with `GET /sites/{siteId}/documents` under a differently-named
>    parameter — httprouter requires the same wildcard name at a shared position, full stop, and
>    panics at route-registration time otherwise. This flaw was already latent in content.md's
>    original single-service sketch, not introduced by the service split; moved `getSite` to
>    `GET /units/{congregationUnitId}/site` instead.
> 2. `congregation-admin`'s role (`scripts/bootstrap-registration-org`) was missing `assignment.read`
>    — the exact same defect class M2.3 found and fixed for `registration-operator`, just never
>    applied to the other role. Without it, `Authorize` returns `PermissionDenied` for every real
>    congregation admin regardless of holding `religionorg.manage`. Fixed there; the script's
>    existing idempotent-reconciliation path (`reconcilePermissions`) backfills it onto an
>    already-bootstrapped instance safely — confirmed live ("added missing permissions, now: [...
>    assignment.read]").
>
> **Live-verified end-to-end** against a real `docker compose` stack, driven through the generated
> TypeScript client (not curl — proving the M2.6 codegen pattern extends cleanly to a second
> module): `createSite` → `createDocument(PAGE)` → `putBlocks` (one intentionally-invalid block
> confirmed rejected with `Content:BlockDataInvalid`, then a valid set saved) → anonymous
> `getSite`/`listPublicDocuments` (correctly empty pre-publish) → `transitionDocument(PUBLISH)` →
> anonymous reads now return the published document and its blocks → a second, still-draft document
> correctly returns `Content:DocumentNotFound` to an anonymous `getPublicBlocks` call → anonymous
> `listBlockTypes` returns all 13 seeded types. **Not attempted:** a true cross-tenant
> `Content:Forbidden` proof — needs a second real identity, the same limitation M2.3's own
> acceptance criteria already named as not achievable headlessly.
>
> **Update (2026-08-10): `Verified`, now that a confirmed green CI run exists.** The first
> post-merge run on `main` at this milestone's merge commit
> ([31366317786](https://github.com/olehmushka/open-faith-map/actions/runs/31366317786)) actually
> **failed**: `gödel verify (format + lint)` reported 11 `undefined: werror` errors in the generated
> `internal/conjure/openfaithmap/content/errors.conjure.go`, while `build + vet + unit tests` and
> both `web` jobs passed. Investigated before assuming the generated code was wrong (per D-Conjure,
> it is never hand-edited): `./godelw verify --skip-test` reproduced locally against the same commit
> with **0 issues**, `werror`'s import is identical to the already-passing `registration` module's,
> and golangci-lint v2.12.2's `--timeout` is disabled by default, ruling out a lint deadline. Content
> generates far more error-handling code than registration did (1526 lines / 80 `werror.` call sites
> vs. 622 / 32) — consistent with this being the first time a CI-environment flake in the `verify`
> job's golangci-lint pass (not a real bug) has been large enough to surface. Re-running the exact
> same commit's failed jobs (`gh run rerun 31366317786 --failed`, no code change) produced a clean
> green run on the same run id, confirming the local reproduction was ground truth. `Verified` flips
> in the stage board above.

### M4 · Public discovery site

**Depends on:** M2 (site data exists), go-oikumenea's `religion` sites/schedules being populated as
part of M2's provisioning flow, and — new at the 2026-08-09 audit — **M2.5**, whose measurement
decides whether this milestone's design survives contact with a real instance. **Leaves
deployable:** the first real end-to-end product demo — a visitor can find a congregation on a map
and read its published page.

Builds `discovery`'s cache + search facade per [discovery.md](modules/discovery.md) and the public
site rendering in [web-facade.md](modules/web-facade.md). Also ships `content`'s `post` and `event`
document kinds, deferred here from M3 (see M3's detail and content.md's open seams) — this is the
milestone where a public site exists to surface them.

**`designed` gate closed (2026-08-10).** The two load-bearing assumptions the 2026-08-09 audit
flagged as unverified are both now resolved by M2.5's live measurement:

1. **The cache refresh path — fixed upstream.** M2.5 confirmed `religion` reads deny machine
   subjects (`RequireAnywhere`); the filed upstream request
   ([go-oikumenea#33](https://github.com/olehmushka/go-oikumenea/issues/33)) landed the same day
   (`oikumenea:0.0.2`, `RequireServiceOrPerson`), so the service principal genuinely can call
   `GET /religion/v1/discovery/sites` and `GET /religion/v1/taxa/{id}` now — no on-behalf-of
   workaround needed.
2. **The public read path is permanently denied — by design, not an open question.** M2.5 confirmed
   an anonymous caller gets `401` even after `0.0.2`, deliberately out of #33's scope. The redesign
   this forces — `openfaithmap-web` reads only `discovery_site_cache`, refreshed lazily by the
   service principal on a cache miss, no scheduled job (`DS-OFM-2` resolved for MVP) — is now
   written into [discovery.md](modules/discovery.md) and [web-facade.md](modules/web-facade.md).
   `DS-OFM-13` (the `discovery_site_cache.content_site_id` cross-module FK) is also resolved: a
   real in-schema FK, `ON DELETE SET NULL`, following M3's own same-schema-FK precedent.

> **As implemented (2026-08-10).** `internal/discovery` mirrors `content`'s hexagonal shape and its
> public/authenticated service split: `DiscoveryPublicService` (`GET /discovery/v1/search`, no
> auth) and `DiscoveryService` (`POST /discovery/v1/refresh`, header auth, reusing
> `registration-operator`'s target-scoped root-unit check — no new go-oikumenea permission, same
> reuse precedent M3 set for `content.manage`). `migrations/0005_discovery.sql` adds
> `discovery_site_cache` per the redesign's schema. `content`'s `POST`/`EVENT` kinds turned out to
> need only a small change — the schema and `CreateDocumentInput`'s event fields were already in
> place from M3, so enabling them was one gate replaced
> (`Content:KindNotSupported` → `Content:EventMissingStart`, requiring `eventStartsAt` only for
> `EVENT`) plus wiring the two new request fields through transport/adapters.
> `openfaithmap-web` gained its first-ever backend calls: a token-free SDK client (structurally
> incapable of forwarding a bearer — no `token` field exists on its client options at all, a
> stronger guarantee than an unused optional field), a Leaflet/OpenStreetMap map/search home page,
> and a per-congregation page rendering the MVP block catalog. `scripts/gen-ts-client.sh` now
> generates into both `web/apps/admin` and `web/apps/web`'s own `lib/openfaithmap/generated` trees
> from one Conjure IR extraction, each app keeping its own full copy (no workspace, per M2.6/D-Stack
> precedent) — `make sdk-verify` checks both.
>
> **Four real bugs found only by bringing up a live stack, none caught by `go build`/`go vet`/
> `godelw verify`/`next build`:**
> 1. `docker-compose.yml`'s new service-principal credential mount used a bare relative path
>    (`${GOOGLE_APPLICATION_CREDENTIALS}:/app/var/service-account.json:ro`) — Compose treats a
>    source with no leading `./`/`/`/`~` as a *named volume* reference, not a bind mount, and
>    `docker compose ps` failed outright with "undefined volume" before any container started.
>    Fixed by prefixing `./`.
> 2. `openfaithmap-web`'s home page has no `auth()`/`cookies()` call (it never has a session), so
>    Next.js tried to **statically prerender** it at `next build` time, before
>    `OPENFAITHMAP_API_BASE_URL` is even set — `next build` failed reaching for the env var. Both
>    new data-reading routes now declare `export const dynamic = "force-dynamic"`, which is also
>    the semantically correct choice for a live discovery cache, not just a build-time workaround.
> 3. **A real upstream go-oikumenea bug**, one layer beneath #33's already-fixed PEP check: RLS on
>    `religion_sites` can't recognize an instance-wide (`org_id IS NULL`) principal grant —
>    `authz_principal_org_in_reach` requires an exact `org_id` match, and `NULL = <uuid>` is never
>    `true` in SQL. `GET /discovery/v1/search` succeeded at the API layer but silently returned
>    `{"sites":[]}` for a real, visible site. Filed as
>    [go-oikumenea#34](https://github.com/olehmushka/go-oikumenea/issues/34) (full root cause in
>    [discovery.md](modules/discovery.md)'s own update) and **fixed upstream the same day**,
>    released as `oikumenea:0.0.3` — `docker-compose.yml`'s pin bumped to match, re-verified live.
> 4. Leaflet touches `window` at module-evaluation time, crashing `openfaithmap-web`'s home page's
>    SSR outright (`ReferenceError: window is not defined`, HTTP 500) — invisible to `next build`'s
>    static-page check, only surfaced by loading the real page. Fixed by loading `DiscoveryMap`
>    through `next/dynamic({ ssr: false })` via a small client-component wrapper, since `ssr: false`
>    isn't usable directly from `page.tsx`'s Server Component.
>
> **Fully live-verified against a real `docker compose` stack** (real go-oikumenea `0.0.3`, real
> service principal, real `content_sites`/`religion_sites`/`content_documents` data from earlier
> milestones' testing) — not just the API boundary, the actual pages: `./godelw verify --skip-test`
> (0 issues), `go build`/`vet`/`test` clean, both web apps' `lint`/`build` clean, `make sdk-verify`
> confirms no drift across both generated-SDK copies. `GET /discovery/v1/search` proven end-to-end
> — cache starts empty, a query triggers a real service-principal call to go-oikumenea, the real
> site comes back (correct `latitude`/`longitude`/`contentSiteId`, the last resolved via
> `internal/discovery`'s cross-module `ContentResolver` call to `content`), `discovery_site_cache`
> confirmed populated in Postgres directly, and a repeat query serves from cache. `GET /` (the
> Leaflet map) and `GET /congregations/{unitId}` (the per-congregation page) were both fetched over
> real HTTP and confirmed to render the real data — the congregation page's server-rendered HTML
> contains the exact published blocks (`<h1>Welcome</h1><p>This is the home page.</p>`) sourced
> from a real `content_blocks` row. `Verified` flips in the stage board above.

Do not pass this milestone's `designed` gate again until M2.5 has answered both.

### M4.1 · Jurisdiction units

**Depends on:** M2 (real congregations to re-parent), and best done alongside or after M4 so the
jurisdictions modeled are the ones real registrations turned out to need. **Blocks:** M5's
`designed` gate. **Leaves deployable:** yes, if the re-parenting migration preserves existing
grants — that is the milestone's central risk.

[D-FlatRoot](architecture/decisions.md) accepted one flat root organization as an M2 build-time
simplification and requires real jurisdiction units before M5. Three designs in this doc set assume
an ancestor chain that does not exist today: moderation's `jurisdiction` queue scope, the glossary's
Jurisdiction term mapping, and D-Exclusions' org-level backstop (which has no per-body root unit to
attach a `religion_org_policies` row to).

**Do:**

1. **Model the jurisdiction layer.** Decide what a jurisdiction unit actually is for the Ukraine and
   USA rollouts — a national synod, a diocese, a district? — informed by whatever real registrations
   exist by then. This is a product decision and needs its own `D-` block or an extension of
   D-FlatRoot, not just a schema.
2. **Extend registration.** A submission gains a jurisdiction selection (or an operator assigns one
   at approval time); `approveRequest`'s `createChildOrg` targets that jurisdiction's unit rather
   than the single root. Note this interacts with M2.3's idempotency work — sequence M2.3 first.
3. **Re-parent existing congregations** in the `canonical` graph, preserving each congregation
   admin's `unit`-scoped grant. Verify against a real instance whether go-oikumenea supports
   re-parenting a live unit at all; if it does not, that is an upstream request and this milestone
   blocks on it. **Check this before scoping the rest** — it is the assumption most likely to be
   false, in the same way M2's own org-creation assumption was.
4. **Attach D-Exclusions' org-level backstop** to the excluded bodies' root units, making the
   documented defense-in-depth real for the first time.

**Exit criterion:** a congregation's ancestor walk returns at least one unit between it and the
root, moderation can express a jurisdiction-scoped query against it, and every pre-existing
congregation admin retains exactly the authority they had before the migration.

> **As implemented (2026-08-10).** All four items landed; full detail and rationale in
> [D-JurisdictionUnits](architecture/decisions.md).
>
> **Item 3's unknown, resolved before scoping the rest, as instructed.** go-oikumenea has no
> dedicated reparent endpoint for religion `Unit`s — verified directly against the sibling
> `go-oikumenea` checkout, not inferred. Re-parenting is achievable by composing the generic
> `tenant` module's `addEdge`/`removeEdge` on the `canonical` graph instead: two non-transactional
> calls, not one atomic move. No upstream request needed; this changed the design (a resumable job,
> not a single call) rather than blocking the milestone.
>
> **Item 1's product decision:** jurisdiction is denomination-aware but explicitly not one canonical
> hierarchy per tradition — Catholic polity fits a clean diocese tree, but Orthodox jurisdiction is
> often multiple and parallel even within one country/tradition, and many Protestant congregations
> have none at all. Operator-assigned, variable and optional depth, never inferred from `taxonId`.
> See D-JurisdictionUnits for the full reasoning.
>
> **Exit criterion revised, not silently met:** "at least one unit between it and root" assumed
> every congregation gets a jurisdiction, which turned out to conflict with item 1's own product
> decision (jurisdiction is genuinely optional). Revised to "at least one unit **when a jurisdiction
> applies**" — a congregation with none remains a direct child of root, and `jurisdiction`/`platform`
> moderation queue scopes coincide for it, by design.
>
> **Item 2 (extend registration):** operator assigns jurisdiction at **approval time**
> (`ApproveRegistrationRequest.jurisdictionUnitId`, optional) — the public `/register` wizard is
> unchanged, per an explicit product decision that self-service jurisdiction selection wasn't worth
> the added submitter-facing complexity this early. `internal/registration`'s `ensureUnit` targets
> the chosen unit instead of the configured root; the choice is persisted in the same
> `MarkProvisioning` write M2.3 already made resumable, so a `PROVISIONING` retry reuses the
> original choice regardless of what a later call carries.
>
> **Item 3 (re-parent existing congregations):** `POST/GET /requests/{id}/reparent`, a resumable
> `PENDING → NEW_EDGE_ADDED → OLD_EDGE_REMOVED → VERIFIED` state machine
> (`jurisdiction_reparenting_jobs`, `migrations/0006_registration_jurisdiction.sql`) mirroring
> M2.3's `PROVISIONING` precedent. **Add-before-remove, not remove-then-add** — live-confirmed
> `tenant_unit_edges` is `UNIQUE(graph_id, parent_id, child_id)`, not `UNIQUE(graph_id, child_id)`,
> so a unit can hold two simultaneous `canonical` parents; remove-then-add would open a real window
> where a `subtree`-scoped grant (registration-operator, platform-moderator) loses reach to the
> congregation mid-move. Triggered one congregation at a time from a new admin page
> (`/admin/registrations/reparent`), not a batch script, per this milestone's own framing of item 3
> as the central risk.
>
> **Item 4 (D-Exclusions backstop):** `scripts/bootstrap-exclusion-backstop` seeds a placeholder
> unit for each of the three named exclusions and attaches `excludes_child_creation` — live-verified
> that a subsequent `createChildOrg` beneath one is rejected with `Religion:ChildCreationExcluded`.
>
> **Nine live-verification checklist items, all confirmed against a real `docker compose` stack
> before any product code was written** (not inferred from source alone): `AddEdge`/`RemoveEdge`
> work on a real religion `Unit` in `canonical`; a unit tolerates two simultaneous `canonical`
> parents; the omitted-`graph` default really is `command`, not `canonical` (confirmed empirically —
> an explicit-`canonical` `AddEdge` right after an omitted-graph call still succeeded, meaning the
> first call landed elsewhere); a duplicate edge fails with `Tenant:UnitInvalid` ("edge already
> exists"), not a dedicated conflict type; a `subtree`-scoped grant and a `unit`-scoped grant both
> survive a real add-then-remove unchanged; `registration-operator` genuinely lacked `unit.read`/
> `unit.edges.manage` before this milestone's bootstrap change (confirmed via direct DB read of its
> permission set); `createChildOrg` with the seeded `jurisdiction` `orgKindId` is accepted;
> `AddOrgPolicy(excludes_child_creation)` genuinely blocks a subsequent `createChildOrg` beneath that
> unit.
>
> **Full end-to-end proof, driven through real HTTP calls to `openfaithmap-api`** (M2's own
> curl-proof pattern, same shape as M2/M3/M4's live verification): submit → approve with a
> jurisdiction assigned → confirmed the created unit's real go-oikumenea parent is the jurisdiction,
> not root → re-parent that same congregation onto a second jurisdiction → job reached `VERIFIED` →
> confirmed the real parent moved and the old jurisdiction is no longer an ancestor → confirmed the
> submitter's grant on the congregation is byte-identical before and after → re-called `reparent`
> with the same target and confirmed it resumes/no-ops to `VERIFIED` rather than erroring or
> repeating work.
>
> **One real bug found only by running the real flow, not caught by `go build`/`go vet`:**
> `transport.toAPI` never mapped `JurisdictionUnitID` from the domain type into the generated
> Conjure response struct — `approveRequest` silently returned `jurisdictionUnitId: null` even
> though the write itself correctly targeted the chosen unit. Caught by the end-to-end proof
> asserting the field round-tripped, not by any static check; fixed in `transport/service.go`.
>
> **Not yet run:** the real browser-driven admin UI flow (searching/creating a jurisdiction unit and
> approving through `/admin/registrations`, re-parenting through
> `/admin/registrations/reparent`) — same category of limitation M1/M1.2/M2.1 already carry (needs a
> real Google OAuth round-trip). The admin UI's server-rendered forms were proven to lint, type-check,
> and build clean, and call exactly the same `lib/registration.ts`/`lib/jurisdiction.ts` functions
> the HTTP-level proof above exercised directly — but the actual page has not been loaded in a
> browser. `Verified` also needs a green CI run on `main` at the merge commit (M2.4's gate), not
> attempted here.
>
> **Update (2026-08-10): the CI-green half is now confirmed.** This milestone's merge commit,
> `d6bf833` (pushed directly to `main`, no PR), produced a green run —
> [31387868674](https://github.com/olehmushka/open-faith-map/actions/runs/31387868674). `Verified`
> stays `⬜` — the real browser-driven admin UI proof above is the one remaining blocker.

### M5 · Moderation

**Depends on:** M3, M4 (needs real content and real discoverable congregations to moderate), and
**M4.1** (its `jurisdiction` queue scope has no ancestor chain to walk without it).
**Leaves deployable:** yes — a functioning platform with a safety net, ready for real (if limited)
users.

Builds `moderation` per [moderation.md](modules/moderation.md): reports, actions, appeals, and
wires the D-Exclusions check (already used at M2 registration time) into the moderator console.

**Two blockers the 2026-08-09 audit resolved into decisions**, both of which had left this
milestone specified against mechanisms that did not exist:

- **The moderator roster now has a home.** `moderation.read`/`moderation.act` were described only as
  "held by a small, fixed set of accounts" — no table, no role, no mechanism, in a milestone whose
  every endpoint gates on them (and in M6, which gates on them too).
  [D-PlatformModerator](architecture/decisions.md) makes them a go-oikumenea `platform-moderator`
  Role granted `subtree` on the shared root unit, resolved by a **target-scoped** capability check.
  `scripts/bootstrap-registration-org` gains that role; its permission set is decided here.
- **The audit-trail mirror is withdrawn.** D-Moderation's Correction establishes that `audit.write`
  does not exist and cannot, so `openfaithmap.moderation_actions` is the ledger of record and
  moderation.md's "mirrored into go-oikumenea's audit ledger before it is considered complete"
  invariant is replaced. Build the append-only table with the `reject_mutation()` guard and do not
  design a mirror.

**Still open when this is scoped:** rate limiting on `POST /reports` and `POST /exclusion-check`,
both public and unauthenticated. `DS-OFM-9` parks this at M7, but this milestone is what actually
ships the public endpoints — decide whether that is acceptable or whether basic limiting moves
forward into M5.

> **As implemented (2026-08-10).** Full stack, mirroring `internal/content`'s exact
> domain/adapters/application/transport shape: `api/moderation.conjure.yml` (two services, same
> public/header-auth split `content`/`discovery` already use — `ModerationPublicService` for
> `POST /reports` and `POST /exclusion-check`, `ModerationService` for the queue/action/appeal
> surface), `internal/moderation/{domain,adapters,application,transport}`,
> `migrations/0007_moderation.sql`, and a moderator console (`/admin/moderation`,
> `/admin/moderation/appeals`) plus a public report form on the congregation page
> (`web/apps/web/app/[locale]/congregations/[unitId]`).
>
> **`platform-moderator`'s permission set, resolved:** `unit.lifecycle` (its PDP marker — not
> `religionorg.manage`, which would make it indistinguishable from `registration-operator`/
> `congregation-admin`) plus `assignment.read` (root unit, same self-reach fix M2.3 already applied
> twice). Recorded as an addendum to D-PlatformModerator
> (`architecture/decisions.md`) rather than a new `D-<Name>` block. `scripts/bootstrap-registration-org`
> now seeds the role and prints its first root-unit assignment alongside `registration-operator`'s.
>
> **`reject_mutation()` implemented for real, for the first time in this repo** — ported
> schema-qualified from go-oikumenea's own migration pattern, guarding `moderation_actions`
> unconditionally. That surfaced a real design conflict before it shipped: the Conjure contract
> describes `ModerationAction.reversedByActionId` as if set on the *original* row once reversed —
> which would need an `UPDATE`, exactly what the guard forbids. Resolved by pointing the stored
> column the other way: a `REVERSE` row's own `reverses_action_id` (insert-time-only, backward) is
> the real fact; `reversedByActionId` on the original is derived at read time by looking forward for
> a row that reverses it (`adapters.Store.hydrateReversedBy`) — same fact, no forbidden write.
>
> **`CheckExclusion` runs under the server's own service-principal token**
> (`coreintegration.NewServiceClient`), reusing the same `religion.read` grant M2.5/M4 already
> proved reachable by a machine subject — the caller of the public endpoint is anonymous and has no
> token of its own to forward.
>
> **Scope cuts, all recorded in [moderation.md](modules/moderation.md)'s open seams, not silently
> dropped:** rate limiting stays at M7 per this section's own "still open" question, now decided;
> `queue_scope = 'jurisdiction'` has no query implementation yet (no moderator role is scoped
> narrower than the root unit for it to enforce); no action kind yet causes a real go-oikumenea- or
> content-side effect; appeal filing supports `CONGREGATION`-kind actions only;
> `registration`'s own exclusion check stays independent of this module's new one (reuses its
> `ExcludedTaxonCodes` list directly, not a copy).
>
> First real unit tests for this module's pure business rules (`domain.ValidateReasonCode`,
> `CanReverse`, `CanDecideAppeal`) — `go build ./... && go test ./...` and both apps'
> `npm run lint && npm run build` all pass clean.
>
> **Live-verified against a real `docker compose` stack** (`OIKUMENEA_SRC=../go-oikumenea docker
> compose up --build`), and this pass found and fixed a real bug the way M2's own curl proof once
> did: a `POST /reports` with `reasonCode: "DOCTRINAL_CONCERN"` fell through to a raw Postgres
> `CHECK`-constraint `500`, not the intended `Moderation:DoctrinalReasonNotAllowed` `400` — because
> `ReasonCode.Value()` (the accessor every other module already uses for an incoming enum) collapses
> *any* unrecognized string to `"UNKNOWN"` before `domain.ValidateReasonCode` ever saw the real one.
> Fixed by reading `.String()` instead for this one field (preserves the raw wire value), and by
> adding the `Moderation:DoctrinalReasonNotAllowed` error type that was missing from the Conjure
> contract entirely — confirmed live before and after. Also proven live: two anonymous writes with
> no `Authorization` header at all (`POST /reports`, `POST /exclusion-check`, both `200`);
> `GET /reports` with no token `403`; a real `POST /actions` → `POST /actions/{id}/reverse` →
> confirmed the original row's `reversedByActionId` is correctly derived (not stored) by reading it
> back; a second `reverse` call on the same action correctly `400`s with `Moderation:
> ActionNotReversible`; and, directly against Postgres, both a raw `UPDATE` and a raw `DELETE` on
> `moderation_actions` genuinely fail with `restrict_violation` — `reject_mutation()` really is
> unconditional, not just documented as such — plus the `moderation_actions_reverses_idx` unique
> index independently rejects a second `REVERSE` row for the same original, defense-in-depth behind
> the application's own check.
>
> **Not achievable headlessly, same limitation every prior milestone (M2, M2.1, M2.3, M4.1) already
> named:** the real two-*different*-people proof (a non-moderator refused, a separately-granted
> `platform-moderator` allowed) needs a real browser Google OAuth round trip — go-oikumenea's
> local-dev HS256 bootstrap issuer trusts exactly one subject (`local-admin`, the pre-provisioned
> bootstrap-admin), and that identity is *already* flagged instance-admin in this stack (a real,
> confirmed finding: `IsInstanceAdmin: true` bypasses every unit-scoped `Authorize` check in
> go-oikumenea's own PDP, `internal/authorization/domain/pdp.go` — so it cannot serve as a "before
> the grant" negative control no matter what `platform-moderator` is or isn't granted). A second,
> genuinely non-admin person (`scripts/bootstrap-admin-person -email moderator-test@example.com`)
> was created for this purpose but has no way to authenticate outside a real browser session. `Verified`
> stays `⬜` until that real-browser proof and a green CI run on `main` at the merge commit are both
> done (see the stage board below).
>
> **Update (2026-08-10): the CI-green half is now confirmed.** This milestone's merge commit,
> `d6b40f1` (PR #17), produced a green run —
> [31411640524](https://github.com/olehmushka/open-faith-map/actions/runs/31411640524). `Verified`
> stays `⬜` — the real two-different-people browser proof above is the one remaining blocker.

### M6 · Vouching

**Depends on:** M2 (needs real admins to vouch), M5 (a revoked guarantor's vouches route into the
moderation queue). **Leaves deployable:** yes — reduces manual verification load without changing
what's already live.

Builds `vouching` per [vouching.md](modules/vouching.md). Its two authorization dependencies —
`moderation.read`/`moderation.act` on the guarantor-management endpoints, and the
"`content.manage`-equivalent authority on **some** congregation" gate on `POST /vouches` — both
resolve through [D-PlatformModerator](architecture/decisions.md)'s target-scoped-capability pattern,
which they lacked any mechanism for before the 2026-08-09 audit.

> **As implemented (2026-08-10).** Full stack, mirroring `internal/moderation`'s exact
> domain/adapters/application/transport shape: `api/vouching.conjure.yml` (one authenticated
> `VouchingService` — unlike content/discovery/moderation, vouching has no genuinely-anonymous
> endpoint at all, so there is no public/private service split here), `internal/vouching/{domain,
> adapters,application,transport}`, `migrations/0008_vouching.sql` (`vouching_edges`, append-only,
> `reject_mutation()`-guarded, reusing the function `0007_moderation.sql` already created;
> `vouching_guarantor_status`, a mutable one-row-per-guarantor overlay), and admin UI
> (`/admin/vouching` — a moderator console for guarantor lookup/revoke and a vouches browser;
> `/admin/vouching/new` — the guarantor-facing "vouch for someone" form).
>
> **The "some congregation" gate resolved as a new request field, not a fixed target.**
> `CreateVouchRequest.guarantorCongregationUnitId` is the unit the *caller* proves their own
> `religionorg.manage` standing on — deliberately independent of `congregationUnitId` (the claim),
> per this doc's own "no relationship requirement between guarantor and claim." Implemented as
> `requireCongregationStanding`, a byte-for-byte duplicate of moderation's
> `requireCongregationAdmin` — confirmed live in `cmd/openfaithmap-api/main.go` that this repo's real
> convention is each module holding its own copy of this check (content's `requireManage` and
> moderation's own equivalent are already independent copies), not importing another module's
> `application` package.
>
> **The moderation-report fan-out is an in-process interface call, not a new `GUARANTOR_REVOKED`
> reason code.** `RevokeGuarantor` writes the revoked-status row first (load-bearing — "cannot vouch
> while revoked" must take effect immediately), then best-effort fans out one
> `moderation_reports` row per the guarantor's prior vouches via a `ModerationReporter` interface
> vouching owns; `cmd/openfaithmap-api/main.go`'s `moderationVouchReporter` adapter translates each
> event into moderation's existing `ReasonCode.OTHER` + a descriptive `detail` string, calling
> `moderationapplication.Service.FileReport` directly (same in-process cross-module shape as
> discovery's `ContentResolver`) — `internal/vouching/application` itself never imports moderation's
> domain or application packages. Avoids touching M5's already-migrated `ReasonCode` `CHECK`
> constraint, Conjure contract, and generated SDKs for this one caller.
>
> **One deliberate, scoped case deviation, confined to a single function.** The DB `CHECK` on
> `vouching_guarantor_status.status` is lowercase (`'trusted'`/`'revoked'`, this doc's own literal
> SQL text), while the Conjure `GuarantorStatus` enum stays uppercase, matching every other Conjure
> enum in this repo. The shim lives only in `transport/convert.go`'s `toAPIGuarantorStatusValue` —
> no other module needs it.
>
> **`GetGuarantorStatus` synthesizes `TRUSTED` for a person with no row**, never a not-found error —
> the column's own `DEFAULT 'trusted'` already means that; proven live (see below) rather than just
> reasoned about.
>
> **No claimant-facing "request a vouch" page was built.** This doc names the eventual real caller
> as "the web-admin.md congregation-claim flow" — confirmed absent from this codebase by direct
> search, not assumed. Building a claimant-facing entry point against a flow that doesn't exist yet
> would be scope creep beyond what this milestone specifies; the guarantor-facing form
> (`/admin/vouching/new`) needs no such dependency, since the guarantor is always an existing,
> already-admin caller.
>
> **Live-verified against a real `docker compose` stack** (`OIKUMENEA_SRC=../go-oikumenea docker
> compose up --build`), through real HTTP calls (not just the API boundary): `createVouch` performed
> a real go-oikumenea `Authorize` call and returned a real vouch row; `getGuarantorStatus` for a
> never-seen person RID returned synthesized `{"status":"TRUSTED"}` with no `updatedAt`;
> `listVouches` filtered correctly by claimant+congregation; `revokeGuarantor` wrote the status row
> and, confirmed directly in Postgres, exactly one `moderation_reports` row
> (`target_kind='VOUCHING_EDGE'`, `reason_code='OTHER'`, `detail` naming `guarantor_revoked` and the
> real guarantor/claimant/congregation RIDs) per affected vouch; a subsequent `createVouch` from the
> now-revoked guarantor correctly returned `Vouching:GuarantorRevoked`; a raw `UPDATE` and `DELETE`
> against `vouching_edges` directly in Postgres both failed with `restrict_violation` —
> `reject_mutation()` really is unconditional; and a request with no `Authorization` header at all
> was rejected before reaching the handler. `go build ./... && go test ./...`,
> `./godelw verify` (0 issues), both web apps' `npm run lint && npm run build`, and `make sdk-verify`
> (no generated-SDK drift) all pass clean. `/admin/vouching` and `/admin/vouching/new` were confirmed,
> over real HTTP, to redirect an unauthenticated visitor to `/login` exactly like every other admin
> page.
>
> **Not achievable headlessly, same limitation every prior milestone (M2, M2.1, M2.3, M4.1, M5)
> already named:** the real two-*different*-people proof (a guarantor with standing over one
> congregation vouching for a claim on an unrelated one vs. a guarantor with no standing anywhere;
> a `platform-moderator` account vs. a non-moderator) needs two real, separately-authenticated
> browser Google OAuth sessions — go-oikumenea's local-dev HS256 bootstrap issuer trusts exactly one
> subject (`local-admin`), already flagged instance-admin, which bypasses every unit-scoped
> `Authorize` check and so cannot serve as either a positive or a negative control on its own. All
> of this milestone's live checks above necessarily ran under that one instance-admin identity.
> `Verified` stays `⬜` until that real-browser proof and a green CI run on `main` at the merge
> commit are both done (see the stage board above).
>
> **Update (2026-08-10): the CI-green half is now confirmed.** This milestone's merge commit,
> `48d324a` (PR #18), produced a green run —
> [31421038134](https://github.com/olehmushka/open-faith-map/actions/runs/31421038134). `Verified`
> stays `⬜` — the real two-different-people browser proof above is the one remaining blocker.

### M7 · Hardening / real-user feedback (idea stage)

**Depends on:** M1–M6 all live with real congregations using the platform daily. Not yet decided
or designed — named here as the expected next milestone once real usage surfaces real problems
(rate limiting, moderation-queue UX, observability), matching the "real-user hardening" spirit of
the original FaithMap roadmap's own post-MVP stage, expressed here as a normal numbered milestone
rather than a separately-tracked stage.
