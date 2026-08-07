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
