# Milestones (2026-08-26 – now)

The architecture sequenced into buildable, dependency-ordered milestones. A roadmap, not binding —
[`architecture/decisions.md`](architecture/decisions.md) governs *what*, this governs *in what
order*. Gate definitions are in [`development-process.md`](development-process.md).

## Status

**M0–M13.6 are done** (no row had unbuilt Backend/Migrated/UI work as of 2026-08-26) — see
[`milestones-2026-08-07-2026-08-26.md`](milestones-2026-08-07-2026-08-26.md) for that full history.

**M14 · The site-building arc** is scoped (2026-08-26); **M14.0 through M14.15 (M14.0–M14.9
2026-08-27/28, M14.10–M14.12 2026-08-29, M14.13–M14.14 2026-08-30, M14.15 2026-09-03) are done**, no
other sub-milestone is built yet. It is the second half of the
product: M4/M13 finished **discovery** (the map); M14 finishes **presence** (the per-congregation
site builder), whose bones shipped at M3/M4 and were never built on. Nineteen sub-milestones,
M14.0–M14.18. **M14.0 was the gate for the whole arc** — it wrote the nine `D-` blocks and the
module-doc rewrites, and ruled on **U16** (tightened) and the M14.10 nav assumption (replaced with a
hand-built menu); see [architecture/decisions.md](architecture/decisions.md) and the Unresolved
unknowns table below. **M14.1 closed the live stored-XSS hole** that ran ahead of any other feature
work in the arc. **M14.2 replaced plain-string block text with a structured richText node model.**
**M14.3 normalizes known share-link hosts and requires `alt`.** **M14.4 kills the JSON-textarea
block editor**, replacing it with a generic form derived from `json_schema` + a new `ui_schema`.
**M14.5 adds a categorized inserter and drag-and-drop/keyboard reorder**, retiring the
manually-typed `position` field entirely. **M14.6 adds forward revisions, autosave, and history**,
so editing a live page never touches what visitors see. **M14.8 adds client-side undo/redo, a real
empty state, and a mobile-workable layout**, taken out of milestone order because M14.7 (Preview)
was blocked on M14.9. **M14.9 serves each congregation's site on its own subdomain** (`Host`-header
routing, a reserved-slug blocklist, a real 301 from the old path, and U16's `content.manage`
tightening) — see its own row below. **M14.7 renders a draft revision through the real public
renderer** on the tenant subdomain, behind a short-lived signed token, once M14.9 unblocked it — see
its own row below. **M14.10 gives each Page its own hierarchical URL** on the tenant host and adds a
hand-built nav menu (`content_site_nav_items`), replacing the one-pager's inline Pages section — see
its own row below. **M14.11 gives every tenant site a real header and footer** — congregation
name/logo/nav in the header, address/service-schedule/social-links in the footer, the address and
schedule composed live from religion's data at request time (content's application layer now
depends on religion's, the same direct-interface-call shape discovery already established) — see its
own row below. **M14.12 gives `content_sites.theme` a real, curated schema** — accent color, font
pairing, spacing scale, and header layout are each chosen from a fixed vocabulary, never typed as
raw CSS, with a WCAG AA contrast check rejecting a failing accent/mode pair at write time — see its
own row below. **M14.13 finally builds `content.catalog.manage`** (block-type and pattern CRUD,
platform-moderator-gated) and ships 5 seeded starter patterns with WordPress's unsynced insert
semantics — see its own row below. **M14.14 closes `DS-OFM-7`**, splitting the tenant site's
`[locale]` URL segment into a UI-chrome language and a separate, free-text content locale, adding a
per-page locale picker + `hreflang` and an editor-side translation panel — see its own row below.
**M14.15 adds scheduled publishing with no scheduler**, a `SCHEDULED` state whose visibility is
decided entirely by the public read predicate comparing `publish_at` to `now()` — see its own row
below. Next up: M14.16.

## Unresolved unknowns — read this before building anything

Every place the doc set currently says "we don't actually know." Detail lives where the third
column points; this table exists so nothing is hidden, not to duplicate it.

Everything carried in from the archive has been resolved (see the note below the table). Of the
three items opened by M14's scoping pass on 2026-08-26, **U16 is now resolved (M14.9, 2026-08-28)**;
**U14** and **U15** remain open:

