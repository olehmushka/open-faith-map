# Open questions

The live backlog of deferred seams — parked items, each promotable to a milestone once it matters.
Resolved seams are removed from here; their outcomes move into
[`architecture/decisions.md`](architecture/decisions.md). Each item below is also cross-referenced
from the module doc it was raised in.

- **DS-OFM-1 — No RLS-based isolation on OpenFaithMap's own tables.** OpenFaithMap's database has
  no equivalent of go-oikumenea's `app.readable_units` GUC/RLS backstop; access control for its
  own `content`/`moderation`/`vouching` tables is app-layer only, checked against a live
  go-oikumenea authority probe. Accepted for now (see
  [conventions.md](architecture/conventions.md)); revisit if OpenFaithMap's own data ever needs a
  second line of defense the way go-oikumenea's does.
- **DS-OFM-2 — Discovery cache refresh cadence — resolved for MVP (M4, 2026-08-10): lazy-only, no
  scheduled job.** `GET /search` refreshes `discovery_site_cache` purely as a side effect of a
  cache miss, via the now-unblocked service principal (M2.5, `oikumenea:0.0.2`). A future
  proactive refresh — a timer, or a go-oikumenea webhook once one exists — remains a real option,
  deliberately not designed speculatively; revisit only if real traffic shows lazy refresh leaves
  results stale in practice. See [discovery.md](modules/discovery.md#open-seams).
- **DS-OFM-3 — Location-scoped role assignments** (e.g. a "campus admin" scoped below a
  congregation unit). go-oikumenea's own religion module doc reserves this as its `DS-50`;
  OpenFaithMap has no workaround until it's picked up upstream. See
  [core-integration.md](modules/core-integration.md#open-seams).
- **DS-OFM-4 — Guarantor eligibility threshold for vouching.** MVP ships with no threshold (any
  admin with current standing may vouch); tighten only if real abuse patterns appear. See
  [vouching.md](modules/vouching.md#open-seams).
- **DS-OFM-5 — Full-text content search.** Searching page/post bodies (not just location) has no
  owner yet — deferred until content volume justifies a dedicated index. See
  [content.md](modules/content.md#open-seams).
- **DS-OFM-6 — Automated exclusion enforcement beyond registration-time.** The D-Exclusions taxon
  check runs once, at intake; detecting a congregation that later re-affiliates with an excluded
  body has no design. See [moderation.md](modules/moderation.md#open-seams).
- **DS-OFM-7 — Locale-switching UX for the public site.** Content translation groups
  ([content.md](modules/content.md)) support multi-locale structurally; the visitor-facing locale
  picker/detection UX is undesigned. See [web-facade.md](modules/web-facade.md#open-seams).
- **DS-OFM-8 — Taxon-level exclusion has no go-oikumenea-native home.** Currently facade-side only
  (OpenFaithMap is the only consumer needing it). If a second consuming app ever needs the same
  "block this whole tradition" behavior, it becomes a real go-oikumenea feature request rather than
  something each consumer reimplements. See
  [core-integration.md](modules/core-integration.md#open-seams).
- **DS-OFM-12 — Backfilling `moderation_actions` into go-oikumenea's audit trail.** D-Moderation's
  Correction drops the single-ledger goal because go-oikumenea's audit module has no write endpoint.
  If one ever ships upstream, backfilling becomes a real option — recorded so the intent isn't lost,
  not as a commitment. See [moderation.md](modules/moderation.md).
- **DS-OFM-14 — Per-surface OAuth clients, and WireGuard in front of `oikumenea-console`.** Both are
  recorded as required before any non-local-dev deployment
  ([D-OAuthClients](architecture/decisions.md),
  [D-InstanceAdminConsole](architecture/decisions.md)). ~~Neither has a milestone — because there
  is no deployment milestone at all yet. Whoever creates one inherits both.~~
  **Confirmed (2026-08-11): deliberately not decoupled further right now.** Per-surface OAuth
  clients stay a real prerequisite once a deployment milestone exists. WireGuard specifically is
  infrastructure-shaped, not application-shaped — its concrete choice (WireGuard vs. something
  else) depends on wherever the instance actually ends up hosted, which doesn't exist yet either.
  ~~No action to take here until that milestone is scoped; recorded so this isn't silently reopened
  as if it were undecided.~~
  **Resolved (2026-08-14): that milestone now exists —
  [M9](milestones.md#m9--production-deployment-single-cheap-vm) · D-ProductionDeployment
  (`architecture/decisions.md`).** Both items are scheduled there as M9's own build-phase work,
  still deliberately provider-agnostic (the concrete VM provider remains undecided). Still open in
  practice — nothing is provisioned yet — just no longer homeless.
