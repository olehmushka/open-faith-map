# Architecture decisions

These are OpenFaithMap's binding decisions. If code and a decision recorded here disagree, **the
code is wrong** — change the decision (with rationale) before diverging in code, exactly as
go-oikumenea's own `decisions.md` governs that project. Each decision is a `D-<Name>` block;
**decided-but-not-yet-built** is a normal state — see the [stage board](../milestones.md).

## Decision index

| ID | One-liner |
|---|---|
| [D-Scope](#d-scope--christian-only-discovery--presence-ukraine--usa-first) | Christian-only, discovery + presence, Ukraine + USA first |
| [D-Exclusions](#d-exclusions--a-named-permanent-denomination-exclusion-list) | Named, permanent denomination exclusion list (ROC, JW, LDS) |
| [D-CoreDependency](#d-coredependency--go-oikumenea-is-the-headless-core-consumed-via-its-docker-image) | go-oikumenea is the headless core, consumed via its docker image |
| [D-Facade](#d-facade--thin-on-identity-tenant-person-rbac-location-religion-taxonomy) | Thin on identity/tenant/person/RBAC/location/religion-taxonomy; owns content/discovery-glue/moderation/vouching |
| [D-ContentModel](#d-contentmodel--block-based-site-builder-owned-by-openfaithmap) | Block-based site builder, owned by OpenFaithMap, not go-oikumenea |
| [D-Moderation](#d-moderation--policy-engine--audit-trail-reuse) | Policy engine + reports/appeals, audit trail reused from go-oikumenea |
| [D-Vouching](#d-vouching--web-of-trust-guarantor-verification) | Web-of-trust guarantor verification for congregation-admin claims |
| [D-Stack](#d-stack--the-same-toolchain-as-go-oikumenea) | Same Go/gödel/Conjure/witchcraft/Atlas/Next.js toolchain as go-oikumenea |
| [D-AdminSurface](#d-adminsurface--the-admin-moderator-console-is-a-separate-deployment-from-the-public-site) | The verified/admin console is a separate Next.js app (`openfaithmap-admin`) from the anonymous public site (`openfaithmap-web`) |
| [D-InstanceAdminConsole](#d-instanceadminconsole--reuse-go-oikumeneas-own-console-as-the-third-super-admin-only-surface) | A third UI surface, `oikumenea-console`, is go-oikumenea's own published console reused unmodified — not built by OpenFaithMap |
| [D-BulkImport](#d-bulkimport--hermenea-replays-the-existing-registration-flow-in-bulk-no-new-write-path) | `hermenea`, a CLI, bulk-onboards congregations by replaying registration's existing submit/approve endpoints — no new write path, no new credential |

---

### D-Scope — Christian-only, discovery + presence, Ukraine + USA first

**Decision.** OpenFaithMap is a free, global platform for **Christian** church discovery and
online presence — a map (discovery) and a per-congregation site builder (presence). Eligibility is
**Nicene-affirming Christian traditions**, minus the named exclusions in
[D-Exclusions](#d-exclusions--a-named-permanent-denomination-exclusion-list). "Open" in the
project name means *open-source and free*, not multi-faith — this is a deliberate, narrow product
surface, not a placeholder for a broader one. Geographic rollout: Ukraine + USA first, Poland/UK
next, then the rest of EU/LATAM/Africa/Asia. Three audiences: visitors (anonymous), congregation
admins (verified, manage one or more congregations), platform moderators (seed/verify/moderate
platform-wide).

**Why.** A narrow, named perimeter is cheap to defend and expensive to re-litigate — every
contributor knows immediately what is and isn't in scope. go-oikumenea's own `religion` module is
deliberately multi-faith and catalog-driven at the **core** layer (no faith's vocabulary is
hard-coded there) specifically so that a narrower-scoped consuming app like OpenFaithMap can sit on
top of it without the core needing to know or care about OpenFaithMap's scope decision — the
narrowing happens entirely in this facade, never in the core.

**Why not** broaden to multi-faith: rejected for now. It's a real product pivot (different
moderation policy, different exclusion semantics, different audience), not a documentation change,
and nothing about the core forces it — go-oikumenea's religion module already supports it if a
later, deliberate decision reopens this one.

**Consequences.** The exclusion mechanism ([D-Exclusions](#d-exclusions--a-named-permanent-denomination-exclusion-list))
is enforced facade-side against go-oikumenea's religion taxonomy, not baked into the core. A future
multi-faith pivot would mean relaxing D-Scope and D-Exclusions here — it would not require any
go-oikumenea change.

---

### D-Exclusions — A named, permanent denomination exclusion list

**Decision.** The platform will never permit new congregation registrations under the following
bodies, regardless of Nicene affirmation:

- **Russian Orthodox Church (ROC)** — political exclusion, inherited unchanged from the original
  FaithMap product decision.
- **Jehovah's Witnesses** — doctrinal exclusion (non-Trinitarian; does not affirm the Nicene
  Creed).
- **The Church of Jesus Christ of Latter-day Saints (Mormons)** — doctrinal exclusion (non-Nicene
  Trinitarian theology).

This is a **named, explicit list**, not left implicit in the general Nicene-affirming test — even
though JW and LDS would also fail that test on doctrinal grounds, they are called out here so the
exclusion is discoverable and auditable on its own, independent of how the general eligibility
rule is phrased. The platform does not otherwise adjudicate doctrinal disputes: reports framed as
doctrinal complaints outside this named list are accepted with free text and moderators decline to
act on doctrinal grounds alone (see [moderation.md](../modules/moderation.md)).

**Mechanism.** Two layers, matching how go-oikumenea's `religion` module already models exclusion
for a single already-registered body (`religion_org_policies.excludes_child_creation` /
`excluded_body` — documented there as *"the generic analog of the dropped Christianity-specific
'Nicene gate'"*):

1. **Taxon-level gate (facade-side, primary).** When a prospective admin selects a
   denomination/tradition for a new congregation, OpenFaithMap checks the selected
   `religion_taxa` node (and its ancestors, via the taxonomy closure) against this named exclusion
   list **before** calling go-oikumenea's `POST /religion-orgs` or `POST /units/{id}/child-orgs`.
   This is where the actual product-level "no" happens — go-oikumenea itself has no concept of a
   taxon-level ban, only an org-level one (below).
2. **Org-level backstop (go-oikumenea-native).** If a top-level body for an excluded tradition is
   ever registered as an organization anyway (e.g. seeded for reference, or a future
   multi-tenant deployment shares the same go-oikumenea instance), its root `Unit` is marked
   `religion_org_policies` with policy kind `excludes_child_creation`, which makes go-oikumenea
   itself refuse `POST /units/{id}/child-orgs` beneath it — defense in depth, not the primary
   control.

**Why.** The taxon-level check is the *product* decision and belongs where OpenFaithMap's scope is
decided, not inside a general-purpose directory core that intentionally hard-codes no faith
vocabulary. Layering the go-oikumenea org-level policy on top costs nothing and closes the gap for
any body that does get registered.

**Consequences.** The exclusion list lives in this decision file, not a database row in
go-oikumenea — reopening it means editing this file with a new ADR-style rationale and quorum, the
same governance weight the original FaithMap ADR-0001 gave it.

---

### D-CoreDependency — go-oikumenea is the headless core, consumed via its docker image

**Decision.** OpenFaithMap depends on **go-oikumenea** as its identity/authorization/directory
core, run headless (no public port — D-HeadlessTopology) from its published docker image in this
project's own `docker-compose.yml`, alongside go-oikumenea's own Postgres. OpenFaithMap consumes
it exclusively through the generated Go/TypeScript SDK (D-ClientSDK) — never raw HTTP, never
direct database access to go-oikumenea's schema. Two identity paths, both already built in
go-oikumenea:

- **Interactive (a visitor or congregation admin using the site).** OpenFaithMap's Next.js facade
  owns the browser session (httpOnly cookie via Auth.js v5, Google as the sole OIDC provider — no
  Keycloak, no shared realm; see M1's as-built note and M1.1 item 3 in
  [milestones.md](../milestones.md)) and forwards the end user's Google ID token on every call to
  go-oikumenea — the console-bff pattern, reused verbatim (D-HeadlessTopology).
- **Background (moderation sweeps, vouching-graph checks, exclusion-list sync).** OpenFaithMap's
  backend registers as a **service principal** via OAuth2 client-credentials against the same IdP
  (D-ServiceIdentities), holding narrow, per-permission grants — never a role assignment, never
  unit reach.

**Why.** This is exactly the north-star topology go-oikumenea already ships and verifies (M51–M54)
— reusing it rather than inventing a bespoke integration means zero new trust surface on the core
side, and OpenFaithMap inherits go-oikumenea's own hardening (grant caching, closure-backed reach,
audit) for free.

**Why not** talk to go-oikumenea's Postgres directly: rejected outright — it would bypass the PDP,
violate go-oikumenea's own non-goal ("connectors never touch the core database"), and couple
OpenFaithMap to an internal schema that D-CoreDependency explicitly keeps replaceable.

**Consequences.** OpenFaithMap's own `docker-compose.yml` runs go-oikumenea's `migrate`/`init-role`
bootstrap sequence, then the `app` container unpublished on the internal network, matching
go-oikumenea's own `docker-compose.yml` shape. A go-oikumenea version bump is a dependency bump
(the SDK), not a schema migration OpenFaithMap has to write.

---

### D-Facade — Thin on identity/tenant/person/RBAC/location/religion-taxonomy

**Decision.** OpenFaithMap is **not** a zero-backend BFF and **not** a full independent backend
either. It is thin — owns no tables and makes no independent decisions — for everything
go-oikumenea's `tenant`, `person`, `personprofile`, `authorization`, `identity-federation`,
`location`, and `religion` modules already cover: organizational structure, people, roles,
addresses, and faith taxonomy. It is a genuine backend — its own Go modules, its own Postgres
schema, same toolchain as go-oikumenea — for the domains go-oikumenea has no equivalent for:
content/site-builder, discovery UX glue, moderation, and vouching.

**Why.** go-oikumenea's `religion` module doc says this outright: *"A FaithMap-style
discovery/CMS app sits on top and uses go-oikumenea as its identity/authorization/directory
backend — the CMS (pages/blocks/themes) stays in that app."* Duplicating tenant/person/RBAC modeling
here (as the original FaithMap design did, from scratch) would mean maintaining two authorization
systems that must stay consistent — a maintenance and security liability go-oikumenea's own
north-star was built specifically to avoid.

**Why not** a pure zero-backend facade (Next.js only, no Go service): rejected — content, reports,
and vouching edges are genuinely new, stateful domains with their own invariants (append-only
vouching edges, reversible moderation actions, translation groups) that belong behind a real
backend with migrations and tests, not client-side or crammed into the facade's session store.

**Consequences.** Every module doc under [modules/](../modules/) either (a) is a thin
API-shape-only doc pointing at the go-oikumenea module it delegates to
([core-integration.md](../modules/core-integration.md)), or (b) is a full module doc, same template
go-oikumenea uses, for a domain OpenFaithMap actually owns (content, moderation, vouching).

---

### D-ContentModel — Block-based site builder, owned by OpenFaithMap, not go-oikumenea

**Decision.** Site content (pages, posts, events) is modeled as ordered, typed **blocks** — never
HTML blobs — with per-locale translation groups and draft/published/unlisted states, in
OpenFaithMap's own schema. See [content.md](../modules/content.md) for the full entity model.

**Why.** Block-based content enables structured translation (one block-set, many locales),
targeted moderation (hide one block, not a whole page), and consistent rendering across a public
site and a preview pane — the same rationale the original FaithMap design used, unchanged here
because nothing about depending on go-oikumenea bears on how content is modeled.

**Consequences.** This is the one domain where OpenFaithMap's Atlas migrations, sqlc queries, and
Conjure contract are entirely its own — no go-oikumenea coupling at the schema level.

---

### D-Moderation — Policy engine + audit-trail reuse

**Decision.** Reports, moderation actions, and appeals are OpenFaithMap-owned tables (see
[moderation.md](../modules/moderation.md)), but every moderation action is **also** written through
go-oikumenea's `audit` module (a service-principal-authenticated write) so there is exactly one
append-only, permission-sensitive-action ledger across both services — not two logs to reconcile
during an incident.

**Why.** go-oikumenea's audit module already exists, is already the append-only ledger every other
consumer of go-oikumenea trusts, and duplicating it would fragment incident response across two
systems with no single source of truth.

**Consequences.** OpenFaithMap's backend needs an audit-write grant on its service principal from
day one of the moderation milestone (M5, see [milestones.md](../milestones.md)).

---

### D-Vouching — Web-of-trust guarantor verification

**Decision.** A lightweight web-of-trust mechanism — an immutable `vouching_edges` log plus a
mutable `guarantor_status` overlay — lets an already-verified congregation admin vouch that a new
admin genuinely represents their claimed congregation, reducing the platform's dependence on a
manual moderator check for every single claim. Fully OpenFaithMap-owned; go-oikumenea has no
equivalent concept (it has role assignments, which grant authority — vouching only raises *trust*,
never authority by itself).

**Why.** Kept from the original FaithMap design because it materially reduces moderator load at
scale (thousands of small volunteer-run congregations, few moderators) without granting any
authority a vouch shouldn't carry — a vouch is evidence for a moderator decision, never a
substitute for one.

**Consequences.** A revoked guarantor's outstanding vouches always route to moderator review
(never auto-revoke the vouched congregation's access) — see the invariant in
[vouching.md](../modules/vouching.md).

---

### D-Stack — The same toolchain as go-oikumenea

**Decision.** OpenFaithMap's own backend (content/moderation/vouching) uses the identical
toolchain go-oikumenea does: Go + **gödel** build with `godel-conjure-plugin`, **Conjure** IDL as
the API contract source of truth, **witchcraft-go-server**, **pgx + sqlc**, **Atlas** versioned
migrations, Docker + docker-compose packaging. The public site and congregation-admin console are
**Next.js** (App Router), matching go-oikumenea's own console (`web/`) — React 19, Auth.js v5 for
session ownership, Tailwind.

**Why.** One toolchain across both services means one set of conventions, one CI shape, and a
contributor who knows go-oikumenea's codebase can read OpenFaithMap's immediately. It also means
OpenFaithMap can consume go-oikumenea's generated TypeScript SDK the same way go-oikumenea's own
console does (`file:` dependency in dev, a real npm package once go-oikumenea publishes one).

**Consequences.** OpenFaithMap's own Conjure contract, Atlas migrations, and schema-naming
conventions follow go-oikumenea's `conventions.md` by reference (see
[conventions.md](conventions.md)) rather than restating them.

---

### D-AdminSurface — The admin/moderator console is a separate deployment from the public site

**Decision.** The congregation-admin console, the registration wizard (submitting a registration
already requires being logged in — [registration.md](../modules/registration.md)), the
operator-approval console, and the moderator console move to a **new, separate Next.js app,
`openfaithmap-admin`** — its own deployment, its own host/origin, the *only* place in OpenFaithMap
that ever holds a session or an Auth.js/Google credential. `openfaithmap-web` keeps the anonymous
public site only — discovery, congregation pages, public report filing, the public exclusion
pre-check — and holds no session at all. Both remain thin per
[D-Facade](#d-facade--thin-on-identity-tenant-person-rbac-location-religion-taxonomy): neither owns
identity/tenant/authorization data; only `openfaithmap-admin` ever forwards a user bearer token.

**Why.** [web-facade.md](../modules/web-facade.md)'s original one-app framing was explicitly
conditional on there being "no isolation benefit" to splitting, because both audiences supposedly
shared one identity provider and one session. That premise doesn't hold: the public site never
authenticates anyone at all — no session, no login, no tracking (Google Analytics is deferred, not
built) — so there was never a session to share in the first place. Splitting now is free: no
Auth.js/Google OAuth wiring, no session cookie, no credential of any kind ships to the surface every
anonymous visitor loads, and the one surface that *can* hold a credential is isolated at its own
origin rather than folded into the same deployment as the anonymous one.

**Why not** keep one app with route-based gating (e.g. `/admin/*` behind a middleware check):
rejected — it still ships admin/auth code to every anonymous visitor's bundle, and blurs the
"which surface can possibly hold a credential" boundary at the infrastructure level down to
application-level routing, which is weaker and easier to regress.

**Consequences.**

- `web/` becomes a small workspace: `web/apps/web` (public, renamed from today's `web/`) and
  `web/apps/admin` (new), with shared code (UI primitives, the go-oikumenea/`openfaithmap-api`
  client wrappers both apps need for reads) under `web/packages/*`. Recommend npm workspaces since
  the repo already uses plain npm — no new tool required. Exact package boundaries are decided when
  this is actually built, not in this decision record.
- `openfaithmap-admin` needs its own `AUTH_URL`/Google OAuth callback origin — the Google Cloud
  Console OAuth client's authorized redirect URIs need the new origin added (external to this repo)
  once the admin app has a real host.
- `docker-compose.yml` gains a new `openfaithmap-admin` service once the app exists in code (see
  [milestones.md](../milestones.md)'s M2.1) — not built yet. `openfaithmap-web` loses its
  `AUTH_SECRET`/`AUTH_GOOGLE_*`/`AUTH_URL` env vars at that point, since it no longer runs Auth.js.
- [web-facade.md](../modules/web-facade.md) narrows to the public surface only;
  [web-admin.md](../modules/web-admin.md) is the new module doc for `openfaithmap-admin`.
- **As implemented (M2.1):** no npm workspace, no `web/packages/*` shared-code directory — this
  decision's original npm-workspaces recommendation was reconsidered and explicitly not taken.
  `web/apps/web` and `web/apps/admin` are two fully independent apps, each with its own
  `package.json`/`package-lock.json`/`Dockerfile`; the ~10 lines of boilerplate config
  (`next.config.ts`, `eslint.config.mjs`, etc.) are duplicated rather than shared. Revisit if/when
  that duplication becomes a real maintenance cost.

---

### D-InstanceAdminConsole — Reuse go-oikumenea's own console as the third, super-admin-only surface

**Decision.** A third UI surface exists: **`oikumenea-console`**, go-oikumenea's own published
console image (`architecture/overview.md`'s comparison table already noted go-oikumenea ships an
"optional Next.js admin console, BFF over the public API" — this is that product, reused
unmodified). It is deployed alongside `oikumenea-app` exactly as `oikumenea-app` itself is
(D-CoreDependency: published image, no source in this repo) and is for **super admins only** —
whoever holds go-oikumenea's own instance-admin authority (see go-oikumenea's own `D-Bootstrap`).
It manages exactly what go-oikumenea already owns and nothing OpenFaithMap-specific: the
`religion_taxa` catalog, tenant/organization structure instance-wide, service-principal
registration, and other instance admins. OpenFaithMap builds none of this — same D-Facade reasoning
already applied to the backend (D-Facade), now applied to the third UI: don't build a bespoke admin
surface for concerns a product that already exists already covers.

This makes three total UI surfaces, each a strictly narrower blast radius than the last:

| Surface | Audience | Scope | Built by |
|---|---|---|---|
| `oikumenea-console` | Super admins (instance admins) | Instance-wide: taxonomy, tenants, service principals, other instance admins | go-oikumenea (reused, unmodified) |
| `openfaithmap-admin` | Congregation admins, registration operators, moderators | OpenFaithMap-domain: one or more congregations, registration approval, moderation queue | OpenFaithMap (D-AdminSurface) |
| `openfaithmap-web` | Anonymous visitors | Public read-only: map, search, congregation pages | OpenFaithMap (D-AdminSurface) |

**Why.** `oikumenea-console` already exists, is already maintained as part of go-oikumenea, and
covers concerns (taxonomy management, service-principal issuance, instance-admin bootstrapping)
that today have no human-facing UI at all in this project — only one-off scripts
(`scripts/bootstrap-service-principal`, `scripts/bootstrap-admin-person`,
`scripts/bootstrap-registration-org`). Reusing it closes that gap for free and keeps
OpenFaithMap's own two surfaces free of any instance-wide power, matching D-Facade's "thin on
identity/tenant" framing extended to the UI layer.

**Why not** fold instance-admin capability into `openfaithmap-admin`: rejected — `openfaithmap-admin`
is scoped to OpenFaithMap-domain authority (a congregation, the registration queue, the moderation
queue), never instance-wide go-oikumenea authority. Blurring that line would mean a congregation-admin
console occasionally also being an instance-admin console depending on who's logged in — the same
"which surface can possibly hold how much power" regression D-AdminSurface already rejected once for
the public/admin split.

**Why not** fold operator-approval or moderator functions into `oikumenea-console` instead (the
inverse question): rejected — `registration_requests`, moderation reports, and vouching are
OpenFaithMap-owned tables go-oikumenea's generic console has no knowledge of and never will
(D-Facade); those stay in `openfaithmap-admin`, unchanged from D-AdminSurface.

**Consequences.**
- `docker-compose.yml` gains an `oikumenea-console` service, pinned to
  `docker.io/olegamysk/oikumenea-console:0.0.1` (matching how `oikumenea-app` is pinned to an exact
  version rather than `latest`), reaching `oikumenea-app` the same way `openfaithmap-api` does
  (compose-internal network, self-signed cert, dev-only `NODE_TLS_REJECT_UNAUTHORIZED=0`). It reuses
  `openfaithmap-web`'s existing Google OAuth client rather than provisioning a new one or
  reintroducing Keycloak — `deploy/oikumenea-install.yml`'s Google issuer entry already trusts this
  audience, so no install-config change is needed there. The one external, out-of-repo action is
  adding this surface's own callback origin to that same Google Cloud OAuth client's authorized
  redirect URIs — the same treatment this decision already gives `openfaithmap-admin`'s callback
  origin.
- Its blast radius is strictly larger than either OpenFaithMap surface's (instance-wide, not
  congregation- or public-scoped), so unlike `openfaithmap-web`/`openfaithmap-admin` it must **not**
  get a bare public host port at any real deployment beyond local dev. **Decided:** a WireGuard VPN
  in front of this surface at any real (non-local-dev) deployment, not an IP allowlist or protected
  subdomain. **Not yet implemented:** there is no real deployment target in this project yet, so
  there is nothing to configure the VPN against — local dev continues to publish a normal host port
  (`docker-compose.yml`'s `oikumenea-console` service), the same exposure model
  `openfaithmap-api`/`openfaithmap-web` already use, until a real deployment target exists.
- Becomes the documented, human-facing replacement for what `scripts/bootstrap-service-principal`
  and `scripts/bootstrap-admin-person` do today; those scripts stay unchanged as the
  reproducible/CI bootstrapping path — a human operator no longer has to reach for `psql` or a
  one-off Go script for these actions.

---

### D-BulkImport — hermenea replays the existing registration flow in bulk, no new write path

**Decision.** **`hermenea`**, a small Go CLI (published as its own image,
`docker.io/olegamysk/hermenea`, same D-Stack toolchain, living in this repo as `cmd/hermenea`) lets
a registration operator bulk-onboard many congregations at once — e.g. importing an existing
directory of churches — instead of one submission at a time through `openfaithmap-admin`'s
registration wizard. It does this by **replaying registration's existing Conjure endpoints in a
loop** ([registration.md](../modules/registration.md)'s `POST /requests` then
`POST /requests/{id}/approve`), reading rows from a structured input file (schema decided when
M2.2 is actually built — see [milestones.md](../milestones.md)) and calling exactly the API
surface the wizard and the operator-approval console already call. It introduces **no new write
path and no new credential**: `hermenea` holds no token of its own — an operator hands it their
own real forwarded token for the duration of one run, the same "operator-owned" trust level
`scripts/bootstrap-registration-org` already assumes today ([registration.md](../modules/registration.md)'s
authority-bootstrapping finding).

**Why.** M2's registration flow was designed for one prospective admin submitting their own
congregation — reasonable for organic growth, unworkable for onboarding an existing list of
hundreds of churches (the actual scenario a legacy-directory migration or a partner-diocese bulk
signup needs). Reusing the existing submit/approve endpoints in a loop means every invariant
`registration.md` already guarantees (the D-Exclusions check per row, `unit`-scoped grants only,
the submitter's person RID always resolved from a real token, never client-supplied) holds for a
bulk run exactly as it does for a single manual one — bulk import is a new *client*, not a new
*mechanism*.

**Why not** give `hermenea` its own service-principal grant to write registrations directly (skip
the human-token requirement): rejected — this is exactly the on-behalf-of write
[core-integration.md](../modules/core-integration.md)'s invariants already forbid ("OpenFaithMap's
backend never presents its service-principal token to act as a specific person, even for
automation"). A bulk-import tool is not a special case of that rule; widening it here would create
a second, higher-privilege write path around the same review workflow that exists specifically to
keep a human decision (and the D-Exclusions check) in the loop.

**Why not** a new bulk-specific backend endpoint (e.g. `POST /requests/bulk`) instead of a CLI
replaying the existing ones: rejected for now — the existing per-row endpoints already do
everything a bulk run needs; a batch endpoint would just be the same logic behind a different
transport, with no invariant it adds. Revisit only if per-row HTTP round-trips prove too slow for
real import volumes.

**Consequences.**
- Depends on M2 (the `registration` module's submit/approve endpoints must exist) — sequenced as
  M2.2, not before.
- No new schema, no new backend module of its own — `hermenea` is a new *consumer* of
  `openfaithmap-api`'s existing surface, documented in
  [modules/import.md](../modules/import.md) the same way `web-facade`/`web-admin` are documented
  as schema-less consumers.
- Because it needs a real operator's token, `hermenea` cannot run unattended in the background —
  every run is initiated by a real registration operator, consistent with
  [core-integration.md](../modules/core-integration.md)'s no-on-behalf-of invariant.
