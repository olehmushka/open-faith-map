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
| [D-Facade](#d-facade--thin-on-identitytenantpersonrbaclocationreligion-taxonomy) | Thin on identity/tenant/person/RBAC/location/religion-taxonomy; owns content/discovery-glue/moderation/vouching |
| [D-ContentModel](#d-contentmodel--block-based-site-builder-owned-by-openfaithmap-not-go-oikumenea) | Block-based site builder, owned by OpenFaithMap, not go-oikumenea |
| [D-Moderation](#d-moderation--policy-engine--audit-trail-reuse) | Policy engine + reports/appeals, audit trail reused from go-oikumenea |
| [D-Vouching](#d-vouching--web-of-trust-guarantor-verification) | Web-of-trust guarantor verification for congregation-admin claims |
| [D-Stack](#d-stack--the-same-toolchain-as-go-oikumenea) | Same Go/gödel/Conjure/witchcraft/Atlas/Next.js toolchain as go-oikumenea |
| [D-AdminSurface](#d-adminsurface--the-adminmoderator-console-is-a-separate-deployment-from-the-public-site) | The verified/admin console is a separate Next.js app (`openfaithmap-admin`) from the anonymous public site (`openfaithmap-web`) |
| [D-InstanceAdminConsole](#d-instanceadminconsole--reuse-go-oikumeneas-own-console-as-the-third-super-admin-only-surface) | A third UI surface, `oikumenea-console`, is go-oikumenea's own published console reused unmodified — not built by OpenFaithMap |
| [D-BulkImport](#d-bulkimport--hermenea-replays-the-existing-registration-flow-in-bulk-no-new-write-path) | OpenFaithMap deploys go-oikumenea's own `hermenea` companion service for reference-data seeding (countries, etc.) — no new code, no new write path; corrects the original CLI premise (see Correction) |
| [D-SharedDatabase](#d-shareddatabase--one-postgres-instance-two-schemas) | One Postgres instance, two schemas (`oikumenea` / `openfaithmap`) — not two database instances |
| [D-GoogleDirect](#d-googledirect--google-is-the-sole-identity-provider-no-keycloak) | Google is the sole IdP for every human and machine subject — no Keycloak, no shared realm |
| [D-OAuthClients](#d-oauthclients--one-google-oauth-client-today-one-per-surface-as-the-target) | One shared Google OAuth client across `openfaithmap-admin` and `oikumenea-console` today; per-surface clients are the target state |
| [D-FlatRoot](#d-flatroot--one-flat-root-organization-now-real-jurisdiction-units-before-m5) | Every congregation was a direct child of one flat root org through M4 — superseded by D-JurisdictionUnits at M4.1 |
| [D-JurisdictionUnits](#d-jurisdictionunits--denomination-aware-non-uniform-jurisdiction-layer-operator-assigned) | Jurisdiction is an ordinary, operator-assigned Unit — denomination-aware but not one canonical hierarchy per tradition, variable and optional depth |
| [D-PlatformModerator](#d-platformmoderator--moderator-authority-is-a-go-oikumenea-role-on-the-root-unit) | Platform-moderator authority is a go-oikumenea Role on the shared root unit, checked target-scoped — not an OpenFaithMap roster table |
| [D-Hardening](#d-hardening--in-process-rate-limiting-on-anonymous-writes-reused-witchcraft-observability) | In-process per-IP rate limiting on moderation's two anonymous write endpoints; observability reuses witchcraft's already-wired stack, no new infrastructure |
| [D-CongregationImport](#d-congregationimport--scraped-congregations-provision-as-real-admin-less-units-a-verifiedclaimed-overlay-tracks-their-status) | Scraped/imported congregations provision as real, admin-less go-oikumenea Units under the approving operator's own token; a verified/claimed overlay (proposal, not settled) tracks status. Resolves `DS-OFM-10` |

---

### D-Scope — Christian-only, discovery + presence, Ukraine + USA first

**Decision.** OpenFaithMap is a free, global platform for **Christian** church discovery and
online presence — a map (discovery) and a per-congregation site builder (presence). Eligibility is
**Nicene-affirming Christian traditions**, minus the named exclusions in
[D-Exclusions](#d-exclusions--a-named-permanent-denomination-exclusion-list). "Open" in the
project name means *open-source and free*, not multi-faith — this is a deliberate, narrow product
surface, not a placeholder for a broader one. Geographic rollout: Ukraine + USA first, Poland/UK
next, then the rest of EU/LATAM/Africa/Asia.

> **Update (2026-08-12): rollout priority reordered, direct from the product owner.** The active
> priority list is now **Ukraine, Argentina, Uruguay, Paraguay, Colombia, Chile, Brazil, USA** —
> six LATAM countries moved ahead of Poland/UK, given directly while scoping `congregationimport`
> (D-CongregationImport). The original "Poland/UK next" text above is left as written (an
> append-only correction, this doc set's own convention), not edited in place, since the
> underlying decision — narrow, named markets, expanded deliberately rather than pivoting to
> "everywhere" — is unchanged; only the sequence is.

**Five audiences** (the original three, plus two the build surfaced — see
[glossary.md](../glossary.md) for each one's definition):

| Audience | Authority | Surface |
|---|---|---|
| Visitor | None — anonymous | `openfaithmap-web` |
| Congregation admin | `unit`-scoped, over their own congregation only | `openfaithmap-admin` |
| Registration operator | `subtree`-scoped over the shared root unit; approves/rejects registrations ([registration.md](../modules/registration.md)) | `openfaithmap-admin` |
| Platform moderator | `subtree`-scoped over the shared root unit; reports/actions/appeals ([D-PlatformModerator](#d-platformmoderator--moderator-authority-is-a-go-oikumenea-role-on-the-root-unit)) | `openfaithmap-admin` |
| Super admin | Instance-wide go-oikumenea authority — taxonomy, tenants, service principals ([D-InstanceAdminConsole](#d-instanceadminconsole--reuse-go-oikumeneas-own-console-as-the-third-super-admin-only-surface)) | `oikumenea-console` |

Registration operator and super admin were absent from this decision's original three-audience
framing; both turned out to be real, separately-bootstrapped roles once M2 and M1.2 were built.

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
   **Real as of M4.1.** `scripts/bootstrap-exclusion-backstop` seeds a placeholder `Unit` beneath
   the shared root for each of the three named bodies and attaches `excludes_child_creation` —
   live-verified that a subsequent `createChildOrg` beneath one is rejected with
   `Religion:ChildCreationExcluded`. See
   [D-JurisdictionUnits](#d-jurisdictionunits--denomination-aware-non-uniform-jurisdiction-layer-operator-assigned).
   Before M4.1 this was designed-not-real (no per-body root units existed under
   [D-FlatRoot](#d-flatroot--one-flat-root-organization-now-real-jurisdiction-units-before-m5)), so
   the taxon-level gate was the *only* enforcement — now both layers are real.

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

**Two later decisions narrow this one — read them together with it:**

- [D-SharedDatabase](#d-shareddatabase--one-postgres-instance-two-schemas) — "alongside
  go-oikumenea's own Postgres" became one shared instance with two schemas. Note the open gap it
  records: `openfaithmap-api` currently connects as superuser, so the no-direct-database-access rule
  above is enforced only by convention until M2.4 lands a least-privilege role.
- [D-GoogleDirect](#d-googledirect--google-is-the-sole-identity-provider-no-keycloak) — the
  "shared Keycloak realm" premise this decision originally carried is gone; Google is the sole
  issuer for both identity paths below.

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

**Correction (found at M1.1, while proving M1's service-principal path against a real instance).**
The single-ledger mechanism below **does not exist and cannot be built today.** go-oikumenea's
`audit` module has no write endpoint — "there is no write endpoint; writes happen in-process"
(go-oikumenea's own `docs/modules/audit.md`), and its permission catalog defines only `audit.read`.
`scripts/bootstrap-service-principal` rejected the grant live with
`PrincipalGrantInvalid: unknown permission code`. There is no service-principal-authenticated path
by which OpenFaithMap can append anything to go-oikumenea's audit trail.

**Corrected decision.** Reports, moderation actions, and appeals are OpenFaithMap-owned tables (see
[moderation.md](../modules/moderation.md)), and **`openfaithmap.moderation_actions` is the ledger of
record** for OpenFaithMap's own moderation decisions: append-only, `reject_mutation()`-guarded, the
same discipline go-oikumenea applies to its own append-only tables. The "exactly one platform-wide
ledger" goal is **dropped as unachievable** — go-oikumenea's audit trail covers go-oikumenea's own
permission-sensitive actions (including every write an approval or a moderator's forwarded token
makes *through* go-oikumenea), and OpenFaithMap's covers OpenFaithMap's. Incident response reads
two logs, correlated by `person` RID and timestamp.

**Why accept two logs.** The alternative — blocking M5 until go-oikumenea grows an audit-ingest
endpoint — makes the moderation milestone's schedule depend on upstream work nobody has scoped, for
a property (one query instead of two during an incident) that is convenient rather than
load-bearing. Every moderation action that actually *changes* go-oikumenea state does so through a
real person's forwarded token and is therefore already in go-oikumenea's own trail; what
`moderation_actions` uniquely holds is the OpenFaithMap-side decision record, which go-oikumenea
would have no schema for anyway.

**Why not** an OpenFaithMap-side mirror written through some other go-oikumenea endpoint: rejected —
there isn't one. `POST /import/{objectType}` is reference-data-shaped and belongs to `hermenea`'s
credential (D-BulkImport); repurposing it for audit records would be an abuse of a contract built
for something else.

**Consequences.**
- M5 needs **no** audit-write grant on OpenFaithMap's service principal — that grant does not exist.
  `core-integration.md`'s authorization-touchpoints table already reflects this.
- `moderation.md`'s invariant "every action is mirrored into go-oikumenea's audit ledger before it
  is considered complete" is **withdrawn**; the replacement invariant is that
  `moderation_actions` is append-only and written before the action's effect is applied.
- **Revisit if upstream changes.** If go-oikumenea ever ships an audit-ingest endpoint, backfilling
  `moderation_actions` into it becomes a real option — tracked as `DS-OFM-12` in
  [open-questions.md](../open-questions.md), not a commitment.

**Impersonation — decided against (2026-08-11).** `glossary.md` once defined "Impersonation" (a
moderator logging in as a congregation admin for support debugging) with no `D-` block, no endpoint
in this module's API surface, and no milestone — an orphan term that also contradicted
[core-integration.md](../modules/core-integration.md)'s **no-on-behalf-of** invariant (OpenFaithMap
never presents a credential to act as a specific person). The 2026-08-09 audit flagged it as
"decide or delete" (`DS-OFM-15`, `U9`); decided: **delete**. A real version would require
go-oikumenea itself to mint the impersonated session — a feature it does not have — not
OpenFaithMap forging one; nothing in this codebase implements it, and there is no plan to build it.
The glossary entry is removed rather than left as a defined-but-unbuilt term.

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
console does (`file:` dependency in dev, a real npm package once go-oikumenea publishes one) —
confirmed as-built: `web/apps/admin` depends on the published `oikumenea-client` npm package.

**OpenFaithMap's own generated TypeScript SDK (M2.6) — resolved differently, and that's
deliberate.** `U4` asked how a generated package reaches two workspace-less apps whose Dockerfiles
each copy only their own directory. Mirroring go-oikumenea's shape (a separate `clients/typescript`
package, consumed via `file:`) doesn't fit here: there is no npm/pnpm workspace anywhere in this
repo, `web/apps/admin/Dockerfile`'s build context is deliberately isolated to its own directory (no
`../` `COPY`), and — unlike go-oikumenea, which serves its SDK to two separate consumers
(`web`, `oikumenea-console`) — OpenFaithMap's own registration client has exactly **one** consumer,
`web/apps/admin`, in the same repo. **Decided: generate directly into
`web/apps/admin/lib/openfaithmap/generated`** (`scripts/gen-ts-client.sh`, `make sdk`) — no
separate package, no `file:` dependency, no npm publish. This satisfies the Dockerfile's
single-directory build context for free (the generated code is just part of `web/apps/admin`'s own
tree), and avoids a real bug class go-oikumenea's own CI comment names outright: a separately-built
package's `dist/` going stale relative to its `src/generated` while local checks stay green,
caught only by the container rebuild — structurally impossible here since there is no separate
build step to go stale from. Should a second in-repo consumer of this SDK ever appear, revisit
this — go-oikumenea's own `file:`/package shape is the fallback, not ruled out permanently.

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
[D-Facade](#d-facade--thin-on-identitytenantpersonrbaclocationreligion-taxonomy): neither owns
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
- `docker-compose.yml` gains a new `openfaithmap-admin` service (**built at M2.1**, host port
  `3004`). `openfaithmap-web` lost its `AUTH_SECRET`/`AUTH_GOOGLE_*`/`AUTH_URL` env vars at the same
  time, since it no longer runs Auth.js.
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

**Correction (found while scoping M2.2 for real, before any code landed).** The decision and prose
below describe a CLI OpenFaithMap was going to build. That premise was wrong on two counts, found
in this order:

1. **The mechanism didn't actually work.** `registration`'s `SubmitRegistrationRequest` carries no
   contact-person field, and `submittedByPersonId` is *always* resolved server-side from the
   caller's own bearer token via go-oikumenea's `whoami` — never client-supplied
   ([registration.md](../modules/registration.md)). Since a bulk-import CLI can only ever hold the
   *operator's* token (this decision's own "why not on-behalf-of" already forbids anything else),
   every imported row would have been submitted, approved, **and granted `congregation-admin`** to
   the operator — not to each congregation's real contact. "Bulk-onboard congregations on behalf of
   their own admins" was never actually buildable on top of `registration`'s existing endpoints as
   they stand today.
2. **The name was already taken, by something that already does what M2.2 actually needs.**
   go-oikumenea ships its own service, also named `hermenea` (sibling repo, `cmd/hermenea` +
   `internal/hermenea/*`) — a persistent, already-built reference-data companion (countries,
   languages, external orgs, geo places) with its own database, its own migrations, its own
   `hermenea-importer` service-principal credential, coupled to go-oikumenea's core purely over
   HTTP (`POST /import/{objectType}`). It self-seeds nothing OpenFaithMap needs to write code for —
   OpenFaithMap's real job for M2.2 is **deploying** it (compose wiring + an install config), never
   building a CLI, and never touching `registration`'s submit/approve endpoints at all. Full detail:
   [modules/import.md](../modules/import.md).

The original congregation-bulk-import scenario this decision was chasing is still real — a future,
separately-named, not-yet-scoped tool near `openfaithmap-api` for scraped church data is expected to
address it eventually — but it is out of scope for M2.2 and has no design yet (see
[open-questions.md](../open-questions.md)'s `DS-OFM-10`). The Decision/Why/Why-not/Consequences
text below is kept as historical record of the rejected CLI design, not as current guidance.

**Decision (superseded — see Correction above).** **`hermenea`**, a small Go CLI (published as its own image,
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

**Consequences (as originally written — see Correction above for what's actually true).**
- Depends on M2 (the `registration` module's submit/approve endpoints must exist) — sequenced as
  M2.2, not before. **Superseded:** the corrected M2.2 has no dependency on `registration` at all.
- No new schema, no new backend module of its own — `hermenea` is a new *consumer* of
  `openfaithmap-api`'s existing surface, documented in
  [modules/import.md](../modules/import.md) the same way `web-facade`/`web-admin` are documented
  as schema-less consumers. **Still true, for a different reason:** the corrected M2.2 also adds no
  OpenFaithMap backend module, schema, or code of any kind — it's deploy wiring only, for a service
  OpenFaithMap doesn't own.
- Because it needs a real operator's token, `hermenea` cannot run unattended in the background —
  every run is initiated by a real registration operator, consistent with
  [core-integration.md](../modules/core-integration.md)'s no-on-behalf-of invariant. **Superseded:**
  go-oikumenea's actual `hermenea` holds its own service-principal credential and is designed to run
  unattended, on cron or triggered via its own `POST /sync/{source}` — the opposite of this
  constraint.

---

### D-SharedDatabase — One Postgres instance, two schemas

**Decision.** go-oikumenea and OpenFaithMap share **one** Postgres instance, separated by schema
(`oikumenea` and `openfaithmap`), rather than running two independent database instances as
[D-CoreDependency](#d-coredependency--go-oikumenea-is-the-headless-core-consumed-via-its-docker-image)'s
original "alongside go-oikumenea's own Postgres" phrasing implied. `hermenea` (D-BulkImport) adds a
third logical store — a separate **database** named `hermenea` on that same instance, with its own
Atlas migration set. Recorded here retroactively: this was decided in conversation while building
M1 and has governed `docker-compose.yml` ever since without a decision block of its own.

**Why.** One instance is materially simpler for local dev and for a small single-operator
deployment: one backup target, one connection string family, one health check, one thing to size.
Schema separation already gives the property that actually matters to the design — OpenFaithMap's
tables live under `openfaithmap.*`, never inside go-oikumenea's namespace — and
[conventions.md](conventions.md)'s no-cross-database-FK rule is unchanged by the collapse (it is now
literally a no-cross-*schema*-FK rule; OpenFaithMap still stores every go-oikumenea RID as an opaque
`TEXT` foreign value).

**Why not** two instances: rejected for now as premature operational cost. Revisit when either
service's load, backup window, or availability requirement stops fitting a shared instance —
splitting later is a connection-string change plus a data move, not a schema redesign, precisely
because nothing crosses the schema boundary.

**Consequences.**

- **Physical isolation was gone; only convention separated the two schemas — fixed by M2.4.**
  `openfaithmap-api` connected as the `postgres` superuser (`docker-compose.yml`'s `DATABASE_URL`),
  which meant it *could* read and write every table in the `oikumenea` schema directly. That
  contradicted D-CoreDependency's "never direct database access to go-oikumenea's schema," which
  that decision calls "rejected outright — it would bypass the PDP." Nothing in the code did this;
  nothing prevented it either. **Fixed:** `migrations/0003_least_privilege_role.sql` adds an
  `openfaithmap_app` role with `USAGE`/DML on the `openfaithmap` schema only and no grant of any
  kind on `oikumenea`; `docker-compose.yml`'s `openfaithmap-init-role` creates the login role that
  holds it, mirroring `oikumenea-init-role`'s own shape, and `openfaithmap-api`'s `DATABASE_URL` now
  connects as it. Verified live: `SELECT` against any `oikumenea.*` table from that role fails with
  `permission denied for schema oikumenea`, while the registration module's own reads/writes/updates
  (including the `set_updated_at` trigger) all still succeed.
- **No independent backup, restore, or failover.** A point-in-time restore of OpenFaithMap's data
  necessarily rolls back go-oikumenea's too, and vice versa. Acceptable at present scale; a real
  reason to revisit this decision once either side has data whose recovery objective differs.
- **Two Atlas revision tables in one database** (`--revisions-schema oikumenea` and
  `--revisions-schema openfaithmap`), both applied with `--allow-dirty` — permanently, not a
  narrowing target. M2.4 checked the original guess above (each migrate service sees a database the
  other has already written to) against a genuinely fresh volume with the flag removed from both:
  `oikumenea-migrate` fails before `openfaithmap-migrate` ever runs a single statement, because the
  `postgis/postgis` base image bootstraps its own `tiger` schema, which Atlas's dirty-check flags
  regardless of `--revisions-schema`. The flag stays; the reason was wrong.

---

### D-GoogleDirect — Google is the sole identity provider, no Keycloak

**Decision.** go-oikumenea is configured to trust **Google directly**
(`deploy/oikumenea-install.yml`) as the sole OIDC issuer for every subject in this deployment:
humans (via `openfaithmap-admin`'s and `oikumenea-console`'s Auth.js Google provider) and machines
(OpenFaithMap's service principal, which mints its own Google-signed ID token from a GCP
service-account key — `internal/coreintegration`). There is **no Keycloak**, no self-hosted IdP, and
no shared realm. Subjects are distinguished by `(issuer, subject)` and audience. Recorded here
retroactively: built at M1, corrected into D-CoreDependency's prose at M1.1, but never given a
decision block of its own despite being a load-bearing product constraint.

**Why.** Standing up and operating Keycloak is real infrastructure — its own database, its own
upgrade cadence, its own backup story, its own attack surface — for a project whose entire
architectural premise (D-Facade) is not building what someone else already runs. Google's issuer is
free, already trusted by go-oikumenea's issuer catalog, and covers both identity paths with one
configuration. Proven end-to-end at M1: a real browser OAuth round-trip resolves a real
`personId` through go-oikumenea's PDP.

**Why not** Keycloak (the original D-CoreDependency premise): rejected as unjustified operational
cost at this stage. It remains the natural answer if either consequence below becomes binding.

**Consequences.**

- **Every human user needs a Google account.** There is no email/password path, no Apple/Microsoft
  login, no institutional SSO. For a platform whose first rollout market is **Ukraine**, serving
  small, often volunteer-run congregations, this is a real adoption constraint and not merely a
  technical one — it is the most likely reason this decision gets reopened.
- **Single point of identity failure.** A Google outage, a suspended OAuth client, or a policy
  change locks every human and machine subject out simultaneously. There is no second issuer to
  fail over to.
- **Reopening means adding an issuer, not replacing one.** go-oikumenea's issuer catalog is a list;
  a second IdP (Keycloak, or another public provider) is an install-config addition plus an
  Auth.js provider entry — existing Google-linked accounts keep working. That is what makes
  deferring this cheap.
- Install config is read once at boot and is **not** hot-reloaded from the bind-mounted file —
  changing an issuer entry requires restarting `oikumenea-app` (found the hard way at M1).

---

### D-OAuthClients — One Google OAuth client today, one per surface as the target

**Decision.** `openfaithmap-admin` (host port `3004`) and `oikumenea-console` (host port `3003`)
currently share **one** Google OAuth client — the same `AUTH_GOOGLE_ID`/`AUTH_GOOGLE_SECRET`, with
two authorized redirect URIs and two independent Auth.js session secrets. This is accepted for local
dev and recorded as a known weakening of
[D-InstanceAdminConsole](#d-instanceadminconsole--reuse-go-oikumeneas-own-console-as-the-third-super-admin-only-surface)'s
blast-radius tiering. **Target state: one OAuth client per surface**, required before either surface
is deployed beyond local dev.

**Why the shared client today.** `deploy/oikumenea-install.yml`'s Google issuer entry already pinned
this audience, so reusing it meant M1.2 and M2.1 could each land without an install-config change or
a `oikumenea-app` restart — and without provisioning a second client in an external console the
repo can't automate.

**Why it's a weakening.** The three surfaces are documented as having strictly ascending blast
radius (public → congregation-scoped → instance-wide). Two of them now mint tokens with an
**identical `aud`**, which means go-oikumenea cannot tell from a token which surface issued it. The
PDP still gates instance-admin power by the caller's own role assignments, so this is **not** a
privilege escalation — a congregation admin's token does not become an instance-admin token by
sharing a client. What is lost is any token-layer enforcement of the tiering at all: the separation
is a property of who is logged in, not of which surface they logged into. A leaked client secret
also compromises the login flow of both surfaces at once.

**Why not** fix it now: the fix is entirely outside this repo (provision a second OAuth client in
Google Cloud Console, add its id/secret to `.env`, add the issuer audience to
`deploy/oikumenea-install.yml`, restart `oikumenea-app`). There is no real deployment target yet, so
it would be configuration with nothing to protect.

**Consequences.**
- Per-surface OAuth clients are a **prerequisite for any non-local-dev deployment**, listed
  alongside D-InstanceAdminConsole's WireGuard requirement. Both are deployment-gate items, not
  milestone-gate items — there is no deployment milestone yet to attach them to.
- `deploy/oikumenea-install.yml` gains a second `audiences` entry when that happens.
- The service principal is unaffected — it authenticates with its own GCP service-account key and
  its own target audience, never this OAuth client.

---

### D-FlatRoot — One flat root organization now, real jurisdiction units before M5

**Decision.** Every congregation registers as a **direct child** of one shared root
`Organization`/`Unit` (`scripts/bootstrap-registration-org`), not beneath a denomination's or a
diocese's own unit. There is no intermediate jurisdiction layer in the `canonical` graph today.
This is accepted as-built for M2–M4, and **must be replaced with real jurisdiction units before M5**
— sequenced as [milestones.md](../milestones.md)'s **M4.1**. Recorded here retroactively: the
flattening was a build-time simplification noted in
[registration.md](../modules/registration.md)'s open seams, never a decision block.

**Why flat, at M2.** go-oikumenea has no self-service org-creation path for an ungranted user
(registration.md's authority-bootstrapping finding), so *some* pre-existing unit had to own the
first assignment. One flat root needed exactly one out-of-band SQL seed; a per-denomination or
per-jurisdiction root would have needed one per branch, each with its own operator, before a single
congregation could register.

**Why it has to change before M5.** Three designs in this doc set assume a real ancestor chain and
silently do not work under a flat root:

1. [moderation.md](../modules/moderation.md)'s `queue_scope = 'jurisdiction'` is defined as
   "walking go-oikumenea's `canonical` religion graph" to a congregation's jurisdictional ancestors.
   Under a flat root every congregation's only ancestor is the shared root, so `jurisdiction` and
   `platform` collapse into the same scope — the enum value would be dead on arrival.
2. [glossary.md](../glossary.md)'s term mapping models Jurisdiction as "a `Unit` one or more levels
   up the `canonical` graph." No such unit exists.
3. [D-Exclusions](#d-exclusions--a-named-permanent-denomination-exclusion-list)'s **org-level
   backstop** (`religion_org_policies.excludes_child_creation` on an excluded body's root unit)
   has nothing to attach to — there are no per-body root units. Only the taxon-level gate is real
   today, so the documented defense-in-depth is currently one layer, not two.

**Why not** build jurisdictions now (at M2/M3): rejected — it would have blocked M2 on modeling a
denominational hierarchy for Ukraine and the USA before a single congregation existed to place in
it. Real registrations are the input that tells us which jurisdictions are actually needed.

**Why not** drop jurisdictions from the product instead: considered and rejected. Jurisdiction-scoped
moderation (a diocese's own staff triaging reports about their own parishes) is the mechanism that
keeps moderator load sublinear as the platform grows — the same load argument
[D-Vouching](#d-vouching--web-of-trust-guarantor-verification) makes. Removing it would push every
report to the small platform-wide roster.

**Consequences.**
- M4.1 must migrate **existing** congregations, not just accept new ones under jurisdictions:
  re-parenting a live `Unit` in the `canonical` graph, preserving each congregation admin's
  `unit`-scoped grant. Scoped in [milestones.md](../milestones.md).
- Until M4.1 lands, `moderation.md` keeps `jurisdiction` in its design but marks it blocked; M5
  cannot pass its `designed` gate while that dependency is open.
- D-Exclusions' org-level backstop stays documented as **designed-not-real** until per-body root
  units exist.

> **Superseded by M4.1.** [D-JurisdictionUnits](#d-jurisdictionunits--denomination-aware-non-uniform-jurisdiction-layer-operator-assigned) records what actually got built: real jurisdiction
> units exist, the org-level backstop is real, and every consequence listed above is closed. This
> block stays as the historical record of the M2–M4 simplification and why it had to change.

---

### D-JurisdictionUnits — Denomination-aware, non-uniform jurisdiction layer, operator-assigned

**Decision.** A jurisdiction is an ordinary go-oikumenea `Unit`, created via the same
`createChildOrg` every congregation already uses, tagged with the seeded `jurisdiction`/`diocese`/
`deanery`/… `orgKindId` family (purely descriptive — "never branched on in code," matching this
decision's own requirement below). A registration operator assigns a congregation's jurisdiction
(or re-parents an existing one) at **approval/re-parent time** — never inferred from `taxonId`, and
the public `/register` wizard is unchanged. Depth and shape are entirely operator-judgment: zero,
one, or several jurisdiction units may sit between root and a congregation, and multiple sibling
jurisdiction units may coexist under the same country with no assumption of exactly one canonical
jurisdiction per denomination.

**Why not a per-denomination canonical tree.** Explicitly rejected. Catholic polity has a clean,
near-universal diocese/national-conference hierarchy that *could* be modeled generically — but
Orthodox jurisdiction is often multiple and parallel even within one country and one broad
tradition (more than one patriarchate/synod claiming overlapping territory), and many Protestant
congregations are independent with no jurisdiction at all. A single "true" hierarchy encoded in
schema would be doctrinally false for exactly the traditions where it matters most. This is a
product decision, not an oversight — see this doc's own instruction (milestones.md's M4.1) that the
jurisdiction model "needs its own D-block... not just a schema."

**Exit criterion revised.** milestones.md's M4.1 originally read "a congregation's ancestor walk
returns at least one unit between it and the root" for *every* congregation — written before this
decision settled jurisdiction as genuinely optional. Revised to: **at least one unit when a
jurisdiction applies**. A congregation with no real denomination-specific jurisdiction remains a
direct child of root, exactly as under the old flat-root design, and moderation's `jurisdiction` and
`platform` queue scopes coincide for that specific congregation — by design, not as a gap.

**Mechanism — nothing new upstream, live-verified against a real instance:**
- **Creation:** `Religion.createChildOrg(parentUnitId, {code, name, orgKindId})` — already accepts
  any existing unit as parent; no upstream change needed.
- **Re-parenting an existing congregation:** go-oikumenea has **no dedicated reparent endpoint** for
  religion `Unit`s (only `reparentTaxon`, for the unrelated taxonomy tree). Composed instead from the
  generic `tenant` module's `Tenant.addEdge`/`removeEdge` on the `canonical` graph — two
  non-transactional calls, the same "sequential, not atomic" shape `createChildOrg`'s own multi-step
  approval flow already has (M2.3). **Add-before-remove, not remove-then-add**: `tenant_unit_edges`
  is `UNIQUE(graph_id, parent_id, child_id)`, not `UNIQUE(graph_id, child_id)` — a unit can hold two
  simultaneous parents in the same graph, live-confirmed. Since `subtree`-scoped reach depends on an
  incrementally-maintained ancestor closure, remove-then-add would open a real window (not just a
  crash-resume edge case) where every root-scoped grant (registration-operator, platform-moderator)
  loses reach to the congregation mid-move. Add-then-remove never has that window.
- **Idempotent resume**, mirroring M2.3's `PROVISIONING`/`created_unit_id` precedent: a
  `jurisdiction_reparenting_jobs` row tracks `PENDING → NEW_EDGE_ADDED → OLD_EDGE_REMOVED →
  VERIFIED`, persisting each durable fact before the next step runs. `AddEdge`'s duplicate-edge
  conflict surfaces as `Tenant:UnitInvalid` with reason "edge already exists" (not a dedicated
  conflict type — checked by substring, live-verified) and is treated as success on resume;
  `RemoveEdge` on an already-absent edge is a documented go-oikumenea no-op, needing no special
  handling at all.
- **`graph` must always be passed explicitly as `"canonical"`.** Live-confirmed: the tenant module's
  own default when `graph` is omitted is `"command"`, an unrelated graph — an omitted `graph` would
  silently operate on the wrong graph, not error.
- **Grant preservation is structural, live-verified, not per-job re-checked:** edge mutations never
  touch `authz_role_assignments`. A `unit`-scoped grant on the moved congregation and a
  `subtree`-scoped grant reaching it from root were both confirmed byte-identical/still-reaching
  before and after a real add-then-remove against a live instance.
- **The picker browses/creates jurisdiction units by calling go-oikumenea directly** from
  `openfaithmap-admin` (`Tenant.listUnits`/`unitAncestors`, `Religion.createChildOrg`) — no new
  `openfaithmap-api` endpoint for browsing. Only the two things coupled to OpenFaithMap's own
  resumable state (the operator's jurisdiction *choice*, and the re-parenting job) go through
  `RegistrationService`.

**Consequences.**
- `registration_requests.jurisdiction_unit_id` (nullable) records the operator's approval-time
  choice — a historical fact, not a live mirror; a later re-parent does not update it (the most
  recent `VERIFIED` `ReparentingJob.newParentUnitId` is the current source of truth for where a
  congregation actually is).
- `registration-operator`'s role gains `unit.read` and the **broad** `unit.edges.manage` — D-EdgePerms
  (go-oikumenea) only seeds dedicated `unit.edges.<graph>.manage` for `command`/`operational`, not
  `canonical`, so the broad fallback is the only option here, not a scoping shortcut.
- D-Exclusions' org-level backstop is now real: `scripts/bootstrap-exclusion-backstop` seeds a
  placeholder unit per excluded body and attaches `excludes_child_creation` — live-verified that a
  subsequent `createChildOrg` under one is rejected with `Religion:ChildCreationExcluded`.
- Moderation's `jurisdiction` queue scope is unblocked — `moderation.md`'s blocked-dependency row
  resolved.
- No local cache of "known jurisdiction units," unlike `discovery_site_cache`: this is a low-traffic,
  operator-only admin control, not a public-latency-sensitive path, so a second driftable mirror of
  go-oikumenea's own graph would cost more than it buys (D-Facade).

---

### D-PlatformModerator — Moderator authority is a go-oikumenea Role on the root unit

**Decision.** Platform-moderator authority is a real go-oikumenea **Role**
(`code = platform-moderator`), granted `subtree`-scoped on the shared root unit — the same
mechanism, the same bootstrap script, and the same trust level `registration-operator` already uses
(`scripts/bootstrap-registration-org`). OpenFaithMap stores **no** moderator roster of its own.
`moderation.read`/`moderation.act` remain OpenFaithMap-defined *names* for what the module gates on,
but each resolves to a live, **target-scoped** capability check against go-oikumenea's PDP, not to a
row in an OpenFaithMap table.

**Why.** [moderation.md](../modules/moderation.md) and [vouching.md](../modules/vouching.md) both
gate on these two codes and both described them only as "held by a small, fixed set of accounts" —
no table, no role, no mechanism. That is an undesigned primitive blocking two milestones. Putting it
in go-oikumenea keeps
[D-Facade](#d-facade--thin-on-identitytenantpersonrbaclocationreligion-taxonomy)'s governing
property intact — OpenFaithMap still makes zero authorization decisions of its own — and reuses a
bootstrap path that already works end-to-end.

**Why not** an OpenFaithMap-owned `platform_roles` table: rejected. It would be a second
authorization system that has to stay consistent with the first, which is the exact liability
D-Facade exists to avoid, and it would need its own management UI, its own audit story, and its own
answer to "who can edit the roster."

**Why not** reuse `registration-operator`: rejected — approving registrations and adjudicating
reports are different jobs with different escalation paths, and an appeal must be decidable by
someone who did not take the original action ([moderation.md](../modules/moderation.md)'s
invariant). Two roles keep that separable.

**Consequences.**

- **Capability checks must be target-scoped — `registration`'s `IsOperator` now is the reference
  implementation.** It used to ask `MyCapabilities()` for a bare permission string with **no target
  unit**, so it answered "does this caller hold `religionorg.manage` *anywhere*" — and
  `congregation-admin` holds `religionorg.manage` on its own unit, so every approved congregation
  admin read as an operator. [milestones.md](../milestones.md)'s **M2.3** fixed this: `IsOperator`
  now calls go-oikumenea's `Authorize` (`POST /authorize`), scoped to the shared root unit — the
  mechanism `MyCapabilities()` couldn't provide (it's deliberately flat/self-only), and one that
  itself required granting `registration-operator` an `assignment.read` reach it didn't have, since
  `Authorize` has no self-exemption. Moderation must not repeat the original bug: `moderation.read`/
  `moderation.act` resolve to a capability check **against the shared root unit specifically**, same
  mechanism.
- **`content.manage` follows the same pattern.** [content.md](../modules/content.md) originally
  defined it as "call `GET /units/{unitId}` with the caller's token and treat a successful read as
  proof of standing." Read authority is not write authority — that would let anyone who can *see* a
  congregation edit its site. Replaced with a target-scoped capability check on that congregation's
  own unit; see content.md's authorization touchpoints.
- `scripts/bootstrap-registration-org` gains the `platform-moderator` Role alongside the two it
  already creates. Its permission set is decided when M5 is scoped, not here.
- A moderator's authority is visible in go-oikumenea's own audit trail and manageable from
  `oikumenea-console` — a real benefit of not owning the roster, and partial compensation for
  D-Moderation's now-corrected two-ledger reality.

> **Addendum (M5 scoping, 2026-08-10): `platform-moderator`'s permission set.** go-oikumenea's
> permission catalog is closed and code-defined
> (`go-oikumenea/internal/authorization/domain/permissions.go` — a write of an unknown code is
> rejected server-side), so M5 cannot mint a new `moderation.*` permission; it must reuse an
> existing one as the PDP marker for "holds platform-moderator standing," the same way
> `registration`/`content`/`discovery` all reuse `religionorg.manage` for their own gates.
> `platform-moderator` is granted **`unit.lifecycle`** (not `religionorg.manage` again — that would
> make it indistinguishable from `registration-operator`/`congregation-admin` at the PDP, defeating
> this decision's own "why not reuse registration-operator" reasoning) plus **`assignment.read`**
> (root unit, `subtree` scope — the same self-reach requirement M2.3 already fixed for
> `registration-operator`, since `Authorize` has no self-exemption). `unit.lifecycle` is the closest
> existing semantic fit — moderation's own `suspend`/`archive` action kinds parallel a unit's
> lifecycle state — and is not already held by either other role, so `platform-moderator` stays
> genuinely distinguishable. `internal/moderation/application/authorize.go`'s `moderatePermission`
> is the implementation; `moderation.read` and `moderation.act` both resolve to this one check —
> there is no second go-oikumenea permission distinguishing read from act, matching how
> `content`/`discovery` already collapse their own read/manage distinctions to one PDP call.

---

### D-Hardening — In-process rate limiting on anonymous writes, reused witchcraft observability

**Decision.** Two mechanisms, both entirely OpenFaithMap-side:

1. **Rate limiting.** `openfaithmap-api`'s two genuinely anonymous *write* endpoints —
   `ModerationPublicService`'s `POST /reports` and `POST /exclusion-check`
   (`api/moderation.conjure.yml`) — get an in-process token-bucket limiter, keyed per
   `(client IP, endpoint)`, attached as `wrouter.RouteMiddleware` scoped to exactly
   `RegisterRoutesModerationPublicService`'s own route registration in
   `cmd/openfaithmap-api/main.go`'s `initServer` — not a router-wide wrapper, and nothing else
   registered on `info.Router` (registration, content, discovery, vouching, or
   `ModerationService`'s authenticated queue/action/appeal surface) is touched. On limit exceeded,
   the middleware returns a raw `429 Too Many Requests` with a `Retry-After` header **before the
   request ever reaches the generated Conjure handler** — see Consequences for why this can't be a
   Conjure-typed error the way every other error in this API is. `ContentPublicService`'s 4 public
   GETs and `DiscoveryPublicService`'s `GET /search` are **not** rate-limited in this pass (see
   [modules/hardening.md](../modules/hardening.md)'s scope boundary).
2. **Observability.** No new logging/metrics infrastructure. witchcraft's already-auto-wired
   `svc1log`/`req2log`/`trc1log`/`metric1log` stack (zero app configuration today,
   `witchcraft.NewServer()`) already gives every request structured logs and a trace ID for free.
   This decision adds a small, fixed set of app-defined counters via the same already-wired
   `palantir/pkg/metrics` registry (`metrics.FromContext(ctx).Counter(name).Inc(1)`, no new
   dependency): `openfaithmap.moderation.reports_filed`,
   `openfaithmap.moderation.exclusion_checks_run`, `openfaithmap.moderation.rate_limit_rejections`
   — nothing bigger.

**D-Facade check.** go-oikumenea's own `decisions.md` has no rate-limiting or client-throttling
decision — and more load-bearing than a bare absence, go-oikumenea is headless with no public port
at all (D-CoreDependency's D-HeadlessTopology sense). Every anonymous caller lands on
`openfaithmap-api`'s own public Conjure services; if those then call go-oikumenea, it's under the
server's own service-principal credential, never the anonymous caller's identity. go-oikumenea
therefore has no anonymous caller of its own to protect — this is unambiguously OpenFaithMap's own
concern, not something to check upstream for first. Observability's D-Facade answer is different in
kind: the underlying infrastructure (witchcraft, the same version D-Stack pins both projects to) is
already shared by construction; the only real work here is a handful of OpenFaithMap-specific
counters go-oikumenea has no visibility into.

**Why.** A coarse, cheap, in-process mechanism closes the actual gap (an anonymous, unauthenticated
`POST` with no cost to the caller) without new infrastructure, matching this repo's own
facade-first/MVP conventions (D-Stack, D-Facade's "err toward reuse" pattern already applied at
DS-OFM-2 and DS-OFM-4). `golang.org/x/time/rate`'s token bucket is already present in `go.sum`
(pulled in transitively; not currently a direct `require` in `go.mod`) — promoting it to a direct
dependency is a one-line `go.mod` change, not a new dependency risk.

**Why not a reverse proxy / API gateway (nginx, Envoy) doing the rate limiting instead.** Rejected.
No such component exists anywhere in `docker-compose.yml` today, and `openfaithmap-api`'s own app
port (3000) isn't even published to the host (M2.4, the same "no public port unless it needs one"
discipline D-HeadlessTopology names elsewhere) — adding a whole new infra tier for one narrow
protection need would be new operational surface this repo has deliberately avoided everywhere
else.

**Why not per-user/per-account limiting instead of per-IP.** Rejected outright for these two
endpoints specifically — both are anonymous by design (no token, no person RID; `fileReport`'s own
docs say "reporterPersonId is unset — this endpoint never asks for identity"), so there is no
"user" to key on. Per-IP is the only signal available. **Known, accepted limitation, not solved
here:** a shared IP (NAT, a church's shared office connection, CGNAT) can throttle innocent
co-tenants alongside a real abuser, and a distributed abuser defeats per-IP limiting entirely —
named rather than engineered around, the same way DS-OFM-4 accepted "no threshold" for vouching
until real abuse data justifies more.

**Why not CAPTCHA or another behavioral/heuristic anti-abuse mechanism.** Rejected as heavier than
a first cut needs — a coarse token bucket is the standard cheap baseline; CAPTCHA changes the
anonymous reporting flow's UX friction, a real product decision deferred until real abuse patterns
are observed, matching DS-OFM-4's own "tighten only if real abuse patterns appear" precedent.

**Why not Prometheus/OpenTelemetry/Grafana, or a `/metrics` scrape endpoint, now.** Rejected —
there is no scrape target or dashboard consumer anywhere in this repo today, so shipping an
exporter would be infrastructure with nothing reading it, the same speculative-building pattern
DS-OFM-2 already rejected for a discovery-cache refresh timer. witchcraft's built-in structured
logs already make "was a report filed, was a rate limit hit" answerable via the metrics log
stream — sufficient for this milestone's exit criterion.

**Why not a distinct structured audit log for rate-limit rejections**, separate from
`moderation_actions`. Rejected — D-Moderation's Correction already established
`openfaithmap.moderation_actions` as this platform's one ledger of record for moderation
*decisions*; a rate-limit rejection involves no moderator or report, so it doesn't belong there.
It's request-level telemetry, which the metrics counter above already covers.

**Mechanism.**
- **Algorithm:** `golang.org/x/time/rate`'s token bucket, promoted from an indirect
  (`go.sum`-only) dependency to a direct `require` in `go.mod`.
- **Key:** `(client IP, endpoint)` — two independent buckets per caller, so `POST /reports` traffic
  can't starve `POST /exclusion-check`'s budget or vice versa. Client IP is read directly from the
  connection (`r.RemoteAddr`), not a forwarded-for header — there is no reverse proxy in front of
  `openfaithmap-api` today, so this is the real client IP. **If a reverse proxy or CDN is ever
  added in front of this service, this must change to trust a specific forwarded-for header from a
  known hop, never blindly** — flagged here so it isn't silently wrong later.
- **Attachment point:** `wrouter.RouteMiddleware(rateLimitMiddleware)` passed as an extra
  `routerParams` argument to `genmoderation.RegisterRoutesModerationPublicService(info.Router,
  moderationPublicTransportSvc, wrouter.RouteMiddleware(rateLimitMiddleware))` — the generated
  registration function already accepts a variadic `...wrouter.RouteParam`
  (`internal/conjure/openfaithmap/moderation/servers.conjure.go`), and `wrouter.RouteMiddleware`
  is a real, already-vendored `RouteParam` constructor — no generated-code change needed.
- **On limit exceeded:** raw `429` + `Retry-After: <seconds>` header + a small JSON body
  (`{"errorCode":"RATE_LIMITED","message":"..."}`), written directly by the middleware before the
  request reaches the generated handler. **This departs from the Conjure-error-body convention**
  every other error in this API follows (`api/moderation.conjure.yml`'s `errors:` block — all
  `NOT_FOUND`/`PERMISSION_DENIED`/`INVALID_ARGUMENT`) — Conjure's error-code system has no code
  that maps to HTTP 429, so a genuinely Conjure-typed 429 isn't expressible in this stack today;
  the middleware intentionally sits outside that system rather than forcing a wrong status code
  through it.
- **Limits (provisional, not data-tuned):** e.g. `rate.NewLimiter(rate.Every(time.Minute/5), 5)`
  per key — roughly 5 requests/minute sustained, burst of 5. Explicitly a placeholder to retune
  once real traffic exists.
- **State:** an in-process map, no external store — acceptable because `openfaithmap-api` runs as
  a single replica; resets on restart; does not coordinate across replicas (Open Seam if that ever
  changes).

**Consequences.**
- `go.mod` gains `golang.org/x/time` as a direct `require`.
- `cmd/openfaithmap-api/main.go`'s `initServer` gains one new middleware value, wired only onto
  `RegisterRoutesModerationPublicService`'s call — every other `RegisterRoutes*` call is untouched.
- The 429 response from these two endpoints is documented in `hardening.md` and
  `api/moderation.conjure.yml`'s own comments as a deliberate, permanent exception to this repo's
  Conjure-error-body convention — not a gap to "eventually fix."
- `modules/moderation.md`'s Open Seam "Rate limiting on anonymous report filing is parked at M7"
  is resolved by this block; `DS-OFM-9` (open-questions.md) and `milestones.md`'s `U10` are both
  closed out.
- Read-side rate limiting (content/discovery public GETs) and multi-replica coordination remain
  open — recorded in `hardening.md`'s Open Seams, not silently dropped.

---

### D-CongregationImport — Scraped congregations provision as real, admin-less Units; a verified/claimed overlay tracks their status

**Decision.** A new module, `congregationimport` (resolving `DS-OFM-10`), ingests congregation data
from external sources (bulk open government data, OpenStreetMap, denomination-locator HTML) and
stages it for a registration operator's review. On approval, the operator's own forwarded token
performs the real go-oikumenea write (`createChildOrg` under the shared root or a chosen
jurisdiction unit, then a site over a new location) — the identical mechanism
`registration.Approve` already uses for `ensureUnit`/`ensureSite`, minus the position/fill/
`congregation-admin`-grant steps, because there is no submitter to grant authority to. **No
`congregation-admin` role is granted at provisioning time** — the congregation is real and
publicly discoverable, but genuinely admin-less until a future claim.

A new overlay table, `congregationimport_congregation_status` (same shape as
`vouching_guarantor_status`: a mutable projection keyed by an immutable go-oikumenea entity),
records `verified_by_person_rid`/`verified_at` (the approving operator) and
`claimed_by_person_rid`/`claimed_at` (nullable — reserved for `vouching.md`'s already-named,
not-yet-built congregation-claim flow). **This status model is a proposal, not settled** — the
product owner has explicitly said the exact state machine (a scraped congregation that was never
reviewed by a platform admin vs. one that was, vs. a real future claim) is still open for
discussion; this decision records the minimal schema needed to not block on it, not a final
design.

**Why this isn't an on-behalf-of write.** `core-integration.md`'s no-on-behalf-of invariant forbids
OpenFaithMap acting *as* a specific person without their authority. Nobody is impersonated here —
the write is honestly attributed to the real operator who reviewed the candidate and decided it's
a legitimate congregation, exactly as `registration.Approve`'s `createChildOrg` call already is.
This is different in kind from D-BulkImport's rejected CLI, which would have attributed every row
to the *operator as the congregation's admin* — the wrong person, holding authority they
shouldn't have. Here, no admin is granted to anyone, so there is no wrong-attribution problem to
have.

**Why not a service-principal write instead of the operator's token.** Rejected —
`createChildOrg`'s real gate is `religionorg.manage`/`assignment.grant` on the target unit,
permissions a human operator holds and a service principal structurally shouldn't (the same
reasoning `core-integration.md` already gives for every other real go-oikumenea write in this
project: the actual judgment call, "is this really a legitimate congregation," needs a human
decision point). The service principal's role in this pipeline stays confined to read-only work
(the D-Exclusions taxon walk, the dedup `SearchSites` check) — never a provisioning write.

**Why not replay `registration`'s submit/approve flow.** Already tried and rejected once
(D-BulkImport's Correction) — `submittedByPersonId` has no field a connector can fill, and there
is no real submitter for a scraped row. `congregationimport` is a **separate creation path**
converging on the same go-oikumenea Unit shape, not a client of `registration`'s endpoints.

**Consequences.**
- `registration.md`'s "every congregation has an admin from creation" invariant now has an
  explicit exception, scoped to this module's path only.
- `content`/`discovery` currently have no UI treatment of verified/unverified/claimed —
  deliberately out of this decision's scope, a follow-on question once the status model above is
  actually settled.
- D-Exclusions applies unchanged: the ancestor-walk check runs under the service principal before
  a candidate is ever presented for approval, and a match is an automatic `REJECTED_EXCLUDED`,
  never a silent drop or a later surprise.
- `DS-OFM-10` (`open-questions.md`) is resolved by this block.
