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
- **DS-OFM-2 — Discovery cache refresh cadence.** Scheduled polling via the service principal vs. a
  future go-oikumenea webhook for religion-site changes — go-oikumenea has no outbound webhook
  today. **Downstream of a larger question:** scheduled polling via the service principal is not
  currently possible at all (M1.1 item 2), and it is unverified whether *anonymous* reads work
  either — see M2.5. Cadence is only worth deciding once that reports. See
  [discovery.md](modules/discovery.md#open-seams).
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
- **DS-OFM-9 — M7 (hardening) is unscoped.** Rate limiting on anonymous report/registration
  endpoints, moderation-queue UX at real volume, and observability are all named but not designed —
  first real work item once M1–M6 are live. See [milestones.md](milestones.md#m7--hardening--real-user-feedback-idea-stage).
  **Note the sequencing risk:** M5 is what actually ships the public, unauthenticated
  `POST /reports` and `POST /exclusion-check` endpoints, a milestone before the hardening that
  protects them. Three other items people might expect here (CI, least-privilege DB role,
  `openfaithmap-api`'s host ports) moved forward into **M2.4** at the 2026-08-09 audit, because they
  gate every intervening milestone's Verified rather than being end-state polish.
- **DS-OFM-10 — A future scraped-church-data importer has no name, schema, or design.** M2.2's
  original congregation-bulk-import scenario is real (importing an existing directory of churches
  from a legacy system or scraped source) but is not `hermenea` — that name belongs to
  go-oikumenea's own reference-data companion, deployed unmodified for M2.2 (see D-BulkImport's
  Correction in [decisions.md](architecture/decisions.md)). A narrower, separately-named tool near
  `openfaithmap-api` is expected to address it eventually; input format, field set, row-attribution
  (how an imported congregation's real-world contact is resolved to a go-oikumenea person RID, if
  at all), and connector shape are all undecided — pick when that milestone is actually scoped. See
  [import.md](modules/import.md#open-seams).
- **DS-OFM-11 — `registration_requests` uses a `uuid` PK, not a composed URN RID.** A deviation from
  [conventions.md](architecture/conventions.md)'s inherited "composed URN RID primary keys" rule,
  unnoticed until the 2026-08-09 audit. Harmless in isolation; the question is whether M3's
  `content_*` tables follow `registration`'s uuid precedent or the stated convention. Pick one and
  make them consistent before there are four modules doing two different things.
- **DS-OFM-12 — Backfilling `moderation_actions` into go-oikumenea's audit trail.** D-Moderation's
  Correction drops the single-ledger goal because go-oikumenea's audit module has no write endpoint.
  If one ever ships upstream, backfilling becomes a real option — recorded so the intent isn't lost,
  not as a commitment. See [moderation.md](modules/moderation.md).
- **DS-OFM-13 — Cross-module foreign keys inside `openfaithmap-api` are undecided.**
  `discovery_site_cache.content_site_id` is a real FK into `content_sites`.
  [conventions.md](architecture/conventions.md) covers cross-*service* references (opaque TEXT, no
  FK) and Go-level module boundaries (interface calls, domain events), but says nothing about a
  foreign key between two modules' tables in one schema. Either it's fine (one schema, one
  deployable, one migration set) or the cache should hold an opaque local id like it does for every
  go-oikumenea reference. Settle before M3/M4 add more. See
  [discovery.md](modules/discovery.md#open-seams).
- **DS-OFM-15 — `Impersonation` is specified nowhere and contradicts a binding invariant.** The
  [glossary](glossary.md) defines it (a moderator logging in as a congregation admin for support
  debugging, time-limited and banner-visible) and nothing else in the doc set mentions it: no
  `D-<Name>`, no endpoint in [moderation.md](modules/moderation.md)'s API surface, no milestone. It
  also contradicts [core-integration.md](modules/core-integration.md)'s **no-on-behalf-of**
  invariant head-on — OpenFaithMap's backend never presents a credential to act as a specific
  person. Any real version would have to be go-oikumenea minting the impersonated session (a
  feature it does not have today), not OpenFaithMap forging one. **Decide or delete;** do not build
  it from the glossary entry. Found by the 2026-08-09 audit.
- **DS-OFM-14 — Per-surface OAuth clients, and WireGuard in front of `oikumenea-console`.** Both are
  recorded as required before any non-local-dev deployment
  ([D-OAuthClients](architecture/decisions.md),
  [D-InstanceAdminConsole](architecture/decisions.md)), and neither has a milestone — because there
  is no deployment milestone at all yet. Whoever creates one inherits both.
