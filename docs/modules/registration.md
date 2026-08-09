# Module: registration

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [architecture/decisions](../architecture/decisions.md)
> Table prefix: `openfaithmap.registration_*`

## Purpose

Congregation self-service registration (M2): a prospective admin submits a request (tradition,
name, address, coordinates); a **registration operator** — a real person holding
`religionorg.manage`/`assignment.grant` on the shared root unit, verified live against
go-oikumenea's PDP, never a locally-cached role — reviews and approves or rejects it. Approval
performs the real go-oikumenea writes with the **operator's own forwarded token**, never a
service-principal on-behalf-of write (D-Facade, D-CoreDependency's no-on-behalf-of invariant).

This is OpenFaithMap's **first real schema** — a correction to `milestones.md`'s original M2
framing ("no dedicated schema of its own — writes go through go-oikumenea directly"), found while
building it: go-oikumenea has no "draft org" or self-service org-creation concept a brand-new,
ungranted user can invoke (see the authority-bootstrapping finding below), so a request needs
somewhere to live between submission and an operator's decision.

Also runs the **D-Exclusions taxon check** at submission time (walking the taxon's ancestor chain
against the named exclusion list) — pulled forward from `moderation.md`'s original plan (which
assumed this landed in M5) because M2 needed it now and M5 doesn't exist yet. `moderation.md`
still owns the *concept* (D-Exclusions); this module owns today's only *implementation* of the
check, cross-referenced here until M5 exists to consolidate it.

## The authority-bootstrapping finding

go-oikumenea's real permission model has no path for a brand-new, ungranted person to create a new
top-level organization or gain authority over one: `POST /religion-orgs` needs
`religion.catalog.manage` (instance-wide, instance-admin-only in practice — see D-InstanceAdmin in
go-oikumenea's own decisions.md), and granting anyone authority over a unit needs `assignment.grant`
**on that unit**, which not even an instance admin holds automatically (instance-admin only
auto-passes *instance-scope* checks; unit-scoped ones still need a real assignment — verified
against `internal/authorization/application/service.go`'s `GrantAssignment`, whose only ungated
path is the internal system/bootstrap-seed one, never reachable via the API).

**Resolution, mirroring go-oikumenea's own `D-Bootstrap` one level down:** `scripts/bootstrap-registration-org`
creates a single shared root organization (a project-specific simplification — every congregation
registers as a *child* unit of one flat OpenFaithMap org, not one root per denomination) and two
Roles (`registration-operator`, `congregation-admin`), then prints the one out-of-band SQL insert
needed to seed the *first* assignment on that brand-new unit — the same "operator-owned DB access"
trust level go-oikumenea's own `D-Bootstrap` uses for the first instance admin. After that single
seed, everything else — including every future congregation admin's own grant — flows through the
normal API, granted by whichever real person the operating org designates as a registration
operator (the seeded person, to start).

## Entities & aggregates

- **Registration request** — a prospective admin's submission: tradition (a go-oikumenea
  `religion_taxa` RID), congregation name, address/coordinates, and its lifecycle status
  (`PENDING` → `APPROVED` | `REJECTED`). OpenFaithMap-local; go-oikumenea has no equivalent.

## Data model

Conventions per [conventions.md](../architecture/conventions.md). No cross-database FKs — every
go-oikumenea RID here (`taxon_id`, `country_id`, `submitted_by_person_id`,
`decided_by_person_id`, `created_unit_id`) is an opaque TEXT foreign value.

**`registration_requests`** (`migrations/0001_registration.sql`) — id (uuid), submitter/taxon/
country RIDs, the submitted address fields + `latitude`/`longitude`, `status`, and the decision
fields (`decided_by_person_id`, `decided_at`, `rejection_reason`, `created_unit_id`) — a CHECK
constraint enforces each status's required fields are present (e.g. `APPROVED` always carries
`created_unit_id`).

## Conjure API surface

`RegistrationService` (`api/registration.conjure.yml`, `base-path: /registration/v1`):

