# Module: moderation

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [architecture/decisions](../architecture/decisions.md)
> Table prefix: `openfaithmap.moderation_*`

## Purpose

Owns reports, moderator decisions, and appeals across everything OpenFaithMap surfaces — a
congregation's content, its claimed identity, or a vouching relationship — plus a standalone
**taxon-level denomination-exclusion check** dry-run (`POST /exclusion-check`, D-Exclusions).
`internal/registration`'s own inline check (added at M2, before this module existed) remains the
one actually enforced at registration time — see this doc's Open seams for why the two are not yet
consolidated. No equivalent exists in go-oikumenea.

`openfaithmap.moderation_actions` is this platform's **ledger of record** for moderation decisions:
append-only, `reject_mutation()`-guarded. It is *not* mirrored into go-oikumenea's audit trail —
see D-Moderation's **Correction**: `audit.write` does not exist and go-oikumenea's audit module has
no write endpoint at all, so the original "exactly one append-only ledger platform-wide" goal was
never buildable. Incident response reads two logs (go-oikumenea's, for everything a forwarded token
did *through* go-oikumenea; this one, for OpenFaithMap's own decisions), correlated by person RID
and timestamp.

## Blocked dependencies (audit 2026-08-09)

This module cannot pass its `designed` gate again until these clear. Two of the three that the
audit found were resolved into decisions; one remains real work.

| Dependency | Status |
|---|---|
| **Who is a moderator?** `moderation.read`/`moderation.act` gate every endpoint here and four in [vouching.md](vouching.md), but were specified only as "held by a small, fixed set of accounts" — no table, no role, no mechanism. | **Resolved.** [D-PlatformModerator](../architecture/decisions.md): a go-oikumenea `platform-moderator` Role, granted `subtree` on the shared root unit, resolved by a target-scoped capability check. No OpenFaithMap roster table. `scripts/bootstrap-registration-org` gains the role at M5. |
| **The audit mirror doesn't exist.** | **Resolved.** D-Moderation's Correction; see Purpose above and the withdrawn invariant below. |
| **`queue_scope = 'jurisdiction'` has no ancestor chain to walk.** Under [D-FlatRoot](../architecture/decisions.md) every congregation is a direct child of one shared root, so `jurisdiction` and `platform` are the same set. | **Resolved.** [D-JurisdictionUnits](../architecture/decisions.md) (M4.1): real, operator-assigned jurisdiction units exist and existing congregations can be re-parented onto one. `jurisdiction` scope now has a real ancestor chain to walk for any congregation a jurisdiction was actually assigned to — still legitimately equal to `platform` for a congregation with none, by design (jurisdiction is optional, never inferred). Wiring that walk into `GET /reports?scope=` is itself deferred — see Open seams below. |

**Built (2026-08-10).** `internal/moderation` (domain/adapters/application/transport),
`api/moderation.conjure.yml`, `migrations/0007_moderation.sql`, and a moderator console +
public report form in `web/apps/{admin,web}` — see [milestones.md](../milestones.md#m5--moderation)
for the stage-board detail. Not yet Verified: needs a green CI run on `main` at the merge commit and
a live two-real-token proof (a non-moderator refused, a `platform-moderator`-granted caller allowed).

## Entities & aggregates

- **Report** — a flag raised by anyone (including an anonymous visitor) against a site, a piece of
  content, a congregation's claimed identity, or a vouching edge.
- **Moderation action** — an immutable record of a moderator decision. Reversible within a grace
  window; the reversal is itself a new action row, never an edit of the original.
- **Appeal** — a congregation admin's structured challenge to an action affecting them, routed to a
  different moderator than the one who took the original action.
- **Denomination-exclusion check** — not a stored entity; a synchronous evaluation run inline
  during the congregation-provisioning flow (step 1 of
  [core-integration.md](core-integration.md#provisioning-a-congregation-the-core-end-to-end-flow)).

## Data model

Conventions per [conventions.md](../architecture/conventions.md).

**`moderation_reports`**
- `id` PK (RID)
- `target_kind TEXT NOT NULL CHECK (target_kind IN ('site','document','congregation','vouching_edge'))`
- `target_ref TEXT NOT NULL` — the RID of the reported thing (local `content_*` RID, or a
  go-oikumenea unit RID for `congregation`)
- `reason_code TEXT NOT NULL` — catalog-typed; `other` accepts free text (see the doctrinal-dispute
  rule below)
- `detail TEXT`
- `reporter_person_rid TEXT` — nullable; a reporter need not be logged in
- `queue_scope TEXT NOT NULL CHECK (queue_scope IN ('platform','congregation','jurisdiction'))`
- `status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','actioned','dismissed'))`
- `created_at`, `updated_at`, `deleted_at`

**`moderation_actions`** (append-only, `reject_mutation()` guarded — no `UPDATE`/`DELETE` ever)
- `id` PK (RID)
- `report_id` FK → `moderation_reports` (nullable — a moderator may act without a prior report,
  e.g. proactively enforcing D-Exclusions)
- `action_kind TEXT NOT NULL CHECK (action_kind IN ('hide','suspend','archive','warn_admin','revoke_vouch','reverse'))`
- `target_kind`, `target_ref` — same shape as `moderation_reports`
- `actor_person_rid TEXT NOT NULL` — the moderator (a go-oikumenea `Person` RID)
- `reason TEXT NOT NULL`
- `reversed_by_action_id` FK → `moderation_actions` (nullable; set on the *original* row once a
  `reverse` action targets it)
- `created_at`

**`moderation_appeals`**
- `id` PK (RID)
- `action_id` FK → `moderation_actions`
- `congregation_admin_person_rid TEXT NOT NULL`
- `statement TEXT NOT NULL`
- `assigned_moderator_person_rid TEXT` — never the original action's actor
- `status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','upheld','overturned'))`
- `created_at`, `updated_at`

## Conjure API surface

`ModerationService` (`/moderation/v1`), `api/moderation.conjure.yml`:

| Op | Intent | Perm |
|---|---|---|
| `POST /reports` | File a report (anonymous allowed) | none (public, rate-limited) |
| `GET /reports?scope=&status=` | List reports in the caller's queue scope | `moderation.read` (platform or congregation/jurisdiction scope) |
| `POST /reports/{id}/actions` | Take an action against a report's target | `moderation.act` |
| `POST /actions` | Take a proactive action (no prior report) | `moderation.act` (platform) |
| `POST /actions/{id}/reverse` | Reverse a prior action within the grace window | `moderation.act` |
| `POST /actions/{id}/appeals` | File an appeal against an action | (the affected congregation admin, verified via go-oikumenea authority check on the unit) |
| `GET·POST /appeals/{id}/decide` | List / decide an appeal | `moderation.act` (must differ from the original actor) |
| `POST /exclusion-check` | Run the D-Exclusions taxon check ahead of registration (used by the registration wizard in [web-admin](web-admin.md), and callable standalone for a dry-run) | none (public) |

`moderation.read`/`moderation.act` are OpenFaithMap-defined *names* for a target-scoped capability
check against go-oikumenea's PDP — specifically, does the caller hold the `platform-moderator`
Role's authority on the **shared root unit** (D-PlatformModerator)? Moderation is platform-wide, and
`subtree` scope on the root unit is exactly how "platform-wide" is expressed in go-oikumenea's model,
so this needs no parallel RBAC. *(Audit 2026-08-09: this previously read "held by platform
moderators — a small, fixed set of accounts, not modeled through go-oikumenea's per-unit
authorization," which named no mechanism at all and left both this module and M6 blocked on an
undesigned primitive.)*

## Dependencies

- **Calls:** go-oikumenea's `religion` module (`religion_taxa` ancestor lookups for the exclusion
  check — note this has **no machine-callable path today**, see
  [core-integration.md](core-integration.md)'s open seams and M2.5; today's only implementation, in
  `registration`, runs under a real submitter's token); [core-integration.md](core-integration.md)
  for the congregation-admin-identity check on appeals and the target-scoped moderator capability
  check (D-PlatformModerator). **Not** go-oikumenea's `audit` module — see Purpose.
- **Called by:** the [web-facade](web-facade.md) (public report-filing UI — filing a report
  requires no login) and [web-admin](web-admin.md) (appeal filing, moderator queue UI — both
  require being logged in); the congregation-registration flow
  ([core-integration.md](core-integration.md), step 1).

## Authorization touchpoints

`moderation.read`, `moderation.act` — both resolve to the caller's authority on the shared root
unit, via the `platform-moderator` Role (D-PlatformModerator). Derived from a real go-oikumenea role
assignment, not a local roster; the check must always name the root unit as its target, never ask
"does this caller hold P anywhere," which is the defect
[registration.md](registration.md#known-defects-audit-2026-08-09) documents.

The exclusion check (`POST /exclusion-check`) is unauthenticated by design — it must run *before*
anyone has proven any standing, as part of deciding whether registration can even begin. **Note the
tension with M2.5:** an unauthenticated caller cannot reach go-oikumenea's taxon reads today (every
`religion` read is `RequireAnywhere`-gated), so this endpoint as specified has no data source until
that resolves. `registration`'s own check sidesteps it by running under the authenticated
submitter's token.

## Invariants

- **Doctrinal disputes are not adjudicated.** `doctrinal_concern` is not a `reason_code`.
  Doctrinally-framed reports are filed under `other` with free text; moderators decline to act on
  doctrinal grounds alone — the only doctrinal line drawn platform-wide is
  [D-Exclusions](../architecture/decisions.md), and that's enforced at registration, not via the
  report queue.
- **Every action is reversible within its grace window**, and a reversal is itself a new,
  append-only row — `moderation_actions` is never edited or deleted in place
  (`reject_mutation()`-guarded, matching go-oikumenea's own convention for its append-only tables).
- **An appeal is never decided by its action's original actor.** Enforced at write time
  (`assigned_moderator_person_rid <> (the original action's actor_person_rid)`), not left to
  moderator discipline.
- **The exclusion check is re-run at registration, never cached from a prior visit.** A taxon's
  exclusion status could in principle change (a future ADR revision); nothing about it is
  memoized client-side.
- ~~**Every moderation action is mirrored into go-oikumenea's audit ledger** before it is considered
  complete.~~ **Withdrawn (audit 2026-08-09).** The endpoint this invariant assumed does not exist —
  see D-Moderation's Correction. **Replacement:** *the `moderation_actions` row is written before
  the action's effect is applied*, so a crash mid-action leaves a recorded decision with an
  unapplied effect (recoverable, reviewable) rather than an applied effect with no record
  (invisible). The row is append-only and `reject_mutation()`-guarded, which is what made the
  original mirror mostly redundant anyway.

## Open seams

- **Rate limiting on anonymous report filing** is parked at M7 (`DS-OFM-9`), but **this module is
  what ships the public endpoints** — `POST /reports` and `POST /exclusion-check` are both
  unauthenticated. Decided at M5 scoping: stays deferred to M7 as originally planned — no basic
  limiting added here.
- **`queue_scope = 'jurisdiction'` still has no query implementation wired to it, and this is now a
  deliberate scope cut, not just an unwired dependency.** M4.1 gave the ancestor chain a real target
  to walk, but M5's only moderator role — `platform-moderator` (D-PlatformModerator) — is granted
  `subtree` on the shared **root** unit, not scoped to any individual jurisdiction. There is no
  moderator authority boundary a jurisdiction-scoped query would actually enforce yet, so every
  report filed by this milestone's code lands in `PLATFORM` regardless of its target's jurisdiction
  (`internal/moderation/application/service.go`'s `FileReport`). The `queue_scope` column, its enum
  values, and `GET /reports?scope=`'s filter all exist and work mechanically — walking
  `Tenant.unitAncestors` to actually classify a report by jurisdiction is real future work, gated on
  a future milestone giving jurisdiction-scoped moderator authority a reason to exist.
- **No real go-oikumenea-side or content-side effect is wired to an action's `action_kind` yet.**
  `HIDE`/`SUSPEND`/`ARCHIVE`/`WARN_ADMIN`/`REVOKE_VOUCH` are recorded in `moderation_actions` as the
  decision of record (D-Moderation's Correction), but none of them yet causes go-oikumenea to change
  a unit's real state, or `content`/`discovery` to actually hide something from public view — this
  module's own spec didn't detail that mechanism in enough depth to build blind, so it's recorded
  here rather than guessed at.
- **Appeal filing only supports `CONGREGATION`-kind actions.** Appealing a `SITE`/`DOCUMENT` action
  would need resolving through `content`'s own site → congregation-unit mapping first (the same
  shape as `discovery`'s `ContentResolver` interface-call cross-module dependency), which this PR
  doesn't wire up — `FileAppeal` returns `Forbidden` for those rather than guessing a unit.
- **Registration's own `checkNotExcluded` is untouched** — this module's `POST /exclusion-check` is
  a new, standalone endpoint reusing `registration`'s `domain.ExcludedTaxonCodes` list directly
  (import, not copy), but the two call sites remain independent. Consolidating them so
  `registration.Submit` calls through this module's check instead of running its own copy is still
  the "when M5 lands" aspiration this doc originally named — not done in this PR, to keep the change
  scoped to moderation's own new surface.
- **Automated exclusion enforcement beyond registration-time** (e.g., detecting a congregation that
  quietly re-affiliates with an excluded body after registration) has no design yet — today the
  check only runs once, at intake.
- **Cross-congregation pattern detection** (the same bad actor registering many fake claims) is
  explicitly deferred — the original FaithMap design never built this either.
