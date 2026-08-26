# Module: hardening

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [architecture/decisions](../architecture/decisions.md) · [moderation](moderation.md)
> Table prefix: none — no new tables (see Data model)

## Purpose

A cross-cutting hardening pass, not a data-owning domain — the first of three concerns
`DS-OFM-9`/`U10` named for M7 and left undesigned. [D-Hardening](../architecture/decisions.md)
covers the first two; this doc's own design covers the third:

1. **Rate limiting** on `moderation`'s two genuinely anonymous write endpoints (`POST /reports`,
   `POST /exclusion-check`) — the abuse surface M5 shipped one milestone before anything protected
   it.
2. **Observability** — a small, fixed set of app-defined metrics on top of witchcraft's
   already-auto-wired logging/metrics stack.
3. **The moderation-queue pagination defect** — `ListReports`/`ListAppeals` already declare
   `pageToken`/`nextPageToken` on the wire (`api/moderation.conjure.yml`'s `ReportPage`/
   `AppealPage`, shipped at M5) but the transport layer receives an incoming `pageToken` and never
   reads it — always `LIMIT`-only, `nextPageToken` always unset
   (`internal/moderation/adapters/report_store.go`'s own doc comment already admits this). This is
   a correctness/UX defect at real queue volume, not a security hole — it stays inside this
   milestone rather than becoming its own numbered one (unlike M2.3's three defects, which blocked
   M2's `Verified`; this blocks nothing today, it just silently ignores a client-supplied cursor).

**Scope boundary — what M7 does *not* cover, and why**, since three concerns this broad invite
scope creep:

- **Read-side rate limiting** on `ContentPublicService`'s 4 public GETs or
  `DiscoveryPublicService`'s `GET /search` is out. Both are read-only, cacheable
  (`discovery_site_cache`), and not the abuse vector `DS-OFM-9`/`U10` named — those named the two
  anonymous *writes* specifically. Revisit only if real traffic shows a problem, the same
  discipline DS-OFM-2 already used for discovery-cache refresh.
- **New moderation-queue filters** (target kind, date range) are out — the two that exist
  (`scope`, `status`) are what the shipped console UI actually uses; adding more before a moderator
  has asked for them is speculative.
- **Multi-replica rate-limit coordination** (a shared store like Redis) is out —
  `openfaithmap-api` runs single-replica today; revisit only if that changes.
- **Numeric tuning** of the rate-limit thresholds, and any dashboard/alerting on the new metrics,
  are out — ship a conservative, explicitly provisional default; only real traffic can tune it
  correctly.

This is the one correction M7's own prior "idea stage" framing needed: it assumed nothing could be
scoped until real usage existed. That premise held for the numeric tuning above, but not for
rate-limiting a known anonymous-write surface or fixing an already-diagnosed pagination defect —
both are ordinary engineering hardening, found by direct code inspection, not by user behavior. See
[milestones-2026-08-07-2026-08-26.md](../milestones-2026-08-07-2026-08-26.md)'s M7 section for the corrected framing.

## Entities & aggregates

None. The only new concept is an opaque page cursor (see Data model) — not a stored entity, not a
go-oikumenea-facing RID.

## Data model

No new tables. Two new composite indexes, added via a new expand-only Atlas migration, back the
keyset pagination fix:

- `moderation_reports(queue_scope, status, created_at DESC, id DESC)`
- `moderation_appeals(status, created_at DESC, id DESC)` (mirroring whatever columns
  `ListAppeals` actually filters on today)

This is why M7 genuinely earns the **Migrated** gate later, not just Backend.

Rate-limit and metrics state are explicitly **➖** — deliberately not persisted. The limiter's
token buckets live in an in-process map, scoped to `openfaithmap-api`'s own process lifetime; they
reset on restart and do not coordinate across replicas (Open seams).

**Page cursor.** Not a table — an opaque token returned in `nextPageToken` and accepted back in
`pageToken`. Encodes `(created_at, id)` of the last row in the previous page: base64 of a small
JSON object. This is the first real cursor implementation anywhere in this repo (no existing module
has one — `registration`'s `ListRequests` set the LIMIT-only precedent `moderation` copied), so
there is no prior convention to match, only one to set for future modules.

## Conjure API surface

No contract change for rate limiting. The `429 Too Many Requests` response from
`POST /moderation/v1/reports` and `POST /moderation/v1/exclusion-check` is **not** a Conjure-typed
error — Conjure's fixed error-code system has no code that maps to HTTP 429, so a genuinely
Conjure-typed 429 isn't expressible in this stack today. The middleware that enforces the limit
sits in front of the generated handler and writes a raw `429` + `Retry-After` header + a small,
non-Conjure JSON body directly — documented in `api/moderation.conjure.yml`'s own comments as a
deliberate, permanent exception to this repo's Conjure-error-body convention, not a gap to
"eventually fix."

No new fields for pagination — `ReportPage`/`AppealPage`'s `nextPageToken` and `ListReports`'/
`ListAppeals`' `pageToken` already exist on the wire from M5. This work makes them actually
function, plus enforces a `maxPageSize` clamp that doesn't exist today (`pageSizeOrDefault` in
`internal/moderation/transport/service.go` currently only guards `nil`/`<=0`, with no upper bound).
A malformed/tampered `pageToken` is rejected with `Moderation:InvalidPageToken` — a new named
error, not the generic `Moderation:InvalidArgument` this doc originally assumed existed. Verified
against the actual contract while implementing: `api/moderation.conjure.yml`'s `errors:` block has
only specific named `INVALID_ARGUMENT`-coded errors (`ActionNotReversible`, `AppealActorConflict`,
`TaxonNotFound`, `DoctrinalReasonNotAllowed`), no generic catch-all — `InvalidPageToken` matches
that convention (see milestones-2026-08-07-2026-08-26.md's M7 "As implemented" note).

## Dependencies

None new into go-oikumenea. `CheckExclusion` continues to run under the server's own
service-principal token exactly as M5 built it; rate limiting and pagination are both entirely
`openfaithmap-api`-internal.

## Authorization touchpoints

**Rate limiting runs pre-auth**, by definition — it applies to unauthenticated callers on
`ModerationPublicService` specifically, and never touches `ModerationService`'s authenticated
queue/action/appeal surface. It is not an authorization mechanism and makes no claim about who may
call — only how often an unauthenticated caller may call before being asked to wait.

**Pagination does not change who can call** `GET /reports`/`GET /appeals` — the existing
`whoami` + target-scoped operator/moderator check (`transport.Service.ListReports`) is untouched.
It only changes what a given authorized call returns.

## Invariants

- Rate-limit state is single-process and ephemeral — a restart clears every bucket; this is a
  known, accepted limitation (Data model), not a bug.
- A malformed or tampered `pageToken` always returns `400 Moderation:InvalidPageToken` — it is never
  silently reinterpreted as "start from page 1," which would just be a different flavor of the
  silent-failure class this fix exists to close.
- Rate limiting never touches `ModerationService`'s authenticated surface, or any other module's
  routes — the middleware is wired onto exactly one `RegisterRoutes*` call
  (`RegisterRoutesModerationPublicService`).
- A rate-limited caller receives no information beyond "try again later" (`Retry-After`) — no
  claim about which of `POST /reports`/`POST /exclusion-check` specifically is under load, no
  distinction between "just you" and "everyone."

## Open seams

- **Read-side rate limiting** on content/discovery public GETs — deferred, see Scope boundary.
- **Multi-replica rate-limit coordination** — the in-process map does not survive a restart or
  coordinate across replicas; fine while `openfaithmap-api` is single-replica, a real gap the day
  it isn't.
- **Provisional numeric thresholds** — the rate-limit bucket size/refill rate in
  [D-Hardening](../architecture/decisions.md) is a placeholder, not data-tuned. Revisit once real
  traffic exists.
- **Additional moderation-queue filters** (target kind, date range) — not built until a moderator
  actually asks for one.
- **A queue-depth gauge** — a reasonable future addition (better sampled periodically than counted
  per-request), but has no dashboard consumer to read it yet; not built speculatively, matching
  D-Hardening's own reasoning against a `/metrics` scrape endpoint.
- **Forwarded-for IP trust** — the rate limiter reads `r.RemoteAddr` directly, correct only because
  there is no reverse proxy in front of `openfaithmap-api` today. If one is ever added, this must
  change to trust a specific, known forwarded-for header — never blindly — or the limiter becomes
  trivially bypassable.
