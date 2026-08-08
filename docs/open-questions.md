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
  today. See [discovery.md](modules/discovery.md#open-seams).
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
- **DS-OFM-10 — `hermenea`'s input file schema and row-attribution.** Format (CSV/JSON), field set,
  and how an imported row's congregation-contact person is resolved to a go-oikumenea person RID
  for attribution are all undecided — pick when M2.2 is actually built. See
  [import.md](modules/import.md#open-seams).
- **DS-OFM-11 — `oikumenea-console`'s network exposure.** Needs a real restriction (VPN, IP
  allowlist, protected subdomain) given its instance-wide power — not a bare public host port the
  way `openfaithmap-web`/`openfaithmap-admin` deliberately are. Pick when M1.2 is actually built.
  See [D-InstanceAdminConsole](architecture/decisions.md#d-instanceadminconsole--reuse-go-oikumeneas-own-console-as-the-third-super-admin-only-surface).
