# Migration plan review #1

Reviewed: `~/.claude/plans/the-goal-is-write-adaptive-engelbart.md` against `0fc2b58` (2026-08-17)
Date: 2026-08-17
Verdict: The migration is the right call and the docs phase (M10, applied to this repo) is sound, but
phases 1–9 as written are **not safe to execute as-is**. One finding is a real, unaddressed
platform-wide-authority gap (#1 below); the rest are scope/estimate gaps that will surface as
schedule or security surprises mid-build, not on day one. Fix #1 and #2 before writing code; the
others can be fixed in the phase they land in, but should be decided now, in the doc, not discovered
during implementation.

## Verification of the plan's factual claims

| Claim | Verdict | Evidence |
|---|---|---|
| PDP is ~260 lines of pure in-memory Go, only I/O is two closure point lookups | **Confirmed** | `go-oikumenea/internal/authorization/domain/pdp.go` is exactly 260 lines; `Decide` calls `p.closure.IsAuthorityBearing` and `p.closure.IsAncestorOrSelf` only (pdp.go:108,120), both single indexed lookups (`tenant/application/closure_port.go:21-32`) |
| `geo_countries` is already a static 249-row seed upstream | **Confirmed** | `go-oikumenea/migrations/0001_platform_core.sql:236` `INSERT INTO oikumenea.geo_countries` — counted exactly 249 rows |
| `membership` needs 2 tables | **Confirmed** | Exactly `membership_memberships`, `membership_positions` exist upstream (grep across `go-oikumenea/migrations/*.sql`) |
| `person` needs exactly 1 table (`person_persons`) of ~50 | **Confirmed** | Exactly 50 `person_*` `CREATE TABLE` statements exist upstream |
| 13 of 22 `religion_*` tables are needed | **Partly refuted** | 22 `religion_*` tables confirmed, but the plan's own prose (`architecture/decisions.md:1497` D-CorePortScope table) names only 6 as dropped (4 clergy + 2 affiliation) and describes "13" kept without listing them precisely; literal counting leaves 3 tables — `religion_classifications`, `religion_taxon_ranks`, `religion_unit_classifications` — named in **neither** list. See Findings → Consistency. |
| go-oikumenea is ~267k LOC, dominated by generated/unused code | **Partly refuted** | Actual checkout at `../go-oikumenea` (commit `76388e9`) is **378,480** total Go LOC / 337,634 non-test — 26–42% higher than the plan's own headline figure. The *qualitative* claim holds (`internal/conjure` 87,921 + `clients/go` 70,166 + `*.sql.go`/`models.go` 87,793 ≈ 245,880 LOC, ~65% of the total, is generated/duplicated), but the raw number repeated in `architecture/decisions.md:1441` and `milestones.md:2441` was not re-verified against the dependency's actual current state. |
| ~7–8k LOC Go + ~1.5k LOC migrations is defensible | **Plausible, internally consistent, but built on the two claims above** | Summing the plan's own per-module estimates (`architecture/decisions.md:61-73`) gives ≈7,900 LOC, self-consistent with the headline — but it inherits the religion-table undercount and (see next row) an undercounted Conjure surface, so it is optimistic rather than defensible as written. |
| Conjure transport generated only for ~12 endpoints is "the single biggest reason the port is small" | **Refuted as stated** | The ~12 figure (`api/core.conjure.yml` section, plan) is explicitly scoped to the 9 operations `openfaithmap-admin` calls oikumenea for **today** (`web/apps/admin/lib/jurisdiction.ts`, `lib/dictionaries.ts`). Phase 8 (D-SuperAdminFold) additionally requires new Conjure-served endpoints for people/role-grant/unit-admin/taxa-admin CRUD — capability `oikumenea-console` gets today via its **own**, separately-published SDK usage, never counted in the 9-operation admin-app survey. See Findings → Consistency. |
| The consumer cutover map is complete | **Partly refuted** | Registration/content/moderation/vouching/discovery rows verified call-for-call against actual SDK usage (exact match, no gaps). `congregationimport`'s row is **incomplete**: it does not distinguish `provision.go`'s human-forwarded-token writes from `jurisdictionsync.go`'s **service-principal-token** writes (`jurisdictionsync.go:41-53,75`), a materially different authorization shape the map collapses into one generic "CreateChildOrg." See Critical Finding #1. |
| The teardown checklist is complete | **Confirmed for infra/config; not verified for prose** | `docker-compose.yml`'s 7 oikumenea/hermenea services, `.env.example`'s 9 vars, `go.mod`, `package.json` all match the checklist exactly. An exhaustive case-insensitive sweep for `oikumenea\|hermenea` also surfaced ~40 files of **doc-comment prose** (Conjure YAML `docs:` strings, generated Go/TS doc comments) describing behavior in terms of go-oikumenea that the checklist doesn't mention — cosmetic, not functional, but real staleness after the cutover. See Unaccounted-for references. |

## Critical findings

### 1. A real, unaddressed platform-wide-authority gap: `RunJurisdictionSync` has no operator gate today, and the plan's SystemContext bucket is the wrong tool to inherit it with

**What breaks.** `POST /congregation-import/v1/jurisdiction-sync/runs` creates **real** go-oikumenea
Units via `CreateChildOrg` (`internal/congregationimport/application/jurisdictionsync.go:75-82`, using
`s.serviceClient(ctx)` — the **service principal's own instance-wide `religionorg.manage` grant**,
not the caller's reach). Its transport handler only resolves `whoami` — it never calls
`requireOperator`:

```
// internal/congregationimport/transport/service.go:209-213
func (s *Service) RunJurisdictionSync(ctx context.Context, authHeader bearertoken.Token, requestArg ...) (...) {
    if _, err := s.whoami(ctx, authHeader); err != nil { ... }
    summary, err := s.appService.RunJurisdictionSync(ctx, requestArg.SourceCode)
```

Compare `RunConnector`, which has the *same* shape but is explicitly justified as safe because it
makes no write (`transport/service.go:63-66`: "RunConnector itself has no operator gate ... it makes
no go-oikumenea WRITE"). `RunJurisdictionSync` copies the pattern but **does** write, and the
justification doesn't carry over — nothing in `jurisdictionsync.go` calls `requireOperator` (confirmed
by grepping every call site of that function across the module: `provision.go` and `geocode.go` only).

**The concrete scenario.** Under the stated threat model (any ordinary Google account, since sign-up
is open): sign in, obtain a valid ID token, call the endpoint above. Today that succeeds and creates
real jurisdiction-tier Units — gated only by whether the *service principal* (not the caller) holds
`religionorg.manage`, which it does, instance-wide, by design (`.env.example:109-119`,
`docker-compose.yml:91-97`).

**Why the plan doesn't fix this — it doesn't know about it.** D-DirectTokenVerification and
D-InProcessAuthz's only named background/no-human-subject paths are "discovery cache refresh,
`POST /exclusion-check`" (`architecture/decisions.md:1585`, plan's own "Authentication" section). Two
outcomes are both bad, and the plan picks neither explicitly:

- If `RunJurisdictionSync`'s in-process writes are folded into `authz.SystemContext()` (the natural
  translation of "no more service principal, background work runs as system") — that construct
  **bypasses the PDP entirely** (`D-InProcessAuthz`'s own text: "SystemContext... bypasses the PDP").
  The trigger endpoint would still only require *any authenticated person*, so the migration would
  turn "any Google account, gated by an elevated-but-checked service-principal permission" into "any
  Google account, gated by nothing at all." That is a strict regression under the plan's own stated
  threat model.
- If instead a future implementer decides (correctly, but *undocumented by this plan*) that
  `RunJurisdictionSync` should require `requireOperator` like every sibling write in the same module —
  that's a legitimate fix, but it's a **behavior change the plan doesn't record**, and it isn't in
  M10.9's verification list (which names congregation content `PATCH`, the moderation queue, and
  vouching for refusal proofs — not congregationimport's `RunConnector`/`RunJurisdictionSync`).

**What to change.** Add an explicit decision (a `D-*` block or an addendum to `D-InProcessAuthz`)
naming `RunJurisdictionSync` as a **third** kind of background caller — distinct from both
"forwarded-token, self-checked" and "no-human-subject, `SystemContext`" — and state whether it becomes
operator-gated (recommended: yes, via `requireOperator`, closing the pre-existing gap for free during
the port) or stays system-context but loses its unauthenticated trigger. Either way, add it to
M10.9's refusal-proof list.

### 2. The plan's own religion-table inventory doesn't total, and the ambiguity sits on top of a real FK dependency

**What breaks.** `D-CorePortScope`'s table (`architecture/decisions.md:1497`) names 6 dropped tables
(4 clergy + 2 affiliation) and describes "13" kept, but the literal nouns in its "kept" cell —
"taxa + closure + classifications, org kinds/profiles/classifications, policies + policy kinds, sites
+ site types, schedules + service types, aliases" — map to only 13 of the **22** actual
`CREATE TABLE oikumenea.religion_*` statements (verified by grep across
`go-oikumenea/migrations/*.sql`), leaving `religion_classifications`, `religion_taxon_ranks`, and
`religion_unit_classifications` in neither the kept nor the dropped list. This isn't cosmetic:
`religion_taxon_classifications.classification_id` and `religion_unit_classifications.classification_id`
both `REFERENCES oikumenea.religion_classifications(id)` (`0008_religion.sql:193,548`) — there are
**four**, not one, differently-named "classification" tables upstream
(`religion_classifications`, `religion_org_classifications`, `religion_taxon_classifications`,
`religion_unit_classifications`), and the plan's bare word "classifications," used twice in the same
cell, doesn't say which two it means. If a migration author reads "classifications" as
`religion_taxon_classifications` while dropping `religion_classifications` (genuinely plausible from
the prose as written), the ported schema ships a dangling FK target on day one.

**The concrete scenario.** M10.1 writes the religion migration from this table as its source of
truth (the plan's own phasing table names it as such). The ambiguity isn't discovered until the
migration fails to apply or, worse, `Atlas` silently accepts a schema nobody intended because the
FK target happened to still exist from an earlier draft.

**What to change.** Replace the prose cell with an explicit, complete table list (the 13 by exact
name) before M10.1 starts, and add one line resolving whether `religion_classifications` /
`religion_taxon_classifications` / `religion_unit_classifications` / `religion_taxon_ranks` are
genuinely unused by OpenFaithMap (plausible — nothing in `internal/congregationimport` or
`internal/discovery`'s `SearchSites` port touches them) or silently load-bearing.

### 3. The "~12 endpoints" cost argument is understated once Phase 8 is included

**What breaks.** D-OwnCore's central affordability argument — "Conjure transport is generated only
for the ~12 endpoints the admin app actually calls... that is the single biggest reason the port is
small" (`architecture/decisions.md:1465-1467`) — is scoped, by the plan's own words in the API-surface
section, to what `openfaithmap-admin` calls oikumenea for **today**: 9 named operations
(`Whoami, GetPerson, GetUnit, ListUnits, UnitAncestors, ListTaxa, ListCountries, CreateChildOrg,
ListMembers`). D-SuperAdminFold then requires the **same** `api/core.conjure.yml` contract to also
serve "managing people, role grants, units and taxa" (`architecture/decisions.md:1707-1710`) —
capability that today lives entirely inside `oikumenea-console`, a separate published binary with its
own direct SDK access, and was never part of the 9-operation admin-app survey. Realistically that's
list/search-people, get/grant/revoke-assignment, list-assignments, unit create/update (beyond the
existing jurisdiction browse), and taxon create/update/delete — plausibly another 8–12 RPCs, roughly
doubling the transport surface the LOC estimate leans on.

**Why it matters.** This isn't pedantry about a number — it's the plan's own stated reason the port
is affordable at all ("the single biggest reason"). An estimate built on undercounting its own biggest
cost driver is the kind of thing that turns into a mid-Phase-7 schedule surprise, not a Phase-9 one.

**What to change.** Enumerate the Phase 8 super-admin endpoint set explicitly (even roughly) and fold
its transport LOC into the estimate before using "~12 endpoints" as a headline number anywhere else.

## Findings by axis

### Consistency

- **Phase ordering / the PDP↔closure dependency is sound only via a pattern the plan never names.**
  Upstream resolves the exact cycle the review brief for this document was worried about — authz
  needs the hierarchy's closure, the hierarchy needs authz to gate its own writes — via a
  domain-owned `ClosurePort` interface (`go-oikumenea/internal/authorization/domain/authorization.go:195`)
  that `tenant/application.Service` satisfies structurally, wired at the composition root
  (`NewPDP(closure ClosurePort)`, `pdp.go:68`). Because `pdp.go` is ported "freely" and near-verbatim
  (confirmed above, 260 lines unchanged), this dependency-inversion pattern comes along *by accident of
  what gets copied*, not because the plan commits to it. Nowhere does the plan's phasing table or design
  section use the words "port," "interface," or name this pattern — a plan-reader who doesn't
  independently go read `closure_port.go` could build Phase 3 against a concrete `internal/directory`
  type instead, which would break the "Phases 1–5 are purely additive" claim the plan makes explicitly.
  Low effort to fix (name the interface, say Phase 3 defines it and Phase 4 satisfies it), real risk if
  unfixed given the codebase's existing convention of hexagonal `domain` → `adapters` boundaries is
  already followed inconsistently at the seams (see `internal/coreintegration` itself, which is a
  straight SDK wrapper, not a port).
- **`religion` table inventory** — see Critical Finding #2.
- **"~12 endpoints" cost claim** — see Critical Finding #3.
- **The plan is silent on where `internal/discovery`'s existing `haversineMeters` client-side radius
  filter (`internal/discovery/application/service.go:190-199`) goes.** That code exists specifically
  because `D-Facade` says OpenFaithMap owns no spatial index of its own — a premise D-OwnCore/
  `D-CorePortScope` directly reverses (PostGIS + `geography` columns move in-process). The plan
  correctly ports `SearchSites` for the *live* discovery path, but `discovery_site_cache`'s own
  cache-hit path still uses the haversine fallback and is explicitly named "out of scope" for
  collapsing (`architecture/decisions.md:1783-1784`, "out of scope" section). That's a legitimate,
  named deferral — flagging only because the plan should say explicitly that the redundant,
  now-provably-inferior haversine code path is *deliberately* kept running post-migration, not
  quietly forgotten.

### Clean architecture

- **The cycle the plan needs to resolve deliberately (authz ↔ directory) is resolved correctly in
  upstream and inheritable — see Consistency above.** No `religion ↔ directory` or `identity ↔ authz`
  cycle was found in the upstream package graph for the ported subset; `religion`'s `SearchSites`
  reads `tenant_units`/`religion_taxa_closure` one-directionally, and `identity` (person/account) has
  no reverse dependency from `authz`.
- **Seven modules is the right count for *this* product, not a trace of upstream's 26-vertical
  boundaries** — the plan's own table (`architecture/decisions.md:61-73`) is a genuine re-derivation
  (e.g. `internal/person` explicitly "rewritten (~500 LOC), not lifted," confirmed: upstream's real
  `internal/person` is 12,068 LOC across ~50 tables, D-CorePortScope keeps exactly 1). This is the one
  place the plan visibly did the harder work of *not* just tracing upstream.
- **Composition root honesty check.** `cmd/openfaithmap-api/main.go` is 394 lines wiring 6 modules
  today (verified: `main.go:1-394`). Adding `identity`, `authz`, `directory`, `religion`, `location`,
  `membership`, `refdata` — 7 more, each needing a store + app service + (for identity/authz) a
  cross-cutting middleware wire — is a believable ~2–3x growth, not the "13 modules with cross-module
  interfaces" ceiling the review brief was probing for. The existing pattern (flat `NewService`
  construction calls, one `RegisterRoutes*` per module, comment-annotated) will hold as literal Go
  structure; whether it stays *readable* at that size is a judgment call, not a defect — flag as
  "watch, don't block."
- **Dropping RLS is defensible and the plan's own reasoning for it is correct, not merely asserted.**
  Verified the actual predicate: `authz_unit_in_reach` (`go-oikumenea/migrations/0011_infra.sql:770-812`)
  reads only `authz_role_assignments`, `authz_roles`, `authz_role_permissions`, `tenant_graphs` — all
  tables `D-CorePortScope` ports — confirming `D-InProcessAuthz`'s own claim ("it *would* port
  cleanly... the reason not to is [GUC plumbing cost]") is accurate, not a rationalization. This is a
  place the plan is honest about a real, present option it's declining, with the actual evidence to
  back the decline. Credit where due.

### Scalability

- **The trigram-index fix for `SearchSites`'s text arm is incomplete.** The actual query
  (`go-oikumenea/internal/religion/adapters/discovery.go:385-391`) does
  `lower(a.alias_text) LIKE '%q%' OR lower(u.code) LIKE '%q%' OR lower(u.name) LIKE '%q%'` — three
  disjuncts, on two different tables. The plan's fix names only "a `pg_trgm` GIN index on the alias
  text and on `directory_units.name`" — `directory_units.code` (the ported name for `tenant_units.code`)
  is left out of the same OR clause it's fixing, so a code-only search term keeps whatever plan
  Postgres was choosing before the fix. Low effort (one more `CREATE INDEX ... USING gin (code
  gin_trgm_ops)`), but it's exactly the kind of thing a "fix while porting" aside is likely to miss if
  not spelled out.
- **Closure-maintenance contention under the Catholic jurisdiction sync is a real, named risk the plan
  correctly flags but doesn't size.** `jurisdictionsync.go` buffers the *entire* source (6,655 nodes
  worldwide, `.env.example:105`) then does topological-order `CreateChildOrg` calls one at a time
  (`jurisdictionsync.go:95-100` — a `for len(remaining) > 0` pass-based loop, not a single transaction).
  In-process, each `CreateChildOrg` extends the closure under an advisory lock inside its own
  transaction (`architecture/decisions.md:108` design detail). On a single small VM, thousands of
  sequential single-row closure extensions under an advisory lock, contending with ordinary user
  traffic on the same lock namespace, is a plausible new latency source that didn't exist when this
  was a separate network hop with its own connection pool. The plan should say whether jurisdiction
  syncs are expected to run during low-traffic windows, since nothing currently enforces that.
- **Single-process blast radius is asymmetric, not just "shared."** A runaway `RunConnector` import
  (ua-edr's real 30,721-record ground truth) and public anonymous map traffic now share one
  Postgres connection pool and one Go process's memory on a ~500MB–1GB VM. The plan's own
  `UAEDR_SOURCE_URL` streaming design (`main.go:280-296` comment) shows this was already thought
  through for *network* memory pressure; it says nothing about *database connection* contention
  between a large import's writes and public read traffic once both go through the same
  `pgxpool.Pool` with no separate pool or priority lane. Worth a one-line decision either way.

### Security

- **Critical Finding #1** (`RunJurisdictionSync`) is the headline item; see above.
- **The removed `Authorize` meta-check is correct for every call site that exists today, verified
  exhaustively** — every `SubjectPersonId:` argument to `Authorization.Authorize` across this
  repo (`content/application/authorize.go:32`, `moderation/application/authorize.go:37,67`,
  `vouching/application/authorize.go:33,65`, `registration/application/service.go:116`,
  `congregationimport/application/provision.go:36`, `discovery/application/service.go:158`) is the
  same `callerPersonID` used to build the client — i.e. every call is a self-check, and the upstream
  gate (`go-oikumenea/internal/authorization/transport/service.go:60`,
  `s.pep.Require(ctx, token, PermAssignmentRead, unitID)`, checked against the **context-resolved
  caller**, not `req.SubjectPersonId`) is provably redundant for this codebase's actual usage. This is
  one of the plan's stronger, better-evidenced arguments — not just asserted, independently
  reproducible from the code.
  **But** the in-process replacement, `Authorize(subject, action, unit)`, is a raw function with no
  structural enforcement that `subject == the actual caller`. The removed remote gate was an
  independent second check that didn't trust the caller to have gotten the self-check convention
  right; the in-process version trusts every future call site to keep doing so, forever, with zero
  tests in the modules that call it (`internal/{registration,content,discovery,platform}` confirmed
  zero `*_test.go` files) and a straight-line, no-rollback cutover. Recommend: make the in-process
  signature take the caller's identity from `ctx` (mirroring upstream's own `pep.Subject(ctx)`,
  `go-oikumenea/internal/authorization/pep/pep.go:69`) rather than a caller-supplied string, so a
  future accidental non-self `Authorize` call is a compile-time impossibility rather than a silently
  unchecked one.
- **Token verification: the plan names two guards; the upstream file it's porting has more than two.**
  Verified `go-oikumenea/internal/identityfederation/middleware/validator.go` also does clock-skew
  leeway (`jwt.WithLeeway`, line 190), HS256 algorithm pinning (`jwt.WithValidMethods`, line 188),
  multi-audience matching (`audienceAccepted`, lines 210-222), and lazy per-issuer JWKS
  caching/rotation via `go-oidc`'s `oidc.NewProvider` (lines 246-263) — none named in
  `D-DirectTokenVerification` or in M10.9's token-verification test list (`milestones.md`:
  "wrong audience refused, HS256 refused when not in dev, unknown `(issuer, subject)` refused,
  `email_verified: false` refused"). If `validator.go`/`authenticator.go` are ported wholesale this is
  moot — the machinery comes along regardless of whether the plan's prose names it. The risk is
  specifically the combination of decision 5 ("port freely... **simplifying as code lands**") with a
  test list that would not catch an algorithm-confusion or clock-skew regression introduced during
  that simplification. One sentence in the plan — "port `validator.go` and `authenticator.go` in full,
  not just the two named guards" — closes this for free.
- **`RunAsSystem`/`SystemContext` boundary is under-specified, and Critical Finding #1 is the concrete
  case where that under-specification bites.** Upstream's `RunAsSystem`
  (`go-oikumenea/internal/platform/db/rls.go:75-87`) is a narrow, named, three-call-site construct
  (first-admin bootstrap, recover-admin CLI, purge-erase subscriber). The plan's equivalent
  (`authz.SystemContext()`) is described only by the two examples already discussed
  (`architecture/decisions.md:1585-1588`) with no closed list of call sites and no stated review bar
  for adding a new one — exactly the gap that makes it possible for `RunJurisdictionSync` to fall into
  it by default rather than by decision.
- **Deterministic seeded RIDs are correctly reasoned about, not a hidden hole.** `D-SeedBootstrap`'s
  own "genuine trade-off" paragraph (`architecture/decisions.md:1697-1699`) already states the actual
  risk (identical structural RIDs across every deployment, revisit if multi-instance federation ever
  happens) — this is the plan being honest about a real property rather than hiding it, and the
  property itself is benign for a single-instance product with no secrets in a RID.
- **Dropping RLS (`D-InProcessAuthz`) is a defensible app-layer-only choice, not a hole** — see Clean
  architecture above; the predicate genuinely would have ported cleanly onto the kept tables, and the
  plan says so with evidence rather than asserting it.
- **The privilege-boundary question (D-SuperAdminFold) is the one place the plan visibly anticipates
  its own risk and commits to closing it, but with no mechanism specified yet.** Verified the *current*
  pattern: `web/apps/admin/app/[locale]/admin/layout.tsx:11-15`'s own comment states plainly that the
  layout "adds no role/permission gate... every mutation still relies exclusively on go-oikumenea's
  live PDP" — i.e., today, zero page-level authorization exists in the Next.js app; every check is a
  per-call backend gate. `D-SuperAdminFold`'s consequences section explicitly names this as a risk to
  close ("Super-admin screens must be role-gated server-side... a named M10.9 verification criterion")
  — which is the right instinct, but Phase 8 has no code yet and the plan gives no detail on *which*
  mechanism (a shared PEP-style enforcer per super-admin handler, vs. relying on the same
  ad hoc per-application-method pattern every other module already uses) will carry the system's
  highest authority. Given every other module's authorization is currently a hand-copied
  `require*` function per file (confirmed: `content`, `moderation`, `vouching`, `discovery`,
  `congregationimport` each maintain their own near-identical copy), betting the widest-blast-radius
  surface in the system on that same copy-paste discipline, with zero tests today, is the single
  highest-value place for the plan to specify a *shared, hard-to-misuse* enforcement helper rather
  than trusting phase 8 to copy the pattern correctly by hand.

## Unaccounted-for references

- **Doc-comment prose.** The exhaustive `oikumenea\|hermenea` sweep (case-insensitive, whole repo)
  found ~40 files whose only match is prose — Conjure YAML `docs:` fields
  (`api/registration.conjure.yml:4-48`), generated Go/TS doc comments derived from them
  (`internal/conjure/openfaithmap/registration/*.go`,
  `web/apps/*/lib/openfaithmap/generated/registration/registrationService.ts:14-38`), and i18n message
  files were checked and **do not** actually reference oikumenea (false-positive-free on that front).
  None of this is a functional dependency, but none of it is in the teardown checklist either — after
  the cutover these comments describe a system that no longer exists ("go-oikumenea's PDP decides for
  real"). Low priority, but real: a regenerated Conjure contract in Phase 7 is the natural place to
  also rewrite the `docs:` strings this prose is copied from.
- **`web/apps/admin/app/[locale]/admin/congregation-import/page.tsx`,
  `.../my-congregation/page.tsx`, `.../register/page.tsx`, `.../whoami/page.tsx` — confirmed as the
  plan's "four page components."** Cross-checked against the grep sweep: these are exactly the four
  admin-app files (besides `lib/jurisdiction.ts`/`lib/dictionaries.ts`) that import `lib/oikumenea.ts`
  directly. The plan's inventory here is accurate — noted as a positive confirmation, not a gap.
- **`religion_classifications` / `religion_taxon_ranks` / `religion_unit_classifications`** — see
  Critical Finding #2; these are absent from both the "kept" and "dropped" halves of the teardown/scope
  accounting.
- **The Phase 8 super-admin endpoint set** — see Critical Finding #3; absent from the "~12 endpoints"
  inventory and therefore from the LOC estimate it feeds.

## What the plan gets right

- **The removed `Authorize` meta-check reasoning** — independently reproducible from the code, not
  just asserted; see Security above.
- **The RLS-drop reasoning** — the plan checks the actual predicate's table dependencies before
  declining to port it, and states the real reason (GUC plumbing cost, duplicated surface) rather than
  a vague "not needed."
- **`SearchSites` port shape** — verbatim structural match to the real upstream query confirmed
  line-for-line; the one gap (code-column indexing) is a one-line omission, not a design flaw.
- **The deterministic-RID trade-off is stated, not hidden** — `D-SeedBootstrap` names its own honest
  limitation.
- **The docs-phase (M10) itself is coherent.** All eight new `D-*` blocks cross-reference correctly,
  every superseded block carries an append-only note (verified: `D-CoreDependency`, `D-Facade`,
  `D-InstanceAdminConsole`, `D-BulkImport`, `D-SharedDatabase` all have "Superseded/Narrowed (M10)"
  trailers), and the stage board's M10.1–M10.9 rows match the phasing table in the plan document
  one-for-one. Do not weaken this while fixing the findings above — the docs discipline is what made
  this review possible to do with actual evidence instead of guesswork.

## Could not assess

- **Whether Phase 8's super-admin screens will in fact get a shared enforcement helper vs. hand-copied
  per-handler checks** — no code exists yet; resolvable only once Phase 8's transport layer is
  written. The artifact that would resolve it: a code review of the first super-admin handler landed,
  checking whether it reuses a common gate or starts a sixth hand-copied `require*` function.
- **Real contention profile of closure extension under the Catholic jurisdiction sync on the target
  500MB–1GB VM** — no load test exists, upstream or in this repo, for concurrent closure writes plus
  read traffic. The artifact that would resolve it: a benchmark run against a seeded closure table
  with the M10.9 refusal-proof stack up, timed against representative anonymous read QPS.
- **Whether `religion_classifications`/`religion_taxon_ranks`/`religion_unit_classifications` are
  genuinely dead for OpenFaithMap's use** — plausible from the read/write call sites checked
  (`SearchSites`, `D-Exclusions`' ancestor walk, `CreateChildOrg`'s primary-taxon set), but not proven
  by exhaustively checking every upstream caller of those three tables (out of scope for this review —
  they're upstream-internal, not called by anything in `open-faith-map`). The artifact that would
  resolve it: confirm no code path this repo actually calls (`ListTaxa`, `GetTaxon`, `CreateChildOrg`,
  `SearchSites`, `ListOrgKinds`) references them, which a full read of `go-oikumenea/internal/religion`
  would settle definitively.
- **The true LOC cost of the Phase 8 super-admin endpoint set** — estimated at "plausibly 8–12 RPCs"
  by inference from what `oikumenea-console`'s stated feature set requires, since `oikumenea-console`'s
  own source isn't in this repo (published image only). The artifact that would resolve it: a page-by-
  page inventory of `oikumenea-console`'s actual API calls, which would need that repo's source.
