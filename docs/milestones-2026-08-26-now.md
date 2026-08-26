# Milestones (2026-08-26 – now)

The architecture sequenced into buildable, dependency-ordered milestones. A roadmap, not binding —
[`architecture/decisions.md`](architecture/decisions.md) governs *what*, this governs *in what
order*. Gate definitions are in [`development-process.md`](development-process.md).

## Status

**M0–M13.6 are done** (no row had unbuilt Backend/Migrated/UI work as of 2026-08-26) — see
[`milestones-2026-08-07-2026-08-26.md`](milestones-2026-08-07-2026-08-26.md) for that full history.

**M14 · The site-building arc** is scoped (2026-08-26); **M14.0 is done (2026-08-27)** and no other
sub-milestone is built yet. It is the second half of the product: M4/M13 finished **discovery** (the
map); M14 finishes **presence** (the per-congregation site builder), whose bones shipped at M3/M4
and were never built on. Nineteen sub-milestones, M14.0–M14.18. **M14.0 was the gate for the whole
arc** — it wrote the nine `D-` blocks and the module-doc rewrites, and ruled on **U16** (tightened)
and the M14.10 nav assumption (replaced with a hand-built menu); see
[architecture/decisions.md](architecture/decisions.md) and the Unresolved unknowns table below.
Nothing else in the arc starts before M14.1, which fixes a live stored-XSS hole.

## Unresolved unknowns — read this before building anything

Every place the doc set currently says "we don't actually know." Detail lives where the third
column points; this table exists so nothing is hidden, not to duplicate it.

Everything carried in from the archive has been resolved (see the note below the table). The three
items below are **new, opened by M14's scoping pass on 2026-08-26**, and are open now:

