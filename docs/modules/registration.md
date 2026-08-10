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
- **Reparenting job** (M4.1) — one attempt at moving an already-`APPROVED` request's congregation
  unit onto a different jurisdiction unit: `PENDING → NEW_EDGE_ADDED → OLD_EDGE_REMOVED → VERIFIED`,
  or `FAILED` with an error. OpenFaithMap-local; go-oikumenea has no equivalent (its own graph state
  is the thing being moved, not tracked as a job).

## Data model

Conventions per [conventions.md](../architecture/conventions.md). No cross-database FKs — every
go-oikumenea RID here (`taxon_id`, `country_id`, `submitted_by_person_id`,
`decided_by_person_id`, `created_unit_id`, `jurisdiction_unit_id`, and the reparenting job's unit
ids) is an opaque TEXT foreign value.

**`registration_requests`** (`migrations/0001_registration.sql`, extended by
`0006_registration_jurisdiction.sql`) — id (uuid), submitter/taxon/country RIDs, the submitted
address fields + `latitude`/`longitude`, `status`, the decision fields (`decided_by_person_id`,
`decided_at`, `rejection_reason`, `created_unit_id`), and `jurisdiction_unit_id` (nullable — the
operator's approval-time choice; not updated by a later re-parent) — a CHECK constraint enforces
each status's required fields are present (e.g. `APPROVED` always carries `created_unit_id`).

**`jurisdiction_reparenting_jobs`** (`migrations/0006_registration_jurisdiction.sql`) — id (uuid),
`registration_request_id` FK (`ON DELETE SET NULL`), `congregation_unit_id`/`old_parent_unit_id`/
`new_parent_unit_id`, `status`, `performed_by_person_id`, `error` (set only when `FAILED`). A partial
unique index on `congregation_unit_id WHERE status <> 'FAILED'` allows at most one live job per unit
at a time.

## Conjure API surface

`RegistrationService` (`api/registration.conjure.yml`, `base-path: /registration/v1`):

| Op | Intent | Gate |
|---|---|---|
| `POST /requests` | Submit a request as the caller (their own resolved go-oikumenea person RID — never client-supplied, always asked of go-oikumenea's own `whoami`). Runs the D-Exclusions check first. | Authenticated |
| `GET /requests` | List — every request for an operator (a target-scoped `Authorize` check against the root unit, D-PlatformModerator), else just the caller's own | Authenticated |
| `GET /requests/{id}` | Read one request | Submitter or operator (same target-scoped check) |
| `POST /requests/{id}/approve` | Approve a `PENDING` request: `createChildOrg` under `jurisdictionUnitId` (or the shared root unit if omitted — M4.1, D-JurisdictionUnits), a location + site, a filled Position, and a `unit`-scoped grant of `congregation-admin` to the submitter — all with the **caller's own forwarded token** | go-oikumenea's PDP decides for real (`religionorg.manage`/`site.manage`/`assignment.grant` on the target unit) |
| `POST /requests/{id}/reject` | Reject with a reason. No go-oikumenea writes. | Authenticated |
| `POST /requests/{id}/reparent` | Start or resume moving an `APPROVED` request's congregation unit onto `newParentUnitId` — a resumable `addEdge`-then-`removeEdge` on the `canonical` graph, add-before-remove (D-JurisdictionUnits) | Same target-scoped operator check as `approveRequest`/`listRequests` |
| `GET /requests/{id}/reparent` | Read the most recent reparenting job for the request, if any | Same as `getRequest` |

## Known defects (audit 2026-08-09)

**Fixed by [milestones.md](../milestones.md)'s M2.3, which also blocks M2's `Verified`.**
The module below is built and works on its happy path (proven by curl at M2). Three defects were
found by reading it against this doc, none of which the happy-path proof would have surfaced. They
are recorded here rather than only in `milestones.md` because this doc is what the next person
reads before touching the module.

**1 · The operator gate is untargeted, and leaks every submitter's PII.** ~~`application.IsOperator`
asks go-oikumenea `MyCapabilities()` whether the caller holds `religionorg.manage` — with **no target
unit**. `scripts/bootstrap-registration-org` grants `religionorg.manage` as part of the
**`congregation-admin`** role. So every approved congregation admin passes the operator check and
`GET /requests` returns them every pending submission platform-wide.~~ **Fixed.** `MyCapabilities()`
is deliberately flat and self-only (confirmed against go-oikumenea's own SDK/docs — U1, now resolved)
and cannot be targeted. `IsOperator` now calls go-oikumenea's `Authorize` (`POST /authorize`) with
`{SubjectPersonId: caller, Action: "religionorg.manage", UnitId: &RootUnitID}`. `Authorize` itself
requires the caller to already hold `assignment.read` reaching the target unit, no self-exemption
(go-oikumenea's own "OQ-5", deliberate) — so `scripts/bootstrap-registration-org` now also grants
`registration-operator` that permission (and reconciles it onto an already-bootstrapped instance's
existing role via `UpdateRole`, not just on a fresh one). `congregation-admin` does **not** get
`assignment.read` — that's what makes it correctly get `Authorization:PermissionDenied` (read by
`IsOperator` as "not an operator") instead of a real `Allow`/`Deny` answer.

**2 · `getRequest` has no authorization at all.** ~~`transport.GetRequest` calls the application
service directly — no `whoami`, no operator check, no submitter comparison. Any authenticated
person can read any request by id.~~ **Fixed.** `GetRequest` now resolves the caller via `whoami`
first; `application.Get` permits iff the caller is the request's `submittedByPersonId` or passes the
same `IsOperator` check as item 1, and returns `domain.ErrNotFound` (mapped to
`Registration:RequestNotFound`, not a distinct "forbidden") otherwise — so the endpoint never confirms
the existence of a request the caller can't see.

**3 · `approveRequest` is a non-atomic distributed write.** ~~Seven go-oikumenea calls followed by a
local `UPDATE`, with no compensation and no idempotency key.~~ **Fixed.** A new `PROVISIONING`
status (`migrations/0002_registration_provisioning.sql`) is persisted, with `created_unit_id`, as
soon as `createChildOrg` returns — the one step that can't be re-derived — so a retry resumes from
the real unit instead of creating a second org. The remaining steps are re-runnable:
`ensurePosition`/`ensureFilled`/`ensureGrant` treat go-oikumenea's own `PositionConflict` /
`PositionAlreadyFilled` / `AssignmentConflict` errors on a repeat call as success; `ensureSite` checks
`listUnitSites` for an existing primary site first, because `createSite` has **no** uniqueness key to
reject a duplicate on — a gap the original ticket text didn't account for.

All three defects are now fixed in code. Live proof against the two-real-token acceptance criterion
(a `congregation-admin`-only account sees only its own requests; a `registration-operator` account
sees all) has not been run yet — that needs a real browser Google OAuth login, not something
achievable headlessly. See `milestones.md`'s M2.3 "As implemented" note.

## Dependencies

- **Calls:** go-oikumenea's `religion` (taxon reads, `createChildOrg`, classification, sites),
  `location` (address), `membership` (position), `authorization` (`Authorize`,
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

**For writes**, `IsOperator`'s result is cosmetic: it decides what a page renders, never what a write
is allowed to do. The real enforcement is `approveRequest`'s actual `createChildOrg`/`grantAssignment`
calls, made with the caller's own token — go-oikumenea's PDP re-decides every one, so this module's
local gate permits nothing on its own, regardless of what it answers.

**For reads, the same `IsOperator` call is the entire access-control decision** — `GET /requests` and
`GET /requests/{id}` (`application.List`/`Get`) call it directly and there is no PDP behind either to
catch a wrong local answer, because the rows being read are OpenFaithMap's own, not go-oikumenea's.
That's why `IsOperator` calls go-oikumenea's real `Authorize` (`POST /authorize`), target-scoped to
`Config.RootUnitID`, rather than the flat, untargeted `MyCapabilities()` it used before M2.3 — a
capability check this module makes about *its own* tables must be target-scoped and load-bearing, not
cosmetic. [D-PlatformModerator](../architecture/decisions.md) generalizes this to every
OpenFaithMap-owned module. One dependency worth naming: `Authorize` requires the caller to already
hold `assignment.read` reaching the target unit (go-oikumenea's own "OQ-5", no self-exemption), which
is why `scripts/bootstrap-registration-org` grants it to `registration-operator` — without that grant,
`IsOperator` fails closed (denies real operators too, never grants anyone extra access).

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
- **Single shared root organization — resolved at M4.1.** Every congregation was a child of one flat
  OpenFaithMap org (not one root per denomination/jurisdiction); see
  [D-FlatRoot](../architecture/decisions.md) for the M2 simplification and
  [D-JurisdictionUnits](../architecture/decisions.md) for what replaced it. `approveRequest` now
  accepts an optional `jurisdictionUnitId`, operator-chosen **at approval time** (never a submission
  field — the public `/register` wizard is unchanged) — omitted falls back to the original flat-root
  behavior unchanged. A new `reparentRequest`/`getReparentStatus` pair moves an already-`APPROVED`
  congregation onto a different jurisdiction, as a resumable job (`jurisdiction_reparenting_jobs`),
  since go-oikumenea has no single atomic "move" call for a religion `Unit`.
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
