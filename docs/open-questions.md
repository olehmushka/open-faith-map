# Open questions

The live backlog of deferred seams — parked items, each promotable to a milestone once it matters.
Resolved seams are removed from here; their outcomes move into
[`architecture/decisions.md`](architecture/decisions.md). Each item below is also cross-referenced
from the module doc it was raised in.

- **DS-OFM-1 — No RLS-based isolation on OpenFaithMap's own tables.** ~~OpenFaithMap's database has
  no equivalent of go-oikumenea's `app.readable_units` GUC/RLS backstop; access control for its
  own `content`/`moderation`/`vouching` tables is app-layer only, checked against a live
  go-oikumenea authority probe. Accepted for now (see
  [conventions.md](architecture/conventions.md)); revisit if OpenFaithMap's own data ever needs a
  second line of defense the way go-oikumenea's does.~~
  **Resolved (2026-08-17, M10): deliberately no — D-InProcessAuthz (`architecture/decisions.md`).**
  The question changed shape when M10 absorbed the core: it is no longer "OpenFaithMap lacks what
  go-oikumenea has," it is "OpenFaithMap owns every table and chose app-layer-only for all of them."
  Upstream's predicate (`authz_unit_in_reach`) is keyed on the authorization reach graph rather than
  on a tenant, so it *would* port cleanly — the reason not to is that it needs session-GUC plumbing
  (`app.person_id`, `app.is_instance_admin`) on every pooled connection, and it backstops the very
  PDP M10 is porting. Recorded as decided rather than open, with the port path known if the answer
  ever changes.
  **Corrected 2026-08-18, after review found the original reasoning incomplete.** Declining RLS is
  only coherent alongside a second change the plan originally got wrong: M10 also **drops the grant
  cache** and reads grants per request. Upstream's cache documents its own safety condition —
  *"The RLS backstop underneath is exact/live, so a stale ALLOW cannot read revoked-away rows"*
  (`grantcache.go:15-17`, TTL 2s). Porting the cache while dropping the backstop would have left an
  out-of-band revocation — raw SQL, an incident-response `UPDATE`, or a migration editing a base
  role, which `D-SeedBootstrap` makes the *normal* path — invisible for two seconds with nothing
  underneath. One indexed join per authenticated request removes the window entirely at this scale.
  The seam stays closed; the reason it is safe to close is different from what was first written.
