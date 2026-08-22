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
| [D-CatholicJurisdictionSync](#d-catholicjurisdictionsync--automated-jurisdiction-tier-unit-creation-from-wikidata-narrowly-scoped) | Automated jurisdiction-tier Unit creation from Wikidata, narrowly scoped (added to this index 2026-08-17 — the block existed since M8 but was never indexed) |
| [D-ProductionDeployment](#d-productiondeployment--single-cheap-vm-docker-compose-caddy-for-tls--provider-agnostic) | Single cheap VM, Docker Compose, Caddy for TLS — provider-agnostic; per-surface OAuth clients, WireGuard, backup, and the `ua-edr` re-run trigger all get scheduled here as M9's own build-phase work |
| [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed) | **Supersedes D-CoreDependency and D-Facade.** The core is absorbed into `openfaithmap-api` as in-repo modules; no oikumenea image, SDK, npm client, or sibling checkout remains |
| [D-CorePortScope](#d-coreportscope--port-the-hierarchy-drop-the-tenancy) | Units, graphs, edges and the closure table are ported; organizations, domains, `pdp_scoped`, audit, RLS, service principals and ~26 unused verticals are not |
| [D-InProcessAuthz](#d-inprocessauthz--the-pdp-runs-in-process-app-layer-only) | The PDP is in-process pure Go over pre-fetched grants; app-layer only, no Postgres RLS. Resolves `DS-OFM-1` as a deliberate no |
| [D-DirectTokenVerification](#d-directtokenverification--google-id-tokens-verified-in-process-no-service-principal) | `openfaithmap-api` verifies Google ID tokens itself with a pinned audience; the GCP service account and the whole service-principal concept are deleted. Supersedes D-ServiceIdentities |
| [D-OwnRIDs](#d-ownrids--uuidv8-resource-identifiers-owned-by-openfaithmap) | The UUIDv8 RID scheme is ported with its structural CHECKs; the registry tables are dropped; RIDs render as `ofm:<service>:<kind>:<type>:<uuid>` |
| [D-SeedBootstrap](#d-seedbootstrap--bootstrap-becomes-deterministic-seed-migrations) | The root unit, base roles, org kinds, site types and the exclusion backstop become seed migrations with fixed RIDs; three instance-specific env vars disappear |
| [D-SuperAdminFold](#d-superadminfold--super-admin-folds-into-openfaithmap-admin-behind-a-role) | **Supersedes D-InstanceAdminConsole.** The third surface is deleted; super-admin screens move into `openfaithmap-admin` behind a role |
| [D-StaticRefData](#d-staticrefdata--reference-data-is-a-static-seed-hermenea-is-removed) | **Supersedes D-BulkImport.** Countries and their locale names ship as seed migrations; the `hermenea` service, database and binary are deleted |

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

> **Update (2026-08-19): the seed data now actually matches this decision.** `migrations/`'s
> `religion_taxa` seed (ported at M10.1) had carried go-oikumenea's full multi-religion taxonomy —
> islam, judaism, hinduism, and 12 more root religions alongside christianity — since the port was a
> straight lift of upstream's curated reference data, not a scope-filtered one. This was a real,
> unnoticed drift from this decision's own "Nicene-affirming Christian traditions" eligibility line,
> caught and fixed in the same session as M10's migration-collapse pass (`docs/milestones.md`'s
> M10.9 row): `migrations/0011_core_religion.sql` now seeds only `christianity`'s subtree, and the
> two denominations that don't confess the Nicene Creed — LDS/Mormon and Jehovah's Witnesses — are
> hard-deleted from the taxonomy, not just excluded at the application layer
> ([D-Exclusions](#d-exclusions--a-named-permanent-denomination-exclusion-list) still covers
> Russian Orthodox Church, whose exclusion is political, not doctrinal — it stays a real taxon).

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

> **Update (2026-08-19): JW/LDS's mechanism changed from exclusion to non-existence.** Same session
> as [D-Scope](#d-scope--christian-only-discovery--presence-ukraine--usa-first)'s update above:
> `jehovahs_witnesses`/`lds_church` are now hard-deleted from `religion_taxa` entirely
> (`migrations/0011_core_religion.sql`), not merely gated by the two layers this decision describes
> — there is no taxon left to select (layer 1 moot) and their org-level backstop placeholder units
> are removed too (`migrations/0015_core_seed.sql`; layer 2 moot for these two specifically). The
> **outcome this decision requires — registration never permitted under either body — still holds,
> now by a stronger mechanism** (not selectable at all, rather than selectable-then-rejected).
> **Russian Orthodox Church is unaffected** — its exclusion is political, not doctrinal (Nicene/
> Trinitarian), so its taxon row and both mechanism layers stay exactly as documented above. If a
> future session ever needs to name a *new* non-Nicene body for this list, the taxon-deletion
> approach is not available (there is no upstream taxon to delete for a hypothetical future case) —
> that case would need the original two-layer mechanism, or a documented reason to delete a live
> taxon with real classified organizations under it, which today's JW/LDS deletion did not have to
> weigh (both were 0-row `religion_org_classifications` per the live check at deletion time).

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

> **Superseded (M10) by [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed).**
> The dependency this decision establishes is removed entirely: the core's capabilities move into
> `openfaithmap-api` as in-repo modules. The text above is left as written (append-only correction,
> this doc set's convention) because the *reasoning* still holds for the period it governed — the
> integration was correct, and reusing go-oikumenea's verified topology is exactly what made M1–M8
> cheap. What changed is the cost side, not the design side: see D-OwnCore's **Why**.

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

> **Superseded (M10) by [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed).**
> "Owns no tables and makes no independent decisions" no longer describes the system: identity,
> authorization, the unit hierarchy, the religion taxonomy, sites, locations and membership all
> become OpenFaithMap tables and OpenFaithMap decisions. One clause of this decision survives and
> is worth restating, because it is the reason the port stays small: the **domains** listed under
> (b) — content, discovery glue, moderation, vouching — were correctly identified as OpenFaithMap's
> own, and none of them change. What changes is that category (a) stops being a delegation and
> becomes an implementation.

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
- `docker-compose.yml` gains an `oikumenea-console` service, pinned to an exact version rather than
  `latest` (matching how `oikumenea-app` is pinned) — `0.0.1` originally, bumped in step with
  `oikumenea-app`/`hermenea` as new fixes land upstream (`0.0.7` as of 2026-08-16, GH-41's fix).
  Reaches `oikumenea-app` the same way `openfaithmap-api` does
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

> **Superseded (M10) by [D-SuperAdminFold](#d-superadminfold--super-admin-folds-into-openfaithmap-admin-behind-a-role).**
> The third surface is deleted along with the rest of the oikumenea artifacts. Note that this
> decision's central **Why not** — "blurring the line would mean a congregation-admin console
> occasionally also being an instance-admin console depending on who's logged in" — is a real
> objection that D-SuperAdminFold has to answer rather than ignore; it does, and the answer is that
> the line moved, not that it stopped mattering.

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

> **Superseded (M10) by [D-StaticRefData](#d-staticrefdata--reference-data-is-a-static-seed-hermenea-is-removed).**
> `hermenea` is deleted — service, binary, database and migration set. What it actually supplied to
> OpenFaithMap was the country list, and that list was *already* a static 249-row seed in
> go-oikumenea's own `0001_platform_core.sql`; hermenea only re-upserted it and added Who's-On-First
> geometry OpenFaithMap never queried.

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

> **Narrowed (M10) by [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed).**
> One instance, **one** schema. The `oikumenea` schema and the `hermenea` database are both dropped;
> everything lives under `openfaithmap.*`. This decision's core claim survives intact and is in fact
> vindicated — "one backup target, one connection string family, one health check, one thing to
> size" is now literally true, and the no-cross-schema-FK rule it defends becomes unnecessary rather
> than violated, since there is no longer a second schema to point at. The least-privilege
> `openfaithmap` role from `0003_least_privilege_role.sql` stays; its "no grants on the `oikumenea`
> schema" assertion becomes moot.

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

> **Amended (M10) by [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed).**
> Read "a go-oikumenea Role" as "a Role" throughout — the mechanism is unchanged, but the roles
> table is now OpenFaithMap's own. The decision's substance (moderator authority is a
> `subtree`-scoped role assignment on the shared root unit, checked target-scoped, **not** an
> OpenFaithMap roster table) is one of the things the port is specifically obliged to preserve, and
> is a named verification criterion in M10.9.

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

---

### D-CatholicJurisdictionSync — Automated jurisdiction-tier Unit creation from Wikidata, narrowly scoped

**Decision.** `congregationimport` gains a second kind of source, `domain.JurisdictionSource`
(distinct from `domain.Connector`, which is always congregation-level), whose job is to create/
resolve **jurisdiction-tier** go-oikumenea Units (`jurisdiction`/`diocese`/`deanery` org-kinds —
never a congregation) from a specific, high-confidence structured dataset: Wikidata, scoped to
entities carrying `wdt:P1866` (a Catholic-Hierarchy.org cross-reference ID), so the scope is
cleanly "actually Catholic" rather than the generic `wdt:P708` ("diocese") property Orthodox/
Anglican bodies also carry. Unlike every other jurisdiction-related write in this codebase, this
one runs **fully automatically and unattended, under the service principal's own token** — a
narrow, deliberate exception to both `D-JurisdictionUnits`'s "jurisdiction is operator-assigned,
never inferred" and `D-CongregationImport`'s "why not a service-principal write instead of the
operator's token — rejected" — requested explicitly by the project owner after being asked (this
session), on the reasoning that the source here is a curated, structured, cross-referenced dataset,
not a fuzzy free-text guess.

**What is and is not in scope for this exception — read carefully, it is narrow on purpose.**
- **In scope**: creating/resolving `congregationimport_jurisdiction_units` rows into real
  go-oikumenea Units, and auto-populating `congregationimport_jurisdiction_aliases` from their known
  name variants (global, `source_code = NULL`) — both automatic, both unattended.
- **Explicitly NOT in scope, unchanged by this decision**: how a specific *congregation* gets
  assigned to a diocese. `Candidate.SuggestedJurisdictionUnitID` is still populated by
  `matchJurisdiction`'s ordinary substring-alias lookup (now with far better aliases to match
  against, but the same advisory-only mechanism), and `ApproveCandidateRequest.jurisdictionUnitId`
  still requires the reviewing operator's own explicit choice — `D-JurisdictionUnits`'s core
  invariant holds exactly as before for every congregation this module has ever staged or will ever
  stage. This decision only automates building the TREE, never assigning a leaf onto it.

**Why Wikidata, and why this counts as "structured" rather than "scraped."** CC0-licensed, queried
via the public SPARQL API (not HTML scraping). Live-verified counts (2026-08-15): 6,655 Catholic
dioceses/eparchies worldwide (Latin **and** the Eastern Catholic churches in full communion with
Rome, e.g. Ukraine's UGCC — both carry `P1866`), with country (`wdt:P17`) and parent-diocese/
ecclesiastical-province (`wdt:P749`) links forming a real tree, not a flat list. 167,544 parish/
church entities are linked to those dioceses via `wdt:P708`; a future connector consuming them
(explicitly out of scope for this decision — see Open Seams) is a natural low-effort follow-on since
it would reuse the unmodified `Connector` interface.

**robots.txt findings, both resolved explicitly with the owner before any code was written — not
reasoned past silently, per this decision's own inherited discipline
(`D-CongregationImport`'s decision-#4 precedent, `docs/modules/congregationimport.md`).**
1. `catholic-hierarchy.org` (the site Wikidata's `P1866` cross-references) blocks known
   bulk-download tools by name in `robots.txt` (Wget, HTTPClient, HTTrack, WebCopier, Teleport,
   WebReaper) — the same signal class that halted the Brazil/CNPJ connector. **This site is never
   touched directly** — Wikidata supplies its diocese identity for free via the cross-reference.
2. `query.wikidata.org/robots.txt` disallows `/sparql` for every user agent. Judged — and the owner
   explicitly agreed — to be Wikimedia's standard pattern for keeping search-engine crawlers off the
   interactive query-builder HTML page, not a block on the documented public SPARQL API this source
   calls: live-verified response headers on the exact endpoint confirm API-oriented design
   (`access-control-allow-origin: *`, a dedicated `api-user-agent` header Wikimedia's own docs
   reserve for API clients, both of which this source's requests set).

**The real upstream blocker, found by reading go-oikumenea's own source directly, not inferred.**
`ReligionService.CreateChildOrg` (`internal/religion/transport/service.go`) gates with
`pep.Require`, one of the **person-shaped** PEP methods that — per that package's own doc comment —
denies a service-principal subject structurally, regardless of any grant, because a principal
carries no `PersonID`. This is the same defect class already found and fixed three times upstream
(GH-33 `religion.read`, GH-36 the RLS defect, GH-37 `country.read`) — a machine-reachable **write**
that is currently person-only. Filed as
[go-oikumenea#39](https://github.com/olehmushka/go-oikumenea/issues/39), mirroring GH-33/36/37's own
shape, asking for a `RequireServiceOrPerson`-gated path for `CreateChildOrg`, scoped by
`religionorg.manage`. **This module's code is written to be
correct on arrival of that fix, not blocked on it** — `RunJurisdictionSync` exists, compiles, and is
wired end-to-end today; calling it will fail with `PermissionDenied` until the upstream fix ships
and `scripts/bootstrap-service-principal` grants this service principal `religionorg.manage`.

**A second, related finding, recorded honestly rather than quietly worked around.**
`scripts/bootstrap-service-principal`'s `GrantPrincipalPermission` call has no unit/subtree-scoping
parameter today — every grant this project's service principal already holds (`religion.read`,
`country.read`, `connector.read`) is **instance-wide**. The `religionorg.manage` grant this
decision needs will be instance-wide too, not narrowed to a Catholic-jurisdiction subtree — a real,
named blast-radius trade-off, not silently smaller than it actually is. Mitigated in practice, not
in the permission model: `RunJurisdictionSync` only ever calls `createChildOrg` for
`JurisdictionSource`-supplied, topologically-ordered nodes under one pre-existing, human-created
anchor unit (see below) — never anything else, regardless of what the raw grant would technically
allow. Whether go-oikumenea should also grow subtree-scoped principal grants is left as an open
question for the upstream issue to raise, not decided unilaterally here.

**Mechanism.**
- One pre-existing anchor Unit (`CatholicJurisdictionAnchorUnitID`, an env-configured go-oikumenea
  Unit RID) is created **once, out-of-band, by a human operator** through the existing admin "Create
  unit" modal — the same one-time-bootstrap shape `scripts/bootstrap-registration-org`'s root
  already uses. Every top-level (no-parent) node this sync creates goes directly under that unit;
  nothing is ever created above or outside its own subtree.
- `congregationimport_jurisdiction_units` (`source_code`, `external_id`) is the idempotency anchor —
  a re-run recognizes an already-`CREATED` node by natural key and skips it, never calling
  `createChildOrg` twice for the same node. A `FAILED` row is retried on the next run (unlike
  `CREATED`, since the underlying cause — a transient outage, a since-fixed parent — may no longer
  apply).
- The whole node set (a few thousand at most — three orders of magnitude smaller than a
  congregation source) is fetched into memory before any write happens, unlike `RunConnector`'s
  deliberate never-buffer-a-whole-run discipline — a real, documented, scale-driven difference (see
  `RunJurisdictionSync`'s own doc comment), not an inconsistency.
- **Org-kind tiering is topology-derived, not a hand-maintained Wikidata-type→org-kind table**: any
  node another node in the same fetched set points to as its parent is re-tagged `jurisdiction`
  tier; every leaf stays the source's own default, `diocese` tier. Chosen because Wikidata models
  many overlapping ecclesiastical-circumscription types (diocese, archdiocese, eparchy, apostolic
  vicariate, ecclesiastical province, ...) with no single authoritative type→tier mapping to encode
  reliably — the tree's own shape is a more honest signal than guessing from an uncertain type
  vocabulary.

**Consequences.**
- `docs/modules/congregationimport.md` gains a new "Jurisdiction sync" section describing this
  pipeline alongside the existing connector one.
- `congregationimport_jurisdiction_aliases` rows created this way are attributed to a fixed sentinel
  (`system:wikidata-catholic-jurisdiction-sync`), not a real person RID — the first alias-creating
  write in this module with no human behind it; `CreatedByPersonRID`'s own field meaning is
  unchanged (still "who created this alias"), just occasionally answered by a machine identity now.
- Deferred, not built here: the parish-level Wikidata connector (167,544 candidates) that would
  consume this same dataset's leaf entities — a natural follow-on, reusing the unmodified
  `Connector` interface, explicitly out of this decision's build scope (the owner's own call).

> **Update (2026-08-15): go-oikumenea#39 fixed upstream** — merged as
> [PR #40](https://github.com/olehmushka/go-oikumenea/pull/40), released as image `0.0.6`
> (`docker-compose.yml`'s pin bumped from `0.0.5` to match), and
> `scripts/bootstrap-service-principal` now grants `religionorg.manage`. **Correction to this
> decision's own text above**, which anticipated a straight `RequireServiceOrPerson` swap (the
> mechanical fix GH-33/36/37 all used) — the actual fix is more precise, a new
> `pep.RequireServiceOrTarget` gate: a person subject keeps the exact same target-scoped
> `Require(action, unitID)` check as before (unchanged for `registration.Approve`/
> `congregationimport.ApproveCandidate`), and only a machine subject gets a new door, checked
> against its flat grant set. go-oikumenea's own PR description explains why the straight swap this
> decision assumed would have been a real regression: `RequireServiceOrPerson`'s person arm is
> `RequireAnywhere` ("holds the permission somewhere"), not the target-scoped check `CreateChildOrg`
> has always used for a person caller — it would have let any person holding `religionorg.manage` on
> an unrelated unit create a child org under any unit. The instance-wide-grant trade-off this
> decision already named is confirmed, not narrowed: go-oikumenea's own PR description states
> explicitly that principal grants still carry no unit/subtree scope, "an open question this issue
> deliberately leaves unresolved."
>
> **Update (2026-08-15, same day): live-verification attempted for real — found a SECOND, separate
> upstream blocker, not yet fixed.** Brought up a real `docker compose` stack (image `0.0.6`),
> created the anchor unit, granted `religionorg.manage`, and ran the sync scoped to Ukraine
> (`CATHOLIC_JURISDICTION_COUNTRY_QIDS=Q212`) against the real Wikidata API: **40 real Ukrainian
> Catholic dioceses/eparchies fetched** (Latin and Greek-Catholic/UGCC both present, confirmed by
> name), 38 staged as `PENDING` (the other 2 correctly left unattempted — their `P749` parent lies
> outside this country-scoped node set, exactly the documented "left for a later run" behavior, not a
> bug). **All 38 `createChildOrg` calls failed with a real `500`**, root-caused in `oikumenea-app`'s
> own logs, not guessed: `ERROR: new row violates row-level security policy for table "tenant_units"
> (SQLSTATE 42501)`. Read directly against `migrations/0005_document_order_rls.sql`:
> `authz_unit_in_reach` (the predicate every `tenant_units` RLS policy calls) is keyed **entirely** on
> `current_setting('app.person_id')` and `authz_role_assignments.subject_person_id` — no branch
> anywhere in it consults a machine/service-principal subject at all. GH-39/PR #40 fixed the
> **authorization** layer (`pep.RequireServiceOrTarget` now lets a machine subject reach the
> endpoint); this is a separate, deeper **RLS** layer gap the PEP fix could not have touched — the
> same general class GH-36 already fixed once for `tenant_units`, but that fix was scoped to a
> person-caller timing issue (closure not yet seeded before insert), not to machine callers at all.
> One real thing that DID work as designed: the 38 failures were caught and correctly recorded
> (`MarkJurisdictionUnitFailed`, `unitsFailed: 38` in the run summary) rather than crashing the whole
> sync — the resumability design held even though the underlying write never succeeded.
>
> **Update (2026-08-15, same day, continued): the `tenant_units`-level failure above was NOT a new
> upstream bug — it was this decision's own grant misconfigured. Correcting a real research gap in
> this decision's own earlier text** (and in go-oikumenea#39 itself): `GrantPrincipalPermissionRequest`
> has always had an `orgId` field ("omit for an instance-wide grant; set to confine the machine to
> one organization") — the earlier "principal grants carry no unit/subtree scope" claim only checked
> `scripts/bootstrap-service-principal`'s own instance-wide-only call site, not the full Conjure
> contract. go-oikumenea's own `migrations/0011_infra.sql` (`authz_principal_org_in_reach`) requires
> an ORG-scoped grant for a machine subject's `tenant_units` write — an instance-wide one "confers NO
> operational reach" by that migration's own comment, exactly matching what was observed. **Fixed on
> this side**: `scripts/bootstrap-service-principal` gained a `-catholic-jurisdiction-org-id` flag;
> `religionorg.manage` is now granted ORG-scoped to the org owning the anchor unit (confirmed live:
> the anchor unit and the shared root unit are both in org `019fe8bb-3b41-8101-8406-06b65f756132` in
> this environment), and skipped entirely when the flag is omitted — a misleading unusable
> instance-wide grant is worse than none. Re-running the sync with this grant in place, the
> `tenant_units` insert genuinely succeeds now (confirmed: 38 real unit rows created for real
> Ukrainian dioceses/eparchies).
>
> **A second, DEEPER, and still genuinely unresolved bug surfaced once that layer was fixed**: the
> SAME `createChildOrg` call's follow-on `tenant_unit_edges` insert (attaching the new unit under the
> anchor in the `canonical` graph — same transaction, same request) still fails with the identical RLS
> error. Root-caused as far as possible from outside go-oikumenea's own request-handling code: a
> **manual raw-SQL reproduction of the exact same insert, with the RLS session GUCs (`app.principal_id`
> etc.) set to match the real request, succeeds** — proving the policy/grant model itself is correct
> and the gap is in how go-oikumenea's Go server propagates those GUCs to the specific connection/
> transaction `tenant.Service.CreateUnitWithEdge`'s internal `InsertEdge` call runs on, not in the
> RLS design. Filed as a new go-oikumenea issue (see `docs/milestones.md`'s M8 detail for the link) —
> this one genuinely needs the repo owner's own server-side instrumentation to pin down, since it
> could not be isolated further by black-box HTTP/SQL probing alone
> ([go-oikumenea#41](https://github.com/olehmushka/go-oikumenea/issues/41)). `RunJurisdictionSync` is
> **still not live-verified end to end** as a result — real progress (one full layer fixed, live data
> flowing, resumability proven under real failure), but the actual go-oikumenea Unit tree for Ukraine
> does not exist yet.
>
> **2026-08-16: GH-41 fixed upstream and live-verified end to end.** Diagnosed live, not by further
> black-box probing — the repo owner's own server-side instrumentation pinned it down to
> `InsertEdge`'s sqlc query using `RETURNING`: a row whose `WITH CHECK` (write) passes but whose
> table `USING` (read) does not raises the identical RLS error for `RETURNING`, not a silent empty
> result, in this Postgres version. So it was never a GUC-propagation bug (this decision's own
> working theory above) or a policy gap — an org-scoped `religionorg.manage` grant alone satisfies
> the write side but not the implicit read side. Merged to go-oikumenea `main`, published as image
> `0.0.7` (bumped in this repo's `docker-compose.yml`, was `0.0.6`).
>
> Re-verifying against 0.0.7 on this side surfaced a SECOND, real gap: `bootstrap-service-principal`
> already granted `religion.read`, but instance-wide, and a re-run still hit the identical
> `tenant_unit_edges` RLS error — an instance-wide `religion.read` grant does not satisfy
> `InsertEdge`'s read-reach check any more than an instance-wide `religionorg.manage` grant satisfied
> the write-reach check earlier in this same trace. go-oikumenea's own GH-41 regression test grants
> `religion.read` ORG-SCOPED, confirming the fix: `bootstrap-service-principal` now grants a second,
> org-scoped `religion.read` alongside `religionorg.manage`. With both actually in place, a real
> `wikidata-catholic` sync against a live docker-compose stack on image `0.0.7` succeeded completely
> (`nodesFetched: 40, unitsCreated: 38, unitsFailed: 0, aliasesCreated: 486`), confirmed directly
> against go-oikumenea's tables — all 38 units have real `tenant_unit_edges` and
> `tenant_unit_closure` rows, genuinely reachable from the anchor, not orphans — and a second run
> confirmed idempotency (`unitsSkipped: 38`). `RunJurisdictionSync` is no longer blocked.

---

### D-ProductionDeployment — Single cheap VM, Docker Compose, Caddy for TLS — provider-agnostic

**Decision.** OpenFaithMap's first real (non-local-dev) deployment target is a single Linux VM,
sized around M8's own stated budget (~500MB–1GB RAM), running the existing `docker-compose.yml`
stack unmodified as its base. This decision is **provider-agnostic on purpose** — the concrete VM
provider is explicitly undecided and deferred; nothing below depends on which one gets picked.
Scoped as [milestones.md](../milestones.md)'s **M9**, a design-only milestone (no VM is provisioned,
no compose override is written yet — that is M9's own inherited build-phase work, done once a
provider exists).

Each sub-decision below is either new here or an existing decision formally given somewhere to
attach as scheduled work, not re-litigated:

- **Reverse proxy / TLS — new.** No reverse proxy exists anywhere in the stack today
  ([D-Hardening](#d-hardening--in-process-rate-limiting-on-anonymous-writes-reused-witchcraft-observability)
  deliberately rejected one, but for a rate-limiting reason that doesn't apply here — a real
  deployment still needs TLS termination for whatever is public). **Caddy**, in front of
  `openfaithmap-web` and `openfaithmap-admin` only — automatic Let's Encrypt, a single static
  binary, the lowest ops burden available for a one-VM box. `oikumenea-console` gets no
  reverse-proxy entry and no public port of any kind.
- **Per-surface OAuth clients — inherited from
  [D-OAuthClients](#d-oauthclients--one-google-oauth-client-today-one-per-surface-as-the-target).**
  That decision's target state (one Google OAuth client per surface) was already made; it had
  nowhere to attach as real work because no deployment milestone existed. It is now M9's own
  build-phase item, not redesigned here.
- **WireGuard for `oikumenea-console` — inherited from
  [D-InstanceAdminConsole](#d-instanceadminconsole--reuse-go-oikumeneas-own-console-as-the-third-super-admin-only-surface).**
  Same treatment: that decision already ruled out a bare public port or an IP allowlist for this
  surface. Scheduled here as build-phase work, not reopened.
- **Secrets handling — new.** A root-only `.env` file on the VM (`chmod 600`), never committed,
  with every insecure local-dev default (`HERMENEA_OIKUMENEA_TOKEN`/`OIKUMENEA_HERMENEA_TOKEN`,
  both Auth.js `AUTH_SECRET`s, the HS256 bootstrap-issuer key, `crypto.local-dev.kek`) rotated to
  real values before first boot. No secrets manager for v1 — unjustified cost/complexity at this
  scale; revisit only if a real operational need for rotation-at-scale or audit shows up.
- **Backup — new.** No backup mechanism exists today. A `pg_dump` on a systemd timer, to some
  off-VM target (object storage or a second cheap host — left open, deferred with the provider
  choice). Still governed by
  [D-SharedDatabase](#d-shareddatabase--one-postgres-instance-two-schemas)'s existing caveat: one
  backup target, no independent RPO/RTO between go-oikumenea's and OpenFaithMap's own schemas —
  this decision does not reopen that trade-off, only gives the VM a backup where it currently has
  none at all.
- **Process supervision — new.** No service in `docker-compose.yml` carries a `restart:` policy
  today; a crash simply stays down. Add `restart: unless-stopped` to every long-running service,
  plus a systemd unit wrapping `docker compose up -d` as the boot-time entry point, so the stack
  survives a VM reboot without a login.
- **`ua-edr` periodic re-run trigger — new, resolves an item M8 left explicitly open.** A systemd
  timer (weekly, mirroring `hermenea`'s own `cron: "@weekly"` precedent for its reference-data
  sources —
  [D-BulkImport](#d-bulkimport--hermenea-replays-the-existing-registration-flow-in-bulk-no-new-write-path))
  calling `POST /runs {"sourceCode":"ua-edr"}` under a real, non-bootstrap operator identity. Not a
  new in-process scheduler — consistent with
  [D-CongregationImport](#d-congregationimport--scraped-congregations-provision-as-real-admin-less-units-a-verifiedclaimed-overlay-tracks-their-status)'s
  original "manual operator-triggered runs only, no new scheduler" call: this is an
  operator-authorized timer hitting the module's existing API, not a scheduling subsystem built
  into the module itself.

**Why provider-agnostic.** The owner explicitly deferred the concrete VM provider choice — asking
this decision to also pick one would either block on an unrelated question or bake in a choice
that has to be redone later for no reason. Every sub-decision above (Caddy, systemd timers,
`.env` secrets, `pg_dump`) works identically regardless of which Linux VM it runs on.

**Why not decide the OAuth-client/WireGuard mechanics here.** Both were already decided in
principle by their own original entries; what was missing was a milestone for the work to attach
to (`open-questions.md`'s `DS-OFM-14`). Re-deciding them here would duplicate, not extend, those
entries.

**Consequences.**
- `open-questions.md`'s `DS-OFM-14` and `milestones.md`'s `U13` are resolved by this block — both
  said the reason per-surface OAuth clients and WireGuard had no owner was that no deployment
  milestone existed; M9 is that milestone.
- A follow-up build milestone (numbering TBD — likely `M9.1` once a provider is picked) does the
  actual work this decision schedules: write `docker-compose.prod.yml` and a Caddyfile, provision
  the two OAuth clients, stand up WireGuard, add the `restart:` policies, and wire the two systemd
  timers (backup, `ua-edr` re-run). None of that is done by this decision itself.
- `docker-compose.yml` itself is not rewritten by this decision — the production topology is an
  override/addition on top of the existing file, not a replacement.

---

### D-OwnCore — OpenFaithMap owns its core; go-oikumenea is removed

**Supersedes [D-CoreDependency](#d-coredependency--go-oikumenea-is-the-headless-core-consumed-via-its-docker-image)
and [D-Facade](#d-facade--thin-on-identitytenantpersonrbaclocationreligion-taxonomy).**

**Decision.** OpenFaithMap absorbs the capabilities it used from go-oikumenea — identity,
authorization, the unit hierarchy, the religion taxonomy, sites, locations, membership and country
reference data — into `openfaithmap-api` as ordinary in-repo modules under `internal/`, following
the same `transport → application → domain → adapters` convention every existing module uses. After
this decision the repository contains **no** go-oikumenea Go SDK, **no** `oikumenea-client` npm
package, **no** `docker.io/olegamysk/*` image, and **no** `OIKUMENEA_SRC` sibling checkout.

The system becomes **one binary against one Postgres schema**. Calls that were HTTPS round-trips to
`oikumenea-app:8443` become in-process Go function calls.

**Why.** Three costs, all of which grew rather than shrank across M1–M8:

1. **Authorization was a network dependency.** Every authenticated request made two round-trips —
   one `Whoami`, one `Authorize` — to a service that had to be up for OpenFaithMap to serve any
   authenticated page at all. There is no fallback and no degraded mode; an oikumenea outage is a
   total outage.
2. **Delivery velocity was gated on a second repository.** Six upstream issues (#33, #34, #36, #37,
   #39, #41) were found *by this project* and had to be fixed upstream, released as a new image, and
   pulled back in before the milestone that found them could close. #41 is still open. That is a
   multi-repo change for what is, from the product's point of view, a one-line bug.
3. **The artifact channels drifted.** Go SDK `v0.1.0`, npm SDK `0.0.1`, Docker images `0.0.7` —
   three independently-versioned channels for one dependency, with no automated compatibility gate
   between them.

The port is affordable because the *useful* surface is small. go-oikumenea is ~267k LOC, but that is
overwhelmingly generated Conjure transport, per-package duplicated sqlc `models.go` files, and ~26
verticals OpenFaithMap never touches (audit, company, education, finance, order, rank, vehicle,
document, …). The parts that carry the behaviour are tiny: the entire policy decision point is
**260 lines of pure Go**. Total port is ~7–8k LOC of Go plus ~1.5k LOC of migrations — and most of
that is only necessary because it becomes *our* code, not because it is hard.

**Why not** keep the facade and fix the pain upstream: rejected, because the pain is structural. The
network hop, the two-repo release cycle, and the three artifact channels are all consequences of the
split itself, not of any particular upstream bug. Fixing #41 does not make the seventh issue cheaper.

**Why not** a second in-repo service (`openfaithmap-core` + `openfaithmap-api`): rejected. It would
preserve the boundary — and therefore the HTTP hop, two Conjure contract sets, and two deployments —
while giving up the one thing the boundary bought, which was not having to maintain the core. A
single VM running a single binary is also exactly what
[D-ProductionDeployment](#d-productiondeployment--single-cheap-vm-docker-compose-caddy-for-tls--provider-agnostic)
already assumes.

**Why not** fork go-oikumenea wholesale: rejected. It would import the tenancy model, the 26 unused
verticals, the audit partitioning, the RLS policy set, and ~28k LOC of generated transport for
services nothing calls — most of the maintenance burden, with none of the simplification.

**Consequences.**
- [D-Stack](#d-stack--the-same-toolchain-as-go-oikumenea) is **unaffected**. Conjure, witchcraft,
  gödel and Atlas all stay. Crucially, Conjure transport is generated only for the ~12 endpoints the
  admin app actually calls (`api/core.conjure.yml`); everything else is in-process Go with no
  transport layer at all. That is the single biggest reason the port is small rather than enormous.
- `internal/coreintegration/` is deleted in full, along with the empty `clients/go/` placeholder.
- The `oikumenea` schema and the `hermenea` database are dropped;
  [D-SharedDatabase](#d-shareddatabase--one-postgres-instance-two-schemas) collapses to one schema.
- Nine environment variables disappear (see [D-SeedBootstrap](#d-seedbootstrap--bootstrap-becomes-deterministic-seed-migrations)
  and [D-DirectTokenVerification](#d-directtokenverification--google-id-tokens-verified-in-process-no-service-principal));
  two are added.
- `NODE_TLS_REJECT_UNAUTHORIZED=0` and `OIKUMENEA_INSECURE_SKIP_VERIFY=true` are removed from four
  compose services. They existed only to tolerate `oikumenea-app`'s self-signed certificate; with no
  internal TLS hop there is nothing left to skip verifying.
- **One deliberate behaviour change, recorded here so it is not later rediscovered as a bug.**
  Today `Authorize` is called with *the caller's own token*, so go-oikumenea's PDP additionally
  requires the caller to hold `assignment.read` reaching the target unit, with no self-exemption —
  which is why `scripts/bootstrap-registration-org` grants congregation-admin that permission
  (documented at `internal/content/application/authorize.go:21-25`). In-process,
  `Authorize(subject, action, unit)` is a pure function of the subject and that meta-permission is
  meaningless. Role definitions simplify accordingly; `assignment.read` is no longer granted to
  congregation-admin.
- go-oikumenea remains a perfectly good project and this decision is not a judgement on it. The
  facade was the right call for M1–M8: it is what let this project reach nine built modules without
  ever writing an authorization system. This decision is about what the *next* nine cost.

---

> **Amended (2026-08-18) after two independent code-grounded reviews** (`docs/review-result-1.md`,
> `docs/review-result-2.md`), each finding re-verified against the source before adoption. Two of
> this block's numbers were wrong and are corrected here rather than edited above:
>
> - **The port is ~12–15k LOC of Go and ~3–3.5k of migrations, not ~7–8k and ~1.5k.** Hand-written
>   Go in the six source modules, excluding tests and generated sqlc, totals ~21.5k; dropping every
>   `transport/` layer still leaves ~18k before trimming orgs, kinds, clergy, facets and principals.
>   `0008_religion.sql` alone is 1,299 lines and the four locale packs another 1,542. The
>   *qualitative* claim — that the headline 267k is dominated by generated transport, duplicated
>   sqlc models and ~26 unused verticals — is confirmed and unchanged.
> - **`api/core.conjure.yml` is ~25 endpoints, not ~12.** The 9-operation figure was an exact count
>   of what `openfaithmap-admin` calls oikumenea for *today*; it silently omitted the entire
>   super-admin set that [D-SuperAdminFold](#d-superadminfold--super-admin-folds-into-openfaithmap-admin-behind-a-role)
>   requires the same contract to serve — capability that today lives inside `oikumenea-console`
>   and was never part of that survey.
>
> Neither correction changes the decision. "Conjure transport only where a client actually needs
> it" is still the reason the port is tractable; the number attached to it was just too small. The
> cutover remains straight-line on one branch — an informed bet at 12–15k LOC, mitigated by the
> pre-cutover baselines and the authorization matrix in `milestones.md`'s M10.9, not by optimism.

---

### D-CorePortScope — Port the hierarchy, drop the tenancy

**Decision.** The port is a deliberate subset, not a fork. What comes across, and what does not:

| Ported | Dropped |
|---|---|
| `directory_units`, `directory_graphs`, `directory_unit_edges`, `directory_unit_closure`, `directory_closure_status` | `tenant_organizations`, `tenant_domains`, `tenant_unit_kinds`, all lifecycle/code-event tables |
| `identity_persons` (one table), `identity_accounts` | the other 49 `person_*` tables; `personprofile`, `personsensitive` |
| `authz_roles`, `authz_role_permissions`, `authz_role_assignments`, `authz_epoch` | `authz_principal_grants`, `authz_unit_org`, the `authz_readable_units` SQL pushdown helpers |
| 13 `religion_*` tables (taxa + closure + classifications, org kinds/profiles/classifications, policies + policy kinds, sites + site types, schedules + service types, aliases) | the 4 clergy tables and 2 affiliation tables |
| `location_locations`, `location_location_types` | `geo_places` (the Who's-On-First gazetteer) |
| `membership_positions`, `membership_memberships` | `required_rank_id`, `order_item_id`, and all of `internal/rank` |
| `refdata_countries`, `refdata_country_names` | `internal/localization`'s generic `i18n_translations` overlay |
| — | `internal/audit`, all Postgres RLS, service principals, and ~26 unused verticals |

`internal/person` is **rewritten (~500 LOC), not lifted** — upstream's 8,712-LOC package is
structurally wired to `rank`, `order`, `personprofile`, `personsensitive` and `watchlistclient`,
while OpenFaithMap needs exactly one table. The CLDR person-name field set and the trigram
`search_text` column are worth copying verbatim; the package around them is not.

**Why drop tenancy.** OpenFaithMap is one product with one tenant. Organizations and domains exist
upstream to separate customers who share an instance; here they would be a single row that every
query filters on identically.

**Why keep the hierarchy.** It is load-bearing and irreducible:
[D-JurisdictionUnits](#d-jurisdictionunits--denomination-aware-non-uniform-jurisdiction-layer-operator-assigned)
requires variable-depth jurisdiction chains, `subtree`-scoped role assignments
([D-PlatformModerator](#d-platformmoderator--moderator-authority-is-a-go-oikumenea-role-on-the-root-unit))
require ancestor reach, and `congregationimport`'s jurisdiction sync builds real unit trees. None of
that survives flattening.

**Why not** flatten to congregation-specific tables and skip the generic Unit graph entirely:
considered and rejected. It looks simpler until jurisdiction depth, moderation reach and the
Catholic jurisdiction sync all need the same ancestor query, at which point the graph gets
reinvented worse. The closure table is ~1,200 LOC and already debugged.

**Consequences.**
- Dropping organizations is a **design change, not a column drop**: `tenant_units.org_id` and
  `domain_id` are `NOT NULL` upstream, and `pdp_scoped` (the "reference unit exempt from reach
  checks" escape hatch) derives from a unit's domain. The ported schema has no equivalent, so every
  unit is reach-scoped. Any future need for a reference unit is a new decision, not a flag.
- Dropping `authz_principal_grants` removes the machine plane, which is coherent only because
  [D-DirectTokenVerification](#d-directtokenverification--google-id-tokens-verified-in-process-no-service-principal)
  removes machine callers entirely.
- `pkg/{stats,facet,action,crypto,personalcode,events,config}` are not ported. `pkg/facet` alone is
  3,061 LOC serving dashboard/facet endpoints OpenFaithMap has never called, and it drags migrations
  0017–0024 (~800 LOC) with it.

---

> **Amended (2026-08-18) — the ported/dropped table above was incomplete, and one omission was a
> build-stopper.**
>
> **`religion_*` — all 22, explicitly assigned.** The prose cell above named 13 kept + 6 dropped =
> 19, and the bare word "classifications" was ambiguous between four differently-named tables. This
> list supersedes it and is the source of truth for M10.1's migration.
>
> *Keep (15):* `religion_taxon_ranks` · `religion_classifications` · `religion_taxa` ·
> `religion_taxa_closure` · `religion_taxon_classifications` · `religion_org_kinds` ·
> `religion_org_profiles` · `religion_org_classifications` · `religion_policy_kinds` ·
> `religion_org_policies` · `religion_site_types` · `religion_sites` · `religion_service_types` ·
> `religion_service_schedules` · `religion_aliases`
>
> *Drop (6):* the 4 clergy tables (`religion_clergy_credentials`, `religion_clergy_grades`,
> `religion_grade_categories`, `religion_office_types`) and the 2 affiliation tables
> (`religion_affiliations`, `religion_affiliation_types`), with `pkg/crypto`.
>
> *Undecided, resolve before M10.1:* `religion_unit_classifications` — kept only if some path this
> repo actually calls reaches it; otherwise dropped, in which case `religion_classifications` stays
> anyway as `religion_taxon_classifications`' FK target.
>
> **`religion_taxon_ranks` is not optional.** `religion_taxa.rank_id uuid NOT NULL REFERENCES
> oikumenea.religion_taxon_ranks(id)` (`../go-oikumenea/migrations/0008_religion.sql:140`). Omitting
> it produces a migration that does not apply.
>
> **`authz_instance_admins` is ported**, and belongs in the ported column above. `PDP.Decide`'s
> first two branches read `IsInstanceAdmin` (`pdp.go:82-87`); without the table, branch 1 is dead
> code and branch 2 denies every instance-scope action to everyone, permanently. See
> [D-InProcessAuthz](#d-inprocessauthz--the-pdp-runs-in-process-app-layer-only)'s own amendment.
>
> **Also dropped, named so the port does not copy a column without its logic:**
> `tenant_units.visibility` and `ShadowGate`. OpenFaithMap has no shadow-unit concept — site-level
> privacy is `religion_sites.visibility` + `public_precision`, which we keep. Inheriting the column
> without the gate would silently enumerate shadow units to any authenticated caller through
> `ListUnits`; inheriting neither is the decision. Likewise `ReachSet` (pages whole subtrees via
> `DescendantUnitIDs` — for the registration operator's root subtree grant that is every
> congregation in the product) and `scope/scope.go` (192 LOC serving unified search and link
> traversal, neither of which exists here) are **not** ported.
>
> **The closure lock is a row lock, not an advisory lock.** `SELECT id FROM directory_graphs WHERE
> id = $1 FOR NO KEY UPDATE` (`../go-oikumenea/internal/tenant/adapters/queries/tenant.sql:536-540`),
> taken inside the caller's transaction and held to commit. `internal/platform/db/advisorylock.go`
> is a *session-level* lock for boot seeding — a different thing, wanted for a different reason
> (the first-admin seed, see [D-SeedBootstrap](#d-seedbootstrap--bootstrap-becomes-deterministic-seed-migrations)).
>
> Since OpenFaithMap has effectively one authority-bearing graph, every `CreateChildOrg`, `AddEdge`,
> jurisdiction-sync node and candidate approval serialises on **a single row**. Binding invariant:
> **the graph row lock is taken as late as possible, and no network call, geocode or external fetch
> may occur while it is held.** The specific trap is the plausible in-process refactor — today
> `ensureUnit` and `ensureSite` are separate HTTPS calls that lock independently; folding them into
> one "now that it's atomic" transaction holds the lock across all of it, and moving geocoding
> inside holds it across a 1-request/second external call.
>
> **`CreateUnitWithEdge`'s ordering is kept for a different reason than originally recorded.**
> Upstream seeds closure rows before the unit INSERT so the RLS `WITH CHECK` finds a subtree match
> on a brand-new unit. With RLS dropped that motivation is gone; the ordering is harmless and is
> cheap insurance if RLS is ever revisited. Recording the real reason matters — otherwise the next
> person to touch it reorders it and breaks the backstop if it ever comes back.
>
> **`SearchSites` gets one behaviour change, not a verbatim port** — see
> [D-InProcessAuthz](#d-inprocessauthz--the-pdp-runs-in-process-app-layer-only)'s amendment for the
> reasoning; the change itself: `public_precision = 'hidden'` sites are excluded from the public
> search arm entirely, and other non-`exact` sites are filtered and ordered on a generated geometry
> column snapped to their own precision, rather than on the exact geometry.

---

### D-InProcessAuthz — The PDP runs in-process, app-layer only

**Decision.** Authorization is decided by an in-process policy decision point in `internal/authz`.
`Authorize(ctx, subjectPersonID, action, unitID)` is a pure Go function over grants fetched once per
request and cached against `authz_epoch`. There is **no** Postgres row-level security on any
OpenFaithMap table. This resolves `DS-OFM-1` as a deliberate *no* rather than an open seam.

The engine, ported from upstream's `domain/pdp.go`:

1. instance admin → allow;
2. action is instance-scoped and subject is not an instance admin → deny;
3. otherwise, for each active grant carrying the action: `ScopeUnit` matches when
   `target_unit_id == unitID`; `ScopeSubtree` requires the grant's graph to be authority-bearing,
   then matches when `target_unit_id == unitID` or the closure table says target is an
   ancestor-or-self of `unitID`.

The only I/O during a decision is two point lookups against the closure table's primary key.

**Why in-process rather than SQL.** It already is in-process upstream, and for good reason: the
decision needs the subject's whole grant set anyway, so pushing it into SQL means either N queries
or one large join per check. Fetch-once-and-loop is both faster and far easier to test — the engine
becomes a pure function with no database in the way, which is what makes the table-driven test
matrix in M10.9 possible at all.

**Why permissions stay a closed Go catalog** of string constants rather than rows: a permission code
that no code path checks is dead weight, and a code path checking a permission nobody can grant is a
silent hole. Keeping the catalog in Go means the compiler is the integrity check. Upstream's 512-line
catalog is trimmed to what OpenFaithMap actually uses.

**Why not** RLS: rejected for now. Upstream's RLS is genuine defence-in-depth against the
forgotten-`WHERE`-clause bug class, and its predicate (`authz_unit_in_reach`) is keyed on the
authorization reach graph, not on a tenant — so it *would* port. But it needs session-GUC plumbing
on every pooled connection (`app.person_id`, `app.is_instance_admin`, `app.principal_id`), it is
explicitly documented upstream as non-authoritative, and the app-layer PDP it backstops is the same
one we are porting. Adding it now would double the surface of the riskiest phase of this migration
to duplicate a check we are already making. Recorded as a deferred seam, with the port path known.

**Consequences.**
- `Decision.Via []Contribution` (decision-explain) is ported. It is the difference between "403" and
  "403 because grant X on unit Y is `unit`-scoped and you asked about a descendant", and
  authorization bugs are otherwise close to undebuggable.
- The grant cache is invalidated by an epoch counter bumped by every authority-mutating transaction,
  validated with one single-row read. Ported as-is; a stale grant cache is a security bug, not a
  performance bug.
- Background paths that have no human subject (discovery cache refresh, `POST /exclusion-check`) use
  an explicit `authz.SystemContext()` that bypasses the PDP — the in-process equivalent of upstream's
  `RunAsSystem`. It is a named, greppable construct precisely so that "this code path skips
  authorization" is never implicit.

---

> **Amended (2026-08-18) — five changes, one of which reverses a decision made above.**
>
> **1. No grant cache. Grants are read per request.** The block above says the epoch-invalidated
> cache is "ported as-is; a stale grant cache is a security bug, not a performance bug." Porting it
> as-is *is* the security bug, because "as-is" upstream includes a backstop this same decision
> removes. Upstream's own package comment: *"The RLS backstop underneath is exact/live
> (D-RLSLiveReach), so a stale ALLOW cannot read revoked-away rows on RLS-guarded tables"*
> (`../go-oikumenea/internal/authorization/application/grantcache.go:15-17`; `grantCacheTTL = 2 *
> time.Second` at :44). Drop RLS and the cache's 2-second window has no floor under it.
>
> The exposure is concrete: any authority change made *outside* the application bumps the epoch but
> does not reset the local map — raw SQL of the shape `scripts/bootstrap-registration-org` already
> emits, an incident-response `UPDATE`, or a migration editing a base role's permissions, which
> [D-SeedBootstrap](#d-seedbootstrap--bootstrap-becomes-deterministic-seed-migrations) makes the
> *normal* way to change roles. So: one indexed join on `authz_role_assignments` per authenticated
> request. Revocation becomes instant, and ~142 LOC plus the epoch table, singleflight and five
> metrics — all sized for a multi-replica deployment that does not exist — are not ported. Keeping
> RLS instead was the other exit; it needs GUC plumbing on every pooled connection and would enlarge
> the riskiest phase of the migration to duplicate a check we are already making.
>
> **2. Instance admin is a plane, not a role.** `authz_instance_admins` is ported.
> `IsInstanceScope` exists upstream precisely to make instance-scope permissions unsatisfiable by
> any unit-scoped role, so a "super-admin role" is incoherent — and the first admin must be able to
> grant before any unit assignment exists. The permission catalog gains an instance-scope set; see
> [D-SuperAdminFold](#d-superadminfold--super-admin-folds-into-openfaithmap-admin-behind-a-role).
>
> **3. The module-facing entry point takes its subject from context.** The block above proposes
> `Authorize(subject, action, unit)` as "a pure function of the subject". Both reviews confirmed the
> removal of the `assignment.read` meta-check is safe — at all seven call sites the subject argument
> is already the `Whoami`-resolved caller — but the meta-check was also the only thing *binding the
> answer to the authenticated caller*. A subject parameter makes `Authorize` an oracle over
> arbitrary subjects, safe only by call-site convention. That is the same defect class this repo
> already fixed twice (M2.3's untargeted `MyCapabilities`, M3's missing grant). So:
> **`authz.Require(ctx, action, unitID)`** is the only form modules use, subject from context,
> mirroring upstream's own PEP. `authz.DecideFor(ctx, subject, …)` exists solely for the super-admin
> "what can this person do" screen and is itself gated on the instance-admin plane.
>
> **4. `internal/authz/domain` owns a `ClosurePort` interface; `internal/directory` implements it.**
> This is what makes the phase ordering sound and what resolves the authz↔directory cycle — and it
> was load-bearing but unstated. **`internal/authz` imports no other module, and `internal/directory`
> must not import `internal/authz`** (directory's writes are gated at transport). Without this
> written down, the natural implementation imports `internal/directory` from `internal/authz`,
> inverting the dependency, violating this doc set's own cross-module rule, and turning the cycle
> into a compile error found late. Construct the authz service before any route registration so
> upstream's late-`Bind` pattern is unnecessary; if it survives for any reason, port `MustBeBound()`
> and assert at boot — upstream added it after a forgotten `Bind` surfaced as a request-time nil.
>
> **5. `SystemContext()` covers five paths, not two.** The block above names discovery cache refresh
> and `POST /exclusion-check`. A sweep of `coreintegration.NewServiceClient` call sites finds three
> more, and one of them **writes**:
>
> | Path | Site | Kind |
> |---|---|---|
> | discovery cache refresh | `internal/discovery/application/service.go:53,79,123` | read |
> | moderation `CheckExclusion` | `internal/moderation/application/exclusion_check.go:28` | read |
> | `RunConnector` import loop | `internal/congregationimport/application/service.go:90,146` | read |
> | `resolveCountryName` | `internal/congregationimport/application/geocode.go:92` | read |
> | **`RunJurisdictionSync`** | `internal/congregationimport/application/jurisdictionsync.go:75` | **write** |
>
> `SystemContext` must be unforgeable, not merely conventional: a private type in `internal/authz`
> with an unexported key, constructible only by `authz.SystemContext(parent)`; the authentication
> middleware **strips** any system marker from every inbound request context unconditionally;
> `authz.Require` **panics** rather than denies on a system context found in a request-scoped
> context, so the failure is loud in dev and test; and a lint-enforced allowlist of the five files
> permitted to call it.
>
> **`RunJurisdictionSync` additionally gains `requireOperator`**, matching every sibling write in
> its module. Today its transport handler resolves `whoami` and stops
> (`internal/congregationimport/transport/service.go:209-213`), so any authenticated Google account
> can trigger real Unit writes that run under the service principal's instance-wide grant. It copied
> `RunConnector`'s shape, but `RunConnector`'s stated justification — *"it makes no go-oikumenea
> WRITE"* — does not carry over. Its writes stay system-context; its *trigger* stops being
> any-authenticated-account. **This is a live gap on `main` and it stays open until M10.6 lands** —
> a standalone fix first was recommended and deliberately declined in favour of folding it into the
> port. Named in M10.9's refusal-proof list.
>
> **6. `SearchSites` leaks position through its filter, and the fix belongs here.** Coarsening the
> *returned* coordinate does not protect the *predicate*: `ST_DWithin(l.geom, …)` and
> `ORDER BY l.geom <-> pt` run on exact geometry
> (`../go-oikumenea/internal/religion/adapters/discovery.go:354-356`) while `Coarsen` is applied
> app-side afterwards (`internal/religion/application/discovery.go:251`). For a
> `public_precision = 'hidden'` site — a house church, a congregation under harassment, exactly what
> the field is for — membership in an anonymous `GET /search` result set is a boolean oracle on the
> true position, and twenty or thirty varied radius queries binary-search it. KNN ordering leaks it
> faster. This is an inherited upstream defect, not one the migration introduces, but the port must
> not carry it while claiming the opposite property. Fix per
> [D-CorePortScope](#d-coreportscope--port-the-hierarchy-drop-the-tenancy)'s amendment.

---

### D-DirectTokenVerification — Google ID tokens verified in-process, no service principal

**Decision.** `openfaithmap-api` verifies Google ID tokens itself and resolves
`(issuer, subject)` → `identity_accounts` → `identity_persons.id`. A single authentication
middleware, registered via `wrouter.RouteMiddleware` (the same mechanism
[D-Hardening](#d-hardening--in-process-rate-limiting-on-anonymous-writes-reused-witchcraft-observability)'s
rate limiter already uses), does this once per request and puts the caller in the request context.

The GCP service account, its mounted key file, `RegisterServicePrincipal`,
`account_service_principals`, and the whole 473-LOC service-principal feature are **deleted**. This
supersedes `D-ServiceIdentities`.

[D-GoogleDirect](#d-googledirect--google-is-the-sole-identity-provider-no-keycloak) is **unchanged**:
Google remains the sole IdP for humans. What changes is who validates the token.

**Two boot guards are ported, and they are security requirements rather than niceties:**

- **Pinned audience.** Refuse to start with an OIDC issuer configured without a pinned `aud`.
  `https://accounts.google.com` is the same `iss` for *every* Google OAuth client on earth, and `sub`
  identifies a Google account, not an application. Without a pinned audience, a token minted for any
  unrelated Google app authenticates here.
- **No symmetric issuers outside dev.** Refuse HS256 issuers unless the runtime environment is
  local/dev. This is what keeps `scripts/mint-local-token`'s convenience path from becoming a
  production hole.

**Why delete the service principal.** It existed to give a *remote* caller a machine identity the
core could resolve. With no remote caller there is nothing left to identify: background work runs in
the same process, in the same binary, under the same operator's deployment. Keeping a machine
principal would mean minting and verifying a token for a function call.

**Why not** issue our own session JWTs instead of verifying Google's: rejected as scope. It buys real
independence from Google at the cost of key rotation, refresh, and revocation — three things worth
doing deliberately, later, not as a rider on this migration. `next-auth` already owns the browser
session; only the API-side verification moves.

**Consequences.**
- `GOOGLE_APPLICATION_CREDENTIALS` and the mounted `var/service-account.json` are gone. Removing a
  long-lived private key from the deployment is a security improvement independent of this
  migration's other goals.
- Six per-request `Whoami` round-trips collapse into one middleware.
- New configuration: `GOOGLE_OAUTH_CLIENT_ID` (the audience to pin) and a dev-only
  `DEV_ISSUER_HMAC_KEY`.
- JIT account provisioning keeps upstream's `email_verified` requirement. An unverified email claim
  is an account-takeover vector, not an edge case.
- Background work becomes unattributable — there is no principal RID to record. Noted as a deferred
  seam; it matters more once [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed)'s
  dropped audit log comes back.

---

> **Amended (2026-08-18) — "two boot guards" undercounts what has to come across, and the guard as
> described has nothing to key on.**
>
> **Port `validator.go` and `authenticator.go` in full.** The risk is specifically the combination
> of [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed)'s "port freely,
> simplifying as code lands" with a test list that would not catch an algorithm-confusion or
> clock-skew regression introduced during that simplification. Beyond the two guards named above:
>
> - **`GuardReservedIssuer`** — a third guard, not mentioned. It stops an operator pointing a real
>   IdP at the synthetic local issuer, which is exactly the attack invited once that issuer string
>   is a constant in a public repository.
> - `jwt.WithValidMethods` algorithm pinning, and `jwt.WithLeeway` clock skew — noting that upstream
>   applies leeway only on the HS256 path and delegates the OIDC path to go-oidc with none. Whatever
>   OpenFaithMap does should be a decision, not an accident of which branch got the option.
> - `audienceAccepted` multi-audience matching. `SkipClientIDCheck: true` is safe **only** because
>   that check runs unconditionally and the audience guard guarantees a non-empty set. Port the trio
>   together or not at all — the verifier without the guard is a silent no-op audience check.
> - JWKS caching and rotation are entirely go-oidc's, built lazily per issuer and never refreshed
>   for discovery. A known property worth stating rather than assuming.
> - `nonce` is correctly next-auth's job, not the API's. Say so rather than leaving it unaddressed.
> - `azp` is projected but must never be an authorization input. If per-surface OAuth clients
>   (`D-OAuthClients`) ever land, `aud` alone stops distinguishing the two surfaces — decide then,
>   not by default.
>
> **`environment` becomes a real `config.Install` field**, and is the *only* input to the symmetric-
> issuer guard. Upstream's `GuardSymmetricIssuers` fails closed on any environment that is not
> `local`/`dev`, reading it from install config — but `internal/platform/config/config.go:22`
> declares `Install` with **zero fields**, and its own doc comment records why: every setting is read
> straight from the environment via `requireEnv`, bypassing the type. The natural implementation
> ("register the HS256 issuer when `DEV_ISSUER_HMAC_KEY` is non-empty") makes the guard
> self-authorizing — the dev key is permitted because the dev key is present. **Never derive "dev"
> from the presence of a secret.** Boot fails on unknown or empty, exactly as upstream does; ship
> `DEV_ISSUER_HMAC_KEY` commented out, never with a value, and refuse a known placeholder. This
> repo's own track record is the argument: `docker-compose.yml` already carries
> `OIKUMENEA_INSECURE_SKIP_VERIFY: "true"` and three `NODE_TLS_REJECT_UNAUTHORIZED: "0"`, each
> marked DEV-ONLY, each inherited by any production override that does not explicitly unset it. This
> also resolves `U12` for the one setting where it matters most.
>
> **Admin sessions will break at one hour unless Phase M10.2 also fixes the client.**
> `web/apps/admin/auth.ts:41-48` captures `account.id_token` once at sign-in and never refreshes it.
> Google ID tokens live one hour; the next-auth session lives far longer. Today the mismatch is
> masked by the remote core; once `openfaithmap-api` owns the `exp` check it becomes this project's
> problem. The sentence above — "next-auth already owns the browser session; only the API-side
> verification moves" — is what hid it. Refresh-token handling is M10.2 scope, not a follow-up.
>
> **Replay posture, recorded rather than left implicit.** A valid Google ID token is accepted from
> any client; there is no `jti` cache and no binding to a session. That is an acceptable posture for
> this product. Note the migration *improves* on today by deleting a hop that currently runs with
> certificate verification disabled.

---

### D-OwnRIDs — UUIDv8 resource identifiers, owned by OpenFaithMap

**Decision.** Port go-oikumenea's UUIDv8 RID scheme: a native RFC 9562 §5.8 UUID carrying a
millisecond timestamp in bytes 0–5 and app/service/kind/type discriminators in bytes 6–10, minted by
a SQL column `DEFAULT new_id(service, kind, type)` and validated by a per-table structural `CHECK`.
Keep the CHECKs. **Drop** the `platform_rid_services` / `platform_rid_types` registry tables. Render
as `ofm:<service>:<kind>:<type>:<uuid>` at the API boundary only — never stored in that form.

**Why keep the structure.** The timestamp prefix gives b-tree insert locality that random UUIDs
don't, and the structural CHECK catches passing a unit RID into a site column at write time rather
than as a confusing empty result later. In a codebase where cross-module references are deliberately
opaque `TEXT` (`conventions.md`) and therefore *cannot* be foreign keys, that CHECK is the only
type-safety left.

**Why drop the registry tables.** They are documentation stored as rows. The service/kind/type
codes are already Go constants; a second copy in Postgres is a thing to keep in sync, not a
constraint.

**Why not** plain UUIDv7: it is genuinely simpler, and it was a close call. Rejected because the
CHECKs cost one line per table and pay for themselves the first time a RID is passed to the wrong
query — which, with every cross-module reference being untyped `TEXT`, is a matter of when.

**Consequences.** Every existing `*_rid TEXT` column and all six existing Conjure contracts are
untouched — they always treated RIDs as opaque strings, which is exactly why this substitution is
invisible to them. The rendered prefix changes from `oikumenea:` to `ofm:`, which is safe only
because the cutover is greenfield
([D-SeedBootstrap](#d-seedbootstrap--bootstrap-becomes-deterministic-seed-migrations)); there is no
stored RID to rewrite.

---

> **Amended (2026-08-18) — the `ofm:` prefix is dropped; RIDs stay bare uuids on the wire.**
>
> This block claims both that RIDs render as `ofm:<service>:<kind>:<type>:<uuid>` at the API
> boundary *and* that "every existing `*_rid TEXT` column and all six existing Conjure contracts are
> untouched". Those cannot both hold. The *types* are untouched; every *value* changes. Today
> `content_sites.congregation_unit_rid` stores a bare uuid, the admin app round-trips that string,
> and `web/apps/web/app/[locale]/congregations/[unitId]/page.tsx` puts it in a **URL path segment**,
> where `ofm:4:1:1:<uuid>` needs escaping.
>
> So: **keep the UUIDv8 structure and the per-table structural CHECKs — that is where the value
> actually is — and store and expose the plain uuid, exactly as today.** No parse/format boundary,
> no escaping, no value churn, and the type-confusion net this decision exists for is unaffected.
> The human-readable form remains available as a debugging helper in logs; it is never a wire or
> storage value.
>
> One consequence worth naming: with no prefix and no cross-module foreign keys
> ([conventions.md](conventions.md)), a well-formed RID carries no authority and no referential
> integrity. The structural CHECKs catch a unit RID in a site column; they never catch an
> unauthorized one. Every `*_rid` parameter must be authorized, not merely validated — which is what
> [D-InProcessAuthz](#d-inprocessauthz--the-pdp-runs-in-process-app-layer-only)'s
> `authz.Require(ctx, …)` is for.

---

### D-SeedBootstrap — Bootstrap becomes deterministic seed migrations

**Decision.** The root unit, the base roles and their permission sets, org kinds, site types, the
`canonical` graph, the exclusion backstop, and the 249-row country list ship as **Atlas seed
migrations with fixed, hard-coded RIDs** — not as output of manually-run scripts.
`scripts/bootstrap-{service-principal,admin-person,registration-org,exclusion-backstop}` are deleted.

**Why.** Today these are instance-specific RIDs produced by four manual script runs, then pasted
into `.env` as `REGISTRATION_ROOT_UNIT_ID`, `REGISTRATION_CONGREGATION_ADMIN_ROLE_ID` and
`CATHOLIC_JURISDICTION_ANCHOR_UNIT_ID`. That is precisely why environments are not reproducible:
two developers running the documented steps get different identifiers, and no test can name a unit.
Owning the tables makes the values ours to choose, so we choose them once.

**Why not** keep scripts and have them emit deterministic RIDs: rejected — that is a seed migration
with extra steps, and it keeps the "did you remember to run the four scripts, in order" failure mode
that `docs/milestones.md` records as a recurring source of broken local stacks.

**Consequences.**
- Three required environment variables disappear. `docker compose up --build` becomes genuinely
  one-shot: no sibling checkout, no service-account JSON, no manual bootstrap sequence.
- Fixed RIDs become referenceable constants in Go and in tests, which is what makes the M10.9
  authorization matrix expressible.
- The seed is a migration, so changing a base role's permissions is a reviewed, versioned schema
  change rather than an undocumented production `UPDATE`.
- Genuine trade-off: seeded RIDs are identical across every deployment. That is fine here — they
  identify structural rows, not secrets, and OpenFaithMap is a single-instance product — but it
  would need revisiting if the project ever ran multiple independent instances that federate.

---

> **Amended (2026-08-18) — as written, this decision produces an instance nobody can administer.
> Identity is the one place determinism must not apply.**
>
> The block above deletes `scripts/bootstrap-admin-person` and seeds the root unit, base roles, org
> kinds, site types, the exclusion backstop and the country list. It seeds **no person, no account,
> and no instance admin** — and go-oikumenea's JIT is link-on-match-only: *"on no match, reject. JIT
> never creates a person"* (`scripts/bootstrap-admin-person/main.go:4-11`). On a clean volume, the
> first human to sign in with Google is refused, there is no shell account for their identity to
> link onto, and no instance admin exists to grant the first assignment. The instance is
> unadministrable, and M10.9's "`whoami` resolves to the seeded admin person" could not pass.
>
> **The obvious fix is worse than the bug.** Seeding a shell account with a fixed email combines
> badly with this decision's deterministic RIDs: a committed seed email in an open-source repo means
> every deployment that boots the migration unmodified ships the same pre-linked admin address, and
> whoever controls that address — or later registers the domain — is instance admin everywhere.
> `email_verified` does not help; the address is verified, just not yours.
>
> **So: the first admin is seeded at boot from install config, never by a migration.** An
> `environment`-validated install-config field (see
> [D-DirectTokenVerification](#d-directtokenverification--google-id-tokens-verified-in-process-no-service-principal)),
> applied idempotently at startup under the session-level advisory lock
> `internal/platform/db/advisorylock.go` exists for, a no-op once `authz_instance_admins` has any
> active row, and boot **refused** on the placeholder value the way the symmetric-issuer guard
> refuses HS256. A migration is structurally the wrong place regardless of the backdoor problem: the
> admin's email is deployment-specific and the Google `sub` is unknowable until first login.
>
> **Email matching must be exact and normalised identically at write and read.** Port upstream's
> `citext` column and state the normalisation. Google treats `foo@gmail.com`, `f.o.o@gmail.com` and
> `foo+x@gmail.com` as one mailbox but returns the address as registered — a case-insensitive-but-
> not-dot-insensitive match is either safe or a takeover depending on details nobody wrote down.
> Once linked, `(issuer, subject)` — never the email — is the identity key thereafter, so an email
> change at Google cannot re-target the link.
>
> **The rest of this decision stands, and gains an argument it did not make.** Structural RIDs as
> fixed constants are correct, and they are what makes M10.9's table-driven authorization matrix
> expressible at all — a test cannot name a unit whose identifier differs per environment. That is
> the strongest case for this decision and it was missing from the block above. The split is:
> structural RIDs deterministic, **identity RIDs never**.

---

### D-SuperAdminFold — Super-admin folds into openfaithmap-admin behind a role

**Supersedes [D-InstanceAdminConsole](#d-instanceadminconsole--reuse-go-oikumeneas-own-console-as-the-third-super-admin-only-surface).**

**Decision.** The `oikumenea-console` service is deleted. The super-admin capabilities OpenFaithMap
actually uses — managing people, role grants, units and taxa — become screens inside
`openfaithmap-admin`, gated on a super-admin role, served by the same `api/core.conjure.yml`
endpoints. There are now **two** UI surfaces, not three.

**Why.** D-InstanceAdminConsole's entire rationale was "it already exists and is already maintained
as part of go-oikumenea." Once [D-OwnCore](#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed)
removes go-oikumenea, that rationale evaporates: keeping the surface would mean *building* a third
Next.js app to replace a free one, to administer tables we now own, for a handful of screens.

**Answering the objection D-InstanceAdminConsole raised**, because it was a good one. That decision
rejected exactly this fold, on the grounds that it would make "a congregation-admin console
occasionally also an instance-admin console depending on who's logged in." The objection was
correct **against instance-wide authority over a shared, general-purpose core** — a tenant graph and
service-principal issuance spanning consumers beyond OpenFaithMap. That core no longer exists. The
widest authority in the system is now super-admin over OpenFaithMap's own tables, which is a
difference of degree from platform-moderator (already an `openfaithmap-admin` audience under
[D-AdminSurface](#d-adminsurface--the-adminmoderator-console-is-a-separate-deployment-from-the-public-site)),
not a difference of kind. The public/admin split D-AdminSurface actually defends — anonymous surface
versus session-holding surface — is untouched.

**Why not** build `openfaithmap-console` as a third app: rejected. It is a whole deployment,
Dockerfile, OAuth client and CI matrix entry for a screen count in the single digits, and it would
reintroduce the WireGuard requirement this decision otherwise retires.

**Consequences.**
- `DS-OFM-14`'s WireGuard half is resolved by deletion: there is no surface left that must not have
  a public port. Per-surface OAuth clients
  ([D-OAuthClients](#d-oauthclients--one-google-oauth-client-today-one-per-surface-as-the-target))
  narrow from three surfaces to two, and the shared-client problem disappears with
  `oikumenea-console`.
- `openfaithmap-admin` now holds the highest authority in the system, so its own hardening matters
  more than before. Super-admin screens must be role-gated server-side, not merely hidden in
  navigation — a named M10.9 verification criterion, proven with a refused congregation-admin token.
- `OIKUMENEA_CONSOLE_AUTH_SECRET` and the console's compose service, port 3003, and Google callback
  registration all go away.

---

> **Amended (2026-08-18) — "gated on a super-admin role" is incoherent, and the server-side gate
> this decision promises does not exist to be reused.**
>
> **Read "super-admin role" as "the instance-admin plane" throughout.** Instance admin is
> deliberately *not* a role: `IsInstanceScope` exists upstream precisely to make instance-scope
> permissions unsatisfiable by any unit-scoped role, and the first admin must be able to grant
> before any unit assignment exists. `authz_instance_admins` is ported
> ([D-InProcessAuthz](#d-inprocessauthz--the-pdp-runs-in-process-app-layer-only)); the fold is onto
> that plane, not onto a role.
>
> **This decision's consequences section requires that "super-admin screens must be role-gated
> server-side, not merely hidden in navigation" — but there is no such mechanism in
> `openfaithmap-admin` to follow.** Its admin layout says so in its own comment: it *"only removes
> duplicated 'is anyone logged in' boilerplate — it adds no role/permission gate"*
> (`web/apps/admin/app/[locale]/admin/layout.tsx:11-15`). Every authorization check in the product
> today is a per-call backend gate, and `content`, `moderation`, `vouching`, `discovery` and
> `congregationimport` each maintain their own hand-copied `require*` function. Betting the
> widest-blast-radius surface in the system on a sixth hand-copy, with no tests, is the single worst
> place to do it.
>
> So M10.8 builds **two gates, both required**:
> 1. one **shared, hard-to-misuse enforcer** in the API, used by every super-admin handler — not a
>    per-file copy;
> 2. a `requireInstanceAdmin()` check in the super-admin route group's layout, for cosmetic gating
>    only, with the API gate remaining the real one.
>
> Both are proven in M10.9 with a refused congregation-admin token — the layout check and the
> handler check tested separately, because a passing layout with an ungated handler is the failure
> mode that matters.
>
> The objection inherited from `D-InstanceAdminConsole` is answered above and is unchanged by this
> amendment: the line moved because the shared general-purpose core moved, not because it stopped
> mattering.

---

### D-StaticRefData — Reference data is a static seed; hermenea is removed

**Supersedes [D-BulkImport](#d-bulkimport--hermenea-replays-the-existing-registration-flow-in-bulk-no-new-write-path).**

**Decision.** Country reference data ships as two seed tables: `refdata_countries` (the 249-row
ISO-3166-1 alpha-2 list) and `refdata_country_names` (flat `code, locale, name` rows for en/es/pt/uk,
lifted from go-oikumenea's locale-pack migrations). The `hermenea` service, its binary, its separate
database and its migration set are deleted, along with the weekly cron and the two shared-token
environment variables.

**Why.** What hermenea supplied to OpenFaithMap was the country list — and that list was *already* a
static 249-row `INSERT` in go-oikumenea's `0001_platform_core.sql`. Hermenea re-upserted the same
alpha-2 codes and additionally enriched rows with Who's-On-First geometry, `wof_id`, `iso_a3` and
`numeric_code`, none of which OpenFaithMap ever queried. The remaining mappers (Glottolog, CLDR,
Wikidata orgs, Factbook ethnicities, Interpol/sanctions) feed upstream verticals this project does
not have.

Deleting it removes 4,621 LOC, a second database, a second binary, a cron, a job queue, five
`olehmushka/go-*-client` dependencies, and five recurring outbound fetches to
raw.githubusercontent.com, iso639-3.sil.org and query.wikidata.org.

**Why not** port hermenea's importers into the existing `congregationimport` connector pattern:
rejected as speculative. Country codes change on a timescale of years, and the mechanism to refresh
them is "edit a migration and open a PR" — which is *better* than a cron, because it is reviewed and
versioned. Build an importer when there is data that actually moves.

**Why a flat `refdata_country_names` table** rather than porting `internal/localization`'s generic
`i18n_translations` overlay: the generic mechanism exists upstream to translate arbitrary entity
types across an open-ended locale set. OpenFaithMap translates exactly one entity type into exactly
four locales. A two-column lookup is the whole requirement.

**Consequences.**
- The compose stack loses `init-hermenea-db`, `migrate-hermenea` and `hermenea` — three of the seven
  services this migration removes — plus ports 9443/9444 and the
  `HERMENEA_OIKUMENEA_TOKEN`/`OIKUMENEA_HERMENEA_TOKEN` pair (which shipped with committed
  `dev-*-change-me` placeholder defaults).
- Country geometry is no longer available. Nothing uses it; if spatial country lookup is ever
  wanted, `refdata_countries` has room for a `geom` column and the WOF import is a known quantity.
- `D-BulkImport`'s underlying principle is untouched and worth restating, since it long outlived the
  hermenea mechanism it was written about: bulk paths reuse the ordinary write path and never invent
  a second one. `congregationimport` still provisions through the same religion/directory/location
  calls registration uses.

> **Amended (2026-08-18) — one constraint on the seed, surfaced by review.** Nothing in this
> decision was challenged; both reviewers independently confirmed that hermenea's country mapper
> emits `code`/`name`/`alpha3`/`numeric` into a table that already holds 249 static rows, and that
> OpenFaithMap reads only `Id` and `Name`. It goes.
>
> The constraint is on **fidelity of the port, not the decision**: the four locale packs must
> survive **byte-for-byte**. `matchCountry` does exact-string comparison against *every* locale name
> of every country (`internal/congregationimport/application/countrymatch.go:44-50`), and the `osm`
> connector's `CountryHint` is deliberately built to match those strings exactly
> (`adapters/connectors/osm/connector.go:115-127`). A single re-typed diacritic silently breaks
> country matching for one country in one locale — and M10.9's `ua-edr` re-run exercises Ukraine
> only, so it would not be caught there. M10.9 therefore diffs the whole `refdata_country_names`
> table against a pre-cutover `ListCountries()` capture: all four locales, 249 rows.

### D-AccountStatusEnforcement — `identity_accounts.status` is checked at resolution, not just stored

**Decision.** `ResolveBySubject` (`internal/identity/adapters/store.go`) and the authenticator's
resolution path reject any account whose `identity_accounts.status` is not `'active'`, in addition
to the existing `deleted_at IS NULL` check. A disabled account fails at authentication — it never
reaches the PDP, never resolves to a `Resolution{PersonID, AccountID, Email}` at all.

**Why.** The M11 discovery pass found `status` (`active`/`disabled`) has existed in the schema since
M10.1 (`migrations/0008_core_identity.sql`) but was never read anywhere —
`ResolveBySubject` (`internal/identity/adapters/store.go:167-179`) only filters `deleted_at IS
NULL`. Disabling an account today is a no-op: the row still resolves, still authenticates, still
authorizes normally. Not hypothetical — it directly undermines M11.1's own deactivate/reactivate
feature (disabling someone would silently do nothing), and M11.3 layers session revocation on the
same enforcement point, so it would have undermined that too.

**Why at resolution, not in the PDP.** A disabled account is an authentication failure (this
identity no longer gets to *be* anyone), not an authorization failure (this identity isn't allowed
to do *this*). Putting the check in the PDP would still resolve a disabled account to a real
subject and fail it only per-permission — wrong shape, and it would leave the anonymous-route
bypass logic (`isBypassPath`) needing to know about account status, which it has no business
knowing.

**Consequences.**
- Every later M11.x milestone that touches auth (M11.3's session checks, M11.6's
  invite-then-JIT-link path) can assume a disabled account is fully inert, not merely unprivileged.
- No migration needed — the column already exists; this is a code-only fix to a store method and
  its callers.
- Existing tests that create accounts without explicitly setting `status = 'active'` need auditing —
  the column's default (per `migrations/0008_core_identity.sql`) determines whether they still pass
  unchanged.

### D-AuditLogShape — `identity_audit_log` folds into the identity service RID, before/after is curated JSON per call site

**Decision.** `identity_audit_log`'s primary key uses `new_id(1,1,4)` — the identity service (1), not
a new service number — and its `before`/`after` columns hold whatever `map[string]any`/struct each
mutating call site builds by hand, not a generic full-row diff.

**Why the identity service, not a new one.** The migration file is named `0016_core_audit.sql`,
matching the `core_<subdomain>` naming series `0007`–`0015` already established for core absorption,
while the table itself stays `identity_`-prefixed like `identity_persons`/`identity_accounts`. An
audit entry's natural subject is "an identity acted" — the same relationship `authz_instance_admins`
already has to the authz service rather than being its own service number. A new service number was
considered and rejected as unwarranted ceremony for what is, mechanically, one more identity-owned
ledger.

**Why curated per-call-site JSON, not a generic before/after row diff.** Two of the six mutating
paths this table logs (`RevokeRoleAssignment`, `RevokeInstanceAdmin`) had store methods doing a bare
`UPDATE ... WHERE ...` with no `RETURNING` and no `GetByID` on either store — building a generic
"fetch the row, diff it" mechanism would have meant adding read paths to `internal/authz/adapters`
that don't exist for any other reason, and a wrong `WHERE` on a "before" read taken concurrently with
another write could observe post-write state. Cheaper and more precise instead: a small `RETURNING`
addition to each revoke `UPDATE` (and `InsertRoleAssignment` returning a real id, including on its
idempotent-conflict path) gives each call site exactly the fields it needs, hand-assembled into the
audit payload at the point of the mutation.

**Consequences.**
- `action` is free text validated in Go (`internal/core/application/service.go`'s
  `auditAction*`/`auditTarget*` constants today), not a DB `CHECK` enum — M11.3/M11.7/M11.8 can each
  add their own logged actions without a migration.
- The four M10.8 grant/revoke call sites and M11.1's deactivate/reactivate now share one
  `requireSubject` guard that hard-fails on a missing context subject, closing the same "discarded
  `SubjectFromContext`'s `ok` bool" gap in all four at once, rather than leaving three fixed and one
  not.
- `internal/auditlog` is a new, self-contained module (domain/application/adapters, no transport of
  its own) rather than logic folded into `internal/core` — `internal/core/application`'s own header
  already scopes itself as owning no new domain logic beyond one gate, and M11.3's session revocation
  needs `Record` reusable without reaching into `internal/core`.

### D-SessionTracking — auth gains a server-side session record, no longer purely stateless

**Decision.** A new `identity_sessions` table (session id, account id, issuer, `created_at`,
`last_seen_at`, an optional device/user-agent label, `revoked_at`) is introduced. NextAuth
(`web/apps/admin/auth.ts`) issues a session id into its JWT at sign-in; the backend checks that id
is present and unrevoked on every authenticated request, alongside — not instead of — the existing
bearer-token signature/claims verification.

**Why.** M11 scoped "session visibility/revocation" as a real requirement — an admin or user seeing
active sessions and forcing one out — not just the lighter behavior
[D-AccountStatusEnforcement](#d-accountstatusenforcement--identity_accountsstatus-is-checked-at-resolution-not-just-stored)
already provides on its own. That lighter form has no visibility (no list of who's signed in where)
and no way to kill *one* session while leaving others alone; real revocation needs a record to
revoke.

**Why not the lighter option instead, given the added complexity.** Considered and rejected: relying
on account-status-disable plus a short token TTL, and relying on Google's own session management.
Neither meets the actual requirement — account-status-disable is all-or-nothing per person, not
per-session, and Google's session state is invisible to this app either way.

**Consequences.**
- A genuine reversal of the stateless-JWT posture M1's original session design established (no
  adapter/session-store, `web/apps/admin/auth.ts`). Adds one indexed point-lookup to the hot path of
  every authenticated request — same cost class as the existing per-request token verification, not
  a new bottleneck class.
- `identity_sessions.last_seen_at` becomes the natural source for M11.4's last-login/activity
  tracking rather than a duplicate column.
- Revoking a session is a mutation, audit-logged per M11.2 like every other admin action this arc
  introduces.

### D-SessionIdTransport — the session id travels as its own header, checked for every bearer including dev tokens

**Decision.** D-SessionTracking's own text ("alongside — not instead of — the existing bearer-token
verification") left two mechanics unresolved, both settled at build time (M11.3, confirmed with the
user before implementation): (1) `web/apps/admin/auth.ts`'s NextAuth JWT is an encrypted session
cookie entirely separate from the Google ID token forwarded as the API bearer — Google signs that ID
token, so a custom `sessionId` claim can't be injected into it. The session id instead travels as its
own opaque `X-Session-Id` header, sent alongside `Authorization: Bearer <idToken>` via
`lib/openfaithmap`'s existing `fetch` override hook (`lib/core.ts`'s `client()`), and read by
`internal/identity/middleware.Authenticator.Handle` right after bearer resolution — one indexed
`identity_sessions` lookup (`Touch`), cross-checked against the bearer-resolved account id so a
session id for a *different* account can't be substituted in. (2) The check applies to **every**
authenticated request with no issuer-based carve-out — the reserved local/dev HS256 issuer
(`internal/platform/devtoken`) used by `cmd/openfaithmap-api/authorization_matrix_test.go` and
`scripts/mint-local-token` is not exempted, even though neither previously had any session concept.

**Why not exempt dev tokens.** Considered and rejected: the alternative kept the authorization-matrix
test suite and `mint-local-token` unchanged, at the cost of a permanent code-level distinction between
"real" and "test" auth paths that every future session-aware check would have to remember to
replicate. The user chose the more invasive retrofit instead — `seedSubjects` now inserts a real
`identity_sessions` row per minted subject and `doReq` sends `X-Session-Id` on every request;
`mint-local-token` gained optional `-database-url`/`-account-id` flags that insert a session row and
print its id, a deliberate, opt-in expansion of a tool whose doc comment previously promised "no API
calls, no side effects."

**Bootstrapping exemption.** One route, `CoreService.registerSession` (`POST /core/v1/sessions`), is
listed in `internal/identity/middleware.sessionExemptRoutes` — it is what CREATES the session row a
session id would otherwise need to already exist, so it cannot itself require one. This is a
session-check-only exemption, not an authentication bypass (unlike `anonymousRoutes`/`isBypassPath`,
which skip authentication entirely) — `registerSession` still requires a fully valid bearer, and
derives both the owning `accountId` and the recorded `issuer` from that bearer's own resolved
`authz.Subject` (new `SessionID`/`Issuer` fields), never from a client-supplied request field.

**Consequences.**
- `identity_sessions` is `new_id(1,1,5)` — identity's next free object RID slot after
  `identity_audit_log`'s `(1,1,4)` — and is a **mutable** table (`last_seen_at`/`revoked_at` updated
  post-insert), so it follows `identity_accounts`' plain-`UPDATE` shape, not
  `identity_audit_log`/`moderation_actions`' `reject_mutation()` append-only pattern.
- `last_seen_at` is bumped throttled, not on every request: `TouchSession`
  (`internal/identity/adapters`) only issues the `UPDATE` when the existing value is more than 60s
  stale — a build-time call on the tradeoff M11.4's own milestone text flagged as open ("updated on
  each authenticated request or session refresh (decided at build time)").
- `authz.Subject` gained two fields beyond M10's original `PersonID`/`AccountID`/`Email`:
  `SessionID` (the caller's own session, for self-scoped "is this my current session" UI) and
  `Issuer` (so `RegisterSession` never trusts a client-supplied issuer).

### D-InviteLinkMVP — invite-a-teammate ships as a shareable link, not an emailed invite

**Decision.** M11.6's invite flow pre-provisions a person/account row and produces a signed,
one-time invite link the admin copies and shares out-of-band (Slack, email, however they'd already
reach the person). No email is sent by the app.

**Why.** No email-sending infrastructure exists anywhere in this repo today (confirmed by direct
grep — no SMTP/SES/SendGrid client, no notification package); building one is a real new subsystem
(provider account, deliverability, templates) the user deliberately deferred rather than folding
into this milestone's scope.

**Consequences.**
- Recorded explicitly so a future session doesn't mistake the absence of email delivery for an
  oversight: real email sending is wanted, just not now.
- The invite link still has to produce a row M10.2's existing JIT link-on-match logic will actually
  match on first Google sign-in (`IDENTITY_JIT_MATCH=account-email` mode,
  `internal/identity/middleware/validator.go:29-34`) — M11.6 is scoped to verify this against the
  *existing* JIT code path, not just the new invite code.

### D-NoAppLevelMFA — MFA is not built at the application layer

**Decision.** M11 does not add MFA enrollment or enforcement. `identity_accounts.mfa_enrolled_at`
remains unused.

**Why.** MFA was in scope for discussion but dropped once its cost was made concrete: any app-level
second factor (TOTP secrets, backup codes) means storing a new form of credential, which directly
reopens what `identity_accounts_dormant_credentials`'s CHECK constraint was built to close off —
this app deliberately stores no password/MFA material, ever, and delegates all of that to Google.
Requiring Google's own MFA assertion (checking an `amr` claim) was the one variant that wouldn't
reintroduce local credential storage, but was still dropped as not worth building against for now —
Google already enforces this for any account/org that turns it on, with zero code on this side.

**Consequences.**
- If MFA enforcement is wanted later, the cheapest path that doesn't reopen the no-credentials
  guarantee is parsing/requiring the OIDC token's `amr` claim (`internal/identity/middleware
  /validator.go`'s `project()` doesn't read it today) — recorded here so a future session doesn't
  have to rediscover this trade-off from scratch.
