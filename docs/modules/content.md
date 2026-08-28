# Module: content

> Reads: [glossary](../glossary.md) · [architecture/conventions](../architecture/conventions.md) ·
> [core-integration](core-integration.md)
> Table prefix: `openfaithmap.content_*`

## Purpose

Owns a congregation's website: pages, posts, and events, each an ordered list of typed **blocks**,
each translatable into any of the platform's supported locales as an independent document. This is
the one domain in OpenFaithMap with **no go-oikumenea equivalent** (D-ContentModel) — go-oikumenea's
`religion` module explicitly scopes CMS content out of the core.

## Entities & aggregates

- **Site** — the root aggregate for one congregation's web presence. One per congregation
  `Unit` (referenced by go-oikumenea RID — see [conventions.md](../architecture/conventions.md)).
  Holds theme choices and the site's slug.
- **Page** — a long-lived document (home/about/beliefs/contact/staff). Supports parent/child
  nesting up to 3 levels for simple site navigation.
- **Post** — a time-stamped, reverse-chronological feed item.
- **Event** — a scheduled one-off item (start/end/optional recurrence). Distinct from a
  congregation's recurring **service** times, which live in go-oikumenea's
  `religion_service_schedules` and are read live, never copied here (see
  [discovery.md](discovery.md)).
- **Block** — a typed, schema-validated content unit. Pages/posts/events are each an ordered list
  of blocks, never an HTML blob.
- **Block type** — a catalog row (code, JSON Schema, display metadata) naming a valid block shape.
  MVP seed: `heading`, `paragraph`, `image`, `gallery`, `youtube_embed`, `social_embed`, `button`,
  `contact_info`, `map_embed`, `divider`, `staff_card`, `quote`, `columns`, plus `list` (M14.2,
  added by `migrations/0022_content_richtext.sql`).
- **Media normalizer** — a write-time rewrite (M14.3, `internal/content/application/medianormalize.go`,
  called from `Service.PutBlocks` before validation) of `image.url`/`gallery.images[].url` for
  known share-link hosts — a Google Drive or Dropbox share link, or the long-form OneDrive URL —
  into its direct-content form. The pre-rewrite value is preserved in a new, optional
  `originalUrl` field alongside the normalized `url`
  ([D-ExternalMediaOnly](../architecture/decisions.md#d-externalmediaonly--congregations-host-their-own-media-no-first-party-uploads),
  `DS-OFM-17`), so a future normalizer fix is a re-derivation, not a data-loss event. A host it
  doesn't recognize (including OneDrive's short `1drv.ms` links, which only resolve via a
  redirect this arc deliberately never fetches server-side) passes through unchanged. `alt` is now
  schema-required on both block types.
