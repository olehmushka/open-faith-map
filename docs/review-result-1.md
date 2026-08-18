# Migration plan review #1

Reviewed: `~/.claude/plans/the-goal-is-write-adaptive-engelbart.md` against `0fc2b5878b7f5532a5b9cd758cf194af9f612bc8`
Date: 2026-08-17
Verdict: **No — not safe to execute as written.** The strategy is right and most of the estimating is honest, but the plan omits the table its own PDP branches on first (`authz_instance_admins`), has no path by which any human becomes an administrator of a freshly seeded instance, drops Postgres RLS on a stated rationale that upstream's own source refutes, and ports a PostGIS query whose spatial predicate leaks the exact coordinates the plan claims it protects. Four of these are fixable in a paragraph each; none of them are fixable after Phase 6.

---

## Verification of the plan's factual claims

| Claim | Verdict | Evidence |
|---|---|---|
| PDP is ~260 lines of pure in-memory Go, not SQL / not a recursive CTE | **confirmed** | `../go-oikumenea/internal/authorization/domain/pdp.go` is 260 lines; `Decide` at :81–139 is a loop over pre-fetched `[]ActiveGrant` with no SQL |
| …evaluated over pre-fetched grants | **confirmed** | `DecisionInput.Grants` at pdp.go:52–58; assembled once by `cachedAuthority` (`application/grantcache.go:104`) |
| …with only **two closure-table** point lookups as I/O | **partly** | Two calls, but only one touches the closure: `IsAncestorOrSelf` → `ClosureHasPath` (pdp.go:120). `IsAuthorityBearing` (pdp.go:108) is a `GetGraphByID` on `tenant_graphs` (`internal/tenant/application/closure_port.go:26–32`). Both are *per subtree grant*, memoized per graph within one decision (pdp.go:91, 106–114) — not two per decision |
| pdp.go is all that is needed from that file | **refuted** | The same 260 lines also carry `ReachSet` (:163–198, calls unbounded `DescendantUnitIDs`) and `ShadowGate` (:251–260). The plan ports the file and never says which half it wants — see *Clean architecture* below |
| `geo_countries` is already a static 249-row seed upstream | **confirmed** | `../go-oikumenea/migrations/0001_platform_core.sql:236` — `INSERT INTO oikumenea.geo_countries (code, name, sort_order) VALUES` followed by exactly 249 rows |
| hermenea adds nothing beyond WOF geometry OFM never queries | **confirmed** | `../go-oikumenea/internal/hermenea/geocountries/mapper.go:4–8, 41–60`: the country mapper emits `code`, `name`, `alpha3`, `numeric` into the same table. OFM reads only `Id` and `Name` (`internal/congregationimport/application/countrymatch.go:44–50`) |
| `membership` needs 2 tables | **confirmed** | Only `membership_memberships` and `membership_positions` exist upstream |
| `person` needs exactly 1 of ~50 `person_*` tables | **confirmed** (count) | 50 distinct `CREATE TABLE oikumenea.person_*` across `../go-oikumenea/migrations/`. Whether one suffices is a design claim, not checkable — but see the *identity* gap below: the plan's `identity_accounts` silently merges upstream's `account_accounts` + `account_external_identities` (`migrations/0004_authz_identity.sql:203, 247`) and loses the shell-account concept that bootstrap depends on |
| 13 of the 22 `religion_*` tables are needed | **refuted** | 22 confirmed. The plan names 13 kept + 6 dropped (4 clergy, 2 affiliation) = 19. Three are unaccounted: `religion_taxon_ranks`, `religion_taxon_classifications`, `religion_unit_classifications`. **`religion_taxon_ranks` is not optional**: `religion_taxa.rank_id uuid NOT NULL REFERENCES oikumenea.religion_taxon_ranks(id)` (`migrations/0008_religion.sql:140`; table at :62–74). Phase 1 as written produces a migration that does not apply |
| ~7–8k LOC of Go | **refuted as stated; optimistic by ~2×** | Hand-written Go (excluding `*_test.go`, sqlc `*.sql.go`, `models.go`) in the six source modules: authorization 4,368 · tenant 4,381 · religion 5,361 · membership 2,465 · geo 2,015 · identityfederation 2,979 = **21,569**. Dropping every `transport/` layer (authz 676, tenant 1,216, religion 1,536) still leaves ~18k before trimming orgs/kinds/clergy/facets/principals. 7–8k requires a >60% cut on top of that, plus `pkg/{rid,authn,listing,errors}` and ~450 LOC of `platform/db` the plan already counts separately. ~12–15k is the defensible number |
| ~1.5k LOC of migrations | **refuted; ~2× low** | `0008_religion.sql` alone is 1,299 lines for the 22 tables (13–14 kept). Locale packs `0012`–`0015` total 1,542 lines. The 249-row country seed is ~250. Add authz (`0004`: 279), the tenant/units subset of `0001`+`0002`, identity, location, membership, and the base-role/org-kind/site-type/graph seeds → **~3,000–3,500** |
| Only ~12 endpoints need Conjure transport | **partly** | The 9 the admin app calls today is exact — `tenant.getUnit/listUnits/unitAncestors`, `religion.createChildOrg/listTaxa`, `geo.listCountries`, `identityFederation.whoami`, `membership.listMembers`, `person.getPerson` (`web/apps/admin/lib/jurisdiction.ts:42,43,66,81`, `lib/dictionaries.ts:35,46`, `app/[locale]/(chrome)/{whoami,register,my-congregation}/page.tsx`). But "+ the super-admin set (people, role grants, unit admin, taxa admin)" is not 3 more endpoints — replacing `oikumenea-console` needs list/create/update for persons, roles, role-permissions, assignments (grant + revoke + list), instance admins (grant + revoke + list), units, edges, and taxa. ~25 is realistic |
| Only eight capability areas are used from the SDK | **confirmed** | Exhaustive per-file sweep of every `clients/go` importer: `Authorization`, `IdentityFederation`, `Religion`, `Tenant`, `Person`, `Membership`, `Location`, `Geo`. No ninth |
| The consumer cutover map is complete | **partly — see below** | Files are right; three service-principal call sites and one adapter are unaccounted |
| The teardown checklist is complete | **refuted — see below** | 9 categories of surviving reference |

---

## Critical findings

### C1 — The PDP branches on a table the plan never ports, and nobody can become an administrator

**What breaks.** `PDP.Decide`'s first two branches read `IsInstanceAdmin` (`../go-oikumenea/internal/authorization/domain/pdp.go:82–87`), which upstream resolves from `oikumenea.authz_instance_admins` (`../go-oikumenea/migrations/0004_authz_identity.sql:132–152`). That table appears in neither the plan's `internal/authz` table list ("`authz_permissions` (Go catalog), `authz_roles`, `authz_role_permissions`, `authz_role_assignments`, `authz_epoch`") nor in `D-CorePortScope`'s ported column (`docs/architecture/decisions.md:1499`) nor in Phase 1's seed list. Without it, step 1 is dead code and step 2 denies every instance-scope action to everyone, permanently.

