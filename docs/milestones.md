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
| M1 · go-oikumenea integration wiring | ✅ | ✅ | ✅ | ➖ | ✅ | ✅ | **Verified.** `docker-compose.yml` runs a real go-oikumenea instance (published image, migrated, shared Postgres). Service-principal auth (D-ServiceIdentities) proven end-to-end — `internal/coreintegration`, `scripts/bootstrap-service-principal`. `openfaithmap-web`'s session layer (Auth.js v5, Google as sole OIDC provider, ID-token forwarding — `web/auth.ts`, `web/lib/oikumenea.ts`, `/login`, `/whoami`) proven end-to-end with a real browser OAuth round-trip: Google login → `/whoami` resolves a real `personId`/`email` through go-oikumenea's PDP. Required `scripts/bootstrap-admin-person` (go-oikumenea's JIT is link-on-match only — a fresh instance has no person for a new Google identity to link onto) and a restart of `oikumenea-app` after an install-config edit (install config is read once at boot, not hot-reloaded from the bind-mounted file — worth remembering for future config changes). |
| M1.1 · Core-integration doc corrections | ✅ | ✅ | ➖ | ➖ | ➖ | ✅ | **Applied.** Three inaccuracies in `architecture/decisions.md` / `modules/core-integration.md` / `modules/web-facade.md` / `architecture/overview.md`, found by testing M1 against a real go-oikumenea instance rather than assumed from its docs. Items 1 (`audit.write` doesn't exist) and 3 (Keycloak → Google-direct) corrected in the docs themselves. Item 2 (`religion.read` unusable by a service principal) recorded as an upstream go-oikumenea gap — a feature request, not a doc-only fix, needed before M4. |
| M1.2 · Instance-admin console (`oikumenea-console`) | ✅ | ✅ | ➖ | ➖ | ⬜ | ⬜ | **Decided + designed, not yet deployed.** D-InstanceAdminConsole (`architecture/decisions.md`). Deploy go-oikumenea's own published console image as OpenFaithMap's third UI surface, super-admin-only — see prose below. |
| M2 · Church-admin self-service facade | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built, not yet verified — see prose.** `modules/registration.md` (new — corrects the original "no schema of its own" framing). Backend, migration, and UI all built and proven end-to-end via curl against the live stack (submit, D-Exclusions check, list, approve's real go-oikumenea writes, reject, double-approve guard). **Verified is still ⬜:** the real browser round-trip (submit → operator approves → roster renders) hasn't run yet. |
| M2.1 · Split the UI into public and admin surfaces | ✅ | ✅ | ⬜ | ➖ | ⬜ | ⬜ | **Decided + designed, not yet built.** `architecture/decisions.md`'s D-AdminSurface, `modules/web-facade.md` (narrowed to the public surface) + new `modules/web-admin.md`. Revises what M1's session layer and M2's UI routes actually build going forward — see prose below. |
| M2.2 · Bulk congregation import (`hermenea`) | ✅ | ✅ | ⬜ | ➖ | ⬜ | ⬜ | **Decided + designed, not yet built.** D-BulkImport (`architecture/decisions.md`), `modules/import.md` (new). A CLI that replays `registration`'s existing submit/approve endpoints in a loop — see prose below. |
| M3 · Content / site-builder backend | ✅ | ✅ | ⬜ | ⬜ | ⬜ | ⬜ | **Designed.** `modules/content.md` — full entity model, Conjure sketch. (M2's `registration_requests` table was actually OpenFaithMap's first schema — see `modules/registration.md` — this doc's "first genuinely new schema" framing predates that finding.) |
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

### M1.2 · Instance-admin console (`oikumenea-console`)

**Depends on:** M1 (needs `oikumenea-app` running). **Leaves deployable:** yes — adds a third UI
surface with no dependency on anything OpenFaithMap-specific, so it can land any time after M1,
independent of M2's registration work.

Adds `oikumenea-console` — go-oikumenea's own published console image, reused unmodified
(D-InstanceAdminConsole) — as OpenFaithMap's third UI surface, for super admins (go-oikumenea
instance admins) only. Concretely:

1. Add an `oikumenea-console` service to `docker-compose.yml` (image
   `docker.io/olegamysk/oikumenea-console`), reaching `oikumenea-app` over the compose-internal
   network the same way `openfaithmap-api` does today.
2. Decide and implement its network exposure. Unlike `openfaithmap-web`/`openfaithmap-admin`
   (deliberately public), `oikumenea-console` carries instance-wide power — this milestone's exit
   criterion includes picking a real restriction (VPN, IP allowlist, protected subdomain), not
   leaving it as a bare `ports:` mapping the way `openfaithmap-api`'s host-port gap currently is.
3. Document it as the human-facing replacement for what `scripts/bootstrap-service-principal` and
   `scripts/bootstrap-admin-person` do today — those scripts can stay for reproducible/CI
   bootstrapping, but a human operator should be able to do the same actions through
   `oikumenea-console` instead of `psql`/a one-off Go script.

No OpenFaithMap backend or schema work — this is a deployment-and-access-policy milestone, same
shape as M1's `oikumenea-app` addition. `oikumenea-console` has no OpenFaithMap module doc of its
own since OpenFaithMap builds none of it (D-Facade extended to the UI layer — see
D-InstanceAdminConsole in `architecture/decisions.md`).

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

**What this makes stale in M1/M2's own "as built" prose above**, without rewriting it (same
convention M1.1 used): M1's session layer (`web/auth.ts`, `web/lib/oikumenea.ts`, `/login`,
`/whoami`) is described as living in `openfaithmap-web` — going forward, that code's destination is
`openfaithmap-admin`. M2's UI routes (`/register`, `/admin/registrations`, `/my-congregation`) are
described as living in the single `web/` app — going forward, all three move to
`openfaithmap-admin` too, since submitting a registration already requires being logged in.

**Not done in this pass:** no code moved, no `web/` restructuring, no `docker-compose.yml` service
added. `Backend`/`UI` stay `⬜` until an actual `openfaithmap-admin` app exists and the routes above
move into it.

### M2.2 · Bulk congregation import (`hermenea`)

**Depends on:** M2 (the `registration` module's `POST /requests`/`POST /requests/{id}/approve`
endpoints must exist — this milestone calls them, it doesn't add new ones). **Leaves deployable:**
yes — a registration operator can onboard many congregations in one run; nothing else about the
platform changes.

Builds `hermenea` per [modules/import.md](modules/import.md) (D-BulkImport): a Go CLI, `cmd/hermenea`
in this repo, published as `docker.io/olegamysk/hermenea`. For each row of an input file, it calls
`openfaithmap-api`'s existing `RegistrationService` — submit, then approve — using a real
registration operator's own forwarded token for the run. No new backend endpoint, no new schema, no
new credential type: this milestone's work is entirely the CLI itself plus deciding the open seams
`import.md` already names (input file schema, attribution for an imported row's
congregation-contact person, partial-failure reporting shape).

Exit criterion: a batch run against the live stack — N rows submitted and approved, D-Exclusions
still enforced per row (at least one excluded-tradition row in the batch is rejected, not silently
skipped), confirmed the same way M2's curl proof worked (real go-oikumenea writes landed, checked
directly).

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