| # | The unknown | Where it bites | Who resolves it |
|---|---|---|---|
| **U14** | **No apex domain is registered and no DNS-provider API token exists.** A wildcard certificate for `*.<apex>` can only be issued over the ACME **DNS-01** challenge — HTTP-01 cannot issue wildcards. D-ProductionDeployment deliberately left the VM/DNS provider undecided; `D-TenantSubdomains` now constrains that choice for the first time (the provider must expose a DNS API Caddy has a module for). | M14.18 only. Every other M14 milestone is verifiable locally against `*.localhost`, which browsers resolve to loopback with no DNS at all. | The owner, by registering a domain and picking a DNS provider. M14.18 carries `🔶` until then — the same honest gate M1.2/M2 already use for the Google OAuth redirect URI. |
| **U15** | **Google Drive hotlink reliability at volume is unmeasured.** `D-ExternalMediaOnly` makes congregations host their own images on Drive/Dropbox/OneDrive. Direct-content URLs for these hosts are undocumented, have been changed by their vendors before, and are throttled under load — none of which we can measure before real congregations use it. | M14.3's normalizer, and every `image`/`gallery` block on every public site thereafter. A vendor-side change breaks images platform-wide at once. | Only real traffic. M14.3 mitigates rather than resolves: the original URL is preserved alongside the normalized one, so a normalizer fix is a re-derivation, not a data-loss event. Escalation path is the first-party `media` module (`DS-OFM-17`). |
| **U16** | ~~**A registration operator can edit any congregation's website.** Not new — [content.md](modules/content.md#authorization-touchpoints) has recorded it since M3, as a consequence of `content.manage` reusing `religionorg.manage`, which `registration-operator` holds as a subtree grant on the shared root. What is new is the **stakes**: after M14.9 a "site" is a real website on its own subdomain, not an unlinked blob of blocks.~~ **Resolved (2026-08-28, M14.9).** `content.manage` (`PermContentManage`) is now its own per-unit permission, granted to `congregation-admin` only (`migrations/0026_content_manage_permission.sql`); `internal/content/application/authorize.go` checks it instead of `religionorg.manage`. Operators no longer pass this check via their subtree grant — live-verified against `internal/content/content_integration_test.go`'s new M14.9 cases (real Postgres): a `registration-operator` granted the same unit-scoped shape `congregation-admin` holds is denied. See [D-TenantSubdomains](architecture/decisions.md#d-tenantsubdomains--subdomain-per-congregation-wildcard-tls-and-a-reserved-slug-blocklist). | Every `content.manage`-gated write in the arc — which is most of it. | **Decided by the owner (tighten), designed by M14.0, built at M14.9.** Confirmed with the owner: operators are left with no replacement edit path for now — granting a moderation permission instead is a separate, later decision. |

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
| M14.1 · Content security baseline | ✅ | ✅ | ✅ | ➖ | ✅ | ⬜ | **Built (2026-08-27), fixes the live stored XSS; nothing else in the arc ships first.** URL **scheme** allowlist (`https`/`http`/`mailto`/`tel`) on every URL-bearing block field, enforced at write in `blockvalidation.go` (new typed `Content:BlockUrlNotAllowed` error), plus an embed **host** allowlist keyed by platform/block type (`social_embed`, YouTube-only for `youtube_embed` — no Vimeo block type exists yet). Defensive re-validation in the renderer (`web/apps/web/lib/block-security.ts`) because pre-M14.1 rows already existed unvalidated. `sandbox` + `referrerpolicy` on the `youtube_embed` iframe. CSP and security headers in both `next.config.ts` files (previously three lines each, zero headers) — verified present on a real response from both apps against the running stack. New invariant: `dangerouslySetInnerHTML` appears in neither app, ever — enforced by a new ESLint `no-restricted-syntax` rule, not just a point-in-time grep. Verified against a real pre-existing row inserted directly via SQL (bypassing the API): the malicious block renders dropped, not executed. `Verified` awaits CI green on `main`. |
| M14.2 · Rich-text node model | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-27).** A shared `richText` JSON-Schema definition — inline `text` runs carrying `bold`/`italic`/`link` marks, plus `list`/`listItem` — adopted by `paragraph`, `heading`, `quote`, `staff_card.bio` and a new `list` block. The renderer maps nodes to elements, so there is **no HTML parser and no sanitizer**: Drupal's filter-on-output problem is designed out rather than mitigated. Expand-and-data migration (`migrations/0022_content_richtext.sql`) updating those block types' `json_schema`, plus a data migration lifting existing plain strings into single-run nodes, in the same file. `Verified` awaits CI green on `main`. |
| M14.3 · External media URLs, made survivable | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-27).** Normalizer for known share-link hosts (Google Drive, Dropbox, the long-form OneDrive URL) → direct-content URL, applied at write with the original preserved in a new `originalUrl` field (**U15**). `alt` is now schema-**required** on `image`/`gallery`. `loading="lazy"` + `referrerpolicy` on every rendered image (`image`, `gallery`, `staff_card`). **Scoped down from the original text, named and reasoned, not silently dropped:** OneDrive's short `1drv.ms` links are not normalized (would require a server-side redirect-follow — the exact SSRF surface this arc forbids); the editor-side load probe is deferred to M14.4, since the admin editor is still the raw JSON-textarea UI with no per-field surface to attach one to yet. Records the future first-party `media` module as a designed seam so adding it later is additive. |
| M14.4 · Schema-driven block forms | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-27).** New `content_block_types.ui_schema JSONB NOT NULL DEFAULT '{}'` (widget hints, labels, help text, field order) — WordPress's `block.json` lesson: a block's data schema and its editor controls are declared together, so the form is *derived*, never hand-written per type. A generic, recursive form renderer (`web/apps/admin/.../documents/[documentId]/block-data-form.tsx`) replaces the raw-JSON `<Textarea>` for block *data* only — the outer block list's `position`/`blockTypeCode`/add-remove controls are untouched, reserved for M14.5. `Content:BlockDataInvalid` gained a `field` safe-arg (mirroring `Content:BlockUrlNotAllowed`'s existing one), populated by filtering a jsonschema/v6 validation error's instance-location path through the block type's own declared top-level `properties` keys — never a raw, potentially-attacker-chosen path segment; the admin editor now highlights the offending field inline instead of one generic `?error=` banner. **Two named, deliberately-accepted gaps, decided with the owner:** richText fields (`heading.text`, `paragraph.text`, `quote.text`, `staff_card.bio`, `list.content`) stay a schema-aware JSON textarea — no WYSIWYG editor exists in this codebase, and building one is out of scope here; and M14.3's deferred editor-side URL-load probe stays unbuilt (would need its own CSP/SSRF review, unscheduled). A `columns` block's schema-shape failure highlights the whole `columns` field group rather than a specific nested block, since the structural validation pass never descended into nested block data (pre-existing, not new). Migration: `migrations/0024_content_ui_schema.sql`. `Verified` awaits CI green on `main`. |
| M14.5 · Inserter + drag-and-drop reorder | ✅ | ✅ | ➖ | ➖ | ✅ | ⬜ | **Built (2026-08-27).** Categorized block inserter with a one-line description per type — curation over choice, the consistent finding in the editor-UX research (13+ undifferentiated types is already past where an editor picks well). Drag-and-drop replaces the integer `position` input, **with keyboard-accessible move-up/move-down as a first-class path**: drag-only reordering is an accessibility failure, not a polish gap. `Verified` awaits CI green on `main`. |
| M14.6 · Forward revisions, history, autosave | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-28).** New `content_document_revisions`; `content_documents` gains separate *published* and *draft* revision pointers, so **editing a live page never touches what visitors see** (Drupal's forward-revision model) — `PutBlocks`/`GetBlocks` read/write the draft in place, and publishing snapshots the draft into an immutable checkpoint (capped at 50 per document, `internal/content/application/revisionsnapshot.go`) that `GetPublicBlocks` reads instead. New `listRevisions`/`restoreRevision` endpoints back a History panel with per-revision restore. The admin editor autosaves on a ~10s debounce with a visible saved/unsaved indicator, replacing the old redirect-on-save flow. Migration: `migrations/0025_content_revisions.sql`. `Verified` awaits CI green on `main`. |
| M14.7 · Preview | ✅ | ✅ | ✅ | ➖ | ✅ | ⬜ | **Built (2026-08-28).** Renders the draft revision through the **real public renderer** (`components/site-page.tsx`) — an optional `previewToken` prop swaps its two data-fetch calls to token-gated preview reads; the render tree itself never forks. A new `ContentService.createPreviewLink` (content.manage-gated) mints a short-lived (20 min), stateless, site-scoped HS256 token (`internal/content/application/previewtoken.go`, no DB row — a draft is content, not a special code path); `ContentPublicService.listPreviewDocuments`/`getPreviewBlocks` accept it in place of a session, the one deliberate exception to "published/unlisted only," returning a single `Content:PreviewTokenInvalid` for every failure mode (missing/malformed/expired/wrong-site) so a probing caller learns nothing. Reached on the tenant subdomain at plain `/{locale}/preview?token=…` — no `proxy.ts`/`lib/tenant-host.ts` change needed, since that's the same browser-facing shape every tenant page already has; `injectSitesSegment` rewrites it into `app/[locale]/%5Fsites/[slug]/preview/page.tsx` exactly like every other tenant route. `X-Robots-Tag: noindex, nofollow` and `Cache-Control: no-store`, added as a second path-scoped entry in the existing `next.config.ts` `headers()` array (M14.1's baseline headers untouched). A new `components/preview-frame.tsx` client component gives the device-width toggle (mobile/tablet/full) by constraining a wrapping div's `max-width` — no iframe, so there is exactly one render pipeline to keep pixel-identical to publishing. The admin editor's document page mints a link and opens it in a new tab (`target="_blank"`, never embedded) next to Publish/Unlist/Revert-to-draft — no congregation content ever renders inside the admin origin, cross-origin by construction. Verified against a real running docker-compose stack: a document created and left in `DRAFT` (never published) renders through `/en/preview?token=…` on `grace.localhost:3002` while the public root and a tokenless/garbage-token preview request both show nothing of it; a token minted for a different site is rejected on this one. `internal/content/content_integration_test.go` covers the same shapes against real Postgres, plus a dedicated `previewtoken_test.go` for expiry/tampering/malformed-token/alg-confusion cases a live clock can't easily exercise. `Verified` awaits CI green on `main`. |
| M14.8 · Editor polish | ✅ | ✅ | ➖ | ➖ | ✅ | ⬜ | **Built (2026-08-28), out of milestone order** (M14.7 above is blocked on M14.9; this one only depends on M14.5). Session-local client-side undo/redo over the block list (`hooks/use-block-history.ts`), independent of M14.6's server-side revision history — Ctrl/Cmd+Z and Ctrl/Cmd+Shift+Z, plus toolbar buttons, disabled at stack boundaries. A real empty state for a zero-block document, with a CTA into the existing inserter — **scoped down from "start from a template," named rather than silently dropped:** M14.13 (`content_patterns`) doesn't exist yet, so there is no template to start from. The block row's fixed 5-column grid now stacks to a single column below the `sm:` breakpoint, usable at 375px. The two remaining `?error=`-redirect round trips (document details save, new-document create) are now inline via `useActionState`, matching the pattern `people/invite/invite-form.tsx` established — zero `?error=` round trips remain anywhere in the document editor. `Verified` awaits CI green on `main`. |
| M14.9 · Tenant subdomain routing (Phase 1) | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-28).** `web/apps/web/proxy.ts` (Next 16 renamed `middleware.ts` to `proxy.ts` — extended in place, composed with the existing next-intl locale middleware, rather than a second file) resolves the `Host` header to a site slug (`lib/tenant-host.ts`, unit-tested) and rewrites into an internal `/[locale]/_sites/[slug]/…` tree; the apex host keeps serving discovery unchanged. **Direct `/_sites/*` access from the apex 404s** — checked first, before next-intl even runs. **Reserved-subdomain blocklist enforced server-side** (`internal/content/application/slugvalidation.go`, checked in `CreateSite` before the existing `Content:SlugTaken` uniqueness probe) — `content_sites.slug` is now a hostname, so `admin`/`api`/`auth`/`login`/`www`/`app`/`mail`/`static`/`support`/`billing`/`help`/`status` and 25 more are unclaimable, rejected with a new typed `Content:SlugReserved` error. A real 301 (not Next's 307/308 `redirect()`) from the old `/congregations/[unitId]` route, now a Route Handler, to the tenant root. Rendering logic extracted into `components/site-page.tsx`, an "extractable module" reused by the new thin `[locale]/%5Fsites/[slug]/page.tsx` wrapper (the actual directory is `%5Fsites` — Next.js's private-folder convention would otherwise exclude a literal `_sites` folder from routing; `%5F` is the documented URL-encoded-underscore escape) — set up for the owner's Phase 2 (`openfaithmap-sites`) to be a move, not a rewrite. **Also implements M14.0's U16 ruling:** `content.manage` (`PermContentManage`) stops resolving through `religionorg.manage`'s subtree grant and becomes its own per-unit permission granted to `congregation-admin` only (`migrations/0026_content_manage_permission.sql`, same shape as M13.2's `site.manage`); registration operators lose that edit access, confirmed with the owner as leaving them with **no** replacement edit path for now — granting a moderation permission is a separate, later decision. Cross-tenant-denial and operator-denial cases (`docs/modules/content.md`'s previously-named test-coverage gap) covered by new `internal/content/content_integration_test.go` cases against real Postgres. Verified against a real running docker-compose stack: `grace.localhost:3002/` resolves through a real `content_sites` row created over HTTP by a real `congregation-admin` session (307 to `/en` with `NEXT_LOCALE` preserved, then 200); `localhost:3002/_sites/grace` and `/en/_sites/grace` both 404 from the apex; `CreateSite` with `slug: "admin"` returns `400 Content:SlugReserved` over a direct HTTP call (bypassing the admin form entirely); the old `/en/congregations/[unitId]` route returns a real `301` to `http://grace.localhost:3002/`. See [D-TenantSubdomains](architecture/decisions.md#d-tenantsubdomains--subdomain-per-congregation-wildcard-tls-and-a-reserved-slug-blocklist). `Verified` awaits CI green on `main`. |
| M14.10 · Navigation + page routes | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-29).** `/[pageSlug]` and hierarchical nested child routes on the tenant host, honoring the existing 3-level cap — a wrong ancestor segment 404s, never resolved by the last segment alone. **Nav is a hand-built menu (`content_site_nav_items`), not derived from the page tree** — M14.0 replaced the original page-tree-derivation assumption with an independently-curated menu (label, target document or external URL, sort order); `parent_document_id` still governs page nesting/breadcrumbs, just not the nav itself. Breadcrumbs at depth ≥ 2. The site root drops its old inline "every Page rendered inline" section (owner decision) — Pages are reachable only via their own route or the nav menu; Posts/Events feeds are unchanged. `Verified` awaits CI green on `main`. |
| M14.11 · Site chrome — header, footer, template parts | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-29).** Congregation name, logo URL, nav, and a footer whose contact details and service times are read **live from `religion_sites`/`religion_service_schedules`, never copied** — the existing content.md invariant, restated because a footer is exactly where someone would be tempted to denormalize. Social links. Site-level settings on `content_sites`, not content documents. |
| M14.12 · Curated theme tokens | ✅ | ✅ | ✅ | ➖ | ✅ | ⬜ | **Built (2026-08-29).** `content_sites.theme` gets a real schema: accent color from an 8-color vetted palette, one of three system-font pairings, a spacing scale, a header layout, and light/dark/system mode — WordPress's `theme.json` lesson, a fixed vocabulary rather than CSS. Emitted as CSS custom properties consumed by the tenant layout. **A WCAG AA contrast check rejects a failing accent/mode combination at write time**, naming the pair. Live theme preview in the admin. `Verified` awaits CI green on `main`. |
| M14.13 · Starter patterns + block-type catalog admin | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-08-30).** New `content_patterns` (`migrations/0029_content_patterns.sql`) with WordPress's **unsynced** semantics: inserting a pattern is a pure client-side copy of its `blocks` into the document editor's local state (fetched once via the new public `listPatterns`), persisted through the existing `putBlocks` full-replace path — no dedicated "insert" mutation endpoint exists, or is needed, since a pattern's blocks are already the same `BlockInput`-shaped array a document's own block list uses. Seeded 5 church-specific starters — Parish home page, Service times, Meet the clergy, Getting here, Feast-day announcement. Finally builds `content.catalog.manage` (block-type create/update, pattern create/update/delete), gated on `platform-moderator` standing — the exact same `PermModerationStanding`/root-unit-scoped check `internal/moderation`'s `requireModerate` already uses, reused rather than a new authority concept (D-SitePatterns' own explicit call). **Owner decision this session:** `updateBlockType` locks `json_schema`/`ui_schema` after creation — the request type has no such field at all — so a runtime catalog edit can never silently break already-saved blocks of an existing type or the admin form built from its old schema; a moderator wanting a different shape retires the old type and creates a new one. **Named, accepted scope boundary:** a block type added at runtime works in the admin inserter/form (M14.4/M14.5 are schema-driven) but does not render on the public site — `web/apps/web/app/blocks.tsx` dispatches on a hardcoded switch with a no-op fallback for unknown codes; making the public renderer schema-driven too is a separate, larger change. Two new admin routes (`/admin/block-types`, `/admin/patterns`) follow `/admin/moderation`'s own precedent exactly: no local frontend role gate, shown unconditionally in the sidebar nav, the backend's `Content:Forbidden` is the entire authorization decision. Live-verified over real HTTP against the running docker-compose stack (dev-minted tokens, `docker exec ... curlimages/curl` for non-GET per this arc's own verification-notes convention): anonymous 401, an authenticated non-moderator 403, a real `platform-moderator` grant 200 on `GET/POST/PUT /content/v1/catalog/block-types` and the patterns equivalents — a type created, confirmed on the public `listBlockTypes`, retired, confirmed gone; a pattern created, confirmed on the public `listPatterns`, deleted, confirmed gone. `internal/content/content_integration_test.go` covers the same shapes against real Postgres (moderator grant/denial, duplicate-code, not-found, uncompilable-schema cases). `web/apps/admin`'s Vitest suite gained a pattern-insertion test (`block-list-editor.test.tsx`) alongside the existing 42. `Verified` awaits CI green on `main`. |
| M14.14 · Locale switching — closes `DS-OFM-7` | ✅ | ✅ | ✅ | ➖ | ✅ | ⬜ | **Built (2026-08-30).** No migration needed — `content_documents.translation_group_id`/`locale` and their index have existed since M3; this milestone is entirely app-level. **Two owner decisions this session shaped the work:** (1) the picker + `hreflang` are **per-page, in-content** (rendered inside the PAGE route itself), never in the shared site header/footer — the header wraps every route including the root posts/events feed, which has no single translatable document behind it, and a picker that could 404 is explicitly called worse than no picker; (2) the site chrome's UI language (next-intl's `[locale]` route segment, still fixed to 4: en/uk/es/pt) and a document's own **content locale are now decoupled** — a congregation can author a page in any language, not only the 4 chrome locales, so the tenant PAGE route grew a new `[contentLocale]` segment (`/{uiLocale}/{contentLocale}/{pageSlug...}`) independent of the chrome language, which stays put when a visitor switches content locale. `GetPublicDocumentByPath`'s `DocumentWithAncestors` response grew a `translations` list (every `PUBLISHED` sibling in the document's translation group, each with its own resolved href — siblings can sit at a different ancestor hierarchy/slug per locale, so a translation's href is never derived from the leaf's own) — one round trip, same precedent as the response's own `ancestors` field. `generateMetadata` (the first in this app) emits `alternates.languages` from that same list. `CreateDocument`'s existing-but-manual "join an existing translation group" path (`translationGroupId`, already wired since M3/M14.8's own manual admin field) gained two app-level guards with no DB constraint backing either: a duplicate locale within one group (`Content:TranslationLocaleTaken`) and a group belonging to a different site (`Content:TranslationGroupNotFound`) — content.md's own invariant is that a group's documents share nothing but the group id, so nothing in the DB stops either otherwise. The editor's new Translations panel (document editor page) reuses the same "filter what you already have" convention `new/page.tsx`'s own `existingPages` already established — no dedicated list-by-group endpoint — and its "create translation" link reuses the existing `new-document-form.tsx` flow (pre-filling/locking `kind`/`translationGroupId` via query params) rather than a bespoke form. `internal/content/content_integration_test.go` covers the same shapes against real Postgres (a draft sibling excluded from `translations`, both siblings included once published, the duplicate-locale and cross-site guards). `Verified` awaits CI green on `main`. |
| M14.15 · Scheduled publishing, no scheduler | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | **Built (2026-09-03).** `content_documents.publish_at` + a `SCHEDULED` state; the public predicate becomes `state = 'PUBLISHED' OR (state = 'SCHEDULED' AND publish_at <= now())`, applied everywhere a `PUBLISHED` document is already visible (page route, feeds, nav, translations). **Correctness lives in the `WHERE` clause**, so it behaves identically in local dev and on a VM that does not exist yet — no timer, no goroutine, nothing to fire, and no new unattributable background writer (`DS-OFM-16`). The admin UI shows **effective** state, not the raw column; the transitions lookup itself is keyed by effective state too, so `UNLIST`/`REVERT_TO_DRAFT` on a due-but-unsettled `SCHEDULED` document is what settles it. Live-verified against the running docker-compose stack: `publish_at` moved into the past by a direct SQL write (no app call) flips a real tenant-subdomain 404 to 200 with no restart in between. `Verified` awaits CI green on `main`. |
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