It compounds. `D-SuperAdminFold` says super-admin screens are "gated on a super-admin **role**" (`decisions.md:1708–1709`), but instance admin is deliberately *not* a role upstream — it is a separate authority plane, precisely because `assignment.grant` is unit-scoped and the first admin must be able to grant before any unit assignment exists (pdp.go:70–76). Collapsing it into a role means the plan needs a role that carries instance-scope permissions, which `IsInstanceScope` (`internal/authorization/domain/permissions.go:373–378`) exists to make impossible.

**The concrete scenario.** `docker compose up --build` on a clean volume, which is the migration's own headline criterion (Phase 9 item 5). Seeds create the root unit and the base roles. A human signs in with Google. JIT is link-on-match-only — "on no match, reject. **JIT never creates a person**" (`scripts/bootstrap-admin-person/main.go:4–11`) — so the login is refused. Nothing in Phase 1's seed list creates a person, an account, or an instance admin. Even if a person row were seeded, the plan deletes `scripts/bootstrap-admin-person`, so there is no shell account for the Google identity to link onto, and no instance admin to grant the first assignment. **The instance is unadministrable and unusable, and verification step 6 ("`whoami` resolves to the seeded admin person") cannot pass.**

Upstream solved this deliberately and the plan discards the solution without noticing: "The first-admin account/identity is seeded at **BOOT** (or by the recover-admin CLI) by the app on the GUC-bearing pool (D-Bootstrap / D-RIDSeeding) — **not here**" (`../go-oikumenea/migrations/0004_authz_identity.sql`, merged 0008 header). A seed migration is structurally the wrong place: the admin's email is deployment-specific and the Google `sub` is unknowable until first login.

**The trap in the obvious fix.** "Seed a shell account with a fixed email" combines badly with `D-SeedBootstrap`'s deterministic RIDs (`decisions.md:1697–1699`). A committed seed email means every deployment of this open-source repo ships the same pre-linked admin address; whoever controls that Google address (or registers the domain) is instance admin on every instance that boots the migration unmodified. `email_verified` does not help — it is verified, just not yours.

**What to change.**
1. Add `authz_instance_admins` to `D-CorePortScope`, to `internal/authz`, and to Phase 1.
2. Keep instance admin as a plane, not a role. `D-SuperAdminFold` should say "gated on the instance-admin plane"; delete the "super-admin role" phrasing.
3. Replace the deleted `bootstrap-admin-person` with a **boot-time** first-admin seed driven by config, not a migration: `BOOTSTRAP_ADMIN_EMAIL` (or an install-config field), applied idempotently at startup, a no-op once `authz_instance_admins` has any active row. Refuse to boot with the placeholder value the way upstream's guards refuse HS256.
4. Add to Phase 9: *a second, unrelated Google account signs in to a freshly seeded instance and is refused; the configured admin email signs in and is instance admin.*

---

### C2 — RLS is dropped on a rationale that upstream's own source contradicts, and the grant cache is ported without the backstop it depends on

**What breaks.** `D-InProcessAuthz` justifies dropping RLS as "duplicate a check we are already making" (`docs/architecture/decisions.md:1575`). That is not what upstream says RLS is doing. The RLS predicate is *live*: "Because the policies read the authority tables directly, the backstop is **EXACT under revocation** (stronger than the old snapshot-at-request-start GUCs)" (`../go-oikumenea/migrations/0005_document_order_rls.sql:366–368`). The app-layer check it backstops is *not* exact — the grant cache serves a subject's grants for up to `grantCacheTTL` without re-reading them, and its own package comment states the dependency in one sentence: "**The RLS backstop underneath is exact/live (D-RLSLiveReach), so a stale ALLOW cannot read revoked-away rows on RLS-guarded tables**" (`../go-oikumenea/internal/authorization/application/grantcache.go:15–17`; TTL = 2s at :44).

The plan then says of the cache: "Ported as-is; a stale grant cache is a security bug, not a performance bug" (`decisions.md:1582–1584`). Porting it as-is *is* the security bug, because "as-is" upstream includes the backstop the same decision removes.

**The concrete scenario.** A congregation-admin account is compromised or a moderator is removed for cause. An operator revokes the assignment. The revocation bumps `authz_epoch` and resets the local cache — but only if it went through the application. The existing repo already documents the other path: `scripts/bootstrap-registration-org/main.go:243–253` emits raw SQL (`INSERT INTO authz_role_assignments …; UPDATE authz_epoch SET epoch = epoch + 1`) because "go-oikumenea has no API path to grant the first assignment on a brand-new unit". Any out-of-band authority change — the same shape of manual SQL, an `UPDATE` under incident response, a migration that edits a base role's permissions (which `D-SeedBootstrap:1695` explicitly makes the normal way to change roles) — takes effect only after the TTL, with no DB-level floor underneath. In upstream that window is covered; here it is the whole story.

Second, quieter consequence: `conventions.md:64–69` records that OpenFaithMap has **no cross-schema and no cross-table foreign keys**, and `internal/{registration,content,discovery,platform}` contain **zero test files** (verified: the only tests under `internal/` are in `congregationimport`, `moderation/{domain,transport}`, `vouching/domain`, and `coreintegration`). A forgotten `WHERE unit_id = $1` in ported code is caught by nothing at all. That is exactly the bug class `authz_unit_in_reach` exists for.

**What to change.** Either keep RLS on the tables that carry per-unit authority (`directory_units`, `religion_sites`, `religion_org_profiles`, `content_sites`, `moderation_*`) — the plan is right that it needs `app.person_id`/`app.is_instance_admin` GUC plumbing on the pooled connection, which is ~150 LOC in `internal/platform/db` — **or** drop the grant cache and re-read grants per request. One join per authenticated request against `authz_role_assignments_subject_idx` at OpenFaithMap's scale is not a cost worth a correctness hole. What is not defensible is keeping the cache, dropping the backstop, and describing the pair as "duplicate".

Note also that the plan's handling of `pdp_scoped` is described backwards. `D-CorePortScope:1528–1531` says dropping it means "every unit is reach-scoped". `pdp_scoped` appears **nowhere in the PDP** — its only use is as an RLS exemption: `USING (NOT pdp_scoped OR oikumenea.authz_unit_in_reach(id, false))` (`../go-oikumenea/migrations/0005_document_order_rls.sql:463–464`, also `0011_infra.sql:807–810`). With RLS gone, *no* unit is reach-scoped at the database level. Dropping the column removes nothing because dropping RLS already removed the mechanism it modified. The decision block should say so.

---

### C3 — `SearchSites` leaks exact coordinates through the spatial predicate, not the payload

**What breaks.** The plan says: "Results return exact coordinates from SQL; the application coarsens per `public_precision` — keep that split, **it is what makes `visibility`/`public_precision` trustworthy**". The split protects the *returned value*. It does not protect the *filter*, and the filter runs on the exact geometry.

`../go-oikumenea/internal/religion/adapters/discovery.go:354–356`:

```
ST_DWithin(l.geom, <pt>, <radius>)      -- exact geometry, exact radius
orderBy = "l.geom <-> " + pt + ", s.id" -- KNN, exact distance ordering
```

`../go-oikumenea/internal/religion/application/discovery.go:251` then calls `domain.Coarsen`, which for `hidden` returns `ok=false` and omits the coordinate entirely (`internal/religion/domain/discovery.go:216–223`).