- **DS-OFM-2 — Discovery cache refresh cadence — resolved for MVP (M4, 2026-08-10): lazy-only, no
  scheduled job.** `GET /search` refreshes `discovery_site_cache` purely as a side effect of a
  cache miss, via the now-unblocked service principal (M2.5, `oikumenea:0.0.2`). A future
  proactive refresh — a timer, or a go-oikumenea webhook once one exists — remains a real option,
  deliberately not designed speculatively; revisit only if real traffic shows lazy refresh leaves
  results stale in practice. See [discovery.md](modules/discovery.md#open-seams).
- **DS-OFM-3 — Location-scoped role assignments** (e.g. a "campus admin" scoped below a
  congregation unit). ~~go-oikumenea's own religion module doc reserves this as its `DS-50`;
  OpenFaithMap has no workaround until it's picked up upstream.~~ **Reframed (2026-08-17, M10):
  still open, but it is now ours to build rather than ours to wait for.** D-OwnCore makes
  `authz_role_assignments` an OpenFaithMap table, so the two-value `scope` enum (`unit` / `subtree`)
  is a schema we control. Adding a location-scoped third case is a migration plus a PDP branch, not
  an upstream feature request. Deliberately not built in M10 — the port's rule is to preserve
  behaviour, not extend it. See [core-integration.md](modules/core-integration.md#open-seams).
- **DS-OFM-4 — Guarantor eligibility threshold for vouching.** MVP ships with no threshold (any
  admin with current standing may vouch); tighten only if real abuse patterns appear. See
  [vouching.md](modules/vouching.md#open-seams).
- **DS-OFM-5 — Full-text content search.** Searching page/post bodies (not just location) has no
  owner yet — deferred until content volume justifies a dedicated index. See
  [content.md](modules/content.md#open-seams).
- **DS-OFM-6 — Automated exclusion enforcement beyond registration-time.** The D-Exclusions taxon
  check runs once, at intake; detecting a congregation that later re-affiliates with an excluded
  body has no design. See [moderation.md](modules/moderation.md#open-seams).
- **DS-OFM-7 — Locale-switching UX for the public site.** ~~Content translation groups
  ([content.md](modules/content.md)) support multi-locale structurally; the visitor-facing locale
  picker/detection UX is undesigned. See [web-facade.md](modules/web-facade.md#open-seams).
  Scheduled (2026-08-27, M14.14): a visitor-facing picker offering only locales with a published
  variant, `hreflang` alternates, and an editor-side translation panel per document.~~ **Resolved
  (2026-08-30, M14.14): built.** A per-page, in-content locale picker (never the shared site
  header/footer, which has no single translatable document behind the root feed) offering only
  `PUBLISHED` translation-group siblings, `hreflang` alternates via the page route's own
  `generateMetadata`, and an editor-side Translations panel with a "create translation" action. See
  [milestones-2026-08-26-now.md](milestones-2026-08-26-now.md#m1414--locale-switching--closes-ds-ofm-7).
- **DS-OFM-8 — Taxon-level exclusion has no go-oikumenea-native home.** ~~Currently facade-side only
  (OpenFaithMap is the only consumer needing it). If a second consuming app ever needs the same
  "block this whole tradition" behavior, it becomes a real go-oikumenea feature request rather than
  something each consumer reimplements.~~ **Resolved by deletion (2026-08-17, M10): there is no
  "facade side" any more.** D-OwnCore leaves exactly one place this could live, and it lives there.
  The hypothetical second consuming app is gone with the shared core. See
  [core-integration.md](modules/core-integration.md#open-seams).
- **DS-OFM-12 — Backfilling `moderation_actions` into go-oikumenea's audit trail.** ~~D-Moderation's
  Correction drops the single-ledger goal because go-oikumenea's audit module has no write endpoint.
  If one ever ships upstream, backfilling becomes a real option — recorded so the intent isn't lost,
  not as a commitment.~~ **Superseded (2026-08-17, M10) by `DS-OFM-15`.** There is no upstream audit
  trail to backfill into; D-OwnCore declines to port `internal/audit` at all. The single-ledger
  intent survives, now as a question about a ledger this project would build itself. See
  [moderation.md](modules/moderation.md).
- **DS-OFM-15 — No audit log in the owned core.** D-OwnCore deliberately does not port
  go-oikumenea's `internal/audit` (1,538 LOC, a monthly-partitioned `audit_log`, and two partition
  maintenance functions) — nothing in OpenFaithMap ever read it, and `moderation_actions` already
  provides the append-only trail the product actually surfaces. The gap is real, though: mutations
  to units, role assignments, taxa and sites now leave no record beyond the row's own
  `created_by`/`updated_by` columns. Promote this to a milestone if operator accountability over the
  new super-admin screens (D-SuperAdminFold) turns out to matter, and fold `DS-OFM-12`'s
  single-ledger intent into it when it does.
- **DS-OFM-17 — No first-party media storage.** Opened 2026-08-27 (M14.0),
  [D-ExternalMediaOnly](architecture/decisions.md#d-externalmediaonly--congregations-host-their-own-media-no-first-party-uploads):
  congregations host images externally (Google Drive, Dropbox, OneDrive, or any direct URL) — no
  upload endpoint, no object storage, no processing pipeline anywhere in the M14 arc. Failure mode:
  a vendor can change or throttle a hotlinked URL with no warning (**U15**, unmeasured at real
  volume); M14.3 mitigates by normalizing known share-link hosts and preserving the original URL
  alongside the normalized one, so a normalizer fix is a re-derivation, not data loss. Escalation
  path: a future first-party `media` module — nothing in M14's schema forecloses adding one later,
  since the URL field it would populate already exists. See
  [content.md](modules/content.md#open-seams).
- **DS-OFM-16 — Background writes are unattributable.** D-DirectTokenVerification deletes the
  service-principal concept, so work with no human subject (discovery cache refresh,
  `POST /exclusion-check`, scheduled imports) runs under `authz.SystemContext()` and records no
  principal identity. Acceptable while the only such paths are idempotent cache/lookup work; it
  becomes a real problem the moment a background path writes something a human would be asked to
  justify. Coupled to `DS-OFM-15` — attribution is only worth adding if there is somewhere to
  attribute it to.
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
  [M9](milestones-2026-08-07-2026-08-26.md#m9--production-deployment-single-cheap-vm) · D-ProductionDeployment
  (`architecture/decisions.md`).** Both items are scheduled there as M9's own build-phase work,
  still deliberately provider-agnostic (the concrete VM provider remains undecided). Still open in
  practice — nothing is provisioned yet — just no longer homeless.
  **Halved (2026-08-17, M10): the WireGuard requirement is resolved by deletion.** D-SuperAdminFold
  removes `oikumenea-console`, which was the only surface that must never hold a public port; its
  screens move inside `openfaithmap-admin` behind a role. Nothing is left to put a VPN in front of.
  The per-surface OAuth-client half remains open and gets easier: three surfaces become two, and the
  shared-client problem disappears with the console that shared it.