| Op | Intent | Gate |
|---|---|---|
| `POST /requests` | Submit a request as the caller (their own resolved go-oikumenea person RID — never client-supplied, always asked of go-oikumenea's own `whoami`). Runs the D-Exclusions check first. | Authenticated |
| `GET /requests` | List — every request for an operator (checked live via `MyCapabilities`), else just the caller's own | Authenticated ⚠️ **broken — see Known defects** |
| `GET /requests/{id}` | Read one request | ⚠️ **ungated in code — see Known defects** |
| `POST /requests/{id}/approve` | Approve a `PENDING` request: `createChildOrg` under the shared root unit, a location + site, a filled Position, and a `unit`-scoped grant of `congregation-admin` to the submitter — all with the **caller's own forwarded token** | go-oikumenea's PDP decides for real (`religionorg.manage`/`site.manage`/`assignment.grant` on the root unit) |
| `POST /requests/{id}/reject` | Reject with a reason. No go-oikumenea writes. | Authenticated |

## Known defects (audit 2026-08-09)

**Fixed by [milestones.md](../milestones.md)'s M2.3, which also blocks M2's `Verified`.**
The module below is built and works on its happy path (proven by curl at M2). Three defects were
found by reading it against this doc, none of which the happy-path proof would have surfaced. They
are recorded here rather than only in `milestones.md` because this doc is what the next person
reads before touching the module.

**1 · The operator gate is untargeted, and leaks every submitter's PII.**
`application.IsOperator` asks go-oikumenea `MyCapabilities()` whether the caller holds
`religionorg.manage` — with **no target unit**. `scripts/bootstrap-registration-org` grants
`religionorg.manage` as part of the **`congregation-admin`** role. So every approved congregation
admin passes the operator check and `GET /requests` returns them every pending submission
platform-wide: congregation names, street addresses, latitude/longitude, and submitter person RIDs.

The "Authorization touchpoints" section below calls this gate "cosmetic only, matching
go-oikumenea's own `D-SelfCapabilities` framing." That reasoning holds for **writes** — approve and
reject are re-decided by go-oikumenea's PDP no matter what the list endpoint rendered — and does not
hold for **reads**, where this check *is* the access-control decision and there is no PDP behind it
to catch a wrong answer. The fix is a target-scoped check against the root unit
([D-PlatformModerator](../architecture/decisions.md) sets the pattern for every module).

**2 · `getRequest` has no authorization at all.** `transport.GetRequest` calls the application
service directly — no `whoami`, no operator check, no submitter comparison. Any authenticated
person can read any request by id. The Conjure contract's own docs say "The submitter or an
operator (verified live) may read it"; neither half is implemented.

**3 · `approveRequest` is a non-atomic distributed write.** Seven go-oikumenea calls followed by a
local `UPDATE`, with no compensation and no idempotency key. Any failure after `createChildOrg`
orphans a real unit (possibly with a location, site, and position attached) while the request stays
`PENDING` — and retrying creates a *second* org, because `slugCode` appends a random suffix so
nothing collides. The invariants below are all true of a successful approval; none of them say
anything about a partial one.

## Dependencies

- **Calls:** go-oikumenea's `religion` (taxon reads, `createChildOrg`, classification, sites),
  `location` (address), `membership` (position), `authorization` (`MyCapabilities`,
  `grantAssignment`), `identityfederation` (`whoami`) — via the Go SDK,
  `internal/coreintegration.NewUserClient` bound to the caller's forwarded token. Never the
  service-principal path for anything in this module — every write is the real person's own
  authority.
- **Called by:** `openfaithmap-admin`'s `/register` (submit), `/admin/registrations` (operator
  approve/reject), `/my-congregation` (reads an approved request's `createdUnitId`, then calls
  go-oikumenea directly for the roster) — via a hand-written fetch client
  (`web/apps/admin/lib/registration.ts`), not a generated TypeScript SDK (openfaithmap-api has no TS
  codegen pipeline yet — see open seams).

## Authorization touchpoints

This module defines no permission codes of its own — see the API surface table above.