**Built (2026-08-27).** Depends on M14.0 only. **This milestone fixes a hole that was live in
`main`** and therefore ran before any feature work in the arc.

The milestone text named "a named YouTube/Vimeo set for `youtube_embed`", but no `vimeo_embed`
block type exists in the seeded catalog (`youtube_embed.videoId` isn't a URL at all — the embed
`src` is server-constructed). Built YouTube-only; the embed-host allowlist is keyed by block type
on both sides (`blockvalidation.go`'s `socialEmbedHosts`-shaped map, `block-security.ts`'s
`EMBED_IFRAME_HOSTS`) so a future `vimeo_embed` (M14.13) is an additive entry, not a rewrite.

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

**Acceptance criteria — met.** Saving a block with `href: "javascript:alert(1)"` is rejected with a
typed error naming the field (Go integration test, `internal/content/content_integration_test.go`,
run against real Postgres). A row carrying that value **inserted directly with SQL** — the
pre-existing data case — renders with the link dropped, not executed (verified against the running
docker-compose stack: a site/document/block seeded by raw SQL, then fetched over HTTP). The CSP
header is present on a real HTTP response from both apps, verified against the running stack rather
than read from config. No `dangerouslySetInnerHTML` in either app — zero occurrences, plus a new
ESLint `no-restricted-syntax` rule enforcing it going forward.

### M14.2 · Rich-text node model

**Built (2026-08-27).** Depends on M14.1 (the URL allowlist that `link` marks are validated
against).

