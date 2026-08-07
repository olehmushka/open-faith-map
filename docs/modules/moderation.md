# Module: moderation

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [architecture/decisions](../architecture/decisions.md)
> Table prefix: `openfaithmap.moderation_*`

## Purpose

Owns reports, moderator decisions, and appeals across everything OpenFaithMap surfaces — a
congregation's content, its claimed identity, or a vouching relationship — plus, eventually, the
**taxon-level denomination-exclusion check** (D-Exclusions). **As of M2, that check's only real
implementation lives in [registration.md](registration.md)** — this module (M5) doesn't exist in
code yet, and M2 needed the check now; consolidate here when M5 lands. No equivalent exists in
go-oikumenea; every moderation action this module records is also written through go-oikumenea's
`audit` module (D-Moderation) so there is exactly one append-only ledger platform-wide, not two
logs to reconcile.

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

`moderation.read`/`moderation.act` are OpenFaithMap-defined permission codes, held by platform
moderators — a small, fixed set of accounts, not modeled through go-oikumenea's per-unit
authorization (moderation is a platform-wide role, not tied to any one congregation's authority
graph).

## Dependencies

- **Calls:** go-oikumenea's `audit` module (every `moderation_actions` write is mirrored there,
  service-principal-authenticated — D-Moderation); go-oikumenea's `religion` module (`religion_taxa`
  ancestor lookups for the exclusion check); [core-integration.md](core-integration.md) for the
  congregation-admin-identity check on appeals.
- **Called by:** the [web-facade](web-facade.md) (public report-filing UI — filing a report
  requires no login) and [web-admin](web-admin.md) (appeal filing, moderator queue UI — both
  require being logged in); the congregation-registration flow
  ([core-integration.md](core-integration.md), step 1).

## Authorization touchpoints

`moderation.read`, `moderation.act` (both platform-scoped, held by a small fixed moderator roster
— not derived from go-oikumenea role assignments, since platform moderation authority is
orthogonal to any single congregation's authority graph). The exclusion check
(`POST /exclusion-check`) is unauthenticated by design — it must run *before* anyone has proven any
standing, as part of deciding whether registration can even begin.

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
- **Every moderation action is mirrored into go-oikumenea's audit ledger** before it is considered
  complete — a write that succeeds locally but fails to reach go-oikumenea's audit endpoint is
  retried, not silently dropped (background job, service-principal authenticated).

## Open seams

- **Rate limiting on anonymous report filing** is a hardening item (M7), not designed in detail
  yet — see [milestones.md](../milestones.md).
- **Automated exclusion enforcement beyond registration-time** (e.g., detecting a congregation that
  quietly re-affiliates with an excluded body after registration) has no design yet — today the
  check only runs once, at intake.
- **Cross-congregation pattern detection** (the same bad actor registering many fake claims) is
  explicitly deferred — the original FaithMap design never built this either.