**For writes**, the `listRequests`/operator-view gate is genuinely cosmetic (matching go-oikumenea's
own `D-SelfCapabilities` framing): it decides what a page renders, never what a write is allowed to
do. The real enforcement is `approveRequest`'s actual `createChildOrg`/`grantAssignment` calls, made
with the caller's own token — go-oikumenea's PDP re-decides every one, so this module's local gate
permits nothing on its own.

**For reads, the same gate is the entire access-control decision, and it is currently wrong** — see
Known defects above. There is no PDP behind `GET /requests` or `GET /requests/{id}` to catch a
wrong local answer, because the rows being read are OpenFaithMap's own, not go-oikumenea's. Any
capability check this module makes about *its own* tables must be target-scoped and must be treated
as load-bearing, not cosmetic. [D-PlatformModerator](../architecture/decisions.md) generalizes this
to every OpenFaithMap-owned module.

## Invariants

- **No on-behalf-of writes.** `approveRequest` never uses a service-principal or elevated
  credential — the operator's own forwarded token performs every go-oikumenea call, so an operator
  can only approve what go-oikumenea's PDP already lets them.
- **A congregation admin's grant is always `unit`-scoped**, never `subtree` — approving one
  request never gives its submitter reach over any other congregation, even though the shared root
  unit's own operator grants are `subtree`.
- **The submitter's person RID is never client-supplied** — always resolved from the caller's own
  forwarded token via go-oikumenea's `whoami`, both at submission and at every subsequent
  operator-gate check.

## Open seams

- **No generated TypeScript SDK for openfaithmap-api.** `web/apps/admin/lib/registration.ts` is a
  hand-written fetch client — go-oikumenea's TS-SDK pipeline (`scripts/gen-ts-client.sh`,
  `tools/ir2openapi`, `conjure-typescript`) doesn't exist in this repo yet. Revisit once a second
  module needs the same treatment.
- **No sqlc for this module's queries** — `internal/registration/adapters/store.go` is hand-written
  pgx, deviating from D-Stack's "pgx + sqlc" convention. A deliberate simplification for this
  module's small, single-table surface; revisit if the query count grows.
- **Single shared root organization.** Every congregation is a child of one flat OpenFaithMap org
  (not one root per denomination/jurisdiction). **Resolved** into
  [D-FlatRoot](../architecture/decisions.md), which accepts it for now and requires real
  jurisdiction units before M5 ([milestones.md](../milestones.md)'s M4.1) — moderation's
  `jurisdiction` queue scope and D-Exclusions' org-level backstop both need an ancestor chain that
  does not exist under a flat root. M4.1 also changes this module: a submission gains a jurisdiction
  selection, and `approveRequest` targets that unit instead of the single root.
- **No submitter-facing status list beyond `listRequests`'s own-submissions fallback** — a
  submitter sees their requests via the same list endpoint an operator uses; no dedicated
  "my submissions" UX polish yet.
- **`churchSiteTypeID` falls back to the first available site type** if go-oikumenea's seeded
  `church` code ever changes — worth a config value instead of a hardcoded code if that catalog
  becomes operator-editable. Note the fallback is silent: an instance whose first site type is not a
  church would attach every congregation to the wrong type with no error. Prefer failing loudly.
- **D-Exclusions' check lives here, not in `moderation.md`** (see Purpose) — consolidate when M5
  lands.
- **No unit tests.** `checkNotExcluded`'s ancestor walk, `slugCode`, and the status-transition
  guards have zero coverage; the repo's only test is the skipped `coreintegration` integration
  test, so `go test ./...` passes vacuously. First tests land with M2.3.
- **The D-Exclusions check runs under the submitter's own token.** `checkNotExcluded` calls
  `Religion.GetTaxon` through the caller's client, which works because a real person's token can
  read the taxonomy. There is no machine-callable path (M1.1 item 2), so any future non-interactive
  caller of this check — a scheduled re-validation, a bulk importer — has no mechanism today. See
  [milestones.md](../milestones.md)'s M2.5.