A shared `richText` definition (`docs/modules/content.md`'s own Entities section spells out the
node shapes) in the block-type schema vocabulary: an ordered array of inline nodes — `text` runs
carrying `bold`/`italic`/`link` marks, plus `list`/`listItem` — validated by the existing block
validator with no new validation machinery. `content_block_types.json_schema` has no cross-row
`$ref`, so the same `$defs` block is repeated literally in each of the five adopting schemas.
Adopted by `paragraph`, `heading`, `quote`, `staff_card.bio`, and a new `list` block type. The
renderer (`web/apps/web/lib/rich-text.tsx`) maps node types to React elements directly; there is no
HTML string anywhere in the pipeline, hence no parser and no sanitizer. A `link` mark's `href`
goes through M14.1's allowlist like any other URL — extended in
`internal/content/application/blockvalidation.go` with a `checkRichTextLinks` walk (recursing into
`list` nodes' items) that calls the same `checkScheme` closure the flat URL fields already used.

`migrations/0022_content_richtext.sql`: an expand-and-data migration in one file, per this repo's
convention for a schema change that would otherwise reject rows already in the table — the same
`UPDATE`s that loosen `json_schema` also lift every existing plain-string `text`/`bio` value into a
single-text-run node. Nested blocks inside a `columns` block already bypass
`content_block_types.json_schema` entirely (pre-existing, noted in `blockvalidation.go`'s own
comment) and are not rewritten by this migration; no such nested fixture data exists today, and the
renderer degrades a non-array legacy value to "render nothing" rather than crashing if that's ever
hit in practice.

**Acceptance criteria — met.** A `paragraph` with a bolded run and an inline `link` round-trips
through `PutBlocks` and renders as real `<strong>`/`<a>` elements on the public site (verified
against the running stack: a site/document seeded directly via SQL, fetched with a real headless
Chrome via Playwright). A `list` block round-trips and renders as a real `<ul>`/`<li>` list. A
`link` mark with a `javascript:` href is rejected at write with `BlockUrlNotAllowedError{Field:
"text"}` (Go integration test, `internal/content/content_integration_test.go`, run against real
Postgres) — and a row carrying that value inserted directly via SQL (the pre-existing-data case)
renders with the link dropped, not executed, confirmed against the running stack the same way
M14.1 verified its own write-time gate.

### M14.3 · External media URLs, made survivable

**Built (2026-08-27).** Depends on M14.1. Exists because of `D-ExternalMediaOnly` (**U15**).

A share-link normalizer (`internal/content/application/medianormalize.go`) applied at write time
in `Service.PutBlocks`, before `validateBlockData`, for known hosts — Google Drive, Dropbox, and
the long-form OneDrive URL — rewriting a viewer-page URL to its direct-content form. A Drive share
link (`drive.google.com/file/d/<id>/view`) is an HTML page, not an image: pasted into an `image`
block today it renders nothing, with no feedback anywhere. The original URL is stored in a new,
optional `originalUrl` field alongside the normalized `url` (`migrations/0023_content_media_urls.sql`),
so a future normalizer fix is a re-derivation rather than data loss. Pure string rewriting only —
no host is ever resolved over the network. Only top-level `image`/`gallery` blocks are rewritten;
nested blocks under a `columns` block already bypass `content_block_types.json_schema` entirely
(`blockvalidation.go`'s own documented gap) and are left as-is, mirroring the same
deliberately-accepted gap M14.2's migration recorded for richText.

**OneDrive scope, decided with the owner rather than assumed:** only the long-form
`onedrive.live.com/redir?resid=...` URL is normalized, by a pure `redir`→`download` path
substitution. The short `1drv.ms` links most people actually get from OneDrive's Share button
resolve only via a redirect — following it server-side would be exactly the SSRF surface ("no
server-side fetch of a user-supplied URL") this milestone's own acceptance criteria forbids, so
short links pass through unchanged, with no `originalUrl` set.

`alt` becomes schema-**required** on `image` and `gallery` (`migrations/0023_content_media_urls.sql`)
— structurally enforced rather than requested, which is the only version of alt text that survives
contact with real editors; the existing `content_blocks` are backfilled defensively (a no-op today
— no live congregation content exists yet). `loading="lazy"` and `referrerPolicy="no-referrer"` on
every rendered image (`image`, `gallery`, and `staff_card`'s `photoUrl` — the milestone text says
"every rendered image," not just the two block types named in its title).

**The editor-side load probe is deferred to M14.4, decided with the owner rather than assumed.**
The admin editor is still the raw JSON-textarea UI (`web/apps/admin/app/[locale]/admin/sites/[unitId]/documents/[documentId]/page.tsx`)
— M14.4 is what builds the real per-field forms — so there is no "image URL field" a probe could
sensibly attach to yet. This is the one acceptance criterion M14.3 does not meet; named here rather
than silently dropped, same as M14.1's YouTube-only embed-host narrowing.

Records the future first-party `media` module as a designed seam (`DS-OFM-17`): what it would own,
and why nothing in M14's schema forecloses it.

**Acceptance criteria — mostly met, one explicitly deferred.** A pasted Drive or Dropbox share link
normalizes to a direct-content URL at write time, original preserved — verified against real
Postgres (`internal/content/content_integration_test.go`, `TestContentIntegration`'s M14.3
section) and against the running stack. Saving an `image` block with no `alt`, or a `gallery` item
with no `alt`, is rejected with `Content:BlockDataInvalid` — same test. No server-side fetch of a
user-supplied URL exists anywhere in the arc (OneDrive short links are proof: normalizing them was
explicitly declined for this reason, not overlooked). **Not met: "an unreachable URL is reported as
such in the editor before publishing"** — deferred to M14.4 for the reason above.

### M14.4 · Schema-driven block forms

**Built (2026-08-27).** Depended on M14.2 (rich-text fields) and M14.3 (URL fields) only in the
sense that both those milestones' schema shapes needed to exist first, not on either building a
widget for this one to reuse. **This is the milestone that removes the JSON textarea** for block
*data* — the outer block list's `position`/`blockTypeCode`/add-remove controls are unchanged,
reserved for M14.5.

New `content_block_types.ui_schema JSONB NOT NULL DEFAULT '{}'` (`migrations/0024_content_ui_schema.sql`):
widget hints, field labels, help text, and field ordering, sitting beside the existing
`json_schema` — WordPress's `block.json` lesson, that a block's data shape and its editing
affordances belong in one declaration. A generic, recursive form renderer in `web/apps/admin`
(`.../documents/[documentId]/block-data-form.tsx`) builds each block's form from the pair, so
adding a block type in a future M14.13 will produce a working editor form with no admin-app code
change at all — verified locally by inserting a 15th block type row directly via SQL and confirming
it served a correct `uiSchema` through `listBlockTypes` and validated writes per-field with no
admin-app code change.

Typed Conjure validation errors are now surfaced on the field that caused them:
`Content:BlockDataInvalid` gained a `field` safe-arg (mirroring `Content:BlockUrlNotAllowed`'s
existing one), populated by `topLevelFieldFromValidationError`
(`internal/content/application/blockvalidation.go`) filtering a jsonschema/v6 validation error's
instance-location path through the block type's own declared top-level `properties` keys — a
required-field violation's own `InstanceLocation` points at the *parent* object, not the missing
property, so the `required` keyword's `Missing` list is consulted specially. Never a raw path
segment, which on an `additionalProperties:false` schema could otherwise leak an attacker-chosen
key into a safe-arg — covered by a dedicated unit test
(`internal/content/application/blockvalidation_test.go`). The admin editor reads the redirect's new
`position`/`field` query params and highlights the one field on the one block row that failed,
replacing the old single generic `?error=Content:BlockDataInvalid` banner.

**Two named, deliberately-accepted gaps, decided with the owner rather than assumed:**
- **richText fields** (`heading.text`, `paragraph.text`, `quote.text`, `staff_card.bio`,
  `list.content`) stay a schema-aware JSON textarea (label + help text from `ui_schema`, but still
  raw JSON) — no rich-text/WYSIWYG editor exists anywhere in this codebase, and building one is a
  substantially larger, separate piece of work than this milestone's scope.
- **M14.3's deferred editor-side "is this URL reachable" probe stays unbuilt.** M14.4 replaces the
  JSON-textarea UI the probe would have attached to, but the probe itself would mean the admin
  origin fetching/rendering an admin-supplied external URL, which needs its own CSP/SSRF review
  distinct from this milestone's form-rendering work — unscheduled.

A `columns` block's schema-shape validation failure highlights the whole `columns` field group
rather than a specific nested block/field, since `blockvalidation.go`'s structural pass never
descends into nested block data (a pre-existing gap carried from M14.2/M14.3's own migrations, not
new here). The `block-list` widget (`columns.columns[].blocks`) is the one recursive case: each
nested item's own `blockTypeCode` is looked up against the full catalog and rendered with that
type's own `json_schema`+`ui_schema`, which is how nested block data can be a real form at all
despite never being schema-constrained itself.

**Acceptance criteria — met, verified against the running dev stack over the real HTTP/Conjure
boundary (curl/node, not a browser session — no headless path to a real Google OAuth login exists
for this app, so the rendered React form itself was verified by build/typecheck/lint against the
same widget-switch code, not by eye).** A missing-required-field write (`image` with no `alt`,
`gallery` with an item missing `alt`) returns `Content:BlockDataInvalid` with `field` set to `alt`/
`images` respectively; a disallowed-scheme write (`button.href = javascript:...`) returns
`Content:BlockUrlNotAllowed` with `field: "href"`; a well-formed write across scalar/array/nested
(`columns` with a nested `paragraph`) block types succeeds. A block type inserted directly via SQL
(simulating a future M14.13 catalog write, which doesn't exist yet) immediately serves its
`uiSchema` through `listBlockTypes` and accepts/rejects writes against its own `json_schema` with
the same per-field error behavior — with zero admin-app code change.

### M14.5 · Inserter + drag-and-drop reorder

**Built (2026-08-27).** Depends on M14.4 only in the sense that its schema-driven per-block form
needed to exist first, not on reusing any of its widgets. **This is the milestone that removes the
JSON textarea's last neighbors** — the block list's own `position` input and its single hardcoded
"new block" row, both explicitly left in place by M14.4.

A categorized, searchable block inserter (`block-inserter.tsx`) replaces the old fixed "new block"
row entirely: a shadcn `Command` palette (already installed, unused until now) grouped into **Text
/ Media / Layout / Info & contact**, each item showing a one-line description alongside its name,
filterable by either. The editor-UX research is consistent that curation beats choice — an editor
facing 14 undifferentiated types will not pick the right one — and this also fixes, as a side
effect, M14.4's documented "the new row can't preview the type you just picked" limitation: the
inserted block's type is real controlled state from the moment it's added, not a native/uncontrolled
`<Select>` pinned to `blockTypes[0]`.

**The category/description mapping is a frontend-only static table**
(`block-catalog.ts`), not a new `content_block_types` column — this milestone's own Backend/Migrated
stage-board cells are `➖`. **Corrected at M14.13, not carried out as originally anticipated here:**
M14.13 built `content.catalog.manage` without adding a `category`/`description` column to
`content_block_types` — `docs/modules/content.md`'s own canonical entity list for that table never
called for one, and this static map already degrades a block type it doesn't recognize (e.g. one a
moderator adds at runtime) to an "Other" group with just its name, never crashing or disappearing —
good enough that promoting it to a real column wasn't worth the migration for M14.13's scope.

Drag-and-drop (`@dnd-kit/core` + `@dnd-kit/sortable`, new dependencies — nothing in this codebase
did drag-and-drop before) replaces the integer `position` input, which is now computed from array
order and rendered only as a `type="hidden"` field — never visible, never editable. **Keyboard-
accessible move-up/move-down buttons are the first-class reorder path** (always visible, disabled at
the list's boundaries), not dnd-kit's own keyboard-drag sensor, which is included as a secondary
bonus but requires discovering a grab/arrow/drop gesture — materially worse-documented than a plain
icon button. Both paths call the same `arrayMove` helper so they can't drift apart. A visually-hidden
`aria-live` region announces every move/add/remove for screen-reader users. A remove button per row
(never built before this milestone) rounds out the block list's own add-remove controls, matching
what M14.4 left reserved.

Converting the block list from a server-rendered, uncontrolled `<form>` to client-managed state
surfaced a pre-existing bug, fixed in the same milestone: a failed save previously redirected back to
a page that re-fetched the *last successfully saved* list, silently discarding whatever the user had
just edited or reordered — harmless when position was a rarely-touched number, but a real
data-loss risk now that reordering is a common, one-click action. Fixed with a `sessionStorage`
snapshot taken on submit and restored on the error redirect, which also fixes the error-highlight
mechanism itself: `Content:BlockDataInvalid`'s `position` parameter indexes into the exact array
that was POSTed, which a stale server refetch can no longer be trusted to reproduce after any
reorder/add/remove.

`web/apps/admin` gained its first test runner (Vitest + React Testing Library, decided with the
owner rather than assumed) to cover the new keyboard/drag interaction logic offline — this admin
route has no headless Google OAuth login path for a real browser proof, the same constraint M14.4's
own verification already ran into.

**Acceptance criteria — met.** Blocks reorder by drag and by keyboard alone (move-up/move-down),
both persisting through the unchanged `PutBlocks` full-replace endpoint — proved directly against the
real HTTP/Conjure boundary (a site/document/blocks seeded via SQL, a dev-minted bearer token per
M14.4's own technique, then PUT with a new order / one block removed / one block appended, GET back
to confirm). The `position` integer never appears in the UI — the only `position`-named input in the
DOM is `type="hidden"`, confirmed structurally and by a dedicated Vitest assertion. The inserter is
fully keyboard-operable (Tab to open, type to filter, arrow keys, Enter to select), confirmed by a
keyboard-only Vitest interaction test — no mouse click anywhere in that test.

### M14.6 · Forward revisions, history, autosave

**Built (2026-08-28).** Depended on M14.4. The largest schema change in the arc.

New `content_document_revisions`: `document_id`, `revision_no`, a blocks snapshot, author,
`created_at`, optional label (`migrations/0025_content_revisions.sql`). `content_documents` gains
separate **published** and **draft** revision pointers. Editing a live page mutates the draft
revision only, so what visitors see cannot change until an explicit publish — Drupal's
forward-revision model, and the reason the old single-row state flip was genuinely risky for a
congregation editing its own homepage. `PutBlocks`/`GetBlocks` now read/write the draft revision in
place; publishing snapshots the draft into an immutable checkpoint (`internal/content/application/revisionsnapshot.go`,
capped at the 50 most recent per document) that a new `GetPublicBlocks` reads instead of the old
single-state row.

The admin editor autosaves into the draft on a ~10s debounce (`hooks/use-debounced-autosave.ts` —
polls the form's live serialized state, since several of `block-data-form.tsx`'s widgets don't
uniformly bubble change events a listener could catch), with a visible saved/unsaved indicator and a
manual "Save now" trigger, replacing the old `<form action={action}>` + redirect-on-save flow
entirely — a full-page navigation every ~10s would itself be the "silently over live content"
disruption the UX research warns against. New `listRevisions`/`restoreRevision` endpoints back a new
History panel with per-revision restore. `ContentPublicService` reads the published revision.

**Acceptance criteria — met.** Publish a page; edit it heavily without publishing; **the public
subdomain still serves the old text**. Publish; it flips. Restore a prior revision; the public site
follows. Autosave survives a browser refresh mid-edit without having touched the live page — the
server persists the draft on its own debounce, independent of the browser tab. `Verified` awaits CI
green on `main`.

### M14.7 · Preview

**Built (2026-08-28).** Depended on M14.6 (the draft/published split to preview) and M14.9 (the
tenant origin to preview it on). M14.8 was picked up first, out of milestone order, while this was
still blocked; both are now done.

**Design call made at build time, not spelled out in the original scoping text:** the preview token
is scoped to a **site**, not a single document. `components/site-page.tsx` is already a one-pager
that renders every document on the site together — per-page routing doesn't exist yet (M14.10) — so
there is no isolated single-document view to preview separately. "Preview the draft" means
rendering that same one-pager with every document's draft revision substituted for its published
one, which also means a document that has never been published becomes visible in preview, the
whole point of the feature. `D-ContentRevisions`'s own consequence already anticipated this: "a
draft is content, not a special code path."

The draft revision rendered through the **real public renderer** — no second, drifting preview
renderer. `components/site-page.tsx` gained an optional `previewToken` prop; when set, its two
existing data-fetch call sites (the document list, each document's blocks) swap to token-gated
reads instead of the public ones. The render tree (`app/blocks.tsx` and everything under it) is
completely unaware which source it came from.

Backend: `ContentService.createPreviewLink` (`POST /content/v1/sites/{siteId}/preview-link`,
content.manage-gated like every other draft-adjacent read) mints a stateless HS256 token
(`internal/content/application/previewtoken.go`, `golang-jwt/jwt/v5` — the same library
`internal/platform/devtoken` already uses, applied to a different problem) carrying only a site id
and a 20-minute expiry — no DB row, no revocation path, deliberately simpler than invite links'
DB-backed random-token scheme (`internal/identity/application/service.go`), since a leaked preview
link is only a problem until it expires. `ContentPublicService.listPreviewDocuments`/
`getPreviewBlocks` accept the token in place of a session — this service's caller
(`openfaithmap-web`) never holds one (D-AdminSurface) — checking the token's site against the
site/document actually being read, never trusting it just because it verifies. Every failure mode
(missing, malformed, expired, wrong-site) collapses to one `Content:PreviewTokenInvalid`, mirroring
`Forbidden`'s empty safe-args, so a caller probing the endpoints learns nothing about what exists.
`ContentPreviewHMACKey` joins `config.Install` (`var/conf/install.yml`) following `DatabaseURL`'s
ECV-encrypt-in-real-deployment precedent — unlike `DEV_ISSUER_HMAC_KEY` (dev/local-only, gated by
`GuardSymmetricIssuers`), this key is needed in every environment.

Routing needed **no change to `proxy.ts`/`lib/tenant-host.ts`**: a visitor requests plain
`/{locale}/preview?token=…` on the tenant host — the same browser-facing shape every other tenant
page already has, with `_sites/{slug}` injected invisibly by the existing `injectSitesSegment`
rewrite. The new route lives at `app/[locale]/%5Fsites/[slug]/preview/page.tsx`, calling
`SitePage(...)` as a plain awaited function rather than JSX specifically so a thrown
`Content:PreviewTokenInvalid` can be caught here and turned into a clear "this preview link is
invalid or has expired" message — never the app's generic error boundary, and never a silent
fall-back to published content. `X-Robots-Tag: noindex, nofollow` and `Cache-Control: no-store` are
a second, path-scoped entry in the `next.config.ts` `headers()` array M14.1 already established
(matched on both the pre-rewrite and post-rewrite path shapes, to be safe regardless of exactly
when Next resolves `headers()` relative to a middleware rewrite — verified against the real running
stack either way). A new `components/preview-frame.tsx` client component supplies the device-width
toggle (mobile/tablet/full) by constraining a wrapping `<div>`'s `max-width` — deliberately not an
iframe, so there remains exactly one render pipeline, not two that could drift.

The admin editor's document page (`sites/[unitId]/documents/[documentId]/page.tsx`) mints a link
via `createPreviewLink` on every render and opens it in a new tab (`target="_blank"`, never
embedded) next to the existing Publish/Unlist/Revert-to-draft buttons — the WordPress CVE pattern
this milestone exists to avoid is untrusted content rendering **inside the admin origin**, and a
new-tab link on the tenant origin makes that impossible by construction rather than by convention.
`buildPreviewUrl` (`lib/content.ts`) points at a new `TENANT_APEX_HOST` env var
(`openfaithmap-web`'s own published host:port in local dev, "localhost:3002" — the real apex once
**U14** resolves), since this admin app had no prior reason to know that host.

**Acceptance criteria — met, verified against a real running docker-compose stack.** A document
created and left in `DRAFT` (never published) — via a direct authenticated HTTP call, no admin UI in
the loop — renders through `http://grace.localhost:3002/en/preview?token=…` with its draft text
visible, while the same site's public root (`/en`) shows nothing of it. The identical request with
no token, or with a garbage token, renders the "invalid or expired" message instead — confirmed by
grep that the draft text appears zero times in either response. A token minted for a different site
is rejected (`403 Content:PreviewTokenInvalid`) when used against this site's document. The response
carries `X-Robots-Tag: noindex, nofollow` and `Cache-Control: no-store`. `internal/content/content_integration_test.go`
covers the same shapes against real Postgres (never-published doc visible via `GetPreviewBlocks`,
excluded from `ListPublicDocuments`, rejected for missing/garbage/wrong-site tokens); a dedicated
`internal/content/application/previewtoken_test.go` covers expiry, HMAC-key mismatch, wrong
`purpose` claim, malformed input, and `alg: none` confusion — cases a real clock in an integration
test can't cleanly exercise. No congregation content is ever rendered inside the admin origin: the
preview link is a plain external anchor, not embedded. `go test ./internal/content/... -run TestContentIntegration`,
`make verify`, and `make sdk-verify` all pass; `web/apps/web` and `web/apps/admin`'s
`npm run lint && npm run test && npm run build` all pass. `Verified` awaits CI green on `main`.

### M14.8 · Editor polish

**Built (2026-08-28), out of milestone order.** Depended on M14.5 only — M14.7 above is the actual
next milestone in numeric order, but it's blocked on M14.9 (not started), so this one was picked up
instead since it carries no such dependency. No backend change.

A new hook, `hooks/use-block-history.ts`, gives `block-list-editor.tsx` session-local undo/redo:
it polls the same serialized form snapshot `useDebouncedAutosave` already reads (independent poll/
settle constants, no coupling between the two hooks), and once a change holds steady past a short
settle window, the previous snapshot becomes an undo step. This one mechanism covers add, delete,
reorder, and edit with no changes to the block list's existing mutator functions. Restoring a
snapshot re-keys every row so `BlockDataForm` — which owns each block's field data as local state,
never lifted into the parent list — remounts and re-seeds correctly; the alternative (lifting that
state up to make the form fully controlled) would have meant rewriting its JSON-textarea fields'
own deliberately-separate draft-state mechanism for no additional benefit. Ctrl/Cmd+Z and
Ctrl/Cmd+Shift+Z (mirroring `components/command-palette.tsx`'s existing global-shortcut pattern) sit
alongside toolbar Undo/Redo buttons, both disabled at stack boundaries; the keyboard shortcut is
skipped while focus is inside a text field so native per-field undo isn't hijacked. One genuine
timing hazard, caught by testing against real Radix `Select` components rather than mocks: a
mount-time baseline would sometimes latch before Radix's hidden native `<select>` mirror (used for
each block's type field) had rendered, producing an incomplete snapshot that looked like an edit
had already happened at page load. Fixed by latching the baseline on the first poll tick instead of
a separate mount effect.

A document with zero blocks now shows a real empty state (heading, body copy, and a CTA into the
existing block inserter) instead of an empty form. **Scoped down from "start from a template," named
here rather than silently dropped, matching this arc's own convention (M14.1/M14.3/M14.4 each did
the same):** M14.13 (`content_patterns`, starter patterns) doesn't exist anywhere in this repo yet,
so there is no template to start from — M14.13, when built, is what would make this empty state
offer real starter layouts.

The two `?error=`-redirect round trips still left in the document editor after M14.4/M14.5/M14.6
already moved block-level errors inline — the document-details save form and the new-document
create form — are now inline too, via `useActionState`, the same mechanism `people/invite/invite-form.tsx`
already established in this app (there for a different reason: a one-time secret that can't survive
a URL). New `document-details-form.tsx` and `new-document-form.tsx` client components carry the
forms; both pages drop their `searchParams: { error? }` prop entirely.

The block row's fixed `grid-cols-[auto_auto_10rem_1fr_auto]` layout (`SortableBlockRow` in
`block-list-editor.tsx`) — the literal "desktop form grid" the milestone text names — now stacks to
a single column below the existing `sm:` breakpoint convention already used elsewhere in this app
(e.g. `audit-log-list.tsx`), reverting to the fixed-column grid at `sm:` and above. `block-data-form.tsx`
itself needed no changes — already `flex flex-col`/`w-full` throughout.

**Acceptance criteria — met.** Undo/redo works across add, delete, reorder, and edit (including
restoring a field's actual data, not just list shape) — covered by `hooks/use-block-history.test.ts`
and extended `block-list-editor.test.tsx` cases, run against real Radix/dnd-kit components via
Vitest + RTL fake timers (this admin route still has no headless OAuth login path for a real
browser-level proof, the same constraint M14.4/M14.5 already worked around). The block row stacks to
a single column below `sm:` — a jsdom regression test can only assert the responsive class strings
are present, since jsdom doesn't evaluate real media queries; **a real-browser 375px check was not
performed in this session** (this admin route has no headless OAuth login path, the same constraint
M14.4/M14.5 already named for their own browser-level verification), named here rather than silently
assumed. Zero `?error=` query-string round trips remain anywhere in the document editor route tree.

### M14.9 · Tenant subdomain routing (Phase 1)

**Built (2026-08-28).** Depends on M14.0 (`D-TenantSubdomains`). Implements the owner's routing
design and U16's `content.manage` tightening.

**Filename correction, found at build time, not scoping time.** The milestone's own text (and
`D-TenantSubdomains`, since corrected) named `web/apps/web/middleware.ts` — that file never existed
under that name in this codebase: Next.js 16 renamed the middleware entrypoint to `proxy.ts` before
this decision was written, and `web/apps/web/proxy.ts` already existed, running `next-intl`'s own
locale middleware. Host-based tenant resolution is composed into that existing file rather than
added as a second one — confirmed with the owner rather than assumed.

`proxy.ts` resolves the `Host` header to a site slug (`lib/tenant-host.ts`, pure functions,
vitest-covered in isolation — a mistake here is either "the apex stops serving discovery" or "a
tenant site leaks onto the apex," not something to get right only by eyeballing the proxy) and
rewrites into an internal `/[locale]/_sites/[slug]/…` tree; the apex host continues to serve
discovery, search, and the registration entry point unchanged. **Direct `/_sites/*` access from the
apex host 404s** — checked first, before `next-intl` even runs, so it is a hard boundary rather
than something the app-router tree could accidentally bypass. Composing with `next-intl`'s own
`localePrefix: "always"` middleware needed one real subtlety worked out: injecting `/_sites/{slug}`
must be a rewrite (invisible in the address bar), but it can only happen once the locale prefix is
already settled, or the redirect that adds it would itself leak the un-prefixed tenant path into
the browser. `proxy.ts` runs `next-intl`'s middleware on the original request first; if it wants to
redirect (adding the prefix), that redirect goes out as-is and the browser lands back here already
prefixed on the next pass; only then is the rewrite built, copying over any headers `next-intl` had
already set (notably the `NEXT_LOCALE` cookie) onto the response actually returned — verified with
a real `curl -v` against the running stack that the cookie survives.

**The actual route directory is named `%5Fsites`, not `_sites`.** Next.js treats a folder prefixed
with `_` as a private, unroutable implementation-detail folder — `app/[locale]/_sites/[slug]/`
would silently never register as a route at all. `%5F` (the URL-encoded form of an underscore) is
Next's own documented escape for a URL segment that starts with an underscore while the folder
itself stays routable — confirmed against the Next.js docs before building, not discovered by a
failing request. The real rendering logic was extracted out of the old
`congregations/[unitId]/page.tsx` into `components/site-page.tsx`, taking an already-resolved
`Site` rather than fetching one itself — "an extractable module," so the owner's Phase 2
(`openfaithmap-sites` as its own container) is a move, not a rewrite. The new
`app/[locale]/%5Fsites/[slug]/page.tsx` is a thin wrapper: `getSiteBySlug` (new
`ContentPublicService` endpoint, `GET /site-by-slug/{slug}` — its own top-level path, not nested
under `/sites/{id}/...`, for the same static-vs-wildcard httprouter conflict `getSite`'s own
existing comment documents) then render.

A **reserved-subdomain blocklist enforced server-side**
(`internal/content/application/slugvalidation.go`), because `content_sites.slug` stops being a path
segment and becomes a hostname the moment this ships: the milestone-named
`admin`/`api`/`auth`/`login`/`www`/`app`/`mail`/`static`/`support`/`billing`/`help`/`status`, plus
25 more spanning this repo's own other surfaces, mail/DNS hygiene, auth-adjacent phishing-lookalike
names, generic ops/infra, and environment/lifecycle names (`dev`/`staging`/`preview`/…) — decided
concretely rather than left at the milestone's own "and more." Checked in `CreateSite`, before the
existing `Content:SlugTaken` uniqueness probe, returning a new typed `Content:SlugReserved` error —
enforced in `internal/content`, not the UI (the admin form's `pattern="[a-z0-9-]+"` stays
format-only). Live-verified with a direct HTTP call against the running stack, bypassing the admin
form entirely: `POST /content/v1/sites {"slug":"admin"}` from a real `congregation-admin` session
returns `400 Content:SlugReserved`.

**301s from `/congregations/[unitId]`.** The milestone spec says a real 301; Next's own
`redirect()`/`permanentRedirect()` send 307/308, not 301 (RFC 7538). `app/[locale]/congregations/[unitId]/page.tsx`
became a Route Handler (`route.ts`) instead, since a real 301 needs one — looks up the site by unit
RID (`getSite`, unchanged), builds a same-port URL on `{slug}.{host}`, and redirects to the tenant
root with `NextResponse.redirect(target, 301)`. Deliberately targets the root rather than trying to
preserve the caller's original locale/path in one hop — `next-intl`'s own redirect re-adds the
locale prefix on the next hop, simpler and correct in every case since the old route was never
itself locale-aware beyond the prefix.

**U16 (D-TenantSubdomains' ruling): `content.manage` is now its own permission.**
`internal/authz/domain/permissions.go` gained `PermContentManage` ("content.manage");
`migrations/0026_content_manage_permission.sql` grants it to `congregation-admin` only — the same
unit-scoped shape M13.2 already used for `site.manage`, not a subtree grant, and
`registration-operator`'s existing grants (including `religionorg.manage`, still needed for
approving registrations) are untouched. `internal/content/application/authorize.go`'s
`requireManage` now checks `PermContentManage` instead of `PermReligionOrgManage` — the entire fix
is in which permission code gets checked, not in revoking anything from the operator role.
**Confirmed with the owner: registration operators are left with no replacement edit path for
now** — granting them a moderation permission instead is a separate, later decision, not part of
this milestone. `docs/modules/content.md`'s previously-named test-coverage gap (a cross-tenant
`content.manage` denial was untested) is closed: new cases in
`internal/content/content_integration_test.go`, run against real Postgres, prove both that a
`congregation-admin` granted on one unit is denied on an unrelated one, and that a
`registration-operator` granted the exact same unit-scoped shape is denied too — proving
`content.manage` itself gates the write, not incidental scope.

**Acceptance criteria — met, verified against a real running docker-compose stack.**
`http://grace.localhost:3002/` serves that congregation's site — a real `content_sites` row with
`slug: "grace"` created over HTTP by a real `congregation-admin` session (`*.localhost` needs no DNS
change; browsers resolve it to loopback), then `curl -H "Host: grace.localhost:3002"` returns a 307
to `/en` with `Set-Cookie: NEXT_LOCALE=en` preserved, then a 200 rendering the tenant page.
`http://localhost:3002/_sites/grace` and `http://localhost:3002/en/_sites/grace` both **404**, from
the apex host, while the apex root still serves discovery unchanged. A reserved slug is rejected at
the API, not just in the form — a direct `POST /content/v1/sites {"slug":"admin"}` (no admin UI in
the loop at all) returns `400 Content:SlugReserved`. An old `/en/congregations/[unitId]` URL returns
a real `301` to `http://grace.localhost:3002/`, which then 307s to `/en` and renders 200.
`go test ./internal/content/... -run TestContentIntegration`, `make verify`, `make sdk-verify`, and
`web/apps/web`'s `npm run lint && npm run test && npm run build` all pass. `Verified` awaits CI
green on `main`.

### M14.10 · Navigation + page routes

**Built (2026-08-29).** Depends on M14.9 (the tenant-subdomain route tree `/_sites/[slug]/…`). Two
decisions confirmed with the owner before building, since the milestone text left them open: (1) the
site root (`/`) drops its old "every published Page rendered inline" section entirely — Pages are
reachable only via their own route or the nav menu, while the Posts/Events feed and discovery header
on the root are untouched; (2) nested child pages get **hierarchical URLs**
(`/parent-slug/child-slug/grandchild-slug`) that mirror the real `parent_document_id` tree exactly,
not a flat `/slug` regardless of depth — chosen because `content_documents.slug` is unique only per
`(site_id, kind, locale)`, not per parent, so a flat scheme would technically work but defeat the
point of a real path; a wrong ancestor segment anywhere in the URL 404s, never silently resolved by
the last segment alone.

**Nav model resolved at M14.0 (2026-08-27): a hand-built menu, not page-tree-derived.** New
`content_site_nav_items` (`migrations/0027_content_nav_items.sql`): `site_id`, `label`,
`target_document_id`/`target_url` (a same-table `CHECK` enforces exactly one is set;
`target_document_id` is `ON DELETE CASCADE` since a `SET NULL` would violate that same `CHECK`),
`sort_order` (unique per site). No soft-delete, no `updated_at` — mirrors
`content_document_revisions`' own precedent for a table whose rows are wholesale-replaced, never
mutated in place. `parent_document_id` still governs page nesting and breadcrumbs; it no longer
drives the nav menu itself.

**Backend** (`api/content.conjure.yml` + `internal/content`): `ContentService.putNavItems`
(content.manage-gated, full replace — mirrors `putBlocks`' own precedent, not per-item CRUD) and
`listNavItems`; `ContentPublicService.listPublicNavItems` (returns each item's target already
resolved to a ready-to-render `href`, silently omitting an item whose target document is missing or
still `DRAFT` — never a broken link) and `getPublicDocumentByPath` (resolves the leaf `PAGE`
document by `site+kind+locale+slug` plus its real ancestor chain in one round trip, for the
catch-all route's breadcrumbs). Three new typed errors: `Content:NavTargetInvalid` (target isn't a
`PAGE` in this same site), `Content:NavTargetAmbiguous` (neither or both of
`targetDocumentId`/`targetUrl` set), `Content:DuplicateNavItemSortOrder` (rejected up front, same
discipline as `DuplicateBlockPosition`). `application.Service.resolveAncestorChain` (bounded at 3,
mirroring `checkParentDepth`'s own bound) is shared by both `listPublicNavItems` and
`getPublicDocumentByPath` so the walk is never duplicated. `ReplaceNavItems`
(`adapters/repository.go`) is a real transaction (delete-then-insert), mirroring `InsertDocument`'s
tx shape rather than `PutBlocks`' — since M14.6, `PutBlocks` is a single-row JSON update, the wrong
precedent for a genuinely relational table.

**`web/apps/web`**: new catch-all route
`app/[locale]/%5Fsites/[slug]/[...pageSlug]/page.tsx` (1-to-3 segments; `>3` 404s before any API
call) renders one document via new `components/page-document.tsx`, with a new
`components/breadcrumbs.tsx` shown only at depth ≥ 2 (ancestor labels are the ancestor's own slug,
humanized — documents have no title field in this schema). New
`app/[locale]/%5Fsites/[slug]/layout.tsx` fetches the site + nav menu once and renders bare nav
chrome around every route nested under `[slug]/` (deliberately minimal — full header/footer chrome
is M14.11's job). `components/site-page.tsx` drops its inline Pages section; Posts/Events sections
are unchanged. `getSiteBySlug` wrapped in React's `cache()` so the new layout and its sibling routes
dedupe to one fetch per request. No `proxy.ts`/`lib/tenant-host.ts` change was needed —
`injectSitesSegment` already passed arbitrarily deep paths through untouched.

**`web/apps/admin`**: new `sites/[unitId]/nav/page.tsx` + `nav-item-list-editor.tsx` — reuses
`block-list-editor.tsx`'s dnd-kit reorder **idiom** (not its code, a parallel component: this table
is a different shape than blocks). An explicit **Save** button, not autosave — a menu with a
handful of rows edited rarely doesn't earn M14.6's debounce/indicator complexity. Per row: a label
input, a Page/External-URL toggle, and either a page `<Select>` (reusing the same picker pattern
`document-details-form.tsx` already established) or a URL input — never both, enforcing the
exactly-one-of invariant at the UI layer too. A "Manage navigation" button joins the existing
"Manage pages" one on `sites/[unitId]/page.tsx`. New `nav-item-list-editor.test.tsx` (Vitest +
RTL, following `block-list-editor.test.tsx`'s exact setup) covers add/remove/keyboard-reorder,
mode-toggle clearing the other field, and the submitted payload shape.

**Acceptance criteria — met, verified against a real running docker-compose stack (`grace.localhost:3002`,
a real 3-level published page tree created over direct authenticated HTTP calls, no admin UI in the
loop).** A depth-3 page (`/en/verify-top/verify-child/verify-grandchild`) renders its own content
with a breadcrumb showing both ancestors linked correctly; a depth-1 page shows no breadcrumb. A
wrong **middle** URL segment 404s (`/en/verify-top/wrong-slug/verify-grandchild`), proving positional
ancestor matching, not last-segment-only resolution. `>3` segments 404s. A never-published (`DRAFT`)
page 404s on its direct URL, same "draft is never public" invariant as `GetPublicBlocks`. The nav
menu renders in `sort_order` with an external item opening in a new tab
(`target="_blank" rel="noopener noreferrer"`) and an internal item at its real hierarchical href; a
nav item targeting the still-`DRAFT` page is silently absent from the rendered menu. The site root
no longer inline-renders any Page content (confirmed by grep: none of the three pages' body text
appears in the root response), only the existing Posts/Events feed. A cross-site
`targetDocumentId` is rejected with `400 Content:NavTargetInvalid` over a direct HTTP call to
`putNavItems`, bypassing the admin form entirely. `go test ./internal/content/... -run
TestContentIntegration`, `./godelw verify`, and `make sdk-verify` all pass; `web/apps/web` and
`web/apps/admin`'s `npm run lint && npm run test && npm run build` all pass. `Verified` awaits CI
green on `main`.

### M14.11 · Site chrome — header, footer, template parts

**Built (2026-08-29).** Depends on M14.10 only in the sense that the header renders the same nav
menu M14.10 built (`listPublicNavItems`) — no code from that milestone was reused otherwise.

**Two scoping calls made with the owner before building, since the milestone text left both open:**
"contact details" turned out to have no backing field at all — `religion_sites` has no phone/email
column anywhere in the schema — so the footer's contact details are the site's address only for
now; adding phone/email is a separate future milestone, not silently substituted with something
else. And the live religion-data path (address + service-schedule *times*, which no endpoint
exposed before this milestone — only bare day-of-week ints via `DiscoverySite`) is composed
**server-side, inside `content`**: `internal/religion/application.Service` is now injected into
`internal/content/application.Service`'s constructor, the same direct-interface-call shape
`internal/discovery` already established against `religion` (`docs/architecture/conventions.md`) —
this is the first time `content` itself, not just `discovery`, reads religion's live data. The
alternative (a second public endpoint on religion or discovery, fetched separately by the frontend)
was rejected in favor of one bundled call the tenant layout makes once.

**Backend.** `content_sites` gains `logo_url text` and `social_links jsonb NOT NULL DEFAULT '{}'`
(`migrations/0028_content_site_chrome.sql`) — a small named-field `SocialLinks` struct
(facebook/instagram/youtube/twitter/website), not a free-form map, so the renderer can show a known
icon per field deterministically. New `ContentService.updateSiteChrome` (`PUT
/sites/{siteId}/chrome`, content.manage-gated, full replace — same shape as `updateSiteTheme`). New
`religionapplication.Service.GetPrimarySiteByUnit` (a hand-written query reusing `SearchSites`' own
`siteCols`/`siteFrom` projection for the `directory_units` name join, deliberately **not**
`SearchSites`' own `visibility='public'`/non-hidden filtering — a congregation's own site must show
its own name regardless of discovery visibility, only the *address* is precision-gated) and
`ListServiceSchedulesByUnit` (a new query against `religion_service_schedules` — the first place
this table's individual rows, not just `SearchFacets`' aggregated day/language facets, are read
back out), both left ungated like `ListSitesByUnit` — public-safe data, not owner-only like
`GetSiteByUnit`/`site.manage`. New `ContentPublicService.getSiteChrome` (`GET
/sites/{siteId}/chrome`, anonymous) composes `content_sites`' own logoUrl/socialLinks with
religion's live congregationName/address (address run through the existing
`CoarsenAddress`/`PublicPrecision` gate — a `hidden`-precision site shows no address, but its name
still shows, since a site's own name on its own subdomain isn't the discovery-search leak
`D-DiscoveryAddressPrecision` guards against) and schedules; degrades to
`{congregationName: <slug>, address: nil, schedules: []}` rather than erroring if the unit has no
religion site yet at all.

**`web/apps/web`.** New `components/site-header.tsx` (logo/name, and the nav list extracted out of
`layout.tsx`'s old bare `<nav>`) and `components/site-footer.tsx` (address, service schedule
grouped by day using the same `DiscoveryMap.day0`–`day6` labels `site-page.tsx` already uses,
social links) — both fed by one new `getSiteChrome` call the tenant layout makes alongside its
existing `getSiteBySlug`/`listPublicNavItems` fetches. `SiteFooter` renders nothing at all if the
chrome has no address, no schedules, and no social links, rather than an empty shell.

**`web/apps/admin`.** No new route: `logoUrl`/`socialLinks` are a handful of fixed optional fields
— structurally identical to the existing `theme` settings, not a dynamic list like the nav menu —
so a new "Site chrome" card joins the existing Theme/Accessibility cards directly on
`sites/[unitId]/page.tsx`, reusing that same plain `<form action={...}>` server-action shape rather
than reaching for `nav-item-list-editor.tsx`'s client-state machinery, which exists specifically for
add/remove/reorder that this form doesn't need.

**Acceptance criteria — met, verified two ways.** `internal/content/content_integration_test.go`
(run against real Postgres) covers: `GetSiteChrome` degrading gracefully before any religion site
exists; `UpdateSiteChrome` content.manage-gated the same way `UpdateSiteTheme` is, persisting
logoUrl/socialLinks; `GetSiteChrome` composing a real `congregationName`/coarsened `address`/one
`ServiceSchedule` row (day, start/end time, language) once a real `religion_sites` +
`religion_service_schedules` fixture exists for the unit; and a `hidden`-precision religion site
hiding the address while the congregation's own name still renders. `go test ./internal/content/...
./internal/religion/...`, `./godelw verify`, and `make sdk-verify` all pass; `web/apps/web` and
`web/apps/admin`'s `npm run lint && npm run test && npm run build` all pass. **Not verified this
session:** a live browser check against the running docker-compose stack — this session ran
alongside other active sessions sharing the same stack, and rebuilding the shared `openfaithmap-api`
image to hot-swap in this branch's code risked disrupting their work, so that check is left for a
dedicated verification pass rather than silently skipped. `Verified` awaits CI green on `main`.

### M14.12 · Curated theme tokens

**Built (2026-08-29).** Depends on M14.11.

**Correction to the original scoping text, found at build time:** `content_sites.theme` was not
actually unread — an M3-era editor already existed
(`web/apps/admin/app/[locale]/admin/sites/[unitId]/page.tsx`) with three **raw text inputs**
(`accentColor`/`fontPairing`/`headerLayout`) and zero validation, and the backend
(`UpdateSiteTheme`) passed the submitted bytes straight to storage. `web/apps/web` never read
`.theme` at all — the public renderer applied no theme styling before this milestone. M14.12's real
job was retrofitting a real, curated schema onto an already-live but unvalidated feature, not
greenfield work — safe to do as a hard replacement, since no live congregation content exists yet
(the same fact M14.3's migration already relied on).

A real schema for the column, enforced structurally by a fixed JSON Schema (`enum` per field, the
same `santhosh-tekuri/jsonschema/v6` library `blockvalidation.go` already uses) rather than merely
requested in the UI: **accent** from an 8-color curated palette (indigo/violet/rose/amber/emerald/
teal/sky/slate), **mode** (light/dark/system), **fontPairing** from three system-font-stack pairings
(Modern Sans/Classic Serif/Friendly Rounded — no web fonts, no new dependency, no CSP allowlist
change, confirmed with the owner), **spacing** (compact/comfortable/spacious), and **headerLayout**
(logo-left/centered/stacked) — WordPress's `theme.json` lesson, a fixed vocabulary, never free-form
CSS or a raw hex/font entry. Emitted as CSS custom properties (`internal/content/application/
themetokens.go` is the Go source of truth; `web/apps/{admin,web}/lib/theme-tokens.ts` duplicate the
token→value table, the same "frontend-only static table" precedent M14.5's `block-catalog.ts`
established, since no shared package exists between the two Next apps) — overriding `--primary`/
`--primary-foreground` (already read by every shadcn-derived Tailwind utility used throughout
blocks, so accent propagates with no new CSS classes anywhere) and `--font-heading` on a wrapper the
tenant layout renders around header/children/footer; a forced light/dark mode also overrides
`--background`/`--foreground`/`--card`/`--border` to the same values `globals.css`'s own media-query
block already uses. `SiteHeader` gained a `layout` prop branching the three curated arrangements —
previously the component had exactly one, undifferentiated layout.

**A WCAG AA contrast check runs at write time and rejects a failing combination**
(`internal/content/application/themevalidation.go`'s `checkThemeContrast`, real relative-luminance
math against the actual `#FFFFFF`/`#0A0A0A` background values `globals.css` renders, not a
placeholder) — computed, not curated by hand, so the palette has genuine pass/fail variation rather
than every entry being pre-guaranteed safe: a dark-saturated accent (e.g. indigo) fails against the
near-black dark background, a bright accent (e.g. amber) fails against the white light background,
and `system` mode is checked against both, making it the most restrictive of the three. A typed
`Content:ThemeContrastFailed{accent, mode}` names the failing pair; a value outside the curated
vocabulary gets a separate `Content:ThemeInvalid{field}` — both mirror `BlockDataInvalidError`'s
existing safe-arg discipline (a curated token name, never a raw submitted value, ever reaches the
error).

Live theme preview in the admin: `theme-form.tsx` is a client component (the M14.12 acceptance
criterion needs the preview to update before the form is ever submitted, unlike the plain
`<form action={...}>` chrome card next to it) rendering curated `<select>`s plus a swatch that
recomputes from the same duplicated token table as the public site.

**Acceptance criteria — met, verified against the running docker-compose stack** (the API's real
app port isn't published to the host per `D-HeadlessTopology`, so verification ran from inside the
`openfaithmap-web` container, on the compose network, against `https://openfaithmap-api:3000`): a
curated theme (`emerald`/`light`/`classic-serif`/`spacious`/`centered`) written over a real
authenticated HTTP call renders as real CSS custom properties (`--primary:#047857`, the correct
auto-computed white foreground, the classic-serif `--font-heading` stack, `--of-space-scale:1.35`)
and the `centered` header markup on `grace.localhost:3002`'s live response; a raw-hex accent
(`"#ff00ff"`) is rejected with `400 Content:ThemeInvalid{field: "accent"}`; an individually-valid
but contrast-failing pair (`indigo`/`dark`) is rejected with `400
Content:ThemeContrastFailed{accent: "indigo", mode: "dark"}`, and neither rejected write mutated the
stored theme. A pre-M14.12 site with the old free-text shape (`{"accentColor": "F2F230", ...}`,
found live in the local dev database) still renders with no crash — `parseTheme` degrades an
unrecognized shape to "no theme set," the same defensive-parsing precedent M14.2's renderer
established for a legacy shape it doesn't recognize. `internal/content/content_integration_test.go`
covers the same three cases against real Postgres; `themevalidation_test.go` unit-tests the
curated-vocabulary gate, the contrast gate in both directions, and the WCAG math itself against
known reference ratios (pure black/white ≈ 21:1). `go test ./...`, `./godelw verify`, and
`make sdk-verify` all pass; both Next apps' `tsc --noEmit`, `eslint`, and `next build` all pass.
`Verified` awaits CI green on `main`.

### M14.13 · Starter patterns + block-type catalog admin

**Built (2026-08-30).** Depended on M14.4 (patterns insert blocks that need working forms). Finally
builds what M3 deferred.

New `content_patterns` (`migrations/0029_content_patterns.sql`, same table shape/conventions as
`content_block_types`: plain uuid PK, soft-delete, an `updated_at` trigger) with WordPress's
**unsynced** semantics. **Design call made at build time:** inserting a pattern needs no dedicated
backend mutation at all — `content_patterns.blocks` is already a full `BlockInput`-shaped snapshot,
the same shape a document's own block list uses, and the admin editor already holds its current
block list client-side and persists the whole thing through the existing `putBlocks` full-replace
endpoint (manual save or the ~10s autosave). So "insert a pattern" is: fetch every pattern once via
a new public `listPatterns` (`ContentPublicService`, no auth — not sensitive data, same reasoning
`listBlockTypes` already uses), and on selection, map the chosen pattern's `blocks` into fresh-keyed
`ClientBlock`s and append them via the exact same `setItems` path `insertBlock` already uses
(`block-list-editor.tsx`'s new `insertPattern`) — one undo/redo step, autosaved and schema-validated
identically to a hand-authored block, with no special-casing anywhere. Seeded 5 church-specific
starters: Parish home page, Service times, Meet the clergy, Getting here, Feast-day announcement.

Builds the `content.catalog.manage` endpoints M3 left unbuilt: `listBlockTypesForCatalog` (every
status), `createBlockType`, `updateBlockType`, `createPattern`, `updatePattern`, `deletePattern` —
all on `ContentService`, gated by a new `requireCatalogManage`
(`internal/content/application/authorize.go`) that mirrors `internal/moderation`'s own
`requireModerate` exactly: the same `PermModerationStanding` permission, checked against the shared
root unit (`Config.RootUnitID`, wired the same way `register_moderation.go` already does), not a new
authority concept — D-SitePatterns' own explicit call. `content.Service` gained a `Config` struct
and `cfg` field to carry `RootUnitID`, the first content-module addition of its kind.

**Owner decision (asked and confirmed this session):** `updateBlockType` locks `json_schema`/
`ui_schema` after creation — `UpdateBlockTypeRequest` has no such field at all, so this is structural,
not just an application-layer check. Only `name`/`status`/`sortOrder` are editable; a moderator
wanting a different shape retires the old type and creates a new one. Rationale: a runtime catalog
edit has no migration/backfill safety net the way a real repo migration does, so a schema change to
an *existing* type could silently break already-saved blocks or the admin form built from its old
shape. `createBlockType` still smoke-tests a new type's submitted schema compiles
(`compileBlockTypeSchema`, reusing `blockvalidation.go`'s own `jsonschema/v6` compile call) before
seeding the row, so a broken schema is rejected at creation, not discovered on the first `putBlocks`
that references it.

Two new top-level admin routes, `/admin/block-types` and `/admin/patterns` — following
`/admin/moderation`'s own precedent exactly: no local frontend role gate, shown unconditionally in
the sidebar nav (`components/admin-sidebar.tsx`'s main `NAV`, not the unrelated `SUPER_ADMIN_NAV`/
`isInstanceAdmin` group, a structurally different authority), a non-moderator's call simply comes
back `Content:Forbidden` server-side. Both use the plain `?error=`-redirect form-action shape still
used elsewhere in this app outside the document editor (e.g. `sites/[unitId]/page.tsx`) — M14.8's
"zero `?error=` round trips" discipline was scoped to the document editor specifically, not declared
app-wide. `jsonSchema`/`uiSchema` are entered as raw JSON textareas at block-type creation (no
per-field form exists for the catalog's own schema-of-schemas); a pattern's `blocks` are edited the
same way, in the identical wire shape `putBlocks` already accepts.

**Named, accepted scope boundary, found by reading the code rather than assumed:**
`web/apps/web/app/blocks.tsx` dispatches on a hardcoded `switch (blockTypeCode)` with a documented
no-op fallback for unknown codes. A block type a moderator adds at runtime works immediately in the
admin inserter and form (M14.4/M14.5 are already schema-driven) but will **not** render on the
public site until a developer adds a new `case` — this milestone's acceptance criteria only
required the inserter/form half, not full end-to-end runtime rendering; making the public renderer
schema-driven too is a separate, materially larger change.

**Acceptance criteria — met.** A document with zero blocks builds a plausible homepage from a
pattern without hand-authoring a block (`PatternInserter`, mirroring `BlockInserter`'s Command-
palette shape). An inserted pattern is immediately just ordinary, freely-editable blocks — unlinked
from its source pattern by construction, never referenced again. A moderator adds a block type at
runtime and it appears in the inserter with a working form; a non-moderator is refused — proved two
ways, not inferred: `internal/content/content_integration_test.go`'s M14.13 section (real Postgres,
direct `application.Service` calls, mirroring `internal/moderation/moderation_integration_test.go`'s
own grant-then-call pattern) and a live-HTTP pass against the real running docker-compose stack
(dev-minted tokens per `scripts/mint-local-token`, `docker exec ... curlimages/curl` for non-GET
methods since BusyBox `wget` is GET-only): anonymous 401, an authenticated non-moderator 403, a real
`platform-moderator` grant 200 on both `/content/v1/catalog/block-types` and `/catalog/patterns` —
a block type created, confirmed present on the public `listBlockTypes`, retired, confirmed absent;
a pattern created, confirmed present on the public `listPatterns`, deleted, confirmed absent.
`web/apps/admin`'s Vitest suite (M14.5's own precedent) gained a dedicated pattern-insertion test.
`Verified` awaits CI green on `main`.

### M14.14 · Locale switching — closes `DS-OFM-7`

**Built (2026-08-30).** Depended on M14.10 (per-page routes to attach the picker to). Closes an
open seam the arc would otherwise walk past — no migration needed, since
`content_documents.translation_group_id`/`locale` and their index have existed since M3.

**Two design forks, resolved with the owner before building:**

1. **The picker + `hreflang` are per-page, in-content — never the shared site header/footer.** The
   header/footer (`layout.tsx`) wraps every route under a site, including the root posts/events
   feed, which has no single translatable document behind it — only individual `PAGE` documents
   (the `[...pageSlug]` route) have a translation group to offer. A picker that could lead to a 404
   is explicitly worse than no picker, so precision won over a more prominent but riskier chrome
   placement. `ContentLocalePicker` (`components/content-locale-picker.tsx`) renders inline inside
   the page route itself, right below the breadcrumb.
2. **Content locale and the site chrome's UI language are decoupled.** Previously one URL segment
   did double duty: next-intl's `[locale]` (fixed to 4: en/uk/es/pt) was also the exact value
   matched against a document's own `locale` column. The owner wants a congregation to author a
   page in *any* language, not only the 4 chrome locales, so the tenant PAGE route grew a second
   segment: `/{uiLocale}/{contentLocale}/{pageSlug...}` — the moved route is now
   `app/[locale]/%5Fsites/[slug]/[contentLocale]/[...pageSlug]/page.tsx`. `uiLocale` never changes
   when a visitor switches content locale via the picker, or when `hreflang`'s alternates are built.
   `proxy.ts`/`lib/tenant-host.ts` needed no change — `injectSitesSegment` already passes
   arbitrary-depth paths through untouched.

**Backend.** `DocumentWithAncestors` (Conjure) grew a `translations: list<DocumentTranslation>`
field — every `PUBLISHED` document sharing the leaf's `translationGroupId` (leaf included), each
resolved through its own ancestor-chain walk and `buildPublicHref` call, since a sibling can sit at
a different hierarchy/slug per locale — a translation's href is never derived from the leaf's own.
One round trip, the same precedent `ancestors` itself already set (M14.10). A new store query,
`ListDocumentsByTranslationGroup` (deliberately not scoped by `site_id` — see below), backs both
this and `CreateDocument`'s new guard.

`CreateDocument`'s "join an existing translation group" path has existed, if manually, since M3 —
`web/apps/admin`'s new-document form already had a raw `translationGroupId` text input an admin
could type into by hand. It had two gaps, both app-level since `content.md`'s own invariant is that
a group's documents share nothing but the group id (no DB constraint backs either): a caller could
join a group that already has a document at the requested locale, or a group belonging to a
*different* site entirely (`translation_group_id` has no FK to site). `checkTranslationGroup`
(`application/service.go`) rejects both, with two new typed errors, `Content:TranslationLocaleTaken`
and `Content:TranslationGroupNotFound`.

**Public site.** `generateMetadata` — the first anywhere in `web/apps/web` — emits
`alternates.languages` from the same `translations` list, absolute URLs preserving the current
`uiLocale`. `getPublicDocumentByPath` is now `cache()`-wrapped (matching `getSiteBySlug`/
`getSiteChrome`'s own precedent) since `generateMetadata` and the page component both resolve the
same document per request. `Breadcrumbs` now takes `uiLocale`/`contentLocale` separately instead of
one `locale`.

**Editor-side.** A new Translations panel on the document editor page, following History/Preview's
own inline-`Card` shape (no tabs, no modal): lists this document's translation-group siblings by
filtering `listDocuments(site.id)` — the same "no dedicated endpoint, filter what you already have"
convention `new/page.tsx`'s own `existingPages` established — with each sibling's locale/state and
a link into its own editor page. "Create translation" reuses the existing `new-document-form.tsx`/
`new/page.tsx` flow entirely (no bespoke form): the link passes `?translationGroupId=&kind=` as
query params, and the page pre-fills a read-only `translationGroupId` field and replaces the kind
`Select` with a locked, read-only value — a translation must match its group's existing kind. The
locale field stays exactly as free-text as it always was — the owner's explicit call: congregations
should be able to author in any language, not just the platform's 4 chrome locales, even though
those 4 are all next-intl's UI chrome currently supports.

**Acceptance criteria — met.** A page published in Ukrainian and drafted (unpublished) in English
offers only Ukrainian in the picker and `hreflang`; publishing the English variant makes it appear
with no other change (proved directly against real Postgres —
`internal/content/content_integration_test.go`'s M14.14 section creates the same two-sibling,
one-draft-then-published shape and asserts on `GetPublicDocumentByPath`'s returned `translations`
before and after). The duplicate-locale and cross-site `CreateDocument` guards are covered the same
way. `hreflang` tags are present and correct (verified via `generateMetadata`'s
`alternates.languages` against a real two-locale page on the running docker-compose stack).
`DS-OFM-7` is struck in `open-questions.md`. `Verified` awaits CI green on `main`.

### M14.15 · Scheduled publishing, no scheduler

**Built (2026-09-03).** Depends on M14.6. `migrations/0030_content_scheduled_publishing.sql` adds
`content_documents.publish_at timestamptz` and widens the `state` CHECK to add `SCHEDULED`. The
public read predicate becomes `state = 'PUBLISHED' OR (state = 'SCHEDULED' AND publish_at <= now())`
— applied consistently everywhere a `PUBLISHED` document is already treated as visible, per the
owner's scope call this session: the tenant page route (`GetPublicDocumentByPath`/`GetPublicBlocks`),
the `POST`/`EVENT` feed (`ListPublicDocuments`, SQL-level), the nav menu (`ListPublicNavItems`), and
the translation-sibling list (`resolvePublishedTranslations`) — not just the single-document route.
Sitemap exclusion stays deferred to M14.17 as originally scoped.

Correctness lives entirely in the `WHERE` clause / the new `domain.Document.EffectiveState`/
`IsPubliclyVisible` helpers (`internal/content/domain/content.go`) — nothing ever fires. New
`ActionSchedule` transition (`DRAFT`/`UNLISTED → SCHEDULED`, requires a strictly-future `publishAt`
in `TransitionDocumentRequest`, new typed `Content:ScheduleMissingPublishAt`/
`Content:SchedulePublishAtNotFuture` errors) snapshots the draft into a checkpoint revision at
schedule time — the same publish-promotion transaction M14.6 built
(`Repository.snapshotAndPromote`, shared by `PublishDocument` and the new `ScheduleDocument`) —
since nothing runs later to do it when `publishAt` actually arrives. **The one non-obvious design
point:** `TransitionDocument`'s legal-action lookup is keyed by a document's *effective* state, not
its raw column, so a `SCHEDULED` document past its `publishAt` can be `UNLIST`ed or
`REVERT_TO_DRAFT`ed exactly like a real `PUBLISHED` one — taking either action is what settles the
stale row, with no scheduler ever having touched it. `Document` gained both `publishAt` and a
computed `effectiveState` field over the wire; every admin UI surface renders the latter, never the
raw `state`, per the acceptance criterion. Admin editor gets a new `ScheduleForm` (`useActionState`,
inline validation) alongside the existing Publish/Unlist/Back-to-draft buttons — relabeled "Publish
now"/"Cancel schedule" when a document is genuinely still pending — and the document list/editor
badges now share one `documentStateLabel`/`DOCUMENT_STATE_TONE` helper instead of each page's own ad
hoc ternary.

**Acceptance criteria — met.** `internal/content/content_integration_test.go`'s new M14.15 section
covers the full lifecycle against real Postgres: both new typed errors, every public read path
staying blind to a not-yet-due schedule, all of them picking it up once `publish_at` is moved into
the past **by a direct SQL write, never through the app** (the "no job has run" proof), the admin
list showing raw `SCHEDULED` with `effectiveState() == PUBLISHED`, `UNLIST` settling a due row, and
`REVERT_TO_DRAFT` cancelling a pending one. `./godelw verify` (fmt/lint/`go test ./...`) and `make
sdk`/`sdk-verify` clean; both Next.js apps' `tsc`/`eslint`/`next build` clean, `web/apps/admin`'s
Vitest suite (43 tests) clean. **Live-verified against the running docker-compose stack** (a
dev-minted congregation-admin token + session, `docker exec`/`curlimages/curl` for non-GET calls per
this arc's own convention): a real document scheduled for `2026-12-31` over HTTP 404s on
`grace.localhost:3002`; `publish_at` moved into the past via a direct `psql` `UPDATE` (no restart, no
deploy, no app call in between) makes the exact same URL return `200` with the scheduled paragraph
rendered; the admin `listDocuments` response shows `state: SCHEDULED, effectiveState: PUBLISHED` at
that point; a subsequent `UNLIST` call over HTTP settles the row to a real `state: UNLISTED` with
`publishAt` cleared. Test document/session cleaned up afterward.

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