- **Rich text** — the shared `richText` node array (M14.2,
  [D-RichTextNodes](../architecture/decisions.md#d-richtextnodes--structured-inline-nodes-never-html-strings)),
  adopted by `paragraph.text`, `heading.text`, `quote.text`, `staff_card.bio` and `list.content`. An
  ordered array whose elements are either a **text node** (`{"type":"text","text":"…","marks":[…]}`,
  each mark one of `bold`/`italic`/`link` — a `link` mark carries `href`, checked against the exact
  same URL-scheme allowlist as every other URL field) or a **list node**
  (`{"type":"list","style":"bullet"|"ordered","items":[…]}`, each item
  `{"type":"listItem","content": richText}` — recursive, so nested lists work structurally for
  free). Never an HTML string, at any point in the pipeline.
- **Translation group** — a UUID shared across every locale variant of one conceptual page/post/
  event. Each variant is its own row with its own `locale`, edited independently.
- **Document revision** — a snapshot of one document's blocks (M14.6,
  [D-ContentRevisions](../architecture/decisions.md#d-contentrevisions--forward-revisions-with-a-draft-and-a-published-pointer)).
  A document has a **published** revision (what `ContentPublicService` reads) and a **draft**
  revision (what the editor mutates) — editing never touches what's live until an explicit publish.
- **Pattern** — a pre-built starting layout (M14.13,
  [D-SitePatterns](../architecture/decisions.md#d-sitepatterns--unsynced-starter-patterns-and-a-real-block-type-catalog)).
  Inserting one copies its blocks into the document and detaches immediately — unsynced, no
  ongoing link back to the source pattern.
- **Nav item** — an entry in a site's hand-built navigation menu (M14.10), independent of the
  `parent_document_id` page tree: a label, a target (an internal document or an external URL), and
  a sort order.
- **Form submission** — an anonymous contact-form entry (M14.16,
  [D-InAppInbox](../architecture/decisions.md#d-inappinbox--contact-submissions-stay-in-app-no-email-sent)),
  read through `openfaithmap-admin`'s Messages screen. Untrusted, plain-text only, never rendered
  as content.

## Data model

Conventions per [conventions.md](../architecture/conventions.md): plain `uuid` PKs (U6, resolved at
M3 — see conventions.md's corrected RID entry: go-oikumenea's actually-deployed schema uses bare
`uuid`, not the composed-URN scheme this doc used to assume), `TIMESTAMPTZ`, `set_updated_at`,
soft-delete, `TEXT` + `CHECK` for fixed states. Enum values (`kind`/`state`/`status`) are uppercase,
matching `api/content.conjure.yml`'s enum values and `registration_requests.status`'s existing
convention exactly — no case conversion anywhere across the transport/domain/adapters boundary.

**`content_sites`**
- `id` PK (`uuid`)
- `congregation_unit_rid TEXT NOT NULL` — the go-oikumenea `Unit` RID this site belongs to
  (opaque foreign value, no local FK — see conventions.md)
- `slug TEXT NOT NULL UNIQUE` — **as of M14.9, a hostname component** (`<slug>.<apex>`), not a
  path segment — validated server-side against a reserved-subdomain blocklist (`admin`, `api`,
  `auth`, `login`, `www`, `app`, `mail`, `static`, `support`, `billing`, `help`, `status`, …) per
  [D-TenantSubdomains](../architecture/decisions.md#d-tenantsubdomains--subdomain-per-congregation-wildcard-tls-and-a-reserved-slug-blocklist)
- `theme JSONB NOT NULL DEFAULT '{}'` — accent color (vetted palette), font pairing, spacing
  scale, header layout, light/dark; a fixed, contrast-checked-at-write-time vocabulary as of
  M14.12 ([D-CuratedTheme](../architecture/decisions.md#d-curatedtheme--a-fixed-token-vocabulary-contrast-checked-at-write-time)),
  emitted as CSS custom properties — data, never a per-tenant code fork
- `created_at`, `updated_at`, `deleted_at`

**`content_block_types`** (catalog)
- `id` PK · `code TEXT UNIQUE` · `name TEXT` (translatable, OpenFaithMap's own admin-UI label
  store) · `json_schema JSONB NOT NULL` · `ui_schema JSONB NOT NULL DEFAULT '{}'` (M14.4, built
  2026-08-27, `migrations/0024_content_ui_schema.sql` — widget hints, labels, help text, field
  order, sitting beside `json_schema`; a block type's data shape and its editing affordances are
  declared together, so the admin form is derived, never hand-written per type; `NOT NULL DEFAULT
  '{}'` — matching `content_sites.theme`'s own precedent — so a block type inserted without one
  still renders a degenerate but non-crashing form) · `status TEXT CHECK (status IN
  ('ACTIVE','RETIRED'))` · `sort_order` · timestamps + soft-delete

**`content_document_revisions`** (M14.6, new)
- `id` PK (`uuid`) · `document_id` FK → `content_documents` · `revision_no INT NOT NULL` · a
  blocks snapshot (`data JSONB NOT NULL`) · `author_person_rid TEXT` · `created_at` ·
  `label TEXT` (optional)
- `content_documents` gains `published_revision_id`/`draft_revision_id` FKs into this table.
  Editing mutates the draft revision only; `ContentPublicService` always reads the published one.

**`content_patterns`** (M14.13, new)
- `id` PK (`uuid`) · `name TEXT` · `description TEXT` · `blocks JSONB NOT NULL` (a blocks snapshot,
  same shape as a document's block list) · `sort_order` · timestamps + soft-delete
- Insert-time behavior is unsynced (D-SitePatterns): the blocks snapshot is copied into the target
  document's draft revision and the pattern row is never referenced again by that document.

**`content_site_nav_items`** (M14.10, new)
- `id` PK (`uuid`) · `site_id` FK → `content_sites` · `label TEXT NOT NULL` ·
  `target_document_id` FK → `content_documents` (nullable) · `target_url TEXT` (nullable — set
  when the item points off-site instead) · `sort_order INT NOT NULL`
- Independent of `content_documents.parent_document_id` — M14.0 replaces the original
  page-tree-derived-nav assumption with a hand-built menu (see the M14.10 note in
  [milestones-2026-08-26-now.md](../milestones-2026-08-26-now.md)); `parent_document_id` still
  governs page nesting/breadcrumbs, just not the nav menu itself.

**`content_form_submissions`** (M14.16, new, [D-InAppInbox](../architecture/decisions.md#d-inappinbox--contact-submissions-stay-in-app-no-email-sent))
- `id` PK (`uuid`) · `site_id` FK → `content_sites` · `name TEXT` · `email TEXT` ·
  `message TEXT NOT NULL` (plain text only, never rendered as a block) · `created_at`
- Written via a genuinely anonymous `ContentPublicService` endpoint, rate-limited by
  `internal/platform/ratelimit` (M7).

**`content_documents`** (a Page, Post, or Event — one row per locale variant)
- `id` PK (`uuid`)
- `site_id` FK → `content_sites`
- `kind TEXT NOT NULL CHECK (kind IN ('PAGE','POST','EVENT'))`
- `translation_group_id UUID NOT NULL` — shared across locale variants of the same document
- `locale TEXT NOT NULL` — this variant's locale
- `parent_document_id` FK → `content_documents` (nullable; pages only, ≤3 levels deep, enforced in
  the application)
- `slug TEXT NOT NULL` — unique within `(site_id, kind, locale)`
- `state TEXT NOT NULL DEFAULT 'DRAFT' CHECK (state IN ('DRAFT','PUBLISHED','UNLISTED'))`
- `published_at TIMESTAMPTZ` — set on first transition to `PUBLISHED`
- `event_starts_at`, `event_ends_at`, `event_recurrence_rrule TEXT` — populated only when
  `kind = 'EVENT'` (schema-ready; M3's application layer rejects any non-`PAGE` kind outright —
  see open seams)
- `created_at`, `updated_at`, `deleted_at`

**`content_blocks`**
- `id` PK (`uuid`)
- `document_id` FK → `content_documents` **ON DELETE CASCADE**
- `block_type_id` FK → `content_block_types`
- `position INT NOT NULL` — ordering within the document, unique per `document_id`
- `data JSONB NOT NULL` — validated against `content_block_types.json_schema` at write time
- `created_at`, `updated_at`, `deleted_at`

**Design call — `columns` nests without a `parent_block_id`.** `columns` is the one MVP block type
with real internal structure (a row of sub-columns, each its own list of blocks), but
`content_blocks` has no self-referential parent column. Its children live as inline JSON *inside*
the `columns` block's own `data`, validated as part of that one block's `json_schema` — never as
additional `content_blocks` rows. Simpler than a real tree, and sufficient for one level of nesting;
revisit if a second nesting block type is ever added.

## Conjure API surface

**Two services, not one — a real deviation from this table's original one-row-per-endpoint shape,
found at M3 by actually trying to build it.** Conjure's auth is a fixed per-*service* setting
(`default-auth: header` or nothing); there is no per-endpoint "authenticated if a token happens to
be present" mode (confirmed against the Conjure compiler's own `AuthType` — only `Header`/`Cookie`
variants exist, no `None`). `openfaithmap-web` never holds a session at all (D-AdminSurface), so the
"content.manage for draft / none for public" split this table used to put on single endpoints is
split into two services instead, sharing one `types:` block in `api/content.conjure.yml`:

**`ContentService`** (`/content/v1`, `default-auth: header` — `openfaithmap-admin` only):

| Op | Intent | Perm |
|---|---|---|
| `POST /sites` | Create a site for a congregation the caller administers | `content.manage` (on the unit) |
| `PUT /sites/{siteId}/theme` | Update theme choices | `content.manage` (on the unit) |
| `GET /sites/{siteId}/documents?kind=&locale=&state=` | List a site's documents, every state | `content.manage` (on the unit) |
| `POST /sites/{siteId}/documents` | Create a document (starts a new translation group, or joins one via `translationGroupId`) | `content.manage` (on the unit) |
| `PUT /documents/{documentId}` | Update slug/parent | `content.manage` (on the unit) |
| `POST /documents/{documentId}/transition` | `DRAFT → PUBLISHED → UNLISTED` and back | `content.manage` (on the unit) |
| `GET·PUT /documents/{documentId}/blocks` | Read / replace the ordered block list, any document state | `content.manage` (on the unit) |

**`ContentPublicService`** (`/content/v1/public`, no auth at all — `openfaithmap-web`'s only path in):

| Op | Intent |
|---|---|
| `GET /units/{congregationUnitId}/site` | Read a congregation's site by unit. **Not** `/sites/{congregationUnitId}` — that path shares `ContentService`'s `/sites/{siteId}/...` wildcard tree position under a *differently-named* parameter, which witchcraft's underlying `httprouter` rejects at server startup (a hard panic, not a soft warning — found by actually booting the server, not by review). |
| `GET /sites/{siteId}/documents?kind=&locale=` | List a site's `PUBLISHED`/`UNLISTED` documents only |
| `GET /documents/{documentId}/blocks` | Read a document's blocks — `Content:DocumentNotFound` if it's `DRAFT`, never distinguishing "draft" from "doesn't exist" |
| `GET /block-types` | Active block types only |

`content.catalog.manage` (platform-wide block-type catalog writes) has **no endpoint as of M13** —
the catalog is migration-seeded only (13 MVP types). **Scheduled to M14.13**
([D-SitePatterns](../architecture/decisions.md#d-sitepatterns--unsynced-starter-patterns-and-a-real-block-type-catalog)):
block-type and pattern CRUD, gated on platform-moderator authority.

Public reads never expose `draft` documents or blocks belonging to them. `content.manage` is
**not a go-oikumenea permission code** — it is OpenFaithMap's name for a **target-scoped capability
check** against go-oikumenea's PDP: "does this caller hold write authority over *this specific*
congregation unit?" See [D-PlatformModerator](../architecture/decisions.md) for the pattern, which
every OpenFaithMap-owned module follows. **Resolved at M3:** the underlying permission is
go-oikumenea's existing `religionorg.manage` — already granted to `congregation-admin` on their own
unit, the same permission `registration`'s own operator gate reuses (at a different scope). No new
go-oikumenea permission was requested; see the authorization-touchpoints section below for the
consequence of that reuse.

> **Corrected at the 2026-08-09 audit.** This paragraph previously defined `content.manage` as
> "call `GET /units/{unitId}` with the caller's forwarded token and treat a successful,
> authoritative read on that unit as proof of standing." That is wrong in a way worth spelling out,
> because it is an easy mistake to re-make: **read authority is not write authority.** Under
> go-oikumenea's model a person can hold `unit.read` over a unit they have no business editing —
> `congregation-admin` itself holds `unit.read`, and a future jurisdiction-level or moderator role
> would hold it over many units. Treating a successful read as proof of standing would let anyone
> who can *see* a congregation rewrite its website. The check must name the write authority it
> actually requires, against the unit it actually concerns.

## Dependencies

- **Calls:** [core-integration](core-integration.md) for the congregation-unit authority check on
  every write; go-oikumenea's `localization` module is **not** used — content translation is a
  separate-document model, not a label map (see [conventions.md](../architecture/conventions.md)).
- **Called by:** [discovery.md](discovery.md) (a site's published pages/posts feed the public
  search index for full-text content matches — content-only, never location); the
  [web-facade](web-facade.md) (renders the published public site) and
  [web-admin](web-admin.md) (the site editor, authenticated).

## Authorization touchpoints

`content.manage` (per congregation unit — a target-scoped capability check against go-oikumenea's
PDP for write authority on *that* unit, see above) and `content.catalog.manage` (platform-wide,
block-type catalog edits — restricted to platform moderators, whose authority is a
`platform-moderator` Role on the shared root unit per
[D-PlatformModerator](../architecture/decisions.md), not a local roster; unused in M3, no endpoint
exists yet to gate).

Neither name is ever consulted by go-oikumenea's PDP — it has no knowledge of the `content` schema.
What the PDP answers is the underlying capability question; these names are how this module refers
to that answer. That distinction matters for the same reason it does in
[registration.md](registration.md)'s known defects: when the rows being protected are
OpenFaithMap's own, the local check is the *entire* access-control decision, with no PDP behind it
to catch a wrong answer. Get the target and the verb right.

**`content.manage` = `religionorg.manage` through M14.8 — tightened at M14.9.** Through M14.8, the
target-scoped check (`internal/content/application/authorize.go`'s `requireManage`) called
go-oikumenea's `Authorize` with `Action: "religionorg.manage"` and `UnitId` set to the *specific
site's* congregation unit — same call shape as `registration`'s `IsOperator`, live-verified: a
`congregation-admin`-held grant on its own unit passes, and (as a byproduct of reusing the exact
permission `registration-operator`'s subtree grant already carries on the shared root) a
registration operator also passed `content.manage` for *any* unit within that subtree — every
congregation, not just ones they submitted or approved. Accepted through M14.8 (operators are a
small, trusted set — D-PlatformModerator — and a site was still "an unlinked blob of blocks").

**Ruled on at M14.0 (U16), built at M14.9 (2026-08-28): tightened, not restated, because a site is
now a real public website on its own subdomain.**
[D-TenantSubdomains](../architecture/decisions.md#d-tenantsubdomains--subdomain-per-congregation-wildcard-tls-and-a-reserved-slug-blocklist)'s
U16 ruling is now live: `content.manage` (`internal/authz/domain.PermContentManage`) is its own
permission, checked per-unit and granted explicitly to `congregation-admin` on their own unit
(`migrations/0026_content_manage_permission.sql`) — the same shape M13.2 already used for
`site.manage` (`PermSiteManage`) rather than reusing a broader existing grant.
`internal/content/application/authorize.go`'s `managePermission` now points at `PermContentManage`,
not `PermReligionOrgManage`. A registration operator's platform-wide subtree authority no longer
implies content-write authority — confirmed with the owner, deliberately left with **no** replacement
edit path for now: operators simply lose `content.manage`, and granting them a moderation permission
is a separate future decision, not part of this milestone. `CreateSite` over a real HTTP call against
a running docker-compose stack, by a real `congregation-admin` session, was live-verified to succeed
(and to reject a reserved slug — see the open-seams entry below); the cross-tenant-denial and
registration-operator-denial cases are covered by
`internal/content/content_integration_test.go`'s M14.9 section, run against real Postgres.

**Also live-verified, and worth recording as a go-oikumenea characteristic rather than an
OpenFaithMap bug:** a `subtree`-scoped `Authorize` grant on a real unit returns `{"allow":true}` for
a syntactically valid but **nonexistent** target unit id (confirmed directly against
`POST /authorization/v1/authorize`, bypassing this module's code entirely) — subtree containment
appears to fail *open* on an unresolvable unit rather than fail closed. Not exploitable through this
module today (every unit id this module ever passes to `Authorize` comes from a real, already-loaded
`content_sites`/`content_documents` row — never a client-supplied id used unchecked), but worth
flagging for whoever next builds a target-scoped check against an id that hasn't been verified to
exist first.

## Invariants

- **A document's blocks are always schema-valid.** `content_blocks.data` is validated against its
  `block_type`'s `json_schema` at write time — never left to render-time discovery of a malformed
  block.
- **Draft is never public.** The public read path filters on `state IN ('PUBLISHED','UNLISTED')`
  at the query level, not as an application-layer afterthought.
- **A translation group's documents share nothing but the group id.** Each locale variant is
  independently editable, independently publishable — a Ukrainian page can be published while its
  English translation is still a draft.
- **Page nesting is capped at 3 levels**, enforced in the application on `parent_document_id`
  assignment (not a DB constraint — a shallow product rule, not a schema invariant).
- **Congregation-unit authority is always re-checked live**, never cached across requests — see
  [core-integration.md](core-integration.md#invariants).
- **Inline text is always a structured node tree, never an HTML string** (M14.2,
  [D-RichTextNodes](../architecture/decisions.md#d-richtextnodes--structured-inline-nodes-never-html-strings)).
  There is no HTML parser and no sanitizer anywhere in this module's pipeline, by construction.
- **Every URL-bearing block field is scheme-allowlisted at write and re-validated at render** (M14.1,
  [D-PublicSiteCSP](../architecture/decisions.md#d-publicsitecsp--url-scheme-allowlist-embed-allowlist-and-security-headers)).
  `dangerouslySetInnerHTML` never appears in either Next.js app.
- **Publishing a document never mutates what's already live** (M14.6,
  [D-ContentRevisions](../architecture/decisions.md#d-contentrevisions--forward-revisions-with-a-draft-and-a-published-pointer)).
  Edits land on the draft revision; `ContentPublicService` always reads the published one.

## Open seams

- **Post and Event — enabled at M4 (2026-08-10).** `createDocument` accepts all three kinds now;
  the schema (`event_starts_at`/`event_ends_at`/`event_recurrence_rrule`) was already in place from
  M3, so this was a small change: `Content:KindNotSupported` is gone (genuinely unreachable once
  every `DocumentKind` value is accepted), replaced by `Content:EventMissingStart` — `EVENT`
  requires `eventStartsAt`; `PAGE`/`POST` are unaffected. Public listing
  (`listPublicDocuments`) orders `EVENT` by `event_starts_at ASC` (soonest-upcoming first) and
  everything else by `created_at DESC` (reverse-chronological feed), same query, no new endpoint.
- **`content_sites.slug` collisions — resolved at M3.** Admin-chosen slug, probed for uniqueness at
  write time (`INSERT ... ON CONFLICT`-shaped: insert first, catch the unique-violation, translate
  to a typed `Content:SlugTaken` error — race-safe, never check-then-insert). No silent suffixing,
  no per-country namespacing. The same pattern covers `content_documents.slug` collisions within one
  `(site_id, kind, locale)`.
- **`content.catalog.manage` has no endpoint yet — resolved by schedule (2026-08-27, M14.0):
  M14.13 builds it.** The block-type catalog is migration-seeded only (13 MVP types,
  `migrations/0004_content.sql`) until then; see
  [D-SitePatterns](../architecture/decisions.md#d-sitepatterns--unsynced-starter-patterns-and-a-real-block-type-catalog).
- **A cross-tenant `content.manage` denial was untested — resolved at M14.9 (2026-08-28).** The
  `religionorg.manage` reuse this row used to describe is gone (see the authorization-touchpoints
  section); `internal/content/content_integration_test.go`'s M14.9 section now proves both that a
  `congregation-admin` granted on one unit is denied on an unrelated one, and that a
  `registration-operator` granted the same unit-scoped shape is denied too — `content.manage` itself
  gates the write, not incidental scope.
- **A reserved-subdomain slug is rejected server-side — added at M14.9 (2026-08-28).**
  `content_sites.slug` is a hostname component (`<slug>.<apex>`) as of this milestone;
  `internal/content/application/slugvalidation.go`'s blocklist is checked in `CreateSite` before the
  reserved-slug scenario ever hits M3's own `Content:SlugTaken` uniqueness check, and returns the
  new typed `Content:SlugReserved` error. Live-verified over a real HTTP call against a running
  docker-compose stack (bypassing the admin form entirely, not just declining to submit it).
- **go-oikumenea's `Authorize` appears to fail open on a nonexistent target unit under a subtree
  grant** — live-verified (see authorization-touchpoints), not exploitable through this module
  today, but worth a note to whoever next builds a target-scoped check against an unverified id.
- **A live stored-XSS hole existed in `main` — resolved (2026-08-27, M14.1), first in the M14 arc.**
  `web/apps/web/app/blocks.tsx` rendered `button.href`/`image.url`/`social_embed.url`/
  `staff_card.photoUrl` with no scheme validation at any layer. Closed by a write-time scheme/
  embed-host allowlist in `blockvalidation.go` (typed `Content:BlockUrlNotAllowed` error) plus
  render-time re-validation in `blocks.tsx` — belt-and-braces, since rows written before this
  landed were already unvalidated in the DB. See
  [D-PublicSiteCSP](../architecture/decisions.md#d-publicsitecsp--url-scheme-allowlist-embed-allowlist-and-security-headers).
- **Full-text content search** (searching page/post bodies, not just location) has no owner yet —
  a candidate for a dedicated search index once content volume justifies one; not needed at MVP
  scale. Tracked as `DS-OFM-5`.
- **Media beyond images** (audio/video hosting, livestreaming) is a deliberate non-goal, inherited
  from the original FaithMap scope — see [D-Scope](../architecture/decisions.md). The
  `youtube_embed`/`social_embed` block types cover the embed case. **No first-party image storage
  either, as of M14 —** congregations host images externally
  ([D-ExternalMediaOnly](../architecture/decisions.md#d-externalmediaonly--congregations-host-their-own-media-no-first-party-uploads)).
  A future `media` module is a designed-but-unbuilt seam, tracked as `DS-OFM-17`. **Mitigated,
  not eliminated, at M14.3 (2026-08-27):** a write-time normalizer rewrites known Google
  Drive/Dropbox/OneDrive(long-form) share links to their direct-content form, preserving the
  original URL. **Known limitation:** OneDrive's short `1drv.ms` links are not normalized — doing
  so would require following a redirect server-side, i.e. fetching an admin-supplied URL, which
  this arc's own SSRF discipline forbids; they pass through unchanged. The editor-side "did this
  URL load" probe named in the milestone was not built at M14.3 (the admin editor was still the raw
  JSON-textarea UI, with no per-field surface to attach a probe to) **and stays unbuilt at M14.4
  too, decided with the owner rather than assumed** — M14.4 replaces the JSON-textarea UI the probe
  would attach to, but building the probe itself would mean an admin-configured `fetch`/`<img>`
  load of an external URL from the admin origin, which needs its own CSP/SSRF review; unscheduled.
- **Schema-driven block forms, built 2026-08-27 (M14.4).** `content_block_types.ui_schema` (widget
  hints, labels, help text, field order) plus a generic, recursive form renderer in
  `web/apps/admin` (`block-data-form.tsx`) replaced the raw-JSON `<Textarea>` editor for block
  `data` — the block list's own `position`/`blockTypeCode`/add-remove controls are unchanged,
  reserved for M14.5. Two named, deliberately-accepted gaps: **richText fields** (`heading.text`,
  `paragraph.text`, `quote.text`, `staff_card.bio`, `list.content`) stay a schema-aware JSON
  textarea — no rich-text/WYSIWYG editor exists anywhere in this codebase yet, and building one is
  out of scope for this milestone; and a **`columns` block's schema-shape validation failure**
  highlights the whole `columns` field group rather than a specific nested block/field, since
  `blockvalidation.go`'s structural pass never descends into nested block data (a pre-existing gap,
  not new — see the `columns` design-call note above). `Content:BlockDataInvalid` gained a `field`
  safe-arg (mirroring `Content:BlockUrlNotAllowed`'s existing one), populated by
  `topLevelFieldFromValidationError` filtering a jsonschema/v6 validation error's instance-location
  path through the block type's own declared top-level `properties` keys — never a raw,
  potentially-attacker-chosen path segment.