| # | The unknown | Where it bites | Who resolves it |
|---|---|---|---|
| **U14** | **No apex domain is registered and no DNS-provider API token exists.** A wildcard certificate for `*.<apex>` can only be issued over the ACME **DNS-01** challenge — HTTP-01 cannot issue wildcards. D-ProductionDeployment deliberately left the VM/DNS provider undecided; `D-TenantSubdomains` now constrains that choice for the first time (the provider must expose a DNS API Caddy has a module for). | M14.18 only. Every other M14 milestone is verifiable locally against `*.localhost`, which browsers resolve to loopback with no DNS at all. | The owner, by registering a domain and picking a DNS provider. M14.18 carries `🔶` until then — the same honest gate M1.2/M2 already use for the Google OAuth redirect URI. |
| **U15** | **Google Drive hotlink reliability at volume is unmeasured.** `D-ExternalMediaOnly` makes congregations host their own images on Drive/Dropbox/OneDrive. Direct-content URLs for these hosts are undocumented, have been changed by their vendors before, and are throttled under load — none of which we can measure before real congregations use it. | M14.3's normalizer, and every `image`/`gallery` block on every public site thereafter. A vendor-side change breaks images platform-wide at once. | Only real traffic. M14.3 mitigates rather than resolves: the original URL is preserved alongside the normalized one, so a normalizer fix is a re-derivation, not a data-loss event. Escalation path is the first-party `media` module (`DS-OFM-17`). |
| **U16** | ~~**A registration operator can edit any congregation's website.** Not new — [content.md](modules/content.md#authorization-touchpoints) has recorded it since M3, as a consequence of `content.manage` reusing `religionorg.manage`, which `registration-operator` holds as a subtree grant on the shared root. What is new is the **stakes**: after M14.9 a "site" is a real website on its own subdomain, not an unlinked blob of blocks.~~ **Ruled on (2026-08-27, M14.0): tightened, not restated.** [D-TenantSubdomains](architecture/decisions.md#d-tenantsubdomains--subdomain-per-congregation-wildcard-tls-and-a-reserved-slug-blocklist) decides `content.manage` becomes its own per-unit permission, granted to `congregation-admin` only; operators keep just the existing moderation path. Still 🔶 in practice — the decision is written, but the code isn't: implementation is scheduled to M14.9, which is when the stakes this row named actually go live. | Every `content.manage`-gated write in the arc — which is most of it. | **Decided by the owner (tighten), designed by M14.0.** Code lands at M14.9; a second real identity is still needed to test the new per-unit denial path once it exists (M2.3's own known limitation, unresolved by this ruling). |

**Carried in from the archive, all resolved — empty as of 2026-08-26:**

- **Group 1** (U2, U3 — must be measured against a real instance): both were actually resolved back
  in 2026-08-09/2026-08-10 (M2.5 and M4.1 each measured their own question and built on the answer)
  but were never struck in the original table — caught and corrected in the archive while splitting
  this file.
- **Group 2** (U7 — deferred decisions): resolved 2026-08-26. Cross-module FKs formalized as
  permitted in `architecture/conventions.md` (every module already shares one Postgres
  instance/schema since D-SharedDatabase/M10; the precedent already existed in the schema).
- **Group 3** (U11, U12 — contradictions/orphans): both resolved 2026-08-26. U11:
  `churchSiteTypeID` (both the `registration` and `congregationimport` copies) now fails loudly
  instead of silently falling back to the first available site type. U12: the two settings still
  bypassing `config.Install` (`DATABASE_URL`, `GOOGLE_OAUTH_CLIENT_ID`) are now real, schema-
  validated fields, matching `Environment`'s own M10.2 precedent.

Full detail for all five, including what was actually changed, is in the archive's own
[unresolved-unknowns table](milestones-2026-08-07-2026-08-26.md#unresolved-unknowns--read-this-before-building-anything)
(struck entries with a **Resolved (date)** note).

## Stage board

**Gate legend.** ✅ passed · ⬜ not started · ➖ not applicable · 🔶 **passed once, now blocked on a
named dependency** — always named in that milestone's prose; 🔶 without a named blocker is just ⬜.
`Verified` additionally requires CI green on `main` — see
[development-process.md](development-process.md).

| # | Decided | Designed | Backend | Migrated | UI | Verified | Stage |
|---|---|---|---|---|---|---|---|
| M14 · The site-building arc | ✅ | ✅ | ➖ | ➖ | ➖ | ⬜ | **Scoped jointly with the owner (2026-08-26); M14.0 (2026-08-27) writes the nine `D-` blocks and rules on both open questions below.** A codebase audit plus external research into WordPress and Drupal found the site builder has good bones (typed JSON-Schema-validated blocks, translation groups, draft/published states, an unused `theme` JSONB — M3/M4) and nothing built on them: admins author pages by typing **raw JSON into a textarea**, every published page is dumped onto **one route as a one-pager** keyed by a UUID, there is **no media path anywhere**, `content.catalog.manage` still has **no endpoint**, and `button.href`/`image.url`/`social_embed.url` render with **no scheme validation at all** — a live stored XSS. Twelve owner decisions fixed the shape: full-parity arc; **no media uploads** (external URLs only); **forward revisions**; **structured rich-text nodes, never HTML**; **subdomain per congregation**; **curated contrast-checked theme tokens**; platform subdomains only; all four optional surfaces in scope; one web app with Host middleware, extractable later; contact form to an **in-app inbox, no SMTP**; **publish-on-read** scheduling with no scheduler; wildcard TLS designed but gated. Nineteen sub-milestones below. |
| M14.0 · Decisions + designs for the arc | ✅ | ✅ | ➖ | ➖ | ➖ | ⬜ | **Done (2026-08-27).** Writes the nine `D-` blocks (`D-TenantSubdomains`, `D-ExternalMediaOnly`, `D-RichTextNodes`, `D-ContentRevisions`, `D-CuratedTheme`, `D-SitePatterns`, `D-InAppInbox`, `D-PublishOnRead`, `D-PublicSiteCSP`), rewrites [content.md](modules/content.md) to the fixed template, updates [web-facade.md](modules/web-facade.md)/[web-admin.md](modules/web-admin.md)/[glossary.md](glossary.md), schedules `DS-OFM-7` to M14.14, opens `DS-OFM-17` (no first-party media storage), and rules explicitly on **U16**. Flips the first two columns for every row below. Docs-only — no code. |
| M14.1 · Content security baseline | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | **Fixes the live stored XSS; nothing else in the arc ships first.** URL **scheme** allowlist (`https`/`http`/`mailto`/`tel`) on every URL-bearing block field, enforced at write in `blockvalidation.go`, plus an embed **host** allowlist; defensive re-validation in the renderer because pre-M14.1 rows already exist. `sandbox` + `referrerpolicy` on every iframe. CSP and security headers in both `next.config.ts` files (currently three lines each, zero headers). New invariant: `dangerouslySetInnerHTML` appears in neither app, ever. |
| M14.2 · Rich-text node model | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | A shared `richText` JSON-Schema definition — inline `text` runs carrying `bold`/`italic`/`link` marks, plus `list`/`listItem` — adopted by `paragraph`, `heading`, `quote`, `staff_card.bio` and a new `list` block. The renderer maps nodes to elements, so there is **no HTML parser and no sanitizer**: Drupal's filter-on-output problem is designed out rather than mitigated. Expand-only migration updating those block types' `json_schema`, plus a data migration lifting existing plain strings into single-run nodes. |
| M14.3 · External media URLs, made survivable | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Normalizer for known share-link hosts (Google Drive, Dropbox, OneDrive) → direct-content URL, applied at write with the original preserved (**U15**). `alt` becomes schema-**required** on `image`/`gallery`. `loading="lazy"` + `referrerpolicy` on every rendered image. Editor-side "this URL loaded / did not load" probe **from the browser**, never a server-side fetch — that would be an SSRF surface. Records the future first-party `media` module as a designed seam so adding it later is additive. |
| M14.4 · Schema-driven block forms | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | **The milestone that kills the JSON textarea.** New `content_block_types.ui_schema JSONB` (widget hints, labels, help text, field order) — WordPress's `block.json` lesson: a block's data schema and its editor controls are declared together, so the form is *derived*, never hand-written per type. Generic form renderer over `json_schema` + `ui_schema`. Typed Conjure validation errors land on the offending field instead of the current `?error=` query-string round trip. |
| M14.5 · Inserter + drag-and-drop reorder | ⬜ | ⬜ | ➖ | ➖ | ⬜ | ⬜ | Categorized block inserter with a one-line description per type — curation over choice, the consistent finding in the editor-UX research (13+ undifferentiated types is already past where an editor picks well). Drag-and-drop replaces the integer `position` input, **with keyboard-accessible move-up/move-down as a first-class path**: drag-only reordering is an accessibility failure, not a polish gap. |
| M14.6 · Forward revisions, history, autosave | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | New `content_document_revisions`; a document gains separate *published* and *draft* revision pointers, so **editing a live page never touches what visitors see** (Drupal's forward-revision model). Autosave writes into the draft on a debounce with a visible saved/unsaved indicator — never silently over live content. Publish promotes draft→published. History list with restore. `ContentPublicService` reads the published revision. |
| M14.7 · Preview | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | Renders the draft revision through the **real public renderer** — not a second, drifting preview renderer — reached on the tenant subdomain via a short-lived signed token. `X-Robots-Tag: noindex`, no caching. Device-width toggle. Carries the WordPress CVE lesson directly: untrusted congregation content must never render inside the admin origin, and here it is cross-origin by construction. Depends on M14.6 and M14.9. |
| M14.8 · Editor polish | ⬜ | ⬜ | ➖ | ➖ | ⬜ | ⬜ | Client-side undo/redo history stack. Real empty states ("Start from a template" → M14.13). Inline validation. A mobile-workable editor layout — the current one is a desktop form grid. |
| M14.9 · Tenant subdomain routing (Phase 1) | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Next.js middleware resolves the `Host` header to a site slug and rewrites into an internal `/_sites/[slug]/…` tree; the apex host keeps serving discovery. **Direct `/_sites/*` access from the apex is blocked** (owner's guardrail). **Reserved-subdomain blocklist enforced server-side in the slug validator** — `content_sites.slug` becomes a hostname, so `admin`/`api`/`auth`/`login`/`www`/`app`/`mail`/`static`/`support`/`billing`/… must be unclaimable. 301s from `/congregations/[unitId]`. Rendering code structured as an extractable module for the owner's Phase 2 (`openfaithmap-sites`). **Also implements M14.0's U16 ruling:** `content.manage` stops resolving through `religionorg.manage`'s subtree grant and becomes its own per-unit permission granted to `congregation-admin` (same shape as M13.2's `site.manage`); registration operators lose blanket edit access and keep only the existing moderation path. See [D-TenantSubdomains](architecture/decisions.md#d-tenantsubdomains--subdomain-per-congregation-wildcard-tls-and-a-reserved-slug-blocklist). |
| M14.10 · Navigation + page routes | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | `/[pageSlug]` and nested child routes on the tenant host, honoring the existing 3-level cap. **Nav is a hand-built menu (`content_site_nav_items`), not derived from the page tree** — M14.0 replaced the original page-tree-derivation assumption with an independently-curated menu (label, target document or external URL, sort order); `parent_document_id` still governs page nesting/breadcrumbs, just not the nav itself. Breadcrumbs at depth ≥ 2. |
| M14.11 · Site chrome — header, footer, template parts | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Congregation name, logo URL, nav, and a footer whose contact details and service times are read **live from `religion_sites`/`religion_service_schedules`, never copied** — the existing content.md invariant, restated because a footer is exactly where someone would be tempted to denormalize. Social links. Site-level settings on `content_sites`, not content documents. |
| M14.12 · Curated theme tokens | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Gives the entirely-unused `content_sites.theme` a real schema: accent color from a vetted palette, one of a few font pairings, a spacing scale, header layout, light/dark — WordPress's `theme.json` lesson, a fixed vocabulary rather than CSS. Emitted as CSS custom properties. **A WCAG contrast check rejects a failing combination at write time**, so no congregation can ship an unreadable site. Live theme preview in the admin. |
| M14.13 · Starter patterns + block-type catalog admin | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | New `content_patterns` with WordPress's **unsynced** semantics: an inserted pattern detaches into ordinary blocks and is freely edited. Seeded church-specific starters — Parish home page, Service times, Meet the clergy, Getting here, Feast-day announcement. Finally builds the `content.catalog.manage` endpoints M3 left unbuilt (moderator-gated per D-PlatformModerator), so block types and patterns stop being migration-only. The single biggest onboarding lever in the arc. |
| M14.14 · Locale switching — closes `DS-OFM-7` | ⬜ | ⬜ | ⬜ | ➖ | ⬜ | ⬜ | Visitor-facing locale picker offering **only locales that actually have a published variant**, plus `hreflang` alternates. Editor-side translation panel per document showing which locales exist and their state, with "create translation" seeding a variant into the same translation group. Translation groups have worked structurally since M3 with no UI on either side. |
| M14.15 · Scheduled publishing, no scheduler | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | `content_documents.publish_at` + a `SCHEDULED` state; the public predicate becomes `state = 'PUBLISHED' OR (state = 'SCHEDULED' AND publish_at <= now())`. **Correctness lives in the `WHERE` clause**, so it behaves identically in local dev and on a VM that does not exist yet — no timer, no goroutine, nothing to fire, and no new unattributable background writer (`DS-OFM-16`). The one cost: the admin UI must show **effective** state, not the raw column. |
| M14.16 · Contact form + in-app inbox | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | `content_form_submissions` plus an anonymous write on `ContentPublicService`, reusing `internal/platform/ratelimit` (M7) rather than adding a second limiter. Spam handled without a third party: honeypot field, minimum time-to-submit, per-IP rate limit. Messages screen in `openfaithmap-admin`, `content.manage`-gated. **No SMTP anywhere** — follows D-InviteLinkMVP's precedent exactly. Submission text is untrusted and renders as plain text only. |
| M14.17 · SEO, structured data, caching | ⬜ | ⬜ | ➖ | ➖ | ⬜ | ⬜ | Per-page `<title>`/description/canonical/OG. Per-tenant `sitemap.xml` + `robots.txt`. JSON-LD: `Church` for the site, `Event` for events, `BreadcrumbList`. Replaces `force-dynamic` with tag-based revalidation invalidated on publish — today every anonymous page view re-queries the API, which is both slow and a free DoS lever. |
| M14.18 · Deployment wiring | ⬜ | ⬜ | ➖ | ➖ | ➖ | 🔶 | **Blocked on U14: a registered apex domain + a DNS-provider API token.** Caddyfile with the DNS-01 wildcard block (wildcards cannot be issued over HTTP-01 — a new, real constraint on the provider choice D-ProductionDeployment deliberately left open), wildcard DNS record, HSTS with `includeSubDomains`, per-tenant read rate limiting. Confirms the backup story is unchanged: no blobs to back up, because there are no uploads. Records the `openfaithmap-sites` extraction as the named Phase 2 trigger. |

## Per-milestone detail

### M14 · The site-building arc

**Depends on:** M3/M4 (the `content` module — sites, documents, blocks, block-type catalog),
M13 (`web/apps/web`'s shadcn/Radix setup and component conventions, reused rather than re-chosen),
M7 (`internal/platform/ratelimit`, reused by M14.16). **Leaves deployable:** yes at every M14.x
boundary — each sub-milestone is independently buildable and verifiable, the same discipline
M10.x/M11.x/M12.x/M13.x used. **Blocks:** every M14.x below.

OpenFaithMap's product is two halves. The map — discovery — was finished by M4 and redesigned by
M13. The other half, **presence**, is a per-congregation site builder whose foundations shipped at
M3/M4 and were then left alone for the rest of the archive.

**Discovery** (a codebase audit of `internal/content`, `web/apps/admin`'s site editor and
`web/apps/web`'s public renderer, plus external research into how WordPress and Drupal actually do
site building, the security literature on CMS content and uploads, editor-UX research, and the
infrastructure side) found the bones are genuinely good and nothing was built on them:

- **Authoring is raw JSON.** `web/apps/admin/app/[locale]/admin/sites/[unitId]/documents/[documentId]/page.tsx`
  renders, per block, a numeric `position` input, a block-type `Select`, and a `Textarea`
  containing `JSON.stringify(b.data)`. A volunteer parish secretary is expected to hand-edit JSON.
  There is no per-type form, no inserter, no drag-drop, no preview, no autosave, no undo, and no
  revision history of any kind.
- **The public site is a one-pager.** `web/apps/web/app/[locale]/congregations/[unitId]/page.tsx`
  fetches every published `PAGE` and renders them all inline on a single UUID-keyed route. The
  parent/child page nesting `content_documents` supports is never surfaced as routes. No
  navigation, no header/footer, no `<title>`, no OG tags, no sitemap, no structured data, and
  `export const dynamic = "force-dynamic"` on every render.
- **`content_sites.theme` has never been read.** It has existed since `migrations/0002_content.sql`
  and no code path anywhere writes or applies it.
- **There is a live stored XSS.** `web/apps/web/app/blocks.tsx` renders `data.href` (the `button`
  block), `data.url` (`image`, `social_embed`) and `data.photoUrl` (`staff_card`) with no scheme
  validation at any layer — not at write in `blockvalidation.go`, not at render. A congregation
  admin (or a registration operator, per **U16**) saving `javascript:…` gets script execution on
  the public site. The `youtube_embed` iframe has no `sandbox`. Neither `next.config.ts` sets a
  single security header — both files are three lines long.
- **No media path exists anywhere in the repo.** Grepped: no multipart handler, no upload
  endpoint, no presign, no object-storage client. `image.url` is an external URL an admin types.
- **`content.catalog.manage` still has no endpoint**, exactly as M3 left it — the 13 block types
  are migration-seeded and unmanageable at runtime.

**External research** — what WordPress and Drupal know that this codebase does not:

- **WordPress.** `block.json` declares a block's attribute schema *and* its editor controls
  together, so the editing form is derived rather than hand-written — the direct answer to the JSON
  textarea (M14.4). `theme.json`/Global Styles is a curated token vocabulary (palette, type scale,
  spacing) that lets a user restyle without CSS (M14.12). **Patterns** — pre-built layouts that
  detach on insert — are the onboarding mechanism; nobody faces a blank canvas (M14.13). Templates
  and template parts make header/footer/nav content rather than code (M14.11). Plus post revisions
  with autosave (M14.6), and KSES: HTML filtering that only a trusted capability (`unfiltered_html`)
  escapes — the reminder that an escape hatch is a permission decision, not a convenience.
- **Drupal.** The long Paragraphs-vs-Layout-Builder debate settles on structured components for
  predictable pages and free-form layout only where earned; D-ContentModel already chose structured,
  and the lesson for this arc is **not to drift**. Canvas/Experience Builder 1.0 (default in Drupal
  CMS 2.0 since January 2026) is that model modernized: developer-defined components, drag-drop for
  editors. Two more transfer directly — **text formats** (filter on output, allowlist per role,
  "Full HTML" for administrators only) and **content moderation with forward revisions**, where the
  published revision keeps serving while a draft accumulates edits. OpenFaithMap's single-row state
  flip cannot express that at all (M14.6).
- **Security.** Uploads are the sharpest edge in this space — magic-byte checks over extensions,
  server-side re-encoding, EXIF stripping, SVG sanitization or rejection, a separate cookieless
  serving origin — and the arc sidesteps all of it by not accepting uploads (`D-ExternalMediaOnly`).
  What remains and does apply: URL-scheme allowlisting, CSP with an explicit `frame-src`, sandboxed
  embed iframes, SSRF discipline on any server-side fetch of an admin-supplied URL, and the
  WordPress CVE pattern where untrusted content rendered *inside the admin app* is its own XSS path
  (M14.1, M14.3, M14.7).
- **Editor UX.** The consistent finding is that curation beats choice — intentional defaults and
  guardrails, because an editor facing dozens of undifferentiated block types will not pick well.
  Autosave into a *draft* with an explicit Publish and a visible unsaved-changes state, not silent
  autosave over live content. Undo/redo, revision history, real-time preview with a device toggle,
  and accessibility enforced structurally (required alt text, heading order, contrast-checked
  themes). This weighs more here than in a general CMS: the users are volunteer-run congregations.
- **Infrastructure.** Tag-based cache invalidation on publish instead of per-request rendering;
  sitemap/robots/JSON-LD; and for multi-tenant hostnames, wildcard DNS plus a wildcard certificate
  — which forces DNS-01, because ACME cannot issue a wildcard over HTTP-01 (**U14**).

**Scoped jointly with the owner**, across three rounds of questions (2026-08-26):

| Decision | Call | Consequence for the arc |
|---|---|---|
| Arc size | **Full WordPress-parity** | One long arc, 19 sub-milestones, rather than a small usability pass |
| Media | **No uploads — external URLs only** | The owner is not funding storage; congregations host on Google Drive etc. First-party storage becomes a future module (`DS-OFM-17`), which is why M14.3 must normalize share links (**U15**) |
| Publishing | **Forward revisions + history** | New `content_document_revisions`; the public read path changes shape (M14.6) |
| Rich text | **Structured inline nodes** | Never an HTML string, so there is no sanitizer to get wrong (M14.2) |
| Routing | **Subdomain per congregation** | The owner's own design, chosen over all three offered options: `[slug].<apex>`, wildcard DNS + wildcard TLS, `Host`-header tenant resolution, reserved-slug blocklist, cookies never wildcard-scoped, HSTS. Buys real same-origin isolation between tenants that path-based routing cannot, and makes future CNAME custom domains additive |
| Theming | **Curated tokens, contrast-checked** | A fixed vocabulary, WCAG-gated at write time (M14.12) |
| Custom domains | **Platform subdomains only** | Not built; the subdomain work is deliberately the groundwork |
| Optional surfaces | **All four** | Starter patterns, locale switcher, contact form, scheduled publishing (M14.13–M14.16) |
| App shape | **One web app + Host middleware now; extractable later** | Phase 1 keeps memory inside D-ProductionDeployment's 500MB–1GB VM budget with one build pipeline; Phase 2 splits `openfaithmap-sites` out for process-level blast-radius isolation when budget allows. Guardrail: `/_sites/*` is unreachable from the apex host |
| Contact form | **In-app inbox, no SMTP** | Follows D-InviteLinkMVP's precedent — this stack has no outbound mail and this arc does not add it |
| Scheduling | **Publish-on-read** | No scheduler, no timer, no goroutine; the predicate is in the `WHERE` clause |
| Wildcard TLS | **Designed, milestone gated 🔶** | M14.18 blocks on **U14**; everything else verifies against `*.localhost` |

**Explicitly out of scope for M14**, named rather than silently dropped: first-party media storage
and uploads (`DS-OFM-17`); custom domains via CNAME; the `openfaithmap-sites` extraction (the
owner's Phase 2, triggered by budget and scale, not by this arc); email delivery of any kind;
full-text content search (`DS-OFM-5`, still unowned); and free-form layout building — both the
Drupal consensus and D-ContentModel say structured components, and this arc must not drift there.

**Navigation assumption — resolved at M14.0 (2026-08-27): switched, not confirmed.** Navigation was
originally specified as page-tree-derived with per-page overrides plus appended external links —
a sub-question superseded when the owner replaced the routing options with their own subdomain
design, leaving it an assumption rather than a decision. M14.0 switches M14.10 to a fully
hand-built menu (`content_site_nav_items`) instead of confirming the page-tree assumption.

Sub-milestone build narratives are written **at build time, not at scoping time** — the
M10.x/M11.x/M12.x/M13.x precedent. What follows is each one's scope and its acceptance criteria.

### M14.0 · Decisions + designs for the arc

**Done (2026-08-27).** Docs-only; no code, no migration. This is the `Decided`/`Designed` gate for
every row above it — the rest of the arc (M14.1–M14.18) can now start, each still its own row.

Writes to [`architecture/decisions.md`](architecture/decisions.md), each to the standard
decision/why/why-not/consequences shape:

- **`D-TenantSubdomains`** — subdomain per congregation, wildcard DNS + wildcard TLS, `Host`-header
  resolution, the reserved-slug blocklist, cookies never issued with a wildcard `Domain`, HSTS, and
  the Phase 1 / Phase 2 app-shape split. Must state explicitly that it constrains
  D-ProductionDeployment's deferred provider choice (**U14**) — that is the one thing this decision
  takes away from a later decision, and it should not be discovered later.
- **`D-ExternalMediaOnly`** — no uploads, no first-party storage, why (cost, and the entire upload
  attack surface avoided), why-not (real UX cost to congregations; broken-image failure mode), and
  the future `media` module's shape so it stays additive.
- **`D-RichTextNodes`** — structured nodes over HTML; must name the Drupal filter-on-output
  alternative it rejects and why designing the problem out beats mitigating it.
- **`D-ContentRevisions`** — forward revisions; extends rather than supersedes D-ContentModel.
- **`D-CuratedTheme`** — the token vocabulary and the write-time WCAG gate.
- **`D-SitePatterns`** — unsynced pattern semantics, and moderator governance of the block-type
  catalog (finally giving `content.catalog.manage` something to gate).
- **`D-InAppInbox`** — contact submissions with no email; cites D-InviteLinkMVP's precedent.
- **`D-PublishOnRead`** — scheduling without a scheduler; must reference D-CongregationImport's
  "no new scheduler" call and `DS-OFM-16` (unattributable background writes), both of which this
  avoids rather than accepts.
- **`D-PublicSiteCSP`** — the security-header and embed-allowlist policy, and the
  no-`dangerouslySetInnerHTML` invariant.

Doc rewrites: [content.md](modules/content.md) to the fixed template with the new entities
(revisions, patterns, form submissions, theme schema, `ui_schema`); [web-facade.md](modules/web-facade.md)
gains the tenant-subdomain surface and the `/_sites` boundary; [web-admin.md](modules/web-admin.md)
gains the real editor and the Messages inbox; [glossary.md](glossary.md) gains *pattern*,
*revision*, *tenant subdomain*, *rich-text node*.

[`open-questions.md`](open-questions.md): mark `DS-OFM-7` scheduled to M14.14; open **`DS-OFM-17` —
no first-party media storage**, recording the decision, the failure mode, and the escalation path.

**Acceptance criteria — met.** Nine `D-` blocks exist
([D-TenantSubdomains](../architecture/decisions.md#d-tenantsubdomains--subdomain-per-congregation-wildcard-tls-and-a-reserved-slug-blocklist)
through
[D-PublicSiteCSP](../architecture/decisions.md#d-publicsitecsp--url-scheme-allowlist-embed-allowlist-and-security-headers)).
`content.md` describes the schema M14 will actually build (new entities, tightened authorization,
resolved open seams), not M3's. **U16 is ruled on explicitly: tightened**, not restated —
`content.manage` stops being a byproduct of the operator's subtree grant, decided in
D-TenantSubdomains, implementation scheduled to M14.9 (still 🔶 in the unknowns table above until
that code ships). **The navigation assumption is replaced**, not confirmed — M14.10 now builds a
hand-built menu (`content_site_nav_items`), not a page-tree-derived one.

### M14.1 · Content security baseline

**Not started.** Depends on M14.0 only. **This milestone fixes a hole that is live in `main`
today** and therefore runs before any feature work in the arc.

Write-time validation in `internal/content/application/blockvalidation.go` (which already walks
each block's `json_schema`, so this extends an existing pass rather than adding a new one): a URL
**scheme** allowlist — `https`, `http`, `mailto`, `tel`, nothing else — applied to every
URL-bearing field across every block type, and an embed **host** allowlist (a named YouTube/Vimeo
set for `youtube_embed`, a named set for `social_embed`). A typed `Content:BlockUrlNotAllowed`
error, not a generic validation failure, so the editor can say which field and why.

Render-time re-validation in `web/apps/web/app/blocks.tsx`. This is deliberate belt-and-braces, not
redundancy: rows written before this milestone already exist in every deployed database, and a
future block type added through M14.13's catalog endpoints could reintroduce an unguarded URL
field. `sandbox` (no `allow-same-origin`) and `referrerpolicy="no-referrer"` on every iframe.

Security headers, currently entirely absent, in both `next.config.ts` files: CSP with explicit
`frame-src`/`img-src` allowlists on the public app and a stricter policy on admin, plus
`X-Content-Type-Options: nosniff`, `Referrer-Policy`, `Permissions-Policy`, and
`X-Frame-Options: DENY` on admin.

**Acceptance criteria.** Saving a block with `href: "javascript:alert(1)"` is rejected with a typed
error naming the field. A row carrying that value **inserted directly with SQL** — the pre-existing
data case — renders with the link dropped, not executed. The CSP header is present on a real HTTP
response from both apps, verified against the running stack rather than read from config. No
`dangerouslySetInnerHTML` in either app.

### M14.2 · Rich-text node model

**Not started.** Depends on M14.1 (the URL allowlist that `link` marks will be validated against).

A shared `richText` definition in the block-type schema vocabulary: an ordered array of inline
nodes — `text` runs carrying `bold`/`italic`/`link` marks, plus `list`/`listItem` — validated by
the existing block validator with no new validation machinery. Adopted by `paragraph`, `heading`,
`quote`, `staff_card.bio`, and a new `list` block type. The renderer maps node types to React
elements directly; there is no HTML string anywhere in the pipeline, hence no parser and no
sanitizer. A `link` mark's `href` goes through M14.1's allowlist like any other URL.

Expand-only migration updating those block types' `json_schema`, plus a data migration lifting
existing plain strings into single-text-run nodes. Both are required in the same migration — a
schema that rejects the rows already in the table is not expand-only in any useful sense.

**Acceptance criteria.** Existing pages render identically after the data migration. A paragraph
with a bolded word and an inline link round-trips through save and public render. A `link` mark
with a `javascript:` href is rejected at write.

### M14.3 · External media URLs, made survivable

**Not started.** Depends on M14.1. Exists because of `D-ExternalMediaOnly` (**U15**).

A share-link normalizer applied at write time for known hosts — Google Drive, Dropbox, OneDrive —
mapping a viewer-page URL to its direct-content form. A Drive share link
(`drive.google.com/file/d/<id>/view`) is an HTML page, not an image: pasted into an `image` block
today it renders nothing, with no feedback anywhere. The original URL is stored alongside the
normalized one, so a future normalizer fix is a re-derivation rather than data loss.

`alt` becomes schema-**required** on `image` and `gallery` — structurally enforced rather than
requested, which is the only version of alt text that survives contact with real editors.
`loading="lazy"` and `referrerpolicy` on every rendered image.

An editor-side load probe: the browser attempts the image and the editor reports loaded/not-loaded
inline. **From the browser, never the server** — a server-side fetch of an admin-supplied URL is an
SSRF surface, and this arc adds no such path.

Records the future first-party `media` module as a designed seam (`DS-OFM-17`): what it would own,
and why nothing in M14's schema forecloses it.

**Acceptance criteria.** A pasted Drive share link renders as a real image on the public site. An
unreachable URL is reported as such in the editor before publishing. Saving an `image` block with
no `alt` is rejected. No server-side fetch of a user-supplied URL exists anywhere in the arc.

### M14.4 · Schema-driven block forms

**Not started.** Depends on M14.2 (rich-text fields need a rich-text widget) and M14.3 (URL fields
need the probe widget). **This is the milestone that removes the JSON textarea.**

New `content_block_types.ui_schema JSONB`: widget hints, field labels, help text, and field
ordering, sitting beside the existing `json_schema` — WordPress's `block.json` lesson, that a
block's data shape and its editing affordances belong in one declaration. A generic form renderer
in `web/apps/admin` builds each block's form from the pair, so adding a block type in M14.13
produces a working editor form with no admin-app code change at all.

Typed Conjure validation errors are surfaced on the field that caused them, replacing the current
pattern of redirecting to `?error=Content:BlockDataInvalid` and rendering one generic banner.

**Acceptance criteria.** Every seeded block type is editable with no JSON visible anywhere in the
admin UI. A block type added at runtime through M14.13's catalog endpoint renders a usable form
without a redeploy. A validation failure highlights the offending field.

### M14.5 · Inserter + drag-and-drop reorder

**Not started.** Depends on M14.4.

A categorized block inserter with a one-line description per type. The editor-UX research is
consistent that curation beats choice — an editor facing dozens of undifferentiated types will not
pick the right one — and the catalog is already 13 types and about to grow.

Drag-and-drop replaces the integer `position` input. **Keyboard-accessible move-up/move-down is
built in the same milestone, as a first-class path, not a follow-up** — a drag-only reorder control
is inaccessible to keyboard and screen-reader users, which for a platform whose public sites carry
accessibility badges would be an unusually poor look.

**Acceptance criteria.** Blocks reorder by drag and by keyboard alone, both persisting. The
`position` integer never appears in the UI. The inserter is operable by keyboard.

### M14.6 · Forward revisions, history, autosave

**Not started.** Depends on M14.4. The largest schema change in the arc.

New `content_document_revisions`: `document_id`, `revision_no`, a blocks snapshot, author,
`created_at`, optional label. `content_documents` gains separate **published** and **draft**
revision pointers. Editing a live page mutates the draft revision only, so what visitors see cannot
change until an explicit publish — Drupal's forward-revision model, and the reason today's
single-row state flip is genuinely risky for a congregation editing its own homepage.

Autosave writes into the draft on a debounce, with a visible saved/unsaved indicator. The UX
research is explicit that autosave belongs on a draft with an explicit publish, never silently over
live content. Publish promotes draft → published. A history list offers restore.
`ContentPublicService` reads the published revision.

**Acceptance criteria.** Publish a page; edit it heavily without publishing; **the public subdomain
still serves the old text**. Publish; it flips. Restore a prior revision; the public site follows.
Autosave survives a browser refresh mid-edit without having touched the live page.

### M14.7 · Preview

**Not started.** Depends on M14.6 (there must be a draft revision to preview) and M14.9 (the tenant
origin to preview it on).

The draft revision rendered through the **real public renderer**. A second preview renderer is the
standard way this feature rots — two code paths that must agree and slowly stop agreeing — so
there is exactly one. Reached on the tenant subdomain with a short-lived signed preview token.
`X-Robots-Tag: noindex` and no caching on preview responses. Device-width toggle.

The security shape is the point, not incidental: the WordPress CVE pattern is a low-privileged user
injecting content that executes when a higher-privileged user views it **in the admin origin**.
Previewing on the tenant origin makes that cross-origin by construction.

**Acceptance criteria.** A draft renders pixel-identically to how it will publish. The preview URL
is unusable once expired, and unusable by someone without the token. `noindex` present. No
congregation content is ever rendered inside the admin origin.

### M14.8 · Editor polish

**Not started.** Depends on M14.5. No backend change.

Client-side undo/redo over the editor's block list. Real empty states — an empty site offers "Start
from a template" (M14.13) rather than an empty form. Inline validation throughout. An editor layout
that works on a phone; the current one is a desktop form grid, and a volunteer updating service
times on a Sunday morning is not at a desk.

**Acceptance criteria.** Undo/redo across add, delete, reorder and edit. The editor is usable at
375px width. No `?error=` query-string round trips remain.

### M14.9 · Tenant subdomain routing (Phase 1)

**Not started.** Depends on M14.0 (`D-TenantSubdomains`). Implements the owner's routing design.

A `web/apps/web/middleware.ts` resolving the `Host` header to a site slug and rewriting into an
internal `/_sites/[slug]/…` tree; the apex host continues to serve discovery, search and the
registration entry point. **Direct `/_sites/*` access from the apex host must 404** — the owner's
guardrail against internal-route leakage, and the thing that keeps the two host shapes from
collapsing into one.

A **reserved-subdomain blocklist enforced server-side in the slug validator**, because
`content_sites.slug` stops being a path segment and becomes a hostname the moment this ships:
`admin`, `api`, `auth`, `login`, `www`, `app`, `mail`, `static`, `support`, `billing`, `help`,
`status` and the rest. Enforced in `internal/content`, not in the UI — a client-side check on a
phishing-relevant control is not a check.

301s from `/congregations/[unitId]` so discovery links and any indexed URLs survive. The rendering
code is structured as an extractable module, so the owner's Phase 2 (`openfaithmap-sites` as its
own container, for process-level blast-radius isolation) is a move rather than a rewrite.

**Acceptance criteria.** `http://grace.localhost:3002/` serves that congregation's site — browsers
resolve `*.localhost` to loopback, so this exercises the real middleware with no DNS change.
`http://localhost:3002/_sites/grace` **404s**. A reserved slug is rejected at the API, not just in
the form. An old `/congregations/[unitId]` URL 301s to the tenant root.

### M14.10 · Navigation + page routes

**Not started.** Depends on M14.9.

`/[pageSlug]` and nested child routes on the tenant host, honoring `content_documents`' existing
3-level nesting cap — a structure that has been in the schema since M3 and has never had a URL.

**Nav model resolved at M14.0 (2026-08-27): a hand-built menu, not page-tree-derived.** The
original page-tree-derivation sub-question was superseded when the owner replaced the routing
options with their own subdomain design, leaving it an assumption rather than a decision — M14.0
replaced it rather than confirming it. A new `content_site_nav_items` table
([content.md](modules/content.md)) holds an independently-curated, ordered list per site: a label,
a target (an internal document or an external URL), and a sort order. `parent_document_id` still
governs page nesting and breadcrumbs; it no longer drives the nav menu itself.

Breadcrumbs at depth ≥ 2.

**Acceptance criteria.** A three-level page tree is reachable by URL and by nav. Hiding a page
removes it from the nav but leaves it reachable by direct URL (the `UNLISTED` semantics that already
exist). Slug collisions within a site produce the existing typed `Content:SlugTaken`.

### M14.11 · Site chrome — header, footer, template parts

**Not started.** Depends on M14.10.

Site-level chrome as settings on `content_sites`, not as content documents: congregation name, logo
URL, navigation, social links, and a footer whose contact details and service times are read **live
from `religion_sites` and `religion_service_schedules`, never copied into `content`**. That
invariant already exists in [content.md](modules/content.md) and [discovery.md](modules/discovery.md);
it is restated here because a footer is precisely where someone would be tempted to denormalize
service times for one fewer query.

**Acceptance criteria.** Header and footer render on every tenant route. Changing a service time in
the religion module changes the footer with no content edit and no cache staleness beyond M14.17's
declared revalidation window.

### M14.12 · Curated theme tokens

**Not started.** Depends on M14.11. Gives `content_sites.theme` its first-ever reader.

A real schema for the column: accent color from a vetted palette, one of a small set of font
pairings, a spacing scale, header layout, light/dark. WordPress's `theme.json` lesson — a fixed
vocabulary, not CSS, and not free-form hex entry. Emitted as CSS custom properties consumed by the
tenant layout.

**A WCAG contrast check runs at write time and rejects a failing combination.** A warning would be
ignored; the platform badges its congregations on accessibility elsewhere, and shipping a
congregation an unreadable site would contradict that directly.

Live theme preview in the admin.

**Acceptance criteria.** A theme change is visible on the public tenant site. A combination failing
AA contrast is rejected with a typed error naming the failing pair. No congregation can enter a raw
hex value or an arbitrary font.

### M14.13 · Starter patterns + block-type catalog admin

**Not started.** Depends on M14.4 (patterns insert blocks that need working forms). Finally builds
what M3 deferred.

New `content_patterns` with WordPress's **unsynced** semantics: inserting a pattern copies its
blocks into the document and detaches: no ongoing link, no shared state, freely edited afterwards.
Seeded, church-specific: Parish home page, Service times, Meet the clergy, Getting here, Feast-day
announcement.

Builds the `content.catalog.manage` endpoints M3 left unbuilt — block-type and pattern CRUD, gated
on the platform-moderator authority D-PlatformModerator already defines — so the catalog stops
being migration-only. This is the arc's single biggest onboarding lever: the research is
unambiguous that a blank canvas is where non-technical editors stall.

**Acceptance criteria.** A new congregation builds a plausible homepage from a pattern without
authoring a block by hand. An inserted pattern is fully editable and unlinked from its source. A
moderator adds a block type at runtime and it appears in the inserter with a working form (M14.4).
A non-moderator is refused — tested with a token that should be refused, not inferred.

### M14.14 · Locale switching — closes `DS-OFM-7`

**Not started.** Depends on M14.10. Closes an open seam the arc would otherwise walk past.

Visitor-facing locale picker on the tenant site, offering **only locales that actually have a
published variant** — a picker that leads to a 404 is worse than no picker. `hreflang` alternates
so search engines pair the variants.

Editor-side: a translation panel per document showing which locales exist in the translation group
and each one's state, with "create translation" seeding a new variant into the same group.
Translation groups have worked structurally since M3 with no UI on either side of the product.

**Acceptance criteria.** A page published in Ukrainian and drafted in English offers only Ukrainian
to visitors. Publishing the English variant makes it appear with no other change. `hreflang` tags
are present and correct. `DS-OFM-7` is struck in `open-questions.md`.

### M14.15 · Scheduled publishing, no scheduler

**Not started.** Depends on M14.6.

`content_documents.publish_at` plus a `SCHEDULED` state. The public read predicate becomes
`state = 'PUBLISHED' OR (state = 'SCHEDULED' AND publish_at <= now())`.

Correctness lives in the `WHERE` clause. Nothing has to fire: no systemd timer that does nothing in
local dev and cannot be verified before the VM in **U14** exists, no in-process goroutine of the
kind D-CongregationImport explicitly declined to add and `DS-OFM-16` already flags as
unattributable. The single cost is that the stored state lags reality, so **the admin UI must show
effective state, not the raw column** — a document whose `publish_at` has passed reads as
"Published", because to every visitor it is.

**Acceptance criteria.** A document with a future `publish_at` 404s on the public route; with a
past one it renders — **with no job having run**, verified by not having one. The admin list shows
effective state. Scheduled documents are excluded from sitemaps until due (M14.17).

### M14.16 · Contact form + in-app inbox

**Not started.** Depends on M14.11. Implements `D-InAppInbox`.

New `content_form_submissions`, plus a genuinely anonymous write on `ContentPublicService` — the
third such endpoint in the codebase, after moderation's two. It reuses `internal/platform/ratelimit`
(M7, D-Hardening) rather than adding a second limiter, which also means it inherits the shared-bucket
behaviour noted in the verification section below.

Spam handling with no third-party dependency: a honeypot field, a minimum time-to-submit, and the
per-IP rate limit. A Messages screen in `openfaithmap-admin`, `content.manage`-gated so it follows
the same authority as the rest of the site.

**No SMTP anywhere.** D-InviteLinkMVP shipped invites as shareable links precisely because this
stack has no outbound mail, and this arc does not add that dependency. Submission text is untrusted
and renders as plain text only — never as rich text, never as a block.

**Acceptance criteria.** An anonymous submission appears in the admin inbox. A burst is refused by
the rate limiter. A honeypot-filled submission is silently accepted and discarded, not error'd (an
error teaches the bot). Submission text containing markup renders as literal text. No congregation
admin without `content.manage` on that unit can read the inbox — tested with a refused token.

### M14.17 · SEO, structured data, caching

**Not started.** Depends on M14.10, M14.15 (scheduled documents must not leak into sitemaps).

Per-page `<title>`, description, canonical and OG tags — none of which exist on any public page
today. Per-tenant `sitemap.xml` and `robots.txt`. JSON-LD: `Church` for the site, `Event` for event
documents, `BreadcrumbList` for nested pages.

Replaces `export const dynamic = "force-dynamic"` with tag-based revalidation invalidated on
publish/unpublish. The current setting re-queries the API on every anonymous page view: slow, and a
free amplification lever against `openfaithmap-api` from unauthenticated traffic.

**Acceptance criteria.** A published page carries a real title and OG tags. The sitemap lists
published and due-scheduled documents only. Publishing invalidates the cached page within the
declared window; an unrelated page's cache is untouched. A repeat anonymous page view does not
re-query the API.

### M14.18 · Deployment wiring

**Not started. 🔶 Blocked on U14: a registered apex domain and a DNS-provider API token** — neither
exists, and no VM is provisioned (D-ProductionDeployment is design-only). Everything else in M14 is
verifiable locally without any of it, which is why this milestone is last and alone.

A Caddyfile with the DNS-01 wildcard block for `*.<apex>`. This is the one genuinely new
infrastructure constraint the arc introduces: **ACME cannot issue a wildcard certificate over
HTTP-01**, so the DNS provider must expose an API Caddy has a module for — narrowing a provider
choice D-ProductionDeployment deliberately left open. The wildcard is what avoids Let's Encrypt's
per-account new-order ceiling entirely, which per-subdomain issuance would run into as soon as
`congregationimport` provisions congregations in bulk.

Also: the wildcard DNS record, HSTS with `includeSubDomains`, per-tenant read rate limiting, and a
restatement that the backup story is unchanged — there are no blobs to back up, because
`D-ExternalMediaOnly` means there are no uploads. Records the `openfaithmap-sites` extraction as the
named trigger for the owner's Phase 2.

**Acceptance criteria.** A real congregation subdomain serves over HTTPS with a wildcard
certificate. HSTS present. The reserved-slug blocklist holds against a real registration attempt.
`🔶` clears only when a real domain is serving — not when the Caddyfile is written.

## Verification notes for this arc

Ground every `✅` in a real artifact, and remember that **a happy-path proof is not a Verified
proof** — M2 shipped three defects past a curl-based happy-path demo. The authorization and
failure-mode criteria above must be exercised specifically.

- **Stack.** `docker compose up -d --build openfaithmap-api openfaithmap-web openfaithmap-admin`.
  The `--build` is mandatory after *any* edit, Go or TypeScript — `up -d` alone silently keeps the
  previous image, which has cost real debugging time before.
- **Reaching the API.** Port 3000 is not host-published (D-HeadlessTopology). Use
  `docker exec open-faith-map-openfaithmap-admin-1 wget …  https://openfaithmap-api:3000/…`
  (BusyBox `wget`, GET only) or a throwaway `curlimages/curl` on the compose network. Port 3001 is
  the witchcraft management port, not a real API route.
- **Subdomains locally.** Browsers resolve `*.localhost` to loopback with no DNS or hosts-file
  change, so `http://grace.localhost:3002/` exercises M14.9's real Host middleware. `openfaithmap-web`
  is host-published on 3002, `openfaithmap-admin` on 3004.
- **The rate limiter will bite during verification.** `internal/platform/ratelimit` is ~5 req/min
  per client IP per endpoint, and every call from `web/apps/web` is a server-to-server call from
  that container's own IP — so all local browser traffic shares one bucket. A burst of page loads
  (including any `curl` probes just beforehand) exhausts it and surfaces as a generic page error.
  Wait ~60s rather than reading it as a bug. This applies doubly to M14.16, which adds a fourth
  endpoint to the same limiter.
- **Browser proof.** `chromium-cli` is not installed in this environment. Use `playwright-core` in
  a scratch directory with `executablePath: "/usr/bin/google-chrome"`, `headless: true`,
  `--no-sandbox --disable-dev-shm-usage`.
- **Tests.** Extend `internal/content/content_integration_test.go` for revisions, publish-on-read,
  the URL allowlist and the reserved-slug blocklist against real Postgres. `./godelw verify` (fmt,
  lint, conjure-backcompat) and `make sdk`/`sdk-verify` before each gate.
- **CI green on `main` at the merge commit** — no row advances to `Verified` without it.