**The concrete scenario.** A site is published with `public_precision = hidden` — a house church, a congregation under harassment, exactly the case the field exists for. An anonymous caller (`GET /search` is unauthenticated — `internal/discovery/application/service.go:60–61`) issues repeated queries varying `lat/lng/radiusM`. The response omits the coordinate but *membership in the result set* is a boolean oracle on `ST_DWithin(exact_geom, point, r)`. Twenty or thirty queries binary-search the true position to metre precision. KNN ordering leaks it faster still: with `ORDER BY l.geom <-> pt` the site's rank among known-position neighbours is a direct distance comparison.

This is an upstream defect, not one the plan introduces — but the plan ports it "verbatim in shape", asserts the opposite property, and its verification step 9 checks the wrong invariant ("confirm exact coordinates never leave the process"). They don't leave the process; the answers derived from them do.

**What to change.** Coarsen in SQL for non-`exact` sites: filter and order on a rounded/snapped geometry column (a generated `geom_public` maintained alongside `geom`, snapped to the site's own precision), or exclude `public_precision = 'hidden'` from the public search arm entirely. Then rewrite verification 9 to: *a `hidden` site is not distinguishable by any sequence of radius queries*.

---

### C4 — Three system-context paths are unenumerated, and one of them is the only machine **write** in the system

**What breaks.** `D-InProcessAuthz:1585–1588` names exactly two paths that get `authz.SystemContext()`: "discovery cache refresh, `POST /exclusion-check`". A sweep of `coreintegration.NewServiceClient` call sites finds **five**:

| Path | Site | In the plan? |
|---|---|---|
| discovery cache refresh | `internal/discovery/application/service.go:53–54, 79, 123` | yes |
| moderation `CheckExclusion` | `internal/moderation/application/{service.go:48, exclusion_check.go:28}` | yes |
| `RunConnector` — the whole import loop (dedup `SearchSites`, exclusion `GetTaxon`) | `internal/congregationimport/application/service.go:90–91, 146` | **no** |
| `resolveCountryName` during geocoding | `internal/congregationimport/application/geocode.go:92` | **no** |
| `RunJurisdictionSync` — **creates real Units** | `internal/congregationimport/application/jurisdictionsync.go:75` | **no** |

The third is the dangerous omission. `RunJurisdictionSync` is documented as "the one deliberate, narrowly-scoped exception in this module to the 'go-oikumenea writes always use the human operator's own token' precedent" (`jurisdictionsync.go:40–43`), and upstream needed a **new PEP gate** to allow it at all — `pep.RequireServiceOrTarget`, added for GH-39, "deliberately NOT a straight `RequireServiceOrPerson` swap" (`jurisdictionsync.go:47–52`; the gate itself at `../go-oikumenea/internal/authorization/pep/pep.go:221–226`). The entire reason that gate exists is that machine authority and person authority must not be the same door.

**The concrete scenario.** During Phase 6, `RunJurisdictionSync` is cut over. The obvious in-process move is to drop the machine identity and run under the triggering operator's context, because the endpoint has one. That silently converts an unattended job with instance-wide standing into an operator-scoped write, and the first time an operator without `religionorg.manage` at the anchor triggers a sync, several thousand jurisdiction units are half-created and marked `FAILED` (`jurisdictionsync.go:219`). The equally obvious alternative — hand it `SystemContext()` — creates a third unaudited bypass that the decision record does not list, in the one code path that writes to `directory_units` unattended.

**What to change.** Enumerate all five in `D-InProcessAuthz`. State per path whether it is a read bypass or a write bypass. Then build `SystemContext()` so it cannot leak into a request path — see S4 below.

---

### C5 — The dev HS256 issuer has no "dev" to key on

**What breaks.** `GuardSymmetricIssuers(issuers, environment)` fails closed on any environment that is not `"local"`/`"dev"`, including empty and unknown (`../go-oikumenea/internal/identityfederation/middleware/validator.go:42–57`). It reads `environment` from upstream's install config. **OpenFaithMap has no such config value and no mechanism to obtain one.** `internal/platform/config/config.go:22–30` declares `Install` with zero fields, and its own doc comment records why: "cmd/openfaithmap-api reads every one of them straight from the environment via `requireEnv`, bypassing this type. That deviates from the witchcraft install-config convention this package exists to follow — env vars get no schema, no validation, and no ECV encryption path for the secrets among them" (`config.go:9–15`). `var/conf/install.yml` carries no environment field.

The plan's only concrete artifact is a new env var, `DEV_ISSUER_HMAC_KEY`. The natural implementation — register the HS256 issuer when the key is non-empty — makes the guard self-authorizing: the dev key is permitted because the dev key is present. There is no independent signal to refuse on.

**The concrete scenario, and why "someone would notice" is not an answer here.** `D-ProductionDeployment` states the production target is "running the existing `docker-compose.yml` stack **unmodified** as its base" and "`docker-compose.yml` itself is not rewritten by this decision — the production topology is an override/addition on top of the existing file" (`decisions.md`, D-ProductionDeployment). That file today ships `OIKUMENEA_INSECURE_SKIP_VERIFY: "true"` (`docker-compose.yml:245`) and `NODE_TLS_REJECT_UNAUTHORIZED: "0"` three times (`:311, :333, :368`), each with a DEV-ONLY comment, each inherited by any override that does not explicitly unset it. This repo's own track record is the evidence: flags marked DEV-ONLY in this file are the ones the production plan carries forward by construction. An operator who copies `.env.example`, sets a real `GOOGLE_OAUTH_CLIENT_ID`, and leaves `DEV_ISSUER_HMAC_KEY` at whatever `.env.example` ships gets a production instance where anyone holding a committed constant mints a token for **any** person RID — including the instance admin.

**What to change.**
1. Add a real `environment` field to `config.Install` (schema-validated, ECV-capable), defaulting to nothing, and make it the *only* input to the guard. Fail boot on unknown/empty exactly as upstream does.
2. Never derive "dev" from the presence of a secret.
3. Do not ship `DEV_ISSUER_HMAC_KEY` with a value in `.env.example`; ship it commented out, and make boot refuse a known placeholder.
4. Port `GuardReservedIssuer` too (`validator.go:63–70`). The plan says "two boot guards"; upstream has three, and the third exists to stop an operator pointing a real IdP at the synthetic local issuer — the exact attack this configuration invites once the dev issuer string is a constant in a public repo.

Also missing from the plan's token-verification section, all real requirements it does not mention:
- **Clock skew is applied only on the HS256 path** upstream (`jwt.WithLeeway(v.cfg.ClockSkew)` at validator.go:190); the OIDC path (`validateOIDC`, :224–243) delegates to go-oidc with no configured leeway. Whatever OFM does here should be a decision, not an accident of which branch got the option.
- **`SkipClientIDCheck: true`** (validator.go:260) is only safe because `audienceAccepted` runs unconditionally and `GuardIssuerAudience` guarantees a non-empty set. Port the trio together or not at all; porting the verifier without the guard is a silent no-op audience check.
- **JWKS caching/rotation** is entirely go-oidc's (`oidc.NewProvider` cached per issuer at :246–262) and the provider is built **lazily and never refreshed for discovery**. Worth stating so it is a known property.
- **`azp`** is projected but explicitly "NEVER an authorization input" (validator.go:130–135). If the plan's per-surface OAuth clients ever land, `aud` alone stops distinguishing the two surfaces; decide now whether that matters.
- **`nonce`** is correctly not the API's job (next-auth owns it), but the plan should say so rather than leaving it unaddressed.
- **Token lifetime.** `web/apps/admin/auth.ts:41–48` captures `account.id_token` once at sign-in and never refreshes it. Google ID tokens live one hour; the next-auth session lives far longer. Today the mismatch is masked by whatever the remote core does; after the migration `openfaithmap-api` owns the `exp` check and every admin session breaks at the one-hour mark. The plan's "next-auth already owns the browser session; only the API-side verification moves" (`decisions.md:1622–1625`) is the sentence that hides this.

---

### C6 — Phase 3 before Phase 4 works only through a port the plan never commits to

**What breaks.** The plan asserts Phases 1–5 are "purely additive... independently shippable", and orders the PDP (3) before the directory (4). That ordering is sound upstream for one reason the plan never states: the PDP depends on an interface **owned by authorization's own domain**, not on the tenant module — `ClosurePort` at `../go-oikumenea/internal/authorization/domain/authorization.go:192–207`, consumed at `pdp.go:63–68`, implemented by tenant at `internal/tenant/application/closure_port.go`. That is also how the authz ↔ directory cycle is resolved (authorization needs the closure; directory needs authorization to gate its own writes), and how upstream's PDP is testable at all — `pdp_test.go` supplies a `fakeClosure`.

The plan mentions `domain/pdp.go`, `pep/pep.go`, `scope/scope.go`, `grantcache.go` by name and LOC, and never mentions `ClosurePort`. Without the explicit commitment, the natural Phase-3 implementation imports `internal/directory` from `internal/authz`, which (a) doesn't exist yet, (b) inverts the dependency, and (c) violates `conventions.md:20–21`, which requires cross-module queries to be interface calls.

The plan also drops the second half of upstream's resolution. Because tenant/person routes are registered *before* the authz service can be constructed, upstream threads one unbound `pep.Enforcer` and `Bind`s it later — with an explicit boot assertion whose comment names the incident: "`MustBeBound` reports whether the enforcer was wired via `Bind`. The composition root calls it at boot (**review-2026-07 R-11**) so a forgotten `Bind` fails startup instead of surfacing as a request-time nil" (`../go-oikumenea/internal/authorization/pep/pep.go:56–63`). OpenFaithMap's composition root has exactly the same shape — `RegisterRoutes*` calls interleaved with service construction (`cmd/openfaithmap-api/main.go:163–383`) — so it will need the same pattern, and it is not in the plan.

**What to change.** Add to `D-InProcessAuthz`: *`internal/authz/domain` owns `ClosurePort`; `internal/directory` implements it; the composition root wires it. `internal/authz` imports no other module.* Add `MustBeBound()` (or construct authz before any route registration and delete the late binding entirely — cleaner, and OFM has fewer constraints than upstream here). Then Phase 3 is genuinely independent.

---

## Findings by axis

### Consistency

**A1 — Phase 0 is marked done but `conventions.md` was not touched, and it now contradicts three new decisions.** `git status` shows `decisions.md`, `milestones.md`, `open-questions.md`, `overview.md`, `README.md`, `docs/README.md` modified; `docs/architecture/conventions.md` is not. It still states:
- "Primary keys are plain `uuid` (`gen_random_uuid()`), **not** composed URN RIDs — decided at M3" (`conventions.md:49–62`) — directly contradicted by `D-OwnRIDs`, which ports `new_id(service, kind, type)` and per-table structural CHECKs.
- "**No RLS-based tenant isolation** … This is a **known, accepted gap** … see open-questions.md" (`:81–87`) — `DS-OFM-1` is now closed as a deliberate no.
- "Authorization for OpenFaithMap-owned tables is a target-scoped capability check **against go-oikumenea's PDP**" (`:70–76`).
- "Generated **TypeScript** does not exist — there is no codegen pipeline" (`:23–25`) — false since M7; `web/apps/{admin,web}/lib/openfaithmap/generated/` both exist.

`development-process.md:66–71` is binding on this: "Update the stage board in the same commit/PR that passes a gate — not as a follow-up." M10's Verified column explicitly requires the doc set "coherence-checked — no dangling links, no contradiction with the decisions it supersedes" (`milestones.md:98`). It is not coherent. Same for `docs/modules/core-integration.md`, `import.md`, `web-admin.md` and `glossary.md`, none of which Phase 0 names.

**A2 — `D-OwnRIDs` is internally incoherent about wire values.** It says RIDs render as `ofm:<service>:<kind>:<type>:<uuid>` at the API boundary and simultaneously that "Every existing `*_rid TEXT` column and all six existing Conjure contracts are untouched" (`decisions.md:1664–1666`). Those cannot both hold without a parse/format boundary the plan never describes. Today `content_sites.congregation_unit_rid` stores a bare uuid, the admin app round-trips that string, and `web/apps/web/app/[locale]/congregations/[unitId]/page.tsx:24` puts it in a **URL path segment** — where `ofm:4:1:1:<uuid>` needs escaping. The types are untouched; every *value* changes. Either state the boundary (parse on ingress, format on egress, in exactly one place, with the storage form normative) or drop the rendering and keep bare uuids.

**A3 — Phantom and missing entries in the cutover map.**
- `jurisdictionsync.go` is listed under congregationimport's row, but its SDK calls are behind a locally-declared `serviceClient` interface (`jurisdictionsync.go:288–297`), not the shape the row implies; `jurisdictionmatch.go` is not listed at all.
- The `discovery` row says "Authorize, SearchSites ×2" but omits that one of those two is reached from the **anonymous** public endpoint (`internal/discovery/application/service.go:60–74`) — a materially different cutover.
- `internal/congregationimport/adapters/jurisdiction_unit_store.go` is not in the map. It is the OFM-side state machine tracking `PENDING → created(real unit RID) → FAILED` for jurisdiction units (`:32, :46, :61`); its whole reason for existing is that the unit write crosses a network boundary that is about to disappear. It is not a no-op cutover.

**A4 — `scripts/mint-local-token` is used by Phase 2 and never appears in the teardown checklist.** The plan says the dev HS256 path replaces "`scripts/mint-local-token`'s issuer" (`milestones.md:100`) but the checklist deletes only `bootstrap-service-principal`. Verification 7 ("mint a second token for a person holding no grant") depends on this script continuing to work against the *new* issuer — which means the refusal proof itself runs through the dev HS256 path, i.e. the one path C5 says must not exist in production. Say explicitly which environment the refusal proof runs in.

**A5 — Nothing in any phase builds `authz_instance_admins`, the first-admin bootstrap, the shadow-visibility gate, or `religion_taxon_ranks`.** (C1, and *Clean architecture* below.)

### Clean architecture

**B1 — The seven module boundaries trace upstream's, and two of them exist only for upstream's problems.**
- `internal/authz` is asked to absorb `scope/scope.go` (192 LOC). That package's own header says what it is: "the D-VisibilityScope adapter (review-2026-09 R-30): ONE interface answering 'which of these objects may the subject read' … cross-type surfaces (**unified search** D-UnifiedSearch now, **generic link traversal** R-27 next) consume it" (`../go-oikumenea/internal/authorization/scope/scope.go:4–13`). OpenFaithMap has no unified search, no link traversal, and the plan does not port `internal/search` or `internal/links`. It is 192 LOC of pure abstraction with zero consumers on day one — the textbook "upstream idiom that no longer fits" the migration should be pruning.
- `internal/refdata` is two tables and ~200 LOC and exists as a module only because upstream had a `geo` vertical. Fold it into `internal/directory` or a `platform/refdata` package; a seven-module split where one module is a lookup table is boundary inflation.

Conversely, `internal/religion` at ~1,800 LOC estimated is doing too much: taxonomy (a global catalog), organization profiles (unit-attached), sites and schedules (location-attached), and aliases (a matching index). Those have genuinely different lifecycles here — `congregationimport` writes aliases constantly and taxa never. Upstream's single boundary was drawn when religion was one vertical among 26; it is now the product's core domain and deserves better than a trace.

**B2 — `ReachSet` and `ShadowGate` come along in `pdp.go` and neither is addressed.** `ShadowGate` (`pdp.go:251–260`) implements a real visibility property: a `shadow` unit is invisible unless in the subject's readable reach. `tenant_units.visibility` is a real column the plan's `directory_units` presumably inherits. If it inherits the column without the gate, `ListUnits` — which the admin app's jurisdiction picker calls (`web/apps/admin/lib/jurisdiction.ts:43`) — enumerates shadow units to any authenticated caller. If it drops the column, say so. The plan says neither. `ReachSet` (`:163–198`) is worse to inherit silently: it calls `DescendantUnitIDs`, which pages the *entire* subtree (`closure_port.go:38–56`, batch 1000). For the registration-operator's subtree grant at the root (`scripts/bootstrap-registration-org/main.go:243–253` grants `'subtree'` on the canonical graph at the root unit), that is every congregation in the product — tens of thousands of RIDs materialized into a map. It exists upstream to feed RLS reach, which the plan drops. Delete it explicitly.

**B3 — Everything depending on `internal/authz` is fine; everything depending on `internal/directory` is the actual risk.** Authorization as a leaf that everyone calls is how authorization works. But the plan's target state has `authz → directory` (closure), `religion → directory` (units), `membership → directory`, `content/moderation/vouching/registration/congregationimport → directory + authz`, and `directory → authz` (gating its own writes). That is one genuine cycle plus a hub. Upstream breaks it with `ClosurePort` (C6). The plan needs to state that `internal/directory` **must not import `internal/authz`** — gating happens at transport via the PEP, exactly as upstream does it — or the cycle is real and Go will refuse to compile it, discovered in Phase 6 rather than Phase 3.

**B4 — Composition-root estimate.** `cmd/openfaithmap-api/main.go` is 393 lines wiring 6 modules, of which ~140 lines are the connector/geocoder/jurisdiction-source registries (`:279–373`) and ~90 lines are prose comments. The mechanical wiring is ~120 lines for 6 modules. At 13 modules with cross-module interfaces (`contentSiteResolver` and `moderationVouchReporter` at `:62–100` are the existing pattern, ~20 lines each), plus a `ClosurePort` adapter, an auth middleware, a PEP bind + assert, and a first-admin boot seed, the honest end state is **900–1,100 lines**. The existing pattern — hand-wired, one giant `initServer`, `pool.Close()` repeated at every early return (16 times already) — does not hold at that size; the `pool.Close()` repetition is already a bug waiting to happen. Split `initServer` into per-module `register(ctx, info, deps)` functions before Phase 6, not after.

**B5 — `CreateUnitWithEdge`'s ordering is preserved for the wrong stated reason.** The plan says "Preserve `CreateUnitWithEdge`'s atomicity — that was upstream fix #36". Upstream's own comment says the ordering (closure rows seeded *before* the unit INSERT) exists because "`tenant_units_reach`'s WITH CHECK finds a subtree match on that row instead of racing an unpopulated closure — the **structural reason a genuinely-authorized non-admin person could never pass RLS** on a brand-new unit's first insert" (`../go-oikumenea/internal/tenant/application/service.go:448–452`). With RLS dropped, that motivation is gone. Keeping the ordering is harmless; recording the wrong reason for it means the next person to touch it reorders it and breaks the RLS backstop if C2 is ever fixed.

### Scalability

**D1 — The closure lock is not an advisory lock, and it serialises every unit creation in the product on one row.** The plan says "Extend/shrink run under an **advisory lock** inside the caller's transaction" and budgets `internal/platform/db`'s `advisorylock` into the ported substrate "because the closure lock needs it". Refuted. Upstream's `advisorylock.go:23–39` is a **session-level** `pg_advisory_lock` on a dedicated connection, used for **boot seeding**. The closure lock is a row lock: `SELECT id FROM oikumenea.tenant_graphs WHERE id = @graph_id **FOR NO KEY UPDATE**` (`../go-oikumenea/internal/tenant/adapters/queries/tenant.sql:536–540`), taken inside the caller's transaction and held to commit (`internal/tenant/application/service.go:417, 490, 548, 655`).

**Contention profile.** OpenFaithMap has effectively one authority-bearing graph (`canonical`). Every `CreateChildOrg`, every `AddEdge`, every jurisdiction-sync node and every candidate approval therefore serialises on **a single row of `directory_graphs`** for the duration of its whole transaction. The incremental extend itself is cheap — `anc*(parent) × desc*(child)`, one row for a leaf attach under a shallow parent (`tenant.sql:550–563`) — so the cost is not the closure maintenance, it is the serialisation window.

**What makes it dangerous is in-process-ness.** Today, `ensureUnit` and `ensureSite` are separate HTTPS calls (`internal/congregationimport/application/provision.go:173–244`), so each takes and releases the lock independently. The obvious Phase-6 refactor — "it's in-process now, let's make approval atomic" — puts `CreateChildOrg` + `CreateLocation` + `CreateSite` + classification in one transaction, holding the graph row lock across all of it. Then a single slow write blocks every other unit creation in the product. If anyone ever moves geocoding inside that transaction, the lock is held across a **1-request/second** external HTTP call (`internal/congregationimport/adapters/geocoders/nominatim/geocoder.go:17`) and the system stops.

*What to change.* State the invariant explicitly in `D-CorePortScope`: **the graph closure lock is taken as late as possible and no network call, no geocode, and no external fetch may occur while it is held.** Add a Phase 9 measurement: time a 30k-unit jurisdiction sync and record p99 lock wait.

**D2 — The anonymous public search is an in-process amplifier with no rate limit.** `internal/discovery/application/service.go:64–74`: `GET /search` (no token) falls through to `refreshFromLive` whenever the query carries a tradition/language/dayOfWeek/query filter *or* the cache is empty. `refreshFromLive` (`:76–102`) runs a live `SearchSites` and then **upserts every returned row** into `discovery_site_cache`. The only rate limiter in the binary is wired onto `ModerationPublicService` alone (`cmd/openfaithmap-api/main.go:245–255`).

Today the PostGIS work happens in a different container; on a 500MB–1GB VM after the migration it is the same process, the same connection pool, and the same memory that serves authenticated writes and the import. An unauthenticated caller issuing `GET /search?query=a&tradition=...` in a loop drives an unbounded PostGIS scan plus N upserts per request, sharing a pool with everything else. `docs/modules/hardening.md:38` already rules out Redis for cross-replica coordination, so the in-process limiter is the only tool — extend it to `DiscoveryPublicService` in Phase 6, and make the cache-fill path idempotent-or-skip rather than upsert-on-read.

**D3 — Blast radius of a runaway import.** `RunConnector` streams batches but `RunJurisdictionSync` deliberately does not: "the whole node set is fetched into memory before any write happens… 6,655 worldwide for wikidata-catholic" (`internal/congregationimport/application/jurisdictionsync.go:32–39`), plus `byExternalID`, `resolved` and `remaining` maps over the same set. That is fine at 6.6k. It is in the same 500MB process as the grant cache (`grantCacheMaxEntries = 10_000` entries, each holding a `[]ActiveGrant` with a `map[Permission]struct{}` per grant — `grantcache.go:48–55`), the pgx pool, and PostGIS result sets. There is no `restart:` policy on any compose service today (confirmed: zero matches for `restart:` in `docker-compose.yml`), so an OOM kill during an import takes the whole product down and leaves it down. `D-ProductionDeployment` schedules `restart: unless-stopped` as M9 work; this migration should not land before it, because it is the migration that makes one process's death total.

**D4 — N+1s that the HTTP cost was hiding.**
- `matchCountry` calls `ListCountries()` (249 rows, each with a locale map) **once per record** (`internal/congregationimport/application/countrymatch.go:31`), then linear-scans every locale name of every country (`findCountryMatch:44–50`). At 30,721 ua-edr records that is ~30k full catalogue fetches today — expensive enough over HTTP that it is presumably already a known cost; in-process it becomes ~30k × 249 × 4 string comparisons with no signal that anything is wrong. Hoist to one load per run, like `resolveOrgKindIDs` already does (`jurisdictionsync.go:295–308`).
- `resolveCountryName` (`geocode.go:92–105`) does the same per geocode.
- `churchSiteTypeID` lists site types per approval (`provision.go:245`).
- `web/apps/admin/app/[locale]/(chrome)/my-congregation/page.tsx:43` calls `person.getPerson` inside a loop over members. That one stays a network call after the migration and gets *worse* relative to everything else around it — add a batch `getPersons` to `core.conjure.yml` while the contract is being written.

**D5 — The trigram fix is the wrong index.** The plan: "the alias/name text arm is an un-indexed `lower(...) LIKE '%q%'` upstream… Add a `pg_trgm` GIN index on the alias text and on `directory_units.name`." Three problems with that as written:
1. The predicate is `lower(a.alias_text) LIKE $n` (`../go-oikumenea/internal/religion/adapters/discovery.go:387–392`). A GIN trgm index on the **raw column** does not serve `lower(col) LIKE …`. It must be an expression index — `CREATE INDEX … ON religion_aliases USING gin (lower(alias_text) gin_trgm_ops)` — or the query must switch to `ILIKE`.
2. The unit arm tests **two** columns, `lower(u.code) LIKE … OR lower(u.name) LIKE …` (`:391`). The plan indexes only `name`.
3. Trigram indexes cannot serve patterns shorter than 3 characters; a 1–2 character query falls back to a scan regardless.

And the diagnosis may be wrong anyway: both arms are `EXISTS` subqueries *correlated on `s.org_unit_id`*, so the planner filters by the indexed `unit_id` first and applies `LIKE` to a handful of rows. The index will likely never be chosen. The actual cost is the **outer** scan when `q.Query` is supplied without a spatial window — no `ST_DWithin`, no `ST_Intersects`, so no GiST index, and the plan degrades to a full scan of `religion_sites ⋈ religion_site_types ⋈ location_locations` with a correlated probe per row. Measure that shape before adding an index for a different one.

**D6 — Expensive to reverse if the scale assumptions are wrong.** Ranked:
1. **Dropping RLS.** Retrofitting it later means auditing every ported query for reach-correctness with no test suite. Cheapest to keep now.
2. **One binary.** Splitting later reintroduces the Conjure contract, the token plumbing and the deployment the plan just deleted.
3. **Deterministic seeded RIDs.** Cheap to reverse mechanically, but every test and constant that references them has to change with them.
4. **The graph row lock.** Reversible — replace with a per-subtree lock or an advisory lock keyed on the parent — but only if the code that takes it is not spread across five call sites first.
5. **Dropping the audit log.** Cheap to add; expensive that the data for the intervening period does not exist.

### Security

**S1 — The removed meta-check: interrogated.** The plan's argument is largely right and I could not find a case where removing it narrows nothing that matters — but the *reasoning* is right for the wrong reason, and the replacement signature reintroduces the risk.

What is actually true: at all five call sites, `callerPersonID` is derived from `Whoami` on the caller's own forwarded token (`internal/content/transport/service.go:46–58`, and the identical shape in moderation/vouching/discovery/registration/congregationimport transports). So `req.SubjectPersonId` and the PEP's context subject are always the same person, and requiring `assignment.read` at the target adds nothing that `religionorg.manage`/`unit.lifecycle` at the same target does not already require. All three roles bundle `assignment.read` with the permission being tested (`scripts/bootstrap-registration-org/main.go:141, 203, 227`), so no reach differs.

What the plan misses: the meta-check is the **only** thing binding the *answer* to the *authenticated caller*. Upstream's transport gate reads the subject from context (`pep.Require` → `authn.PersonID(ctx)`, `../go-oikumenea/internal/authorization/pep/pep.go:69, 73–85`) while `Decide` takes the subject as a parameter (`internal/authorization/transport/service.go:58–62`). Remove the gate and `Authorize` becomes an oracle over arbitrary subjects, safe only by call-site discipline. The plan's proposed in-process signature — "`Authorize(subject, action, unit)` is a pure function of the subject" — **preserves the parameter and drops the binding**. That is the same defect *class* as the two the repo already fixed: M2.3's `IsOperator` was checking untargeted `MyCapabilities` — "answers 'does the caller hold this permission *anywhere*'" (`internal/registration/application/service.go:97–101`) — a check answering a broader question than the call site needed. A subject-parameter `Authorize` is a check answering a question about someone the call site did not authenticate.

*What to change.* Make the module-facing entry point `authz.Require(ctx, action, unitID)`, subject from context, exactly as upstream's PEP does. Expose the subject-parameter form only as `authz.DecideFor(ctx, subject, …)`, used by nothing except the super-admin "what can this person do" screen, and gate *that* on the instance-admin plane. Then the removal is genuinely safe, not safe-by-convention.

**S2 — Super-admin folds into an app with no server-side role gating at all.** `web/apps/admin/app/[locale]/admin/layout.tsx:11–15` states it in its own comment: "This only removes duplicated 'is anyone logged in' boilerplate — **it adds no role/permission gate**. Every mutation still relies exclusively on go-oikumenea's live PDP." That is the entire existing pattern: session-exists in the app, authority in the API. `D-SuperAdminFold:1738–1740` requires "Super-admin screens must be role-gated server-side, not merely hidden in navigation", but there is no server-side role-gating mechanism in this app to follow, and the plan does not build one.

*The concrete scenario.* A congregation-admin session hits `/admin/people` or `/admin/roles` directly. The layout admits them (session exists). The page renders shells and fires server-side reads. Whether anything leaks depends entirely on whether every one of the ~25 new `core.conjure.yml` endpoints has its own PEP gate — and the plan's permission catalog ("`religionorg.manage`, `moderation.{read,act}`, `assignment.grant`, `unit.{read,create}`, `religion.read`, `country.read`, plus the super-admin set") never names a single instance-scope permission, and C1 shows the instance-admin plane does not exist at all.

*What to change.* Add a server-side capability check to the admin app — one `requireInstanceAdmin()` helper consulting a `MyCapabilities`-style endpoint (`../go-oikumenea/internal/authorization/transport/service.go:92–110` is the model, including its explicit "cosmetic UI gating only; the PDP still re-decides every guarded operation" caveat), called in the super-admin route group's layout. Then keep the API gate as the real one. Two gates, both required, and Phase 9's item 11 proves both.

**S3 — Deterministic seeded RIDs: what they enable.** The genuine risks are narrower than they look but not zero.
- Structural RIDs (root unit, base roles, `canonical` graph, org kinds, site types) as public constants are fine — they name rows, not secrets, and every check is still an assignment lookup. `D-SeedBootstrap:1697–1699` is correct on this.
- They become an **enumeration aid**: `conventions.md:64–69` guarantees no referential integrity across modules, so an IDOR in any endpoint taking a `unit_rid` is now exploitable with a *known-good* root/anchor RID rather than a guessed one. Every `*_rid TEXT` parameter must be authorized, not merely well-formed — the structural CHECKs `D-OwnRIDs` keeps catch type confusion, never authority.
- The one that actually matters is C1's: a **seeded admin identity** with a fixed RID *and* a fixed email is a universal backdoor across every deployment. Structural RIDs, yes. Identity RIDs, no.

**S4 — `SystemContext()` must be unforgeable, and the plan gives it no shape.** Requirements before I would trust it:
1. It is a **private** type in `internal/authz` with an unexported key, constructed only by `authz.SystemContext(parent)`; no other package can synthesise it.
2. The authentication middleware **strips** any system marker from every inbound request context before dispatch, unconditionally. Upstream's equivalent defends the mirror case explicitly rather than relying on a value happening to be empty: "Stated explicitly rather than relying on `PersonID` happening to be empty, so a future change to `Subject` cannot silently promote a principal" (`../go-oikumenea/internal/authorization/pep/pep.go:122–124`). Copy that instinct.
3. `authz.Require` panics — not denies, panics — if it sees a system context on a request-scoped context, so the failure is loud in test and in dev.
4. One `go vet`-able or lint-enforced rule: `SystemContext` may be called only from a named allowlist of files. There are five call sites (C4); an allowlist of five is maintainable and a grep is not.
5. Phase 9 proves it: *a request carrying every header and body field an attacker controls does not reach a `SystemContext` path.*

**S5 — Cache invalidation as a security control: the race is real but small at one replica.** The epoch protocol is correct as written — miss reads the epoch **first**, then fetches, so "a concurrent bump makes the stored entry conservatively stale, never wrongly fresh" (`grantcache.go:9–12`, implemented at `:117–135`), with `singleflight` for stampede control. In a single-process deployment, a mutation through the application resets the local map after commit (`:93–100`) and the window closes immediately. The exposure is: (a) authority changes made **outside** the application — raw SQL, migrations editing base roles (`D-SeedBootstrap:1695` makes this the normal path), incident-response `UPDATE`s — which bump the epoch but do not reset the local cache, giving a 2-second stale-ALLOW window with no RLS floor (C2); and (b) any future second replica, where the bound is TTL for every process that did not perform the write. The machinery (epoch table, singleflight, five metrics) is sized for a multi-replica world this deployment does not have. Either keep it and document the 2-second bound as a security property in `D-InProcessAuthz`, or drop it — one indexed join per request is not the bottleneck here.

**S6 — JIT and `email_verified`.** `D-DirectTokenVerification:1634–1635` keeps the requirement, and upstream's implementation is sound: `email_verified` must be present **and** true, accepting both the JSON bool and the string `"true"` (`../go-oikumenea/internal/identityfederation/middleware/validator.go:125–128, 274–279`), and it is load-bearing only on the attribute arm. Two things the plan should nail down because C1's fix will touch them:
- Email matching must be **exact and normalised identically at write and read**. Google treats `foo@gmail.com`, `f.o.o@gmail.com` and `foo+x@gmail.com` as one mailbox but returns the address as registered; a seeded shell account matching on a raw string is a near-miss away from silently not linking, and a case-insensitive-but-not-dot-insensitive match is the kind of thing that is either safe or a takeover depending on details nobody wrote down. Upstream uses `citext` for `account_accounts.email` (`migrations/0004_authz_identity.sql`, "Depends on 0001 bootstrap … citext"); port that, and state the normalisation.
- Once linked, the `(issuer, subject)` pair — not the email — must be the identity key thereafter, so a later email change at Google cannot re-target the link.

**S7 — Token replay.** Bearer ID tokens are replayable for their full lifetime by anyone who observes one; today they cross the compose network to `oikumenea-app` over TLS-with-verification-disabled (`docker-compose.yml:245`). The migration *improves* this by deleting the hop, which is worth stating as a benefit. What it does not address: `openfaithmap-api` will accept a valid Google ID token from any client on the network, and there is no `jti` replay cache and no binding to a session. That is an acceptable posture for this product; it should be written down as one, in `D-DirectTokenVerification`, rather than left implicit.

---

## Unaccounted-for references

Exhaustive sweep (`grep -ril oikumenea`, all casings, excluding `.git`/`node_modules`): **138 files**. The plan's teardown checklist and cutover map cover the load-bearing ones. These are not covered:

| # | Reference | Where | Why it matters |
|---|---|---|---|
| 1 | `web/apps/admin/lib/oikumenea.ts` | whole file (`:8, 12–15, 26–29`) | The client factory itself. The plan names `lib/jurisdiction.ts`, `lib/dictionaries.ts`, "four page components" and `package.json` — not the module they all import. Deleting it is the actual cutover. |
| 2 | User-visible i18n strings naming "go-oikumenea" | `web/apps/admin/messages/en.json:11, 12, 73, 263` and the `es`/`pt`/`uk` equivalents | Shipped UI copy: *"Who am I (via go-oikumenea)"*, *"Resolved by go-oikumenea's `identityFederation.whoami()`"*, *"a resumable, two-step move on go-oikumenea's side"*, *"Free-text hint → go-oikumenea jurisdiction unit RID"*. Not mentioned anywhere in the plan. Four locale files × 4 strings. |
| 3 | `web/apps/admin/.env.example` | `:6, 12, 13, 15, 22, 26` — `OIKUMENEA_BASE_URL`, `NODE_TLS_REJECT_UNAUTHORIZED` guidance | The checklist says "Update `.env.example`" (singular). There are three. |
| 4 | `web/apps/web/.env.example` | `:4` | Third one. |
| 5 | `web/apps/admin/package-lock.json` | `oikumenea-client` and its transitive tree | Must be regenerated in the same commit as `package.json` or CI's `npm ci` fails. |
| 6 | `web/apps/admin/Dockerfile:2`, `web/apps/web/Dockerfile:2` | build comments naming the dependency | Cosmetic, but `web/apps/admin/Dockerfile` documents the dependency set the build assumes. |
| 7 | `var/conf/install.yml:7–13` | port map naming `oikumenea-console (3003)` | The install config is baked into the image; this is where C5's `environment` field has to land anyway, so fix both together. |
| 8 | `api/*.conjure.yml` (all six) → `internal/conjure/openfaithmap/**` (14 generated Go files) → `web/apps/{admin,web}/lib/openfaithmap/generated/**` (30 generated TS files) | doc-comment text referencing oikumenea RIDs/units | `D-OwnRIDs:1664` says all six contracts are "untouched". Their **documentation strings** describe a system that will not exist, and they propagate into three copies of generated code that Phase 7 regenerates anyway. Fix the six YAML files in Phase 7 or the stale text is permanent. |
| 9 | Docs not in Phase 0's scope | `docs/architecture/conventions.md` (see A1), `docs/modules/core-integration.md` (the module being deleted), `docs/modules/import.md` (hermenea), `docs/modules/web-admin.md`, `docs/glossary.md`, `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/feature_request.md`, `NOTICE`, `web/apps/{admin,web}/README.md` | Phase 0 names only `README.md` and `docs/architecture/overview.md`. M10's own Verified gate demands a coherent doc set. |
| 10 | `scripts/rewrite-ir-packages.mjs`, `tools/conjure-ir-dump/main.go`, `scripts/gen-ts-client.sh` | codegen pipeline | Phase 7 wires `make sdk-verify` into CI and regenerates three copies. These three files are the pipeline; they are not in the checklist. |
| 11 | `migrations/0003_least_privilege_role.sql:4–10` | the assertion this migration exists to make | Plan says the assertion "becomes moot; drop the schema". The migration also documents `openfaithmap_app` as mirroring `oikumenea_app` (`:6, :16`). Since migrations are expand-only (`development-process.md:44–47`), "drop the schema" needs its own contract-phase migration, not a deletion. |
| 12 | `atlas.hcl:1, 3, 20` | `--revisions-schema` rationale | Compose's `oikumenea-migrate` uses `--revisions-schema oikumenea` (`docker-compose.yml:62`); once that service is gone the comment is wrong and the revisions-schema choice should be re-stated. |

Known false positive, correctly flagged by the plan and confirmed: `connector.Fetch/Normalize/Code` in `internal/congregationimport/application/service.go:158` is a local variable. Left alone.

---

## What the plan gets right

Do not weaken these while fixing the above.

1. **The decision to absorb the core.** The three costs in `D-OwnCore:1427–1439` are real and verifiable — every authenticated request makes two round-trips, six upstream issues gated this project's own milestones, three artifact channels version independently. A facade that makes an authorization decision a network call is a facade whose availability is worse than its dependency's.
2. **Keeping the hierarchy and the closure table.** `D-CorePortScope:1515–1525` is right: `D-JurisdictionUnits` needs variable-depth chains, subtree grants need ancestor reach, and the Catholic sync builds real trees. Flattening reinvents the graph worse.
3. **Keeping the PDP in-memory rather than pushing it into SQL** (`decisions.md:1559–1563`). The engine is a pure function; that is what makes the Phase-9 test matrix expressible at all, and it is why C6's `ClosurePort` fix is cheap.
4. **Keeping `Decision.Via []Contribution`.** `decisions.md:1579–1581` — decision-explain is the difference between "403" and a debuggable 403. Upstream's own `pdp.go:32–40` carries it for the same reason.
5. **Keeping the permission catalog closed and in Go** (`decisions.md:1565–1568`). The compiler as the integrity check is correct, and it is why the catalog must be *extended* to include the instance-scope set C1 needs, not abandoned.
6. **Deleting the service-account key file.** Removing a long-lived private key from the deployment is a net security win independent of everything else (`decisions.md:1628–1630`).
7. **Deleting hermenea.** Verified end to end: it supplied `code + name` into a table that already had 249 static rows. 4,621 LOC, a second database, a cron, five outbound fetches, two placeholder shared secrets — for nothing OpenFaithMap reads.
8. **Insisting the refusal proof uses a second, denied token** (Phase 9 items 7 and 11, per `development-process.md:60–64`). That rule was written because M2 shipped three defects behind a green happy path. It is the single most valuable thing in the verification section.
9. **Refusing to also collapse `discovery_site_cache`.** Correctly scoped out. It would be the fourth thing changing under the one feature with the most manual verification debt.
10. **Wiring `make sdk-verify` into CI.** Overdue independently of this migration, and mandatory once three copies of generated code move together.

---

## Could not assess

| Item | What would resolve it |
|---|---|
| Whether ~7–8k LOC is reachable after trimming, or whether ~12–15k is the floor | A file-level port manifest for `internal/authz` and `internal/directory`: which upstream files come across, which functions are deleted from each. Two modules is enough to calibrate the other five. |
| Whether the `LIKE` arm's real cost is the outer scan or the correlated probe (D5) | `EXPLAIN (ANALYZE, BUFFERS)` for `SearchSites` with `q.Query` set and no spatial window, at ~30k sites. Capture it before Phase 6, alongside the search-parity baseline Phase 9 item 9 already requires. |
| Contention profile of the graph row lock at import scale (D1) | Time a full 6,655-node `wikidata-catholic` sync against the current stack and record p99 transaction duration. That is the pre-migration baseline; without it, "it got slower" is unfalsifiable. |
| Whether `directory_units` keeps `visibility`, and whether the shadow gate is wired (B2) | One line in `D-CorePortScope`'s ported/dropped table. |
| Whether the four locale packs' country names survive the port byte-for-byte | `matchCountry` does exact-string comparison against **every** locale name (`internal/congregationimport/application/countrymatch.go:44–50`), and the osm connector's `CountryHint` is built to match them exactly (`adapters/connectors/osm/connector.go:115–127`). Phase 9's ua-edr re-run exercises only Ukraine. A diff of `refdata_country_names` against the pre-migration `ListCountries()` response, all four locales, 249 rows, would settle it. |
| Realistic probability that a subtle authorization regression ships silently | Given zero tests in `internal/{registration,content,discovery,platform}`, no e2e suite, no dual path, no feature flags, and a single straight-line branch: **high**. The plan's own Phase 9 items 7 and 11 are the right shape but are manual, one-shot, and cover two roles against three surfaces. The concrete check that would catch it is not "more tests" — it is **one table-driven test that enumerates the full cross-product**: {anonymous, congregation-admin@ownUnit, congregation-admin@otherUnit, registration-operator, platform-moderator, instance-admin} × {every guarded endpoint} → {allow, deny}, asserted against a fixture instance built from the Phase-1 seeds. That is ~6 × ~40 = 240 assertions, expressible only because `D-SeedBootstrap` makes the RIDs constants — which is the strongest argument for that decision the plan does not make. Make it a Phase-9 gate, not a Phase-3 nicety. |
