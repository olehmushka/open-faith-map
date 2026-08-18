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
| U11 | **`churchSiteTypeID` fails silently.** If go-oikumenea's seeded `church` site type is ever renamed, `approveRequest` attaches every congregation to whatever the first site type happens to be, with no error. Prefer failing loudly. | [registration.md](modules/registration.md) |
| U12 | **Config bypasses the install-config convention.** `internal/platform/config` exists to hold openfaithmap-api's settings and is empty; `cmd/openfaithmap-api` reads five real settings straight from the environment via `requireEnv` — no schema, no validation, no ECV path for the secrets among them. | [conventions.md](architecture/conventions.md) |
| U13 | ~~**Per-surface OAuth clients and WireGuard have no milestone**, because there is no deployment milestone at all. Both are recorded as prerequisites for any non-local-dev deployment; whoever creates that milestone inherits them.~~ **Resolved (2026-08-14): M9 is that milestone.** Both items are now scheduled as M9's own build-phase work — still open in practice, just no longer homeless. | `DS-OFM-14`, [M9](#m9--production-deployment-single-cheap-vm) |

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
| M4.1 · Jurisdiction units | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **Verified (2026-08-11).** D-JurisdictionUnits (supersedes D-FlatRoot's simplification). Real, operator-assigned jurisdiction units; existing congregations can be re-parented onto one. Proven live end-to-end against a real `docker compose` stack, including the real browser-driven admin UI flow (a real Google OAuth session through `/register` → `/admin/registrations` → `/admin/registrations/reparent`) — see prose. |
| M5 · Moderation | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **Verified (2026-08-11).** `modules/moderation.md` — reports/actions/appeals + a standalone D-Exclusions taxon-check dry-run. All three dependencies the 2026-08-09 audit found are resolved (D-PlatformModerator, D-Moderation's Correction, M4.1). CI green at the merge commit (confirmed 2026-08-10) and the two-real-token proof (non-moderator refused, platform-moderator allowed) both done — the latter via a headless local-dev identity, not a real browser Google OAuth session, accepted as equivalent evidence — see prose. |
| M6 · Vouching | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **Verified (2026-08-11).** `modules/vouching.md` — web-of-trust guarantor model. Its `moderation.read`/`moderation.act` gates and its `content.manage`-equivalent guarantor-standing check both resolved through D-PlatformModerator, the same mechanism moderation already uses. CI green at the merge commit (confirmed 2026-08-10) and the two-different-people proof (guarantor-with-standing vs. guarantor-with-none, moderator vs. non-moderator) both done — via a headless local-dev identity, not a real browser session, accepted as equivalent evidence — see prose. |
| M7 · Hardening / real-user feedback | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-11), not yet Verified.** D-Hardening (`architecture/decisions.md`), `modules/hardening.md`. In-process per-IP rate limiting on moderation's two anonymous write endpoints, a handful of app-defined metrics on witchcraft's already-wired stack, and a fix for the moderation-queue pagination defect (`nextPageToken` silently dropped since M5). Note that the audit moved three items people might expect here (CI, least-privilege DB role, API port exposure) forward into M2.4, because they gate every intervening milestone's Verified rather than being end-state polish. **`Verified` needs a green CI run on `main` at the merge commit and a live authenticated-moderator round trip (a real browser Google OAuth session or a granted moderator token)**, not yet attempted here — see prose. |
| M8 · Congregation import | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Production-hardening pass done (2026-08-12); go-oikumenea#36 RLS blocker fixed (2026-08-13); HTTP-streaming ingestion + D-Scope Christian-name filter added, a real cursor-doubling bug found and fixed, second connector `ar-rnc` (Argentina) added, a real pluggable geocoder (`domain.Geocoder`/`nominatim`) built and a full admin-UI usability pass done, all live-verified; third connector `osm` (OpenStreetMap/Overpass, scoped to Uruguay/Paraguay/Colombia/Chile) added and live-verified — a fourth-connector candidate, Brazil's CNPJ, was fully designed and live-verified but halted on a real `robots.txt` finding rather than built; admin UI gained manual-run parameters (`domain.ConnectorConfigurable`, `osm`'s `countryCodes`) plus a real adjacent bug fix (`domain.Connector.Clone` — a stale-cache bug that made a second manual run of `arrnc`/`osm` silently replay the first run's data forever) (all 2026-08-14); a hierarchical Catholic-church jurisdiction-tree sync (`D-CatholicJurisdictionSync`, `domain.JurisdictionSource`, `wikidata-catholic`) designed and built 2026-08-15, its first upstream blocker (go-oikumenea#39, mirroring GH-33/36/37) fixed the same day (PR #40, image 0.0.6); live verification then found the `tenant_units` write itself needed an org-scoped (not instance-wide) principal grant — fixed on this side (`scripts/bootstrap-service-principal -catholic-jurisdiction-org-id`), 38 real Ukrainian diocese/eparchy units created for real — but surfaced a second, deeper `tenant_unit_edges` RLS gap underneath (filed as go-oikumenea#41) —
**fixed upstream and live-verified end to end 2026-08-16** (a `RETURNING`-needs-read-reach gap, not
a policy/GUC bug, fixed upstream as image `0.0.7`; a second, real gap found on THIS side re-verifying
against it — `bootstrap-service-principal`'s `religion.read` grant was instance-wide, not org-scoped,
also fixed — after which a real sync against a live stack created all 38 Ukrainian dioceses with
real, closure-confirmed `tenant_unit_edges`, idempotent on retry); this session also rewired
`ua-edr`/`ar-rnc`/the Nominatim geocoder to delegate to standalone published packages
(`go-uaedr`/`go-arrnc`/`go-nominatim`) instead of duplicating their parsing logic in-repo; not yet
Verified (admin-UI browser click-through and a green CI run at the merge commit still block that).** D-CongregationImport (`architecture/decisions.md`), `modules/congregationimport.md`. Resolves `DS-OFM-10`. A module stages congregations from external sources (v1 connector: Ukraine's ЄДР open-data export, live-verified at full real scale — the true full-scale count is **30,721**, not the originally-reported 3,000, a real bug in the connector's cursor arithmetic, found live and fixed) for operator review; approval provisions a real, deliberately admin-less go-oikumenea Unit under the approving operator's own token, confirmed live under a genuinely non-admin identity. `ua-edr` can now stream directly from HTTP with no local file ever written to disk (`UAEDR_SOURCE_URL`, for a memory-constrained cloud deployment), and a positive Christian-name keyword filter auto-rejects out-of-scope (Muslim/Jewish/etc.) candidates that the source's own institutional-form filter can't distinguish. Review-queue + alias-management UI in `web/apps/admin`, real keyset pagination, alias-management API, automated tests, metrics. **`Verified` blocked on:** the admin UI's browser click-through (no OAuth session in this environment) and a green CI run at the merge commit. See prose for full live-verification detail. |
| M9 · Production deployment (single cheap VM) | ✅ | ✅ | ➖ | ➖ | ➖ | ✅ | **Verified (2026-08-14), a docs-only milestone — mirrors M0's own shape.** D-ProductionDeployment (`architecture/decisions.md`). Resolves `DS-OFM-14`/`U13`'s "no deployment milestone exists" gap. Single Linux VM (~500MB–1GB RAM), Docker Compose, Caddy for TLS — deliberately provider-agnostic, the concrete VM provider left undecided at the owner's own direction. Schedules two already-decided items (per-surface OAuth clients, WireGuard for `oikumenea-console`) as real build-phase work for the first time, and makes three new calls: `pg_dump` on a systemd timer for backup (none exists today), `restart:` policies + a systemd unit for process supervision (none exists today), and a weekly systemd timer calling `POST /runs` as the `ua-edr` periodic re-run trigger — the item M8's own memory left explicitly open. **`Verified`** needs the new doc set (this row, D-ProductionDeployment, the struck `DS-OFM-14`/`U13` entries) coherence-checked — no dangling links, no contradiction with the decisions it inherits from. **No VM is provisioned and no compose/Caddy/systemd files are written this milestone** — that is explicitly a follow-up build milestone (numbering TBD, likely M9.1) once a provider is picked. |

| M10 · Core absorption — decisions & docs | ✅ | ✅ | ➖ | ➖ | ➖ | ✅ | **Verified (2026-08-18).** **Decided (2026-08-17); amended 2026-08-18 after review.** Eight new decisions in `architecture/decisions.md`: D-OwnCore (supersedes D-CoreDependency + D-Facade), D-CorePortScope, D-InProcessAuthz, D-DirectTokenVerification, D-OwnRIDs, D-SeedBootstrap, D-SuperAdminFold (supersedes D-InstanceAdminConsole), D-StaticRefData. Resolves `DS-OFM-1`/`DS-OFM-8`, reframes `DS-OFM-3`, supersedes `DS-OFM-12`, halves `DS-OFM-14`, opens `DS-OFM-15`/`DS-OFM-16`. **Amendment pass (2026-08-18):** two independent code-grounded reviews (`review-result-1.md`, `review-result-2.md`) found six substantive errors, each re-verified in source before adoption — a missing `NOT NULL` FK target that would have made M10.1's migration fail to apply, an unported `authz_instance_admins` the PDP branches on first, an unadministrable fresh instance, a grant cache ported without the RLS backstop it documents depending on, a coordinate oracle in `SearchSites`, and a row lock described as an advisory lock. All eight decision blocks carry amendment notes; `architecture/conventions.md` corrected (four stale statements the first pass missed). Estimate revised from ~7–8k to **~12–15k LOC Go + ~3–3.5k migrations**. **`Verified`** needs the doc set coherence-checked. |
| M10.1 · Core schema + deterministic seeds | ⬜ | ⬜ | ➖ | ⬜ | ➖ | ⬜ | **Not started.** D-CorePortScope, D-OwnRIDs, D-SeedBootstrap and their amendments. PostGIS, ported `new_id()`, migrations for identity/authz (**including `authz_instance_admins`**)/directory/religion (**the 15 explicitly-named tables, not "13"** — `religion_taxon_ranks` is a `NOT NULL` FK target of `religion_taxa.rank_id` and its omission would have broken the migration)/location/membership/refdata, plus seed migrations with fixed **structural** RIDs. `visibility`/shadow units deliberately not ported. **Blocked on one check:** whether any path this repo calls reaches `religion_unit_classifications`. Purely additive — nothing calls these tables yet. |
| M10.2 · Identity + authentication middleware | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | **Not started.** D-DirectTokenVerification + amendment. Port `validator.go`/`authenticator.go` **in full** — three boot guards not two (the third, `GuardReservedIssuer`, stops an operator pointing a real IdP at the synthetic local issuer), plus algorithm pinning, clock skew, multi-audience matching and JWKS handling. A real `environment` field on `config.Install` as the **only** input to the symmetric-issuer guard — never derive "dev" from the presence of a secret. Boot-time first-admin seed from config, idempotent, refusing the placeholder. One auth middleware. **Also client work:** admin sessions break at the one-hour mark once this service owns the `exp` check, so next-auth refresh-token handling is in scope here, not a follow-up. |
| M10.3 · The policy decision point | ⬜ | ⬜ | ⬜ | ➖ | ➖ | ⬜ | **Not started.** D-InProcessAuthz + amendment. `internal/authz`: the in-memory PDP, the closed Go permission catalog **including the instance-scope set**, the instance-admin plane, decision-explain, `authz.Require(ctx, action, unitID)` (subject from context) with `DecideFor` reserved for one super-admin screen, and an unforgeable `SystemContext` covering **five** paths. **No grant cache** — grants are read per request. `internal/authz/domain` owns `ClosurePort` and imports no other module; that is what makes this milestone independent of M10.4. Highest-risk unit of work; gets a real table-driven test matrix. |
| M10.4 · Directory (units, graphs, closure) | ⬜ | ⬜ | ⬜ | ➖ | ➖ | ⬜ | **Not started.** D-CorePortScope. Adjacency plus the incrementally-maintained materialized closure under advisory lock, with `WITH RECURSIVE` confined to rebuild/verify/re-derive. Keeps `directory_closure_status` and the verify endpoint — the drift check is what makes the denormalisation safe. Preserves `CreateUnitWithEdge`'s atomicity (upstream's GH-36 fix). |
| M10.5 · Religion, location, membership, refdata | ⬜ | ⬜ | ⬜ | ➖ | ➖ | ⬜ | **Not started.** The 15 named religion tables (clergy and affiliation dropped), locations without the `wgs84` coordinate-conversion dependency, membership's two tables, and the country seed — whose four locale packs must survive **byte-for-byte**, since `matchCountry` does exact-string comparison against every locale name and `osm`'s `CountryHint` is built to match them. Includes the ported `SearchSites` PostGIS query with one deliberate behaviour change: `public_precision = 'hidden'` sites are excluded from the public search arm and other non-`exact` sites are filtered and ordered on a snapped geometry column, because coarsening the returned coordinate never protected the predicate. The text-arm index fix from the first draft was wrong three ways (needs an **expression** index for `lower(...)`, covers `code` as well as `name`, and cannot serve patterns under three characters) — and the real cost is probably the outer scan, so **measure before indexing**. |
| M10.5.5 · Split the composition root | ⬜ | ⬜ | ⬜ | ➖ | ➖ | ⬜ | **Not started.** `cmd/openfaithmap-api/main.go` is 393 lines wiring 6 modules, with `pool.Close()` repeated at 16 early returns; the honest end state at 13 modules plus cross-module interfaces, a `ClosurePort` adapter, an auth middleware and a boot seed is 900–1,100. Split `initServer` into per-module `register(ctx, info, deps)` functions **before** the cutover, not after — doing it afterwards means refactoring the composition root while it is the thing most likely to be wrong. |
| M10.6 · Consumer cutover | ⬜ | ⬜ | ⬜ | ➖ | ➖ | ⬜ | **Not started.** Six modules flip from SDK round-trips to in-process calls, `registration` first. `internal/coreintegration` deleted. Carries three behaviour changes: the `Authorize` meta-check disappears and `assignment.read` is no longer granted to congregation-admin; **`RunJurisdictionSync` gains `requireOperator`** (today its transport resolves `whoami` only, so any authenticated account triggers real Unit writes under the service principal's instance-wide grant — a live gap on `main` until this lands); and the anonymous discovery search gains the in-process rate limiter. Also fixes the N+1s the HTTP cost was hiding (`matchCountry` fetches all 249 countries **per record**). |
| M10.7 · `core.conjure.yml` + SDK + admin app | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | **Not started.** **~25 endpoints, not ~12** — the 9 the admin app calls oikumenea for today, plus the super-admin set replacing `oikumenea-console`, plus a batch `GetPersons` (`my-congregation` calls `GetPerson` in a loop). Regenerate the Go server and both TS SDK copies; rewrite the six contracts' `docs:` strings, which describe a system that will not exist and propagate into three copies of generated code; drop `oikumenea-client` and regenerate `package-lock.json` in the same commit. Wires `make sdk-verify` into CI. |
| M10.8 · Super-admin screens + teardown | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | **Not started.** D-SuperAdminFold + amendment — the fold is onto the **instance-admin plane**, not a role. Two gates, both required: one shared, hard-to-misuse API enforcer (not a sixth hand-copied `require*`) and a `requireInstanceAdmin()` layout check, because `openfaithmap-admin` has **no** server-side role gate today and its own layout comment says so. Then the full teardown, which is larger than first recorded: `lib/oikumenea.ts` itself, user-visible i18n strings in four locales, three `.env.example` files, the codegen pipeline scripts, `var/conf/install.yml`, `atlas.hcl`, and a **contract-phase migration** to drop the `oikumenea` schema (migrations are expand-only). |
| M10.9 · Verification | ⬜ | ➖ | ➖ | ➖ | ➖ | ⬜ | **Not started.** Headline: `docker compose up --build` with no sibling checkout, no service-account JSON, no `docker.io/olegamysk/*` pull. The single highest-value gate is a **table-driven authorization matrix** — {anonymous, congregation-admin@own, congregation-admin@other, registration-operator, platform-moderator, instance-admin} × every guarded endpoint → allow/deny, ~240 assertions, expressible only because D-SeedBootstrap makes structural RIDs constants. Plus refusal proofs (now including `RunJurisdictionSync`), the first-admin proof on a clean volume, an unforgeable-`SystemContext` proof, a `hidden`-site oracle proof, discovery and country parity against **baselines captured before M10.6**, the `ua-edr` 30,721 re-run, lock-contention timing, and an admin session surviving past one hour. |

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
`migrate-hermenea`, and a `hermenea` service (at the time, built from a sibling go-oikumenea
checkout via `Dockerfile.hermenea` — no published image existed yet — matching the `OIKUMENEA_SRC`
sibling-checkout pattern `oikumenea-migrate` already uses; **2026-08-16: a published image now
exists** (`docker.io/olegamysk/hermenea`), and `hermenea` is pulled like `oikumenea-app` rather than
built — migrations still read from the sibling checkout, unchanged), plus the two shared-secret env vars
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
>
> **Update (2026-08-13): `approveRequest`'s real write path proven live under a genuine non-admin
> identity for the first time — not a substitute for the two-real-token proof above, still open.**
> M8's own production-hardening pass had found and root-cased a real go-oikumenea RLS defect
> ([go-oikumenea#36](https://github.com/olehmushka/go-oikumenea/issues/36)) that made
> `approveRequest` under any genuinely non-admin `registration-operator` structurally impossible —
> every prior live proof of this endpoint, including this milestone's own, used the instance-admin
> identity, which bypasses RLS *and* PDP checks entirely, so the defect sat unexercised. Once
> go-oikumenea#36 was fixed upstream (image `0.0.4`) and two further permission gaps it had been
> masking (`religion.read`, `location.create` — missing from `registration-operator`'s role) were
> found live and fixed the same way `assignment.read` was above, a real `submitRequest` →
> `approveRequest` round-trip under a single non-admin operator identity succeeded end-to-end:
> `APPROVED`, a real unit created, and a real unit-scoped `congregation-admin` grant to the submitter
> confirmed in `authz_role_assignments`. This proves the write path itself is sound under a real,
> PDP-checked non-admin caller; it does **not** prove the target-scoped PII-disclosure fix (item 1
> above), which needs a *second*, distinct `congregation-admin`-only identity denied where the
> operator is allowed — still unattempted, still the one piece not achievable headlessly. This
> milestone's gates stay exactly as they are; full detail of the RLS fix and its live verification is
> under M8's own entry.

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
>
> **Update (2026-08-11): the real browser-driven admin UI flow is now done.** A real Google OAuth
> session against `http://localhost:3004` (the redirect URI for that origin was already registered
> on the shared OAuth client, closing the open item M1/M1.2/M2.1 all carried) drove the actual
> flow: submitted a registration through `/register`, approved it through `/admin/registrations`
> with a jurisdiction assigned, and re-parented the resulting congregation through
> `/admin/registrations/reparent` onto a different jurisdiction. Confirmed working as expected — no
> new defects found this time. **`Verified` flips to `✅`** in the stage board above; every gate
> this milestone named is now met.

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
>
> **Update (2026-08-11): the two-different-people distinction is now proven, headlessly, against a
> real `docker compose` stack — not through a real browser.** New tool: `scripts/mint-local-token`
> mints an HS256 token for go-oikumenea's local-dev issuer for an arbitrary `(email, subject)`
> pair, exploiting the fact that issuer's validator has no subject allowlist (any signature valid
> against the known dev key is accepted — `deploy/oikumenea-install.yml`'s own comment already says
> so) and that JIT (`account-email` match) links a new subject onto an *existing*, genuinely
> non-admin person's account without ever touching `authz_instance_admins`. This is a different
> proof mechanism than "real browser Google OAuth" — it never exercises Google's OAuth flow (M1
> already proved that separately) — but it exercises the exact same downstream `Authorize` call
> this milestone's own code makes, against a real, distinct, non-instance-admin PDP subject.
>
> Concretely: minted a fresh token for the already-existing `moderator-test@example.com` shell
> account, confirmed `GET /identity/v1/whoami` resolves it to that person's real `personId`
> (`019fec8e-f7d5-...`), distinct from `local-admin`'s. Called `GET /moderation/v1/reports` with
> it before any grant — real `403 Moderation:Forbidden`. Granted `platform-moderator` to that exact
> person (via `scripts/bootstrap-registration-org -moderator-person-code`, which already supported
> a distinct moderator identity — no code change needed there) and re-called — real `200` with the
> report list. This is precisely the "non-moderator refused, platform-moderator allowed"
> distinction this section's text names.
>
> **`Verified` is deliberately left `⬜`.** The milestone's own exit criterion says "real browser
> Google OAuth round trip" — this proof is real and live, but not that. Whether a headless
> local-dev-only identity proof is accepted as equivalent evidence is left for explicit review, not
> decided unilaterally here.
>
> **Decided (2026-08-11): accepted as equivalent evidence.** The headless proof above exercises the
> exact same downstream `Authorize`/PDP logic a real browser session would — go-oikumenea's
> identity layer treats a JIT-linked HS256 subject identically to a JIT-linked Google subject once
> past authentication, and M1 already separately proved the Google-OAuth-to-JIT path itself works.
> `Verified` flips to `✅` in the stage board above.

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
>
> **Update (2026-08-11): both two-different-people distinctions are now proven, headlessly, against
> a real `docker compose` stack** — using the same `scripts/mint-local-token` tool built for M5 (see
> that milestone's own 2026-08-11 update for the mechanism, and why this differs from "real browser
> Google OAuth"). Two identities beyond `moderator-test@example.com` (already `platform-moderator`
> from M5's update): a fresh `guarantor-test@example.com` shell account, granted `congregation-admin`
> scoped to one specific, already-real congregation unit from earlier testing
> (`019febff-3af8-...`, `iglesia-cristiana-evang-lica-los-hermano-buqw`) via the same direct-SQL
> pattern `scripts/bootstrap-registration-org` already documents for the first assignment on a unit
> (go-oikumenea has no API path for it) — not through a fresh registration submit/approve round
> trip, because `checkNotExcluded`'s `GetTaxon` call runs under the *caller's own* token, and a
> genuinely non-admin person needs an explicit `religion.read` grant to pass go-oikumenea's
> `RequireServiceOrPerson` gate on that read; granting congregation-admin directly is the more
> direct proof of vouching's own authorization check specifically.
>
> - **Guarantor standing distinction:** `POST /vouches` as `guarantor-test@example.com` (real
>   `religionorg.manage` standing on its own unit) → `200`, a real vouch row created. The identical
>   call as `moderator-test@example.com` (standing on no congregation at all) → real
>   `403 Vouching:Forbidden`. This is exactly "a guarantor with standing... vs. a guarantor with no
>   standing anywhere," this section's own phrasing.
> - **Moderator vs. non-moderator distinction, on `POST /guarantors/{id}/revoke`:** as
>   `guarantor-test@example.com` (no `platform-moderator` grant) → real `403 Vouching:Forbidden`. As
>   `moderator-test@example.com` (`platform-moderator`, from M5) → real `200`, guarantor status
>   flipped to `REVOKED`. Confirmed directly in Postgres that the revoke's fan-out fired exactly as
>   designed: one new `openfaithmap.moderation_reports` row, `target_kind='VOUCHING_EDGE'`,
>   `target_ref` = the real vouch id created above, `reason_code='OTHER'`, `detail` naming the real
>   guarantor/claimant/congregation RIDs and the revoke reason string passed above.
>
> **`Verified` is deliberately left `⬜`,** for the same reason M5's update gives: this is a real,
> live proof, but not a real browser Google OAuth round trip, which is what this milestone's stated
> exit criterion names. Whether it counts as equivalent evidence is left for explicit review.
>
> **Decided (2026-08-11): accepted as equivalent evidence, same reasoning as M5's.** `Verified`
> flips to `✅` in the stage board above.

### M7 · Hardening / real-user feedback

**Depends on:** M5 (the milestone that actually shipped the two anonymous write endpoints this
one protects). **Blocks:** nothing. **Leaves deployable:** yes — no schema-breaking change; one
expand-only migration adds two composite indexes.

**Decided + Designed (2026-08-10) — corrects this milestone's own prior framing.** The original
text above assumed nothing about M7 could be scoped "until real usage surfaces real problems." That
premise holds for the *numeric tuning* of a rate limiter and for whether more moderation-queue
filters are ever needed — both stay explicitly provisional. It does not hold for the two concerns
actually decided and designed here: rate-limiting a known anonymous-write surface, and fixing an
already-diagnosed pagination defect. Both were found by direct code inspection, not by user
behavior, so there was nothing to wait for.

[D-Hardening](architecture/decisions.md) covers rate limiting and observability;
[modules/hardening.md](modules/hardening.md) covers all three concerns to the full template,
including the moderation-queue pagination fix design. Scope, in brief:

**In scope (first cut):**
1. In-process, per-`(client IP, endpoint)` token-bucket rate limiting on
   `ModerationPublicService`'s `POST /reports` and `POST /exclusion-check` — the only two
   genuinely anonymous *write* endpoints in the whole API (registration and vouching both require
   `default-auth: header` on every endpoint, correcting `DS-OFM-9`'s "anonymous report/
   registration endpoints" phrasing — registration has no anonymous endpoint at all).
2. A small, fixed set of app-defined metrics (`reports_filed`, `exclusion_checks_run`,
   `rate_limit_rejections`) on top of witchcraft's already-auto-wired logging/metrics stack — no
   new infrastructure, no Prometheus/OpenTelemetry/scrape endpoint.
3. The moderation-queue pagination defect: `ListReports`/`ListAppeals` have declared
   `pageToken`/`nextPageToken` on the wire since M5, but the transport layer silently drops an
   incoming `pageToken` and never sets `nextPageToken` — always `LIMIT`-only. Fixed with real
   keyset pagination (see `hardening.md`'s Data model).

**Explicitly out of scope**, recorded as Open seams rather than silently dropped: read-side rate
limiting on content/discovery's public GETs, multi-replica rate-limit coordination, new
moderation-queue filters beyond the existing `scope`/`status`, and the rate-limit thresholds'
actual numeric tuning.

**Exit criterion**, to be proven once Backend/Migrated/UI land in a later ticket, against a real
`docker compose up --build` stack:

1. A scripted burst past the configured limit against `POST /reports` (or `/exclusion-check`)
   receives a real `429` with a `Retry-After` header once the bucket empties; a request from a
   second, distinct source IP in the same window is *not* rejected — proving the limiter is scoped
   per key, not globally tripped.
2. `reports_filed`, `exclusion_checks_run`, and `rate_limit_rejections` all show up incrementing in
   the metrics log stream after driving real traffic through each path.
3. Seed more than one page's worth of reports; a small-`pageSize` `GET /reports` call returns
   exactly `pageSize` rows and a non-null `nextPageToken`; the follow-up call with that token
   returns the next distinct rows with no overlap/gaps against the known-inserted set; a tampered
   `pageToken` is rejected `400`, not silently treated as page 1; the admin console's moderation
   queue page loads a second page through a new "Load more" control against the real stack, not
   just curl.
4. `go build ./... && go test ./...`, `./godelw verify`, both web apps' `npm run lint && npm run
   build`, and `make sdk-verify` all pass clean.
5. A green CI run on `main` at the merge commit.

> **As implemented (2026-08-11).** One correction to `hardening.md`'s original text, found while
> wiring the fix: it named "the existing `Moderation:InvalidArgument` Conjure error" for a tampered
> `pageToken`, but no such generic error exists in `api/moderation.conjure.yml` — every existing
> `INVALID_ARGUMENT`-coded error in this module is a specific named one
> (`ActionNotReversible`/`AppealActorConflict`/`TaxonNotFound`/`DoctrinalReasonNotAllowed`), never a
> catch-all. Added `Moderation:InvalidPageToken` instead, matching that convention;
> `modules/hardening.md` corrected to match.
>
> Exit criteria 1, 2, and 4 above are live-verified against a real `docker compose up --build`
> stack, not just reasoned about: a burst of 5 requests against `POST /reports` from one client IP
> succeeded, the 6th got a real `429` with `Retry-After: 12` and the documented
> `{"errorCode":"RATE_LIMITED",...}` body, and a second, distinct-IP container in the same window
> was unaffected — proven with two long-lived containers on the compose network holding fixed,
> confirmed-distinct IPs, not the ambiguous signal a `docker run --rm` loop gives (Docker can reuse
> a just-freed IP across rapid, sequential ephemeral containers). `reports_filed` and
> `rate_limit_rejections` both showed up incrementing in the real metrics log stream afterward. The
> new `migrations/0009_hardening.sql` applied cleanly against real Postgres in the same run. All
> four gate commands in criterion 4 pass clean.
>
> **Criterion 3 (the authenticated pagination round-trip) and the admin console's "Load more" click
> are not yet run** — same wall M2.3/M4.1/M5/M6 all hit before this milestone: `ListReports`/
> `ListAppeals` require a real person token holding platform-moderator standing, which needs a real
> browser Google OAuth session (or a headlessly-granted moderator token neither exists yet nor was
> set up here) — this environment has neither. The cursor logic itself is unit-tested directly
> (`internal/moderation/transport/cursor_test.go`: encode/decode round-trip, and every tamper
> shape — truncated base64, non-JSON bytes, wrong fields, empty values — rejected with
> `Moderation:InvalidPageToken`), and the underlying SQL is standard keyset pagination applied via
> the same migration already proven against real Postgres above; what's unverified is specifically
> the HTTP-layer round-trip through a real authenticated session. **`Verified` stays `⬜` until
> criterion 3, criterion 5 (CI), and the browser click-through are done.**
>
> **Update (2026-08-10): the CI-green half is now confirmed.** This milestone's own commit,
> `0905032`, produced a green run —
> [31432203685](https://github.com/olehmushka/open-faith-map/actions/runs/31432203685). `Verified`
> stays `⬜` — criterion 3 (the authenticated pagination round-trip) and the admin console's "Load
> more" browser click-through, both named above, are the two remaining blockers.
>
> **Update (2026-08-11): criterion 3's HTTP-layer round-trip is now proven, headlessly**, using a
> `moderator-test@example.com` identity minted via `scripts/mint-local-token` (see M5's own
> 2026-08-11 update for the mechanism). `GET /moderation/v1/reports?pageSize=5` against the real
> queue (20 seeded reports left over from earlier live-verification passes) returned exactly 5 rows
> and a non-null `nextPageToken`; the follow-up call with that token returned the next 5 distinct
> rows with no overlap against the first page's ids; a tampered `pageToken`
> (`pageToken=not-a-valid-token-at-all`) was correctly rejected `400 Moderation:InvalidPageToken`,
> not silently treated as page 1. `GET /moderation/v1/appeals` was also confirmed reachable (`200`,
> empty — no appeals existed to seed a second page from; it shares the identical cursor codec
> already unit-tested in `transport/cursor_test.go`, not independently re-proven with real
> multi-page data here). The admin console's actual "Load more" button click remains untested — a
> real browser is still required for that specific piece, and this proof is headless local-dev, not
> a real browser Google OAuth session, for the same reason M5/M6's own updates already qualify
> theirs. **`Verified` stays `⬜`** — the browser click-through and the same "is a headless proof
> equivalent evidence" question M5/M6 raise are both left for explicit review.
>
> **Decided (2026-08-11): the headless part is accepted as equivalent evidence, same reasoning as
> M5/M6.** That closes criterion 3's HTTP-layer half. `Verified` still stays `⬜` — the admin
> console's actual "Load more" browser click is a genuinely separate requirement (it exercises the
> Next.js UI, not the API), unresolved by any headless proof, and is planned as joint work with the
> real browser session — see M4.1's own remaining browser-UI blocker for the same category of item.
>
> **Update (2026-08-11): the browser click-through found a real bug, now fixed and re-verified.**
> The first real browser load of `/admin/moderation` (same Google OAuth session used for M4.1's own
> browser proof) returned a hard `500`, not the queue. Server logs
> (`docker logs openfaithmap-admin`) named the real cause precisely: `page.tsx` (a Server Component)
> passed a plain closure (`filedAt: (date) => t("filedAt", {date})`) into `<ReportList>` (a Client
> Component) as part of a `labels` prop — Next.js only allows Server *Actions* (functions marked
> `"use server"`) across that boundary, not arbitrary closures, and rejects the render outright. Not
> caught by `next build`'s static generation check, `tsc`, or lint — this route is fully dynamic
> (`ƒ`), so the boundary violation only fires on a real request. Fixed by having `report-list.tsx`
> call `useTranslations("ModerationQueuePage")` directly (a client-side next-intl hook, the same
> pattern already used by this app's own `locale-switcher.tsx`) instead of receiving a formatter
> function as a prop. **Found and fixed the identical bug on `/admin/moderation/appeals`** before it
> was ever reported — `appeal-list.tsx` had the exact same `filedAt` prop shape, same fix applied.
>
> Re-verified after the fix: the page now renders (confirmed both via `curl` — a session-less
> request correctly redirects to `/login` instead of `500` — and by the real browser session
> reloading the page). Seeded 53 real `OPEN` reports (over `ListReports`'s 50-row default page
> size, via the public `POST /reports` endpoint from 6 distinct-IP containers to stay under M7's own
> per-IP rate limit) so a "Load more" button would actually have something to load; the real browser
> click loaded the next page of rows with no duplicates. `tsc --noEmit`, `eslint`, and a full
> `next build` all pass clean on the fix.
>
> **Update: the fix is merged to `main`** (`4af7f12`, PR #20). `Verified` still needs a green CI run
> confirmed at that merge commit (M2.4's gate) before it can flip — not confirmed here.

### M8 · Congregation import

**Depends on:** M2 (the `registration` provisioning pattern this reuses), M2.5/M4 (the
service-principal path this module's D-Exclusions check and dedup search need). **Leaves
deployable:** yes — an operator-only API surface with no public-facing change; nothing scraped
becomes visible until an operator explicitly approves it.

Resolves `DS-OFM-10` — full design in [D-CongregationImport](architecture/decisions.md) and
[modules/congregationimport.md](modules/congregationimport.md). Scoped directly with the product
owner: broad, multi-source from the start (not one connector at a time deferred indefinitely);
scraped congregations provision as real, deliberately admin-less go-oikumenea Units, with an
explicit verified/claimed status overlay left open for further design; manual operator-triggered
runs only, no new scheduler; real government open-data sources over speculative HTML scraping for
v1. Target countries, stated directly mid-design: Ukraine, Argentina, Uruguay, Paraguay, Colombia,
Chile, Brazil, USA — reordering `D-Scope`'s original rollout sequence (recorded there as an
append-only update, not an edit).

**As implemented (2026-08-12).** Full hexagonal module (`internal/congregationimport/{domain,
adapters,application,transport}`), `api/congregationimport.conjure.yml`,
`migrations/0010_congregationimport.sql` (five tables — runs, candidates, taxon aliases, connector
citations, and the verified/claimed overlay), wired into `cmd/openfaithmap-api/main.go`'s
composition root and `docker-compose.yml` (an optional `UAEDR_UO_FILE_PATH` — the module boots
with zero connectors registered if unset, never a hard failure).

**Real, verified sources, not fabricated:** checked live via web search and, for Ukraine, by
downloading the actual dataset's own published schema (`uo_schema.zip`, a real XSD) and classifier
(`kopfg.json`) from data.gov.ua directly — not inferred from a third-party tool, which turned out
to describe an older, superseded version of the same export. Real, load-bearing finding from that
schema: **the current ЄДР export has no address field of any kind** (an older schema version did);
every `ua-edr` candidate lands in `NEEDS_GEOCODE`, address filled in by an operator, not invented.
Filter confirmed against the real classifier: `OPF = "Релігійна організація"`, КОПФГ code `825`.
Brazil (CNPJ, legal-nature code `322-0`) and Argentina (Registro Nacional de Cultos, excludes the
Catholic Church by law) sources were confirmed real but not yet built.

**Two real design corrections made while building, not assumed correct on the first pass:**
1. Taxon-alias matching was originally designed as an exact lookup; a `TaxonHint` is typically a
   full scraped legal name, not a short keyword, so an exact match would never fire against real
   data. Changed to substring matching against a small, fully-loaded alias list before this was
   ever run against real data, not discovered live.
2. Dedup was designed to compare candidate names against live go-oikumenea site names; checked
   directly against the real Conjure `DiscoverySite` struct (`SearchSites`'s response shape) and
   found it carries no name field at all. Simplified to geo-proximity-only before running, with the
   real reason recorded in code and in `congregationimport.md`, not silently dropped.

**Live-verified against a real `docker compose up --build` stack, against genuinely real ЄДР
data — not a fabricated fixture.** Downloaded the actual 326MB `uo.zip` (3.15GB uncompressed) from
data.gov.ua, stream-scanned it with a throwaway script (not the connector itself, to keep this
session tractable) to extract a small, honest subset: 12 real religious-organization records
(including a real Jehovah's Witnesses congregation) and 5 real non-religious ones, repackaged into
a `uo.zip` with the exact real structure/encoding, mounted into the container.

- `POST /runs {"sourceCode":"ua-edr"}` (via a `scripts/mint-local-token`-minted operator identity):
  `recordsFetched: 12, candidatesCreated: 12` — the 5 non-religious real records were correctly
  excluded by the OPF filter; Cyrillic names decoded correctly (cp1251 → UTF-8) in Postgres,
  confirmed by direct query.
- Seeded two real taxon aliases ("свідків єгови" → Jehovah's Witnesses, "баптистів" →
  Protestantism) and re-ran: `candidatesAutoRejected: 1`. Confirmed directly: the real JW
  congregation (`РЕЛІГІЙНА ГРОМАДА СВІДКІВ ЄГОВИ...`) landed `REJECTED_EXCLUDED` with
  `rejection_reason` naming `jehovahs_witnesses` — D-Exclusions firing live, on real data, not a
  synthetic test case. `POST .../approve` on that same candidate correctly returned
  `CongregationImport:NotApprovable`.
- The Baptist candidate: `POST .../edit` (operator fills in missing country/coordinates — ЄДР's
  real no-address constraint, confirmed exactly as documented) → `POST .../approve` under the
  real instance-admin identity → `PROVISIONED`, a real go-oikumenea unit created. Confirmed
  directly in Postgres: **zero rows** in `authz_role_assignments` for the new unit — the
  no-congregation-admin-grant invariant holds for real, not just in the design doc — and the
  `congregationimport_congregation_status` overlay row is correct (`verified_by_person_rid` set,
  `claimed_by_person_rid` null).

**Two real bugs found and fixed by actually running this, neither caught by review or `godelw
verify`:**
1. **The OPF filter would have matched zero real records.** `data.gov.ua`'s own classifier
   resource (`kopfg.json`) gives `"Релігійна організація"` (title case) for code `825`; the real
   export stores it as `"РЕЛІГІЙНА ОРГАНІЗАЦІЯ"` (all uppercase) — found by scanning the actual
   downloaded file, not assumed. Fixed with `strings.EqualFold`.
2. **A second `approveCandidate` call on an already-`PROVISIONED` candidate created a real,
   confirmed duplicate go-oikumenea unit** — the exact defect class M2.3 fixed once for
   `registration`, not fully replicated here: the precondition only excluded
   `REJECTED`/`REJECTED_EXCLUDED`, so `PROVISIONED` fell through to `ensureUnit`, whose own
   resume-check only short-circuits on `PROVISIONING`. Caught live: a real second unit
   (`019ff2bf-...`) existed in `tenant_units` before the fix. Changed the precondition from a
   denylist to an allowlist; re-tested — unit count stayed at 2, correctly rejected before ever
   reaching `createChildOrg`.

**One real, deep limitation found and diagnosed, left unfixed — out of scope for this module:**
attempting the approval under a genuine non-admin `registration-operator` identity (not the
instance-admin) failed with a real Postgres error: `new row violates row-level security policy for
table "tenant_units"`. Traced to the actual cause, not just the symptom: go-oikumenea's
`tenant_units` RLS write-check (`authz_unit_in_reach`) requires `tenant_unit_closure` to already
contain the new unit — which cannot be true for a brand-new unit's first-ever INSERT — and its
person-shaped-principal fallback path requires `authz_principal_grants`, a table
`scripts/bootstrap-registration-org`'s own break-glass raw-SQL grant (documented as necessary
because go-oikumenea has no API path for a first assignment) never populates, unlike whatever the
real `grantAssignment` API call does. **This is not new to `congregationimport`** — it would
equally block `registration.Approve` under a genuinely non-admin operator; every prior live proof
of that path in this project's history (M2/M2.3/M4) used the instance-admin identity, which
bypasses RLS entirely, so this gap was never previously exercised. Recorded here rather than
patched blind — fixing `bootstrap-registration-org`'s SQL (or go-oikumenea's own RLS policy) is a
real, separate, project-wide follow-up, not this milestone's to absorb.

No review-queue UI exists in `web/apps/admin` yet — this milestone is API-only for now.
`Verified` needs that UI, a green CI run on `main` at the merge commit, and a decision on the RLS
finding above — not attempted here.

> **Update (2026-08-12): institutional-hierarchy support (Catholic/Orthodox/Lutheran/
> Anglican-Episcopal).** Added `Candidate.JurisdictionHint`/`SuggestedJurisdictionUnitID` and a
> `congregationimport_jurisdiction_aliases` table, mirroring `congregationimport_taxon_aliases`'s own
> shape and substring-match discipline exactly. Deliberately follows
> [D-JurisdictionUnits](architecture/decisions.md) rather than inventing a parallel model: jurisdiction
> stays operator-assigned, never inferred — a match here is surfaced to the operator as
> `SuggestedJurisdictionUnitID` and nothing more; `ApproveCandidateRequest.jurisdictionUnitId` still
> requires the operator's own explicit choice, unchanged from before this feature existed. Traditions
> with no real institutional hierarchy (Baptist, Pentecostal, most non-denominational bodies)
> correctly produce no hint and no suggestion — by design, not a gap, matching D-JurisdictionUnits'
> own rejection of a single canonical per-denomination tree.
>
> **A real bug, found live, not by review:** the jurisdiction-match call was originally placed after
> the taxon-match step, which returns early (`NEEDS_TAXON_REVIEW`) whenever the taxon hint doesn't
> resolve — silently skipping jurisdiction matching entirely for exactly the candidates most likely to
> carry a useful jurisdiction hint (a still-unaliased denomination keyword sitting next to a
> resolvable diocese name in the same scraped legal name). Found by seeding a real jurisdiction alias
> against a UGCC (Ukrainian Greek Catholic) test record with an intentionally-unaliased taxon hint and
> observing `SuggestedJurisdictionUnitID` stay null after a re-run. Fixed by moving the
> jurisdiction-match call ahead of the taxon-match early return, so it always runs regardless of taxon
> outcome; re-tested, confirmed the suggestion populates correctly.
>
> **Live-verified against a real `docker compose up --build` stack**, using a locally-crafted fixture
> structurally faithful to the real, previously-verified `uo.zip` `<SUBJECT>` schema (this session did
> not re-download the 3.15GB export; the schema/encoding/field verification from the original M8 pass
> above still stands) — three records: a UGCC parish whose real legal name textually names "Львівської
> архієпархії" (the Lviv archeparchy), an independent Baptist congregation, and a non-religious LLC
> (OPF filter regression check).
> - After the fix: the UGCC record correctly got `suggestedJurisdictionUnitId` set from a seeded
>   `"львівської архієпархії"` alias, **while still sitting in `NEEDS_TAXON_REVIEW`** (no taxon alias
>   seeded for UGCC) — direct confirmation the two matches are independent, not sequentially gated.
>   The Baptist record correctly got no suggestion (no jurisdiction substring in its name); the LLC
>   record was correctly never staged at all.
> - **The "never auto-applied" invariant confirmed live, not just by reading `provision.go`:**
>   approved the UGCC candidate under the real instance-admin identity (the genuinely non-admin
>   `registration-operator` path hit the same pre-existing RLS gap recorded above, unrelated to this
>   feature) **without** passing `jurisdictionUnitId`, despite a matched (deliberately fake,
>   unresolvable) `suggestedJurisdictionUnitId` on the candidate. Provisioning succeeded; a direct
>   `oikumenea.tenant_unit_edges` query on the real created unit confirmed its parent is
>   `REGISTRATION_ROOT_UNIT_ID` — not the fake suggested unit, which would have failed outright had it
>   been used. `godelw verify --skip-test` clean; `go build ./... && go vet ./...` clean.

> **Update (2026-08-12): production-hardening pass — depth on `ua-edr`, not new breadth.** Scoped
> directly with the owner: harden the existing pipeline + review flow to real production quality
> (UI, real pagination, alias-management endpoints, automated tests, metrics), root-cause the RLS
> blocker rather than absorb a fix for it, no new connectors/countries this pass.
>
> **Review-queue UI now exists** — `web/apps/admin/app/[locale]/admin/congregation-import/{page.tsx,
> candidate-list.tsx}` (modeled on `/admin/moderation`'s server/client split and "Load more" pattern,
> not `/admin/registrations`'s no-pagination one) plus a secondary `/admin/congregation-import/
> aliases` page for taxon/jurisdiction alias management. `lib/congregation-import.ts` mirrors
> `lib/moderation.ts` exactly. `scripts/gen-ts-client.sh` regenerated both web apps'
> `lib/openfaithmap/generated` trees for the first time with `CongregationImportService`. `tsc
> --noEmit`, `eslint .`, and `next build` all pass clean on both new routes; `web/apps/web`'s own
> `tsc --noEmit` stays clean too (its `generated/` tree also picked up the new service, unused).
>
> **Real keyset pagination** — `ListRuns`/`ListCandidates` previously declared `pageToken` on the
> wire and silently ignored it, the exact M7-fixed defect class, unfixed until now. Ported
> `moderation`'s own M7 solution byte-for-byte (`domain.PageCursor`, `transport/cursor.go`, the
> `(created_at, id) < (...)` keyset predicate, a `CongregationImport:InvalidPageToken` error
> matching `Moderation:InvalidPageToken`'s precedent, a `pageSizeOrDefault` `maxPageSize` clamp that
> didn't exist before). New `migrations/0011_congregationimport_hardening.sql` (a separate file, not
> amending 0010 — mirrors `0009_hardening.sql`'s own precedent) adds the two composite indexes the
> new keyset queries need.
>
> **Alias management is a real API + a simple UI form now**, not SQL-only —
> `listTaxonAliases`/`createTaxonAlias`/`listJurisdictionAliases`/`createJurisdictionAlias`, gated by
> the same `requireOperator` check every other operator endpoint uses. A duplicate `(sourceCode,
> aliasText)` returns a typed `CongregationImport:AliasConflict` (`CONFLICT`), not a bare 500 —
> `adapters`'s `CreateTaxonAlias`/`CreateJurisdictionAlias` now translate the real Postgres
> unique-violation (`INSERT`, catch, translate — never check-then-insert, matching `content`'s own
> `InsertSite` precedent), race-safely. Deliberately `create`+`list` only, no delete — named as an
> open seam, not silently dropped.
>
> **Automated test coverage, zero before this pass.** Pure, DB-free logic was split out of
> I/O-bound functions specifically so it's unit-testable (matching `scripts/bootstrap-registration-
> org`'s own `permissionsToAdd` precedent): `findTaxonAliasMatch`/`findJurisdictionAliasMatch`
> (taxonmatch.go/jurisdictionmatch.go), `isApprovable` (provision.go — a direct regression test for
> the real duplicate-provisioning bug this milestone's own earlier update found and fixed live), plus
> `haversineMeters` and the cursor codec (copied from `moderation`'s own test cases). `go test
> ./internal/congregationimport/...` passes clean; no DB/go-oikumenea mocking framework introduced.
>
> **A small, fixed set of metrics**, matching D-Hardening's already-decided pattern applied to a new
> module for the first time: `openfaithmap.congregationimport.{candidates_staged,
> candidates_auto_rejected, candidates_provisioned, connector_run_failures}`, wired into
> `RunConnector`/`ApproveCandidate`.
>
> **The RLS gap is now root-caused precisely, not fixed** — this session read the actual go-oikumenea
> RLS policy/helper/grant code (not inferred) and confirmed it structurally cannot be satisfied by
> any person-shaped caller's first `CreateChildOrg` INSERT, for reasons unrelated to how the grant
> was made (`docs/modules/congregationimport.md`'s Known limitations has the full trail: exact
> files/lines, why the service-principal fix go-oikumenea already built for this exact problem class
> is unreachable from `CreateChildOrg`, two viable fix directions). Filed as
> [go-oikumenea#36](https://github.com/olehmushka/go-oikumenea/issues/36) per the owner's own
> decision — a go-oikumenea-side fix in a separate session, not absorbed into this module.
>
> **Live-verified against a real `docker compose up --build` stack, including a genuine full-scale
> run against the real ЄДР export** — the one piece no prior M8 pass had done (every earlier proof
> used a small subset or a synthetic fixture):
> - Downloaded the real `uo.zip` fresh from data.gov.ua (326,722,660 bytes, byte-identical in size to
>   the dataset's own published metadata) and mounted it in unmodified. `POST /runs
>   {"sourceCode":"ua-edr"}` completed with `status: SUCCEEDED` in ~4 minutes:
>   `recordsFetched: 3000, candidatesCreated: 2904, candidatesUpdated: 16, candidatesAutoRejected: 80`.
>   Independently cross-checked the claim that this was genuinely the whole file, not an early exit:
>   `unzip -l` confirms 3,158,894,541 bytes uncompressed; a direct byte-level scan of the extracted
>   file counted 2,015,098 real `<SUBJECT>` records — the connector's own Go `encoding/xml` streaming
>   decoder (charset-aware, unlike the ad-hoc byte-grep used only for this cross-check) scanned every
>   one of them and kept the ~0.15% whose `OPF` matched, a plausible real ratio.
> - **Memory stayed at ~16MiB (`docker stats`) for the entire multi-minute run against a 3.15GB
>   uncompressed file** — the batch=500 streaming design's own memory-boundedness claim, now proven
>   at real scale, not just asserted in a doc comment.
> - Resulting real status breakdown: `NEEDS_TAXON_REVIEW: 2713` (only 2 taxon aliases seeded — real
>   UGCC/Orthodox/Adventist/etc. records correctly wait for an operator, exactly the "advisory hint,
>   never inferred" design working as intended at scale), `NEEDS_GEOCODE: 202` (taxon resolved via a
>   seeded alias, no coordinates — ЄДР's real no-address-field constraint), `REJECTED_EXCLUDED: 81`
>   (D-Exclusions firing live on real data, not synthetic).
> - Real keyset pagination proven against real multi-page data: `pageSize=5` returned exactly 5 rows
>   and a real `nextPageToken`; the follow-up page returned 5 distinct rows with no overlap against
>   the first page's ids; a tampered `pageToken=not-a-valid-token-at-all` correctly returned `400
>   CongregationImport:InvalidPageToken`.
> - Real alias-endpoint round-trip: `createTaxonAlias`/`createJurisdictionAlias` both succeeded and
>   the created rows appeared in `listTaxonAliases`/`listJurisdictionAliases`; a duplicate
>   `createTaxonAlias` call on the same `(sourceCode, aliasText)` correctly returned `409
>   CongregationImport:AliasConflict`, not a 500.
> - One real, self-caused hiccup along the way, not a connector defect: the first full-scale attempt
>   failed 500 records in (`getTaxon failed: ... 500 Internal Server Error`) because an earlier
>   alias-conflict test in this same session had seeded a taxon alias pointing at a fake RID
>   (`test-taxon-adventist`). Cleaned up the test alias and re-ran cleanly — a real illustration of
>   why a bad alias can currently abort an entire run rather than just skip one candidate, named as a
>   new, small open seam rather than silently absorbed into this pass's scope.
>
> `godelw verify --skip-test`, `go build ./... && go vet ./... && go test ./...`, and both web apps'
> `tsc --noEmit`/`eslint .`/`next build` all pass clean. **The admin UI's actual browser
> click-through was not run** (no Google OAuth session in this environment) — named explicitly, same
> unresolved category M4.1/M7 both already carry. **`Verified` still stays `⬜`** — the browser
> click-through, a green CI run at the merge commit, and the go-oikumenea-side RLS fix (tracked
> separately, [go-oikumenea#36](https://github.com/olehmushka/go-oikumenea/issues/36)) all remain.

> **Update (2026-08-13): go-oikumenea#36 fixed upstream — the non-admin RLS blocker is closed.**
> go-oikumenea's `main` gained commit `02a1c6f` ("seed closure before INSERT so CreateChildOrg passes
> RLS for non-admin persons"), released as image tag `0.0.4` (`docker.io/olegamysk/oikumenea:0.0.4`,
> published via `scripts/release.sh`/CI this session — the release workflow needed
> `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` repo secrets that had never been configured on
> `go-oikumenea`, found and fixed live). Bumped `docker-compose.yml`'s `oikumenea-app` pin to `0.0.4`.
>
> **Live-verified against a real stack, from a genuinely non-admin identity — not the instance-admin,
> which bypasses RLS *and* PDP checks entirely, so it could never have exercised this path.** Minted a
> real, distinct person (`scripts/bootstrap-admin-person` + `scripts/bootstrap-registration-org` +
> `scripts/mint-local-token`, the same tool chain M5/M8 already established) holding only the
> `registration-operator` grant, no instance-admin flag:
> - Ran a small connector pass (a hand-crafted fixture, structurally faithful to the real `<SUBJECT>`
>   schema already verified at full scale above — no need to re-download the 3.15GB export just to
>   re-prove the RLS path) and called `ApproveCandidate` under this identity. Result: `PROVISIONED`, a
>   real `oikumenea.tenant_units` row created — directly confirmed in Postgres — where every prior
>   attempt (this module's own original live-verification, quoted above) hit `new row violates
>   row-level security policy for table "tenant_units"` outright. The admin-less-provisioning
>   invariant still holds: zero rows in `authz_role_assignments` for the new unit.
> - **The RLS fix unmasked two further, real, open-faith-map-side gaps immediately behind it** —
>   found live, the same discovery pattern this module's own history already established (bugs found
>   by actually running the thing, not by review): `registration-operator`'s role
>   (`scripts/bootstrap-registration-org`) was missing `religion.read` (`ensureSite`'s
>   `ListUnitSites` resumability check, unit-scoped `pep.Require`) and `location.create`
>   (`ensureSite`'s `CreateLocation` call, instance-wide `pep.RequireAnywhere`). Neither permission
>   was ever reachable before this session — both calls sit immediately behind the exact RLS wall that
>   had blocked every prior non-admin attempt, on both `congregationimport` and `registration`. Fixed
>   by adding both permissions to the role definition (`scripts/bootstrap-registration-org/main.go`)
>   and re-running the script — `reconcilePermissions` (already unit-tested, M2.3) added them to the
>   live, already-existing role idempotently. Re-tested: `ApproveCandidate` completed clean.
> - **Confirmed not specific to this module, live, not just by reading the code**: submitted a real
>   `registration` request and approved it under the *same* non-admin identity —
>   `POST /requests` → `POST /requests/{id}/approve` returned `APPROVED`, a real unit created, and
>   (the one place the two modules' provisioning intentionally differs from `congregationimport`) a
>   real unit-scoped `congregation-admin` grant to the submitter confirmed in
>   `authz_role_assignments`. This is *not* M2.3's own still-open "two-real-token" acceptance test
>   (a `congregation-admin`-only account and a `registration-operator` account, both via a real
>   browser Google OAuth session, proving the target-scoped PII-disclosure fix specifically) — that
>   remains unattempted, unrelated to RLS, and is recorded separately under M2.3 below. What this does
>   confirm is that `registration.Approve`'s real go-oikumenea write path now works end-to-end under a
>   genuine, PDP-checked non-admin identity for the first time in this project's history.
>
> `docs/modules/congregationimport.md`'s "Known limitations" RLS bullet updated to record the fix
> (struck through, not deleted, so the root-cause trail stays legible) rather than restating it as
> still-open. **`Verified` still stays `⬜`** — the RLS item is now the one thing on the earlier list
> that's actually closed; the admin UI's browser click-through and a green CI run at the merge commit
> both remain, unchanged from before.

> **Update (2026-08-14): two owner-requested deployment features — HTTP-streaming ingestion and a
> D-Scope Christian-name filter — built, live-verified, and along the way a real, previously
> undiscovered bug was found that corrects this milestone's own earlier "full-scale" numbers.**
> The owner's target deployment is the cheapest available cloud VM (~500MB RAM), and asked two
> things: don't require downloading the ~326MB export to disk at all (no cron, no object storage —
> "too expensive" for this), stream it straight into the database; and add an automatic filter for
> the Muslim/Jewish congregations they'd seen live in the review queue, as a **positive**
> Christian-keyword match rather than a blacklist of every non-Christian religion/sect variant.
>
> **HTTP-streaming connector** (`adapters/connectors/uaedr/connector_http.go`, new) — `uaedr.New`
> now takes either `filePath` or `sourceURL` (mutually exclusive). The real incompatibility: the
> existing `Fetch(ctx, cursor)` reopens the file and reskips from scratch every call, fine for a
> local file, impossible over HTTP (thousands of full re-downloads for a ~2M-record file at
> batch=500). Fixed by making the HTTP mode stateful — one held-open response body + `xml.Decoder`
> across calls, `cursor` becoming an opaque "already started" marker for this mode only, guarded by
> a `TryLock`'d semaphore (a second concurrent run on the same source is rejected with a clear
> error, not raced) — plus a new, additive `domain.ConnectorCloser` interface so `RunConnector`
> releases the lock/stream even when a run ends via a `Fetch` error or `ctx` cancellation, not just
> clean exhaustion (a real deadlock this design would otherwise have). The zip itself streams
> through a hand-written 30-byte local-file-header parser straight into `compress/flate` — no
> `archive/zip`, no `io.ReaderAt` requirement, true forward-only reading (DEFLATE's own end marker
> terminates decoding, independent of the zip's declared sizes); STORE is supported only when its
> length is knowable up front (empirically, Go's own `archive/zip.Writer` never produces that shape
> for a streamed write — verified directly, not assumed, and tested as a deliberate "rejected"
> case, not silently unsupported).
>
> **Live-verified against the real data.gov.ua resource**, not a local fixture: found the actual
> current download URL via data.gov.ua's own CKAN API (`package_show`), confirmed reachable and
> exactly 326,722,660 bytes via a plain `HEAD`, then ran the real connector against it twice through
> a genuinely non-admin operator identity. Both runs: `SUCCEEDED`, `recordsFetched: 30721` — RAM
> stayed in the low tens of MiB throughout (`docker stats`), no local file ever written to disk.
>
> **That very number — 30,721 — is itself the second real finding.** It disagreed by 10x with this
> milestone's own earlier "full-scale" verification (`recordsFetched: 3000`, quoted above), which
> had been treated as the real, complete count. Independently cross-checked the new figure three
> ways before trusting it over the old one: (1) a plain `unzip -p uo.zip | iconv -f windows-1251 -t
> utf-8 | grep -o '<OPF>[^<]*</OPF>' | grep -ic 'релігійна організація'` on the freshly-downloaded
> real file returned **30,721**, exact match, zero Go code involved; (2) direct sampling of the
> resulting DB rows showed real, plausible, geographically-diverse Ukrainian religious-organization
> names — Orthodox, Greek-Catholic, Baptist parishes, correctly alongside a few real Muslim/Jewish
> communities (pre-filter fix) that had prompted this whole session; (3) 30,721 is also simply the
> more plausible figure on its own terms — Ukraine has tens of thousands of registered religious
> organizations, not 3,000. Root-caused the old figure directly rather than assuming the file
> changed (`Last-Modified` on the current resource is 2026-08-11, predating the original 3,000-record
> run): a throwaway diagnostic driving `fetchFile` verbosely against the same real file showed the
> returned cursor **roughly doubling every call** (26455 → 82597 → 283345 → 613009 → 1247060 →
> 2510423 — the last value already exceeding the file's true 2,015,098 total records, which is why
> the very next call found nothing and reported a clean, false "exhaustion"). The bug:
> `fetchFile` returned `cursorOf(skip + seen)`, but `seen` already counts every element from 1
> within the current reopened pass — which itself re-decodes the `skip` prefix from byte zero — so
> `seen` alone already equals the correct new cumulative position; adding `skip` again double-counts
> it, and the error compounds across every subsequent call. Confirmed via direct experiment, not
> just static reading: a plain `io.Copy` of the raw zip entry read all 3,158,894,541 bytes
> correctly; a single continuous XML/charset decode (no skip logic at all) correctly counted all
> 2,015,098 `<SUBJECT>` elements, three times in a row; only the reopen-and-reskip cursor arithmetic
> was wrong. Fixed (`cursorOf(seen)`) and regression-tested
> (`TestFetchFileMultiBatchResume` — confirmed to actually fail against the pre-fix code, not just
> pass trivially against the fixed one). **Every prior run of this connector needing more than one
> batch (over 500 real matches) silently undercounted** — this was a real, live defect in production
> code, not a documentation error, present since the connector was first written, invisible at any
> smaller scale (a single-batch run never exercises `skip > 0` at all) and invisible in status too
> (`SUCCEEDED`, not `FAILED` — the corrupted cursor's false "exhaustion" is indistinguishable from a
> genuinely complete run without an independent count to check against).
>
> **D-Scope Christian-name pre-filter** (`application/christianfilter.go`, new) — a candidate whose
> taxon hint resolves to nothing at all (the exact case Muslim/Jewish congregations hit, since no
> taxon alias exists for them) is now checked against a positive, source-agnostic Ukrainian keyword
> list before falling through to `NEEDS_TAXON_REVIEW`; a miss auto-rejects it (`REJECTED_EXCLUDED`,
> reused — no migration — with a distinct `"D-Scope: ..."` reason, matching `checkExcluded`'s own
> existing precedent for carrying the specific reason in free text, not a dedicated status).
> Deliberately placed only inside the no-match branch, never before taxon-match, so it can never
> shadow `checkExcluded`'s own more specific D-Exclusions reason for a real excluded denomination.
> Live sampling of the real 30,721-record run's `REJECTED_EXCLUDED` rows caught **two further real
> keyword-list bugs** before calling this done: the original `"парафія"` (parish) entry didn't match
> its own genitive form `"парафії"` — the form real registered names actually use — the identical
> declension trap already documented for `"церква"`, just not applied consistently on the first
> pass; and two near-ubiquitous Orthodox abbreviations, `УПЦ` and `ПЦУ`, plus `єпархія`
> (eparchy/diocese), were missing entirely. Fixed and regression-tested
> (`christianfilter_test.go`, 16 cases). Re-running the connector after the fix correctly
> reclassified newly-seen records; rows already `REJECTED_EXCLUDED` from the pre-fix pass stayed
> frozen at their old (wrong) classification, by design — `UpsertCandidate`'s own `WHERE status IN
> (...)` guard deliberately never touches a row already past review, so a re-scrape can't silently
> undo an operator's decision. That's the correct invariant for a real deployment (no
> "pre-fix pass" ever occurs there); confirmed directly (not assumed) that the handful of
> still-visibly-wrong rows in this session's own dev database are exactly the frozen, stale ones —
> `import_run_id` and `updated_at` both point to the pre-fix run — not a live re-manifestation of
> the bug. A residue of genuinely ambiguous names remains by design (a parish named only after a
> saint or icon, or abbreviated `СВ.` for "Holy/Saint" — no keyword list can resolve these from the
> short legal name alone without real false-positive risk) — the explicit, owner-accepted "~99%, not
> 100%" bar, not chased further.
>
> `go build ./... && go vet ./... && go test ./... -race` and `./godelw verify --skip-test` both
> clean. **`Verified` still stays `⬜`** — unchanged blockers (browser click-through, CI green at the
> merge commit).

> **Update (2026-08-14): second connector added — `ar-rnc`, Argentina's Registro Nacional de
> Cultos.** Confirmed with the owner: next in D-CongregationImport's own target-country order
> (Ukraine → **Argentina** → Uruguay/Paraguay/Colombia/Chile → Brazil → USA), one connector, built
> to the same real-verified bar as `ua-edr`. Full design/rationale is in
> [modules/congregationimport.md](modules/congregationimport.md)'s own `ar-rnc` Sources entry; this
> block covers the live-verification evidence.
>
> **Real source, the CKAN-declared URL turned out to be dead** — found live, not assumed:
> `datos.gob.ar`'s own listed resource
> (`https://cancilleria.gob.ar/userfiles/datos/registro-nacional-cultos.csv`) 404s. The ministry's
> own current landing page links the real, working export instead
> (`https://cancilleria.gob.ar/userfiles/datos/registro-culto-export.csv`, confirmed `200`,
> 3,608,415 bytes, `Last-Modified: 2025-08-13`) — both URLs recorded honestly in the connector's own
> `Citation()`, not just silently swapped.
>
> **Two real, consequential findings from directly downloading and inspecting the export** (30,178
> rows, plain 5-column CSV, no header): (1) the `CI` column is the registered institute's own
> registration number, **shared across every branch row of one institute** — not a per-row key, so
> `SourceRecordID` is a SHA-256 hash of the row's own content instead, confirmed live to correctly
> keep distinct branches distinct; (2) **503 of the 30,178 rows are byte-for-byte duplicates** of
> another row in the source itself (a real data-quality artifact, not a bug here) — confirmed live
> that these correctly collapse onto one candidate via the same hash (a direct query for one known
> duplicate pair, "UNION DE LAS ASAMBLEAS DE DIOS" at "Juan Domingo Perón esq. Dinamarca", found
> exactly one candidate row, not two).
>
> **Much simpler connector than `ua-edr` by design, not a shortcut**: at 3.6MB (vs. `ua-edr`'s
> ~3.15GB), the whole export loads into memory once per run — no stateful streaming, no
> `ConnectorCloser`, no reopen-and-reskip cursor arithmetic, so `ua-edr`'s own real 2026-08-14
> cursor-doubling bug class is structurally impossible here. `TestFetchMultiBatch` regression-tests
> this design's own real risk instead (an off-by-one at a batch boundary).
>
> **The D-Scope Christian-keyword filter is now Spanish-aware**, extending the same
> `christianKeywords` list (no per-source dispatch — the Ukrainian/Cyrillic and Spanish/Latin stems
> occupy disjoint Unicode ranges, so merging is safe). Every Spanish stem was checked against real
> grep counts on the live export before being added, the same discipline the Ukrainian block used.
> **A real, consequential diacritics finding**: unaccented `"evangelica"` outnumbers accented
> `"evangélica"` 10,055-to-781 in the live data — the filter strips Spanish diacritics before
> matching (same treatment Ukrainian apostrophe variants already got) so this can't cause a false
> negative. A real short-stem finding too: `"evangelístico"` (evangelistic) does not contain
> `"evangelic"` as a substring, so the keyword is the shorter `"evangel"` instead — found by checking
> real unmatched names, not assumed.
>
> **Live-verified against a real `docker compose up --build` stack, against the genuinely live,
> freshly-downloaded export — not a fixture.** While wiring this up, found and fixed a real,
> pre-existing gap unrelated to this connector's own code: `docker-compose.yml` documented
> `UAEDR_SOURCE_URL` (`.env.example`) but never actually forwarded it into `openfaithmap-api`'s
> container environment — `UAEDR_SOURCE_URL`/`ARRNC_FILE_PATH`/`ARRNC_SOURCE_URL` are now all wired
> through. `POST /runs {"sourceCode":"ar-rnc"}` against the real live URL, no local file ever
> staged: `SUCCEEDED`, `recordsFetched: 30178, candidatesCreated: 26754, candidatesUpdated: 503,
> candidatesAutoRejected: 2921` — the `updated` count matches the 503 real duplicate rows exactly.
> Confirmed directly in Postgres: `26754` real rows `NEEDS_TAXON_REVIEW` (no Spanish taxon aliases
> seeded yet, so nothing resolves a taxon on this first run — the exact same shape `ua-edr`'s own
> first full-scale run had), `2921` `REJECTED_EXCLUDED` with the `"D-Scope: ..."` reason. Spot-checked
> known real rows land correctly: `"ASOCIACION DE LOS TESTIGOS DE JEHOVA"` (JW) rejected by D-Scope
> (no taxon alias seeded yet to route it through D-Exclusions instead — expected), `"CONFEDERACION
> ESPIRITISTA ARGENTINA"` (Spiritism) rejected, `"ASAMBLEA ESPIRITUAL DE LOS BAHAIS..."` (Bahá'í)
> correctly landed as the one **documented, accepted false positive** (contains "asamblea") rather
> than silently miscounted. Real address fields confirmed populated (`street`/`locality`/
> `admin_area1`) for real rows like `"IGLESIA BAUTISTA CALVARIO"` — genuinely better than `ua-edr`'s
> blank-everything case, since this source actually carries address text.
>
> `go build ./... && go vet ./... && go test ./...` and `./godelw verify --skip-test` both clean.
> This does not change M8's own `Verified` status — still blocked on the same, unrelated items
> (browser click-through, CI green at the merge commit).

> **Update (2026-08-14): real geocoder built (`domain.Geocoder`/`nominatim`) plus a full admin-UI
> usability pass — triggered by a real operator hitting a real, confusing `NotApprovable` failure
> live, not planned in advance.** Full design in
> [modules/congregationimport.md](modules/congregationimport.md)'s "Known limitations"; this block
> is the live-verification trail.
>
> **Root cause of the original failure**: the review-queue UI's `taxonId`/`countryId`/
> `jurisdictionUnitId` fields were plain free-text `<input>`s requiring a real go-oikumenea RID
> typed from memory, and Approve required coordinates with no way to get them — `ApproveCandidate`
> returns the identical `CongregationImport:NotApprovable` code for three different real causes
> (wrong status, no taxon, no coordinates/country), which cost real debugging time twice on the
> exact same candidate before the actual missing piece (coordinates) was found.
>
> **Admin UI**: the "Run connector" button was hardcoded to `ua-edr` — `ar-rnc` could not be
> triggered from the UI at all; now a `<select>`. `taxonId`/`countryId` are now real `<select>`s
> populated from `client.religion.listTaxa`/`client.geo.listCountries` (`web/apps/admin/lib/
> dictionaries.ts`, new — mirrors `/register`'s own existing inline calls exactly, `EXCLUDED_TAXON_
> CODES` deduplicated between the two). `jurisdictionUnitId` is now a real search-and-select
> (`JurisdictionField`, reuses `lib/jurisdiction.ts`'s existing `searchJurisdictionUnits`, built for
> `/admin/registrations/reparent`) plus a "Create unit" button opening a native `<dialog>` modal
> (no new dependency — this app has none) pre-filled from the candidate's own `jurisdictionHint`;
> submitting immediately selects the new unit, no page reload, via the same "call the Server Action
> and use its return value" shape `candidate-list.tsx`'s own load-more already established.
> Portaled to `document.body` since a `<dialog><form>` nested inside the outer Approve `<form>`
> would be invalid HTML (forms cannot nest). **No Go/Conjure changes for any of this** — every
> lookup goes straight to go-oikumenea from the Next.js layer under the operator's own session
> token, the same D-Facade architecture `/register`/`/admin/registrations/reparent` already use.
>
> **Real geocoder, designed pluggable from day one per the owner's own direction** (not a one-off
> integration: the stated goal is adding congregations globally over time, at a scale where
> Nominatim's own ToS — 1 req/sec, no bulk/systematic querying — will eventually need a second
> provider registered alongside or instead of it). `domain.Geocoder` mirrors `domain.Connector`
> exactly; `adapters/geocoders/nominatim` is the first implementation (OpenStreetMap, free,
> keyless). New `suggestCoordinates` endpoint, **advisory only** (same invariant as
> `suggestedJurisdictionUnitId` — never writes to the store), structured query
> (`street`/`city`/`state`/`country`, not a concatenated string), `countryId` best-effort resolved
> to a real name via the service principal's `Geo.ListCountries` first. A real, mechanically
> enforced `rate.Limiter` (1 req/sec, `golang.org/x/time/rate`) — not just a policy comment.
>
> **Live-verified against the real public Nominatim endpoint and the real running stack — closing
> the loop on the exact candidate that started this investigation**
> (`2e04778b-540c-4f87-9a5b-e6f9345f0c0b`, "Ministerio Evangelístico Mi Amigo Jesús A Las Naciones",
> `ar-rnc`): its own real address text turned out to have cadastral reference codes mixed into the
> street field — `suggestCoordinates` correctly returned `GeocodeNoMatch` (not a crash, not a wrong
> guess); approved instead with manually-supplied coordinates, confirmed `status: PROVISIONED`, a
> real `createdUnitId`. A second, cleaner-addressed real candidate ("IGLESIA EVANGELICA EL ALFARERO
> CRISTO JESUS," `CASADO 1173, CASILDA, Santa Fe`) round-tripped the full happy path:
> `suggestCoordinates` → a real match (`-33.0513756, -61.1575169`, a `displayName` naming the real
> matched street) → `editCandidate` → `approveCandidate` → confirmed `PROVISIONED`. Also hit real
> connection resets/timeouts calling the public Nominatim endpoint directly, seconds apart, from
> this session's own dev sandbox during research — a genuine reminder it's a best-effort free
> community service, not a guaranteed-uptime API, which is exactly why a non-nil,
> non-`GeocodeNoMatch` error passes straight through to the operator as a clear failure rather than
> being silently swallowed.
>
> `go build ./... && go vet ./... && go test ./...`, `./godelw verify --skip-test`, and both admin
> apps' `tsc --noEmit`/`eslint .`/`next build` all clean. Does not change M8's own `Verified`
> status.

**2026-08-14 (same day, later session): fourth-connector attempt on Brazil (CNPJ/Receita Federal)
fully designed and live-verified, then halted before any code was written — a `robots.txt` finding,
not a technical blocker.** Confirmed live: legal-nature code `3220` = "Organização Religiosa"
directly against IBGE/CONCLA and a real downloaded `Naturezas.zip`; RFB migrated the whole CNPJ
open-data dump to a Nextcloud WebDAV share (`arquivos.receitafederal.gov.br/public.php/webdav/
{YYYY-MM}/`) around January 2026 — the old `dadosabertos.rfb.gov.br` host is now unreachable; the
real current month (`2026-08`) has 10 parts each for Empresas/Estabelecimentos/Socios, far smaller
than an old ~85GB community estimate (~1.37GB/~5.34GB compressed respectively); a real match ratio
measured from a full real part (3,513 of 4,494,860 Empresas rows, ~0.078%); ISO-8859-1 encoding and
the official layout PDF's column order confirmed directly against real data, including one real
doc-vs-data mismatch (a missing house number is literal `"SN"` live, not the documented `"S/N"`).
Then found `arquivos.receitafederal.gov.br/robots.txt` returns `Disallow: /` for all user agents.
Reasoned this was very likely a generic Nextcloud-installation default rather than an RFB directive
aimed at this specific bulk-open-data endpoint, but asked the owner explicitly rather than deciding
unilaterally, per this module's own established discipline (`ar-rnc`'s dead-URL finding, `ua-edr`'s
citation checks) of surfacing robots.txt/ToS findings honestly rather than reasoning past them
silently. **The owner's explicit answer: stop, do not build this connector** — not "proceed with a
documented caveat," the alternative this session offered. No `brcnpj` code exists anywhere in the
repo as a result; full record kept in this project's session memory for if it's ever revisited (the
blocking fact to re-check first is that exact `robots.txt` line).

**2026-08-14 (same day, next session): third connector `osm` (OpenStreetMap, Overpass API) built and
live-verified, chosen specifically because it was less likely to hit the same class of blocker.**
Live-checked robots.txt for two Overpass mirrors before building: `overpass-api.de` (the main
OSM-Foundation-run instance) disallows `/api/` in its own robots.txt — the exact query-endpoint
path — flagged to the owner the same way, who asked to try an alternative mirror rather than stop
entirely. `overpass.kumi.systems` (Private.coffee) has **no robots.txt at all** (404, confirmed
live), with a published policy welcoming reasonable use (no enforced rate limit, asks only that
large-scale projects notify the operator first); `overpass.osm.ch` independently confirmed the same
absence, as a second data point. Chosen as the default endpoint.

A real end-to-end query against `overpass.kumi.systems` for all of Uruguay
(`area["ISO3166-1"="UY"][admin_level=2]`, `nwr["amenity"="place_of_worship"]["religion"="christian"]`,
`out center tags`) returned `200 OK` in ~20s: 566 real elements (290 node, 274 way, 2 relation — every
way/relation carried a real `center` object, 0 missing). Real tag data confirmed every design
decision: `denomination` values `catholic`/`roman_catholic` both appear for the same real
denomination (a real vocabulary inconsistency, documented as a starter-alias-list note, not a
parsing bug); 78 of 566 elements (~14%) had no `name` tag at all — filtered out before ever becoming
a candidate, a deliberate data-quality floor stated explicitly in the connector's own doc comment;
at least one real element carried a `diocese` tag, now mapped to `JurisdictionHint` when present.
Unlike either existing connector, OSM commonly carries both real address text and real coordinates —
`osm` is the first connector to actually populate `Latitude`/`Longitude`, which the existing
nil-check in `processRawRecord` already routes straight to `STAGED`, bypassing `NEEDS_GEOCODE`
entirely, with no code change needed there.

`Code() = "osm"` (`internal/congregationimport/adapters/connectors/osm`), simpler execution shape
than either existing connector: queries once per configured country on the first `Fetch` call
(`ISO3166-1` area match, real, live-verified query shape), keeps every result in memory, serves
batches via plain integer-offset slicing — no reopen-and-reskip step, so `ua-edr`'s real
cursor-doubling bug class cannot occur here. Default scope `OSM_COUNTRY_CODES=UY,PY,CO,CL` — the
D-Scope countries with no confirmed dedicated registry — deliberately **not** Ukraine/Argentina
(already covered) or Brazil (blocked, see above): `application/dedup.go`'s `findPossibleDuplicate`
only checks a new candidate against already-provisioned go-oikumenea sites, never against sibling
`STAGED` candidates in another connector's own still-largely-unreviewed queue (confirmed by reading
`dedup.go` directly), so running `osm` over Argentina today would flood the review queue with
near-duplicates dedup can't yet catch. `SourceRecordID` is OSM's own stable element identity
(`"{type}/{id}"`), genuinely per-element unique.

`go build ./... && go vet ./... && go test ./...` clean, including new `osm`-specific tests
(`connector_test.go`: multi-country load + `CountryHint` assignment, nameless-element filtering,
locale-name fallback, request-shape, batch-boundary/exhaustion, node-vs-way/relation coordinate
extraction, and the full `Normalize` field mapping). Not yet run against a live `docker compose`
stack in this session — `Country.Name`'s real content for the four target countries, and `osm`'s real
per-country totals beyond the one live-verified Uruguay query, remain open per
`docs/modules/congregationimport.md`'s own Open seams entry. Does not change M8's own `Verified`
status.

**2026-08-14 (same day, next session): admin UI gained manual-run parameters, plus a real adjacent
bug fixed in the same pass.** User request: the admin UI needed a way to manually trigger a
connector run, and — when a connector has parameters — a way to actually use them. Explored the
existing `RunConnectorRequest` (`sourceCode` only, no precedent anywhere in this repo's own Conjure
files for a generic params bag — though `map<string, string>` is an established pattern in
go-oikumenea's own contracts, confirmed by reading them directly) and `domain.Connector` (no
parameter channel at all — `Fetch` takes only a cursor) before designing.

**Real, adjacent bug found while designing, fixed in the same pass at the owner's explicit
direction**: `arrnc`/`osm`'s `sync.Once`-cached in-memory rows lived on the SAME long-lived connector
instance registered once at boot and reused for every `RunConnector` call — a second manual run
would silently replay the first run's data forever, never re-querying the real source (`uaedr`'s
own HTTP-streaming design happened to avoid this via its own per-run lock/stream reset, confirmed by
reading `connector_http.go` directly). Fixed by adding a required `Clone() Connector` method to
`domain.Connector`, implemented in all three connectors — `RunConnector` now always runs against a
fresh, run-scoped value (`base.Clone()`, or `configurable.WithParameters(parameters)` when the
caller supplies a non-empty map), never the shared registry instance directly.

**The parameters feature itself**: a new optional `domain.ConnectorConfigurable` interface
(`WithParameters(params map[string]string) (Connector, error)`, mirroring `ConnectorCloser`'s
own "optional, type-asserted for" pattern) — only `osm` implements it (one key, `countryCodes`,
validated against the same `countryNames` map `New` already uses; an unrecognized key or invalid
value is a construction error, fail-loudly). `RunConnectorRequest`/`ImportRun` both gained an
`optional<map<string, string>>` `parameters` field; a new `RunParametersNotSupported` typed error
(`CongregationImport` namespace) for a non-empty map against a connector that doesn't implement
`ConnectorConfigurable`. Persisted on the run row (`migrations/0012_congregationimport_run_
parameters.sql`, a nullable `jsonb` column, expand-only). Regenerated both the Go server code
(`./godelw conjure`) and the TypeScript SDK for both `web/apps/{admin,web}` (`scripts/
gen-ts-client.sh` — the real, previously-undocumented-in-this-file script that does this, found by
reading `CONTRIBUTING.md` and the M2.6 history rather than guessing a command).

**Admin UI**: `SOURCE_CODES` was missing `osm` entirely (a real, separate small gap found while
wiring this up) — added. A new small client component, `run-connector-form.tsx`, conditionally
renders a `countryCodes` text input only when `osm` is selected (`PARAMETERIZED_SOURCES`, manually
kept in sync with the backend — no "list registered connectors + their parameter shape" endpoint
exists, not worth building for one parameterized source). A blank field means "use the connector's
own deploy-time default," never an explicit empty-list override — enforced both in the Server
Action (never sends an empty-string parameter) and in `osm.WithParameters` itself (rejects an
effectively-empty `countryCodes` value).

**Scope note, found while implementing**: `listRuns`/`getRun` are wired in `lib/congregation-
import.ts` but nothing in `web/apps/admin` renders run history at all today — an operator sees the
resulting candidates land in the queue, but no page shows what parameters a given run actually used.
Real, deliberately out of scope for this pass (not asked for, would be a new page, not an extension
of an existing one) — the backend/data model (including the new `parameters` field) is ready
whenever this is built; see `docs/modules/congregationimport.md`'s Open seams.

`go build ./... && go vet ./... && go test ./...` clean across the whole repo, including new
`TestClone` regression tests on all three connectors (`arrnc`/`osm`'s rewrite the fixture/mock
response between two Fetch calls to prove a clone re-reads rather than reusing a cached result) and
`osm`'s `TestWithParameters`. `atlas migrate hash` re-run after the new migration. Both
`web/apps/{admin,web}`'s `tsc --noEmit`/`eslint .` clean; `web/apps/admin`'s `next build` clean.
**Not yet run against a live `docker compose` stack** in this session — the end-to-end proof (two UI
-triggered `osm` runs with different `countryCodes`, confirming fresh candidate sets and a clear
error for an unsupported-parameters attempt on `ua-edr`) remains open. Does not change M8's own
`Verified` status.

**2026-08-14 (same day, next session): the live `docker compose` end-to-end proof above actually ran
— a real operator triggered `osm` with `countryCodes: "CO"` through the rebuilt UI — and it failed.**
Root-caused directly against Postgres (`congregationimport_runs.parameters`/`.error`, the new column
from the session above) and reproduced live moments later: `overpass.kumi.systems` genuinely times
out computing a whole-country query for Colombia (`504`, "the server is probably too busy") — the
identical Uruguay query, re-run at the same time, completed in 6.5s. A second real finding from the
same investigation: the mirror doesn't always fail with a clean `504` — one Colombia attempt came
back `200 OK` with an HTML error page in the body instead of JSON, surfacing as a confusing raw
`invalid character '<'` JSON-decode error rather than a diagnosable one. Fixed:
`osm/connector.go` gained a `regionGrid` concept — a country can be configured with a bbox grid
(real bounds fetched live from Overpass itself via `relation["ISO3166-1"="CO"][admin_level=2]; out
bb;`, not guessed) that splits its query into several smaller bbox-bounded requests, still
intersected with the real country polygon (`area["ISO3166-1"=...]`) so results stay geographically
accurate. Only Colombia got one (3×2=6 cells) — Uruguay/Paraguay/Chile keep their original single
query, since only Colombia has actually been observed to need splitting; the doc comment is explicit
that this is measured, not pre-emptive. `queryRegion` (renamed from `queryCountry`) also now detects
a non-JSON response body explicitly instead of a bare status-code check. New tests:
`TestSplitGrid` (pure grid math), `TestLoadSplitsColombiaOnly` (Colombia issues 6 distinct-bbox
requests, Uruguay still issues exactly 1, in the same run), `TestQueryRegionHTMLErrorPage`
(regression test for the HTML-error-page finding). `go build/vet/test ./...` clean. Does not change
M8's own `Verified` status.

**2026-08-15: a hierarchical Catholic-church import strategy designed and built** — the owner asked
specifically for a hierarchical strategy (real diocese tree, not another flat connector with an
unaliased free-text hint), on branch `feature/congregationimport-catholic-jurisdiction-sync`, not
yet committed. Full design in the new
[D-CatholicJurisdictionSync](architecture/decisions.md) and
[modules/congregationimport.md](modules/congregationimport.md)'s "Jurisdiction sync" section — this
entry is a build summary.

Research done live before any code was written (this session's own discipline, matching `ar-rnc`'s
dead-URL check and `br-cnpj`'s robots.txt halt): **Wikidata** (CC0, public SPARQL API) was chosen
over `catholic-hierarchy.org` and `gcatholic.org` — the former's `robots.txt` explicitly blocks known
bulk-download tools by name (the same signal class that halted `br-cnpj`), the latter blocks
AI-training crawlers specifically. Both findings and the choice were confirmed with the owner, not
decided unilaterally. Live-verified via direct SPARQL queries: 6,655 Catholic dioceses/eparchies
worldwide (scoped via `wdt:P1866`, a Catholic-Hierarchy.org cross-reference — cleanly "actually
Catholic," unlike the generic `wdt:P708` "diocese" property Orthodox/Anglican bodies also carry),
167,544 parish/church entities linked to them, 142,459 (85%) with direct coordinates.
`query.wikidata.org/robots.txt` itself disallows `/sparql` for every agent — a second robots.txt
finding, also explicitly checked with the owner before building against it, judged (and agreed) to
be Wikimedia's standard interactive-query-page crawler block, not a block on the documented public
API this session's live-verified response headers (`access-control-allow-origin: *`, a dedicated
`api-user-agent` header) confirm this endpoint is designed for.

Three real product decisions the owner made explicitly when asked (not assumed):
1. Jurisdiction-tree creation is **fully automatic, no per-diocese operator click** — a deliberate,
   narrow exception to `D-JurisdictionUnits`'s "operator-assigned, never inferred" rule, scoped
   specifically to jurisdiction-tier Units from this one high-confidence structured source; how a
   *congregation* gets assigned to a diocese is completely unchanged.
2. Scope is global (all countries), not hardcoded to one — Ukraine is the natural first
   live-verification target (ties into `ua-edr`'s own already-large, still-largely-unaliased
   candidate set) but isn't special-cased in code.
3. Build scope this pass is the jurisdiction-tree sync + alias population only — the natural
   parish-level `wikidata-catholic` *connector* follow-on (167,544 candidates) is explicitly
   deferred, not built.

A real, load-bearing blocker was found by reading go-oikumenea's own source directly (sibling
checkout, not inferred): `Religion.CreateChildOrg`'s PEP gate is `pep.Require`, a person-shaped
method that — per that package's own doc comment — structurally denies a service-principal subject
regardless of grants. This is the identical defect class already found and fixed three times this
project (GH-33 `religion.read`, GH-36 the RLS defect, GH-37 `country.read`) — a machine-reachable
**write**, this time, not a read. Asked the owner which path to take (an upstream go-oikumenea fix,
mirroring GH-33/36/37, vs. a long-lived person-shaped "bot" credential as a workaround); the owner
chose the upstream-fix path. Filed as
[go-oikumenea#39](https://github.com/olehmushka/go-oikumenea/issues/39), mirroring GH-33/36/37's own
shape (asked first, filed only after the owner explicitly confirmed — a `gh` action visible on
another repo, per this project's own risky-action discipline). A second, related finding recorded
honestly rather than worked around: `scripts/bootstrap-service-principal`'s `GrantPrincipalPermission`
has no unit/subtree-scoping parameter today, so the `religionorg.manage` grant this pipeline will
eventually need is instance-wide, like this principal's three existing grants — not silently
narrower than it actually is.

**Built, this session, all live-verified where a live external call was possible (Wikidata's real
API), the rest unit-tested and `go build/vet/test ./...` clean:**
- `domain.JurisdictionNode`/`domain.JurisdictionSource` — a new, deliberately separate interface
  from `domain.Connector` (tree nodes, not congregation candidates).
- `adapters/jurisdictionsources/wikidatacatholic/` — the SPARQL source, two-query-per-batch design
  (bounded core-metadata page, then a `VALUES`-bounded multilingual-labels query), citation/
  robots.txt discipline matching every existing connector.
- `migrations/0013_congregationimport_jurisdiction_units.sql` — the natural-key
  (`source_code`, `external_id`) idempotency anchor and `PENDING`/`CREATED`/`FAILED` state machine,
  same decision-shape-CHECK-constraint discipline as `congregationimport_candidates`.
- `application/jurisdictionsync.go`'s `RunJurisdictionSync` — fetches the whole node set into memory
  (a deliberate, documented difference from `RunConnector`'s never-buffer discipline: a few thousand
  nodes at most, three orders of magnitude smaller than a congregation source), derives org-kind
  tiering from the fetched set's own topology (a node referenced as another's parent becomes
  `jurisdiction` tier, chosen over a hand-maintained Wikidata-type→org-kind table), creates nodes in
  topological order under one pre-existing, human-created anchor unit, and upserts global
  `congregationimport_jurisdiction_aliases` rows on success.
- A new Conjure endpoint, `POST /congregation-import/v1/jurisdiction-sync/runs` — real codegen run
  (`./godelw conjure`), not hand-edited generated code.
- `cmd/openfaithmap-api/main.go`/`docker-compose.yml`/`.env.example` wiring
  (`CATHOLIC_JURISDICTION_ANCHOR_UNIT_ID`/`_COUNTRY_QIDS`/`_WIKIDATA_BASE_URL`), same
  never-a-boot-failure, opt-in pattern as every existing connector.
- Unit tests for every pure function (`upgradeGroupingOrgKinds`, `jurisdictionSlugCode`'s
  determinism, `qidFromURI`, `primaryAndAliases`, QID validation) — no DB/go-oikumenea mocking
  infrastructure exists in this module (same established testing philosophy), so `RunJurisdictionSync`
  itself is not unit-tested end to end, matching how `RunConnector` has always been verified (a real
  live run), not attempted yet — see below.

**Not yet done, explicitly named, not silently skipped:** the real
`createChildOrg` call under the service principal (blocked on that fix landing, plus a
`scripts/bootstrap-service-principal` grant deliberately not added yet — adding an unusable grant
today would be misleading); any live `docker compose` verification (creating the anchor unit,
running the sync scoped to Ukraine, confirming the real tree + aliases, idempotency on a second run,
and a `ua-edr` candidate's suggestion populating on a backfill re-run); the parish-level connector
follow-on. Does not change M8's own `Verified` status — this is design-and-build-complete-not-yet-
live-verified, the same shape `br-cnpj`'s halt and `osm`'s first build both went through before their
own live-verification passes.

**2026-08-15 (same day, next session): go-oikumenea#39 fixed upstream, this side updated to match —
still not live-verified.** Owner reported the fix directly (merged as
[go-oikumenea PR #40](https://github.com/olehmushka/go-oikumenea/pull/40)). Real, not a mechanical
copy of GH-33/36/37's own fix shape: `CreateChildOrg`'s new gate is `pep.RequireServiceOrTarget`, not
`RequireServiceOrPerson` — go-oikumenea's own PR description explains why the straight swap this
session's own D-block had anticipated would have been a real regression (it would have widened a
PERSON caller's check from target-scoped to "holds it anywhere," letting any person holding
`religionorg.manage` on an unrelated unit create a child org under any unit). A person caller keeps
the unchanged target-scoped check; only a machine subject gets a new door, checked against its flat,
**instance-wide** grant set — go-oikumenea's own principal grants still carry no unit/subtree scope,
confirmed still true, not narrowed by this fix. `D-CatholicJurisdictionSync`'s own text corrected via
an append-only update note (its "Update (2026-08-15)" block), not silently rewritten.

Updated on this side: `docker-compose.yml`'s `oikumenea-app` pin `0.0.5` → `0.0.6`;
`scripts/bootstrap-service-principal` now grants `religionorg.manage` (documented as instance-wide,
same discipline as the `religion.read`/`country.read` grants before it); `application/
jurisdictionsync.go`'s own doc comment and every doc cross-reference (`decisions.md`,
`modules/congregationimport.md`) updated to describe the real fix mechanism rather than the
originally-guessed one. `go build/vet/test ./...` clean.

**Still not live-verified end to end** — a real `docker compose` stack was available in this
environment (a `go-oikumenea-postgres-1` container already running, port `5432` published,
apparently from unrelated prior work), and bringing up this repo's own compose stack alongside it
risked a port/state collision with whatever that container is serving — asked the owner before
attempting rather than guessing it was safe to bring up a competing stack. Once cleared: create the
anchor unit, run the sync scoped to Ukraine (`CATHOLIC_JURISDICTION_COUNTRY_QIDS=Q212`), confirm the
real go-oikumenea tree + alias rows, idempotency on a second run, and a real `ua-edr` candidate's
suggestion populating on a backfill re-run. Does not change M8's own `Verified` status.

**2026-08-15 (same day, later): owner reported go-oikumenea#39 fixed (PR #40, merged); this side
updated and a real live-verification attempt made — found a SECOND, separate upstream blocker.**
Bumped `docker-compose.yml`'s `oikumenea-app` pin to `0.0.6`, added `religionorg.manage` to
`scripts/bootstrap-service-principal`'s grant loop, corrected every doc reference from the originally
guessed "`RequireServiceOrPerson`" mechanism to the real one shipped
(`pep.RequireServiceOrTarget` — deliberately not a straight swap, since that would have widened a
*person* caller's check too; go-oikumenea's own PR description explains why). **A real bug in this
session's own code was also caught here, before any live call was even attempted**: `OrgKindId` on
`CreateChildOrgRequest` is a real go-oikumenea RID, not the stable code string
(`"diocese"`/`"jurisdiction"`) `JurisdictionNode.SuggestedOrgKindID` carries — `ensureJurisdictionUnit`
was passing the bare code straight through. Fixed by resolving codes to RIDs via a new
`resolveOrgKindIDs` (lists `ListOrgKinds` once per sync run, matches by `.Code`), the exact same
list-then-match pattern `provision.go`'s `churchSiteTypeID` already established for site types — a
real, live-verification-only catch, not something `go build`/`go vet`/unit tests could have found
(the type system has no way to know an `OrgKindId string` field secretly expects a specific catalog's
RID shape).

Given the go-ahead to proceed carefully despite the pre-existing (unrelated) `go-oikumenea-postgres-1`
container, brought up this repo's own stack for real
(`OIKUMENEA_SRC=../go-oikumenea docker compose up --build`) — confirmed no port collision (that
container had exited on its own by the time this repo's stack came up). Also found and fixed live: a
new migration needs `atlas migrate hash --env local` re-run before `atlas.sum` (gitignored,
regenerated locally, not committed) matches — `openfaithmap-migrate` failed with a checksum-mismatch
error on the first `docker compose up` attempt until this ran. Created the anchor unit for real
(`POST /religion/v1/units/{rootUnitId}/child-orgs`, `{"code":"catholic-church-root","name":"Catholic
Church"}` under the operator's own token) — a real go-oikumenea Unit now exists, id recorded in this
session's own `.env` (`CATHOLIC_JURISDICTION_ANCHOR_UNIT_ID`), not committed (matches every other
real secret/id in this repo's `.env` handling).

**The sync itself ran against the real Wikidata API and staged 38 real Ukrainian Catholic
dioceses/eparchies correctly** (`nodesFetched: 40`, both Latin- and Greek-Catholic/UGCC dioceses
present by name — e.g. "Erzeparchie Lemberg"/"Archiepiscopal Exarchate of Lutsk" alongside
"Archidiócesis de Leópolis"; 2 nodes correctly left unattempted, their `P749` parent lying outside
this country-scoped fetch, exactly the documented "left for a later run" behavior). **Every one of
the 38 real `createChildOrg` calls failed with a genuine `500`**, root-caused directly in
`oikumenea-app`'s own logs (not guessed): `ERROR: new row violates row-level security policy for
table "tenant_units" (SQLSTATE 42501)`. Traced to `migrations/0005_document_order_rls.sql`'s
`authz_unit_in_reach` — the predicate every `tenant_units` RLS policy calls — which is keyed
**entirely** on `current_setting('app.person_id')` and `authz_role_assignments.subject_person_id`;
nothing in it branches on a machine/service-principal subject at all. GH-39/PR #40 fixed the
**authorization** layer only (`RequireServiceOrTarget` now lets a machine subject reach the
endpoint); this is a separate, deeper **RLS** layer gap, the same general class GH-36 already fixed
once for this exact table, but that fix was scoped to a person-caller timing issue, not to machine
callers at all — a third upstream issue, still open, needed before this can go further.

**What DID work as designed, despite the underlying write failing**: `RunJurisdictionSync`'s
resumability held — all 38 failures were caught and correctly recorded
(`MarkJurisdictionUnitFailed`, `unitsFailed: 38` in the real run summary returned to the caller) with
no crash, no partial/inconsistent state, and no silent swallow. Does not change M8's own `Verified`
status. Stack left running in this environment (not torn down, in case the owner wants to inspect it
directly); `.env`'s `CATHOLIC_JURISDICTION_ANCHOR_UNIT_ID`/`_COUNTRY_QIDS` are this session's own
local additions, not committed.

**2026-08-15 (same day, continued): the `tenant_units` failure traced to this side's own grant, not
a new upstream gap — fixed; a genuinely new, deeper RLS bug found underneath it — filed.**
`GrantPrincipalPermissionRequest.OrgId` has existed all along ("omit for an instance-wide grant; set
to confine the machine to one organization") — the earlier "principal grants are instance-wide-only"
claim (this session's own D-block, and GH-39 itself) was a research gap, not a go-oikumenea fact.
go-oikumenea's own `authz_principal_org_in_reach` (`migrations/0011_infra.sql`) requires an
org-scoped grant for a machine subject's `tenant_units` write — confirmed live: granted
`religionorg.manage` org-scoped to `019fe8bb-3b41-8101-8406-06b65f756132` (the org owning both the
shared root unit and the new anchor unit) via a direct `POST /principal-grants` call, re-ran the
sync, and the `tenant_units` insert succeeded for real — 38 real Ukrainian Catholic diocese/eparchy
units now exist in this environment's database. `scripts/bootstrap-service-principal` gained a
`-catholic-jurisdiction-org-id` flag so this is reproducible without a manual API call next time; the
script's own doc comments and this repo's other docs (`D-CatholicJurisdictionSync`,
`modules/congregationimport.md`) corrected to match.

**Immediately underneath, a second, genuinely new bug**: the same `createChildOrg` call's follow-on
`tenant_unit_edges` insert (attaching the new unit to the anchor, same transaction) still fails with
the identical RLS error — every one of the 38 retried nodes, consistently. Root-caused as far as
possible without go-oikumenea's own server-side instrumentation: a manual raw-SQL reproduction of the
exact same insert (`SET ROLE oikumenea_app`, the three `app.*` GUCs set to match the real request)
**succeeds** — proving the RLS policy and grant model are both correct, and the gap is specifically in
how go-oikumenea's Go request path propagates those GUCs to the connection
`tenant.Service.CreateUnitWithEdge`'s internal `InsertEdge` call actually runs on. Filed as
[go-oikumenea#41](https://github.com/olehmushka/go-oikumenea/issues/41), with the working raw-SQL
repro and a suggested instrumentation point included. **Real consequence of this testing, left
as-is**: 38 orphan `tenant_units` rows now exist in this local-dev database under the anchor unit
(created successfully, no parent edge — unreachable from the graph). Not cleaned up — no safe
delete-unit path exists to do this without risking corrupting go-oikumenea's own closure/RLS
invariants by hand; flagged to the owner rather than attempted. `RunJurisdictionSync` remains not
live-verified end to end. Does not change M8's own `Verified` status.

**2026-08-16: GH-41 fixed upstream, live-verified end to end against a real stack — `wikidata-catholic`
is no longer blocked.** Root cause, diagnosed live against a real Postgres (not the GUC-propagation
bug this entry's own working theory above suspected): `InsertEdge`'s sqlc query uses `RETURNING`,
and in this Postgres version a row whose `WITH CHECK` (write) passes but whose table `USING` (read)
does not raises the identical "new row violates row-level security policy" error for `RETURNING`,
not a silent empty result — so the org-scoped `religionorg.manage` grant satisfies the write side
but not the implicit read side. Never an RLS policy gap or GUC-propagation bug; the policy and
connection plumbing were both correct all along. Merged to go-oikumenea `main`, published as image
`0.0.7` — bumped in this repo's `docker-compose.yml` (was `0.0.6`).

Re-verifying against the bumped image surfaced a SECOND real gap on this side:
`scripts/bootstrap-service-principal` already granted `religion.read`, but instance-wide — re-running
the sync against 0.0.7 with only that grant hit the byte-for-byte identical
`tenant_unit_edges` RLS error, proving an instance-wide grant doesn't satisfy `InsertEdge`'s
read-reach check any more than an instance-wide `religionorg.manage` grant satisfied the write-reach
check earlier in this same investigation. go-oikumenea's own GH-41 regression test grants
`religion.read` ORG-SCOPED, confirming the fix: `bootstrap-service-principal` now grants a second,
org-scoped `religion.read` alongside `religionorg.manage` when `-catholic-jurisdiction-org-id` is
passed.

**With both org-scoped grants actually in place, a real `wikidata-catholic` sync against a live
docker-compose stack on image `0.0.7` succeeded completely**: `nodesFetched: 40, unitsCreated: 38,
unitsSkipped: 0, unitsFailed: 0, aliasesCreated: 486`. Verified past the HTTP response, directly
against go-oikumenea's own tables: all 38 `congregationimport_jurisdiction_units` rows are `CREATED`,
all 38 have a real `tenant_unit_edges` row under the anchor unit, and all 38 have a real
`tenant_unit_closure` row confirming genuine reachability from the anchor — not orphans (the previous
session's 38 orphan `tenant_units` rows no longer exist in this environment's database; the volume
was evidently reset between sessions, so no manual cleanup was needed either). A second sync run
confirmed idempotency: `unitsCreated: 0, unitsSkipped: 38, unitsFailed: 0` — no duplicate
`createChildOrg` calls. `RunJurisdictionSync` is now genuinely live-verified end to end. Does not by
itself change M8's own `Verified` status (the admin-UI browser click-through and a green CI run at
the merge commit remain the blockers for that).

### M9 · Production deployment (single cheap VM)

**Depends on:** D-InstanceAdminConsole, D-OAuthClients, D-SharedDatabase (the decisions this
milestone schedules or inherits work from), D-CongregationImport (the module whose periodic
re-run trigger this milestone decides). **Leaves deployable:** no — this is a design milestone, not
a build one. A follow-up build milestone (numbering TBD, likely **M9.1** once a VM provider is
picked) does the actual provisioning; nothing in this repo changes behavior as a result of M9 by
itself.

Full design in [D-ProductionDeployment](architecture/decisions.md). Confirmed directly with the
owner (2026-08-14): scope this session is a design-only milestone, mirroring **M0**'s own
docs-only precedent — `Backend`/`Migrated`/`UI` are `➖` (not applicable, not just unbuilt), and the
concrete **VM provider is explicitly deferred, not this milestone's question**. The design stays
provider-agnostic on purpose: a single Linux VM, ~500MB–1GB RAM, Docker + Docker Compose available
— the same budget M8's own `UAEDR_SOURCE_URL` work already targets.

**Why this milestone exists at all.** `open-questions.md`'s `DS-OFM-14` and this doc's own
`U13` both said the same thing plainly: per-surface OAuth clients
([D-OAuthClients](architecture/decisions.md)) and WireGuard in front of `oikumenea-console`
([D-InstanceAdminConsole](architecture/decisions.md)) were already decided in principle but had
nowhere to attach as scheduled work, because no deployment milestone existed. Digging further
while scoping this surfaced more real gaps with **no decision on record at all**: no service in
`docker-compose.yml` carries a `restart:` policy (a crash just stays down), there is no reverse
proxy or TLS termination anywhere in the stack, there is no backup mechanism for the shared
Postgres instance, and M8's own memory explicitly named "no decision on what triggers a periodic
re-run of the `ua-edr` connector on a real deployed VM" as still open — a different question from
the download-step no-cron constraint M8 already resolved.

**What M9 actually decides**, full rationale in D-ProductionDeployment:
- **Reverse proxy / TLS:** Caddy (automatic Let's Encrypt), in front of `openfaithmap-web` and
  `openfaithmap-admin` only. `oikumenea-console` gets no public port at all, WireGuard only.
- **Per-surface OAuth clients** and **WireGuard for `oikumenea-console`**: both inherited verbatim
  from their own existing decisions, given real scheduled work here for the first time.
- **Secrets handling:** a root-only `.env` file on the VM (`chmod 600`), rotated from today's
  insecure dev defaults, never committed. No secrets manager for v1.
- **Backup:** `pg_dump` on a systemd timer to an off-VM target (concrete target deferred with the
  provider). Still bound by D-SharedDatabase's existing "one backup target, no independent
  RPO/RTO" caveat — not reopened here.
- **Process supervision:** `restart: unless-stopped` on every long-running service, plus a systemd
  unit wrapping `docker compose up -d` as the boot-time entry point.
- **`ua-edr` periodic re-run:** a weekly systemd timer calling `POST /runs
  {"sourceCode":"ua-edr"}` under a real operator identity — mirroring `hermenea`'s own
  `cron: "@weekly"` precedent, not a new in-process scheduler (consistent with
  D-CongregationImport's original "no new scheduler" call).

**Explicitly out of scope this milestone** — named, not silently dropped: the concrete VM
provider, DNS/domain, and actually writing `docker-compose.prod.yml`/a Caddyfile/the two systemd
units/timers or provisioning the OAuth clients and WireGuard peers. All of that becomes M9's own
inherited build-phase work, done once a provider is chosen.

**`Verified`** — same exit criterion M0 used: the new/changed doc set (this section, the stage
board row, D-ProductionDeployment, and the struck `DS-OFM-14`/`U13` entries) coherence-checked —
no dangling relative link, no contradiction between D-ProductionDeployment's sub-decisions and the
decisions it inherits from. Done directly, same pass (2026-08-14): every new cross-reference
(`D-OAuthClients`, `D-InstanceAdminConsole`, `D-SharedDatabase`, `D-BulkImport`,
`D-CongregationImport`, and the `M9`/`DS-OFM-14`/`U13` anchors themselves) resolved correctly, and
the stage-board row's gate marks match this section's own prose.

---

### M10 · Core absorption — decisions & docs

**Depends on:** nothing in code — a docs milestone, same shape as M0 and M9. **Leaves deployable:**
trivially yes (no code changes). **Blocks:** every M10.x below.

Reverses this project's founding architectural bet. M0 established go-oikumenea as the headless
core (D-CoreDependency) and OpenFaithMap as a thin facade over it (D-Facade); M10 absorbs that core
into `openfaithmap-api` and deletes the dependency — image, SDK, npm client, sibling checkout and
all.

**Why now, and why not earlier.** The facade was the right call for M1–M8 and this milestone is not
a repudiation of it: it is what let the project reach nine built modules without ever writing an
authorization system. What changed is the cost side, and it changed in three measurable ways the
build itself surfaced:

1. **Authorization was a network dependency with no degraded mode.** Every authenticated request
   made two round-trips — one `Whoami`, one `Authorize`. An `oikumenea-app` outage is a total
   outage.
2. **Delivery was gated on a second repository.** Six upstream issues (#33, #34, #36, #37, #39,
   #41) were found *by this project*, each blocking the milestone that found it until it was fixed
   upstream, released as a new image, and pulled back in. Read M8's own stage-board row as the
   evidence — it is largely a chronicle of that loop.
3. **Three artifact channels drifted.** Go SDK `v0.1.0`, npm SDK `0.0.1`, images `0.0.7`,
   independently versioned with no compatibility gate.

**Why it is affordable.** go-oikumenea is ~267k LOC, but that number is misleading: it is dominated
by generated Conjure transport, per-package duplicated sqlc `models.go` files, and ~26 verticals
this project never touches. The behaviour-carrying parts are small — the entire PDP is **260 lines
of pure Go** over pre-fetched grants, and `geo_countries` was already a static 249-row seed. The
estimate is ~7–8k LOC of Go plus ~1.5k LOC of migrations, and the single biggest reason it is not
far larger is that Conjure transport is generated only for the ~12 endpoints `openfaithmap-admin`
actually calls (M10.7). Everything else is in-process Go with no transport layer at all.

**The eight decisions**, in `architecture/decisions.md`: D-OwnCore (supersedes D-CoreDependency and
D-Facade), D-CorePortScope, D-InProcessAuthz, D-DirectTokenVerification, D-OwnRIDs, D-SeedBootstrap,
D-SuperAdminFold (supersedes D-InstanceAdminConsole), D-StaticRefData (supersedes D-BulkImport).
D-Stack is deliberately **untouched** — the Palantir toolchain stays.

**Two findings recorded up front**, so neither is rediscovered later as a bug:

- **The `Authorize` meta-check disappears, and that is correct.** Today `Authorize` is called with
  the caller's own token, so go-oikumenea's PDP additionally requires the caller to hold
  `assignment.read` reaching the target unit — which is why `scripts/bootstrap-registration-org`
  grants congregation-admin that permission (see `internal/content/application/authorize.go:21-25`,
  and M2.3/M3 where the same defect class was fixed twice). In-process,
  `Authorize(subject, action, unit)` is a pure function of the subject; the meta-permission is
  meaningless and the grant goes away.
- **Bootstrap RIDs become deterministic seed data.** `REGISTRATION_ROOT_UNIT_ID`,
  `REGISTRATION_CONGREGATION_ADMIN_ROLE_ID` and `CATHOLIC_JURISDICTION_ANCHOR_UNIT_ID` are
  instance-specific values produced by four manual script runs — which is exactly why environments
  are not reproducible today. Owning the tables makes them ours to fix once (D-SeedBootstrap).

**Cutover shape.** Greenfield and straight-line, on one long-lived branch — nothing is deployed and
there is no production data, so there is no dual code path, no feature flag, and no migration of
existing rows. The `oikumenea` schema and the `hermenea` database are dropped outright. M10.1–M10.5
are purely additive (new modules nothing calls yet) and can land as separate commits; only M10.6
onward is irreversible in the branch's own history.

**Explicitly out of scope** — named, not silently dropped:

- **Collapsing `discovery_site_cache`.** Once `SearchSites` is in-process the cache table is
  arguably redundant — it exists precisely because that join used to cross a network boundary
  (M2.5, M4). A real opportunity, deliberately deferred: widening M10 to include it would put the
  product's most manually-verified feature at risk during its riskiest refactor.
- Postgres RLS (D-InProcessAuthz), the audit log (D-OwnCore, now `DS-OFM-15`), the Palantir
  toolchain (D-Stack, unchanged), and the `go-uaedr`/`go-arrnc`/`go-nominatim` libraries — those
  are standalone parsers with no oikumenea coupling and are not part of this dependency.
- M9's production deployment. M10 makes it materially cheaper — no WireGuard, two surfaces instead
  of three, no `OIKUMENEA_SRC`, one database instead of two, and one fewer long-lived private key
  on the VM — but provisioning remains M9's inherited build-phase work.

**`Verified`** — the same exit criterion M0 and M9 used: the new and changed doc set (this section,
the ten stage-board rows, the eight new decisions, the four superseded ones, and the amended
`DS-OFM-1`/`3`/`8`/`12`/`14` plus new `DS-OFM-15`/`16`) coherence-checked. No dangling relative
link, no contradiction between a new decision and the one it supersedes, and every superseded block
left in place with an append-only note rather than edited away — this doc set's own convention.

---

#### M10 amendment pass — 2026-08-18

The plan was sent to two models for independent review against this repo and `../go-oikumenea`
(`docs/review-result-1.md`, `docs/review-result-2.md`). Every finding below was re-verified in the
source before being adopted; the reviews are point-in-time opinions and bind nothing on their own.

Six substantive errors, in rough order of how much damage each would have done:

1. **`religion_taxa.rank_id uuid NOT NULL REFERENCES religion_taxon_ranks(id)`**
   (`0008_religion.sql:140`). The scope table named 13 kept + 6 dropped = 19 of 22, and its bare
   word "classifications" was ambiguous between four differently-named tables. M10.1 would have
   produced a migration that does not apply. Replaced with an explicit 15-name list.
2. **`authz_instance_admins` was never ported**, though `PDP.Decide` branches on it first
   (`pdp.go:82`). Branch 1 would have been dead code and branch 2 would have denied every
   instance-scope action to everyone, permanently. It also made `D-SuperAdminFold`'s "super-admin
   role" incoherent: instance admin is a separate authority plane precisely because
   `assignment.grant` is unit-scoped and the first admin must grant before any assignment exists.
3. **A freshly seeded instance would have been unadministrable.** M10 deletes
   `bootstrap-admin-person`, seeds no person or account, and go-oikumenea's JIT is
   link-on-match-only. Nobody could ever log in. The obvious fix — a seeded shell account with a
   fixed email — is worse: combined with `D-SeedBootstrap`'s deterministic RIDs it would ship the
   same pre-linked admin address to every deployment of an open-source repo. Resolved as a
   boot-time seed from install config, with identity carved out as the one exception to
   determinism.
4. **The grant cache was ported without the backstop it documents depending on.** Upstream:
   *"The RLS backstop underneath is exact/live, so a stale ALLOW cannot read revoked-away rows"*
   (`grantcache.go:15-17`, TTL 2s). M10 drops RLS, so the window would have had no floor — and
   `D-SeedBootstrap` makes "a migration edits a base role" the *normal* path for authority changes,
   which is exactly the out-of-band case the epoch bump does not cover locally. Resolved by
   dropping the cache and reading grants per request. The two reviews disagreed here; the source
   settled it.
5. **`SearchSites` leaks position through its filter.** `ST_DWithin` and the KNN ordering run on
   exact geometry while `Coarsen` is applied app-side, so result-set membership is a boolean oracle
   on a `hidden` site's true position. Inherited from upstream, but M10 would have ported it while
   claiming the opposite property, and its verification step checked the wrong invariant — the
   coordinates genuinely never leave the process; the answers derived from them do.
6. **The closure lock is a row lock, not an advisory lock** —
   `SELECT id FROM tenant_graphs WHERE id = $1 FOR NO KEY UPDATE`, held to commit. With one
   authority-bearing graph, every unit creation in the product serialises on a single row. Recorded
   with a binding invariant: no network call, geocode or external fetch while it is held.

Also corrected: the estimate (**~12–15k LOC Go and ~3–3.5k migrations**, not ~7–8k and ~1.5k — the
qualitative claim about generated code dominating the headline 267k is confirmed and unchanged);
the Conjure surface (**~25 endpoints**, since the 9-operation figure counted only what the admin app
calls *today* and omitted the entire super-admin set); the RID wire format (the `ofm:` prefix is
dropped — bare uuids, since a prefixed value cannot both render at the boundary and leave existing
contracts untouched, and one such value is a public URL path segment); `ClosurePort` (named
explicitly — it is what makes M10.3 independent of M10.4 and resolves the authz↔directory cycle);
the authorization entry point (`authz.Require(ctx, …)`, subject from context, because a subject
parameter is an oracle safe only by call-site convention — the same defect class this repo already
fixed at M2.3 and M3); and the teardown checklist, which had missed `lib/oikumenea.ts` itself,
user-visible i18n strings in four locales, two of three `.env.example` files, the codegen pipeline
scripts, and the fact that dropping the `oikumenea` schema needs its own contract-phase migration.

**One live gap is knowingly left open.** `RunJurisdictionSync`'s transport resolves `whoami` and
stops (`congregationimport/transport/service.go:209-213`), so any authenticated Google account can
trigger real Unit writes running under the service principal's instance-wide grant. It copied
`RunConnector`'s shape, but `RunConnector`'s justification — *"it makes no go-oikumenea WRITE"* —
does not carry over. A standalone fix ahead of the migration was recommended and deliberately
declined; it is folded into M10.6 and named in M10.9's refusal-proof list. Until M10.6 lands, the
gap is open on `main`.

Finally, `architecture/conventions.md` was corrected — it was not touched by the first Phase 0 pass
and had gone stale in four places (plain-`uuid` PKs, no-RLS as an "accepted gap", authorization
"against go-oikumenea's PDP", and "generated TypeScript does not exist", false since M2.6).

> **`Verified` (2026-08-18).** Coherence-checked per this milestone's own exit criterion, same shape
> as M0/M9: every `[text](path#anchor)` link across the nine changed/added files
> (`README.md`, `docs/README.md`, `docs/architecture/{conventions,decisions,overview}.md`,
> `docs/milestones.md`, `docs/open-questions.md`, `docs/review-result-{1,2}.md`) resolves — checked
> programmatically, both file existence and heading-anchor slugs, not just eyeballed. All four
> decision blocks M10 supersedes (D-CoreDependency, D-Facade, D-InstanceAdminConsole, D-BulkImport)
> carry an append-only `Superseded (M10) by [...]` note with original text left untouched.
> `open-questions.md`'s amended entries (`DS-OFM-1` resolved, `DS-OFM-3` reframed, `DS-OFM-8`
> resolved by deletion, `DS-OFM-12` superseded by new `DS-OFM-15`, `DS-OFM-14` halved, `DS-OFM-15`/
> `16` opened) match this section's own summary paragraph. No dangling reference, no contradiction
> between a new decision and the one it supersedes. M10.1–M10.9 remain `⬜ Not started` — this row's
> `Verified` covers the doc set only, not the code that removes go-oikumenea.
