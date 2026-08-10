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
  `contact_info`, `map_embed`, `divider`, `staff_card`, `quote`, `columns`.
- **Translation group** — a UUID shared across every locale variant of one conceptual page/post/
  event. Each variant is its own row with its own `locale`, edited independently.

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
- `slug TEXT NOT NULL UNIQUE` — the site's public path/subdomain segment
- `theme JSONB NOT NULL DEFAULT '{}'` — accent color, font pairing, header layout; data, never a
  per-tenant code fork
- `created_at`, `updated_at`, `deleted_at`

**`content_block_types`** (catalog)
- `id` PK · `code TEXT UNIQUE` · `name TEXT` (translatable, OpenFaithMap's own admin-UI label
  store) · `json_schema JSONB NOT NULL` · `status TEXT CHECK (status IN ('ACTIVE','RETIRED'))` ·
  `sort_order` · timestamps + soft-delete

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

`content.catalog.manage` (platform-wide block-type catalog writes) has **no endpoint in M3** — the
catalog is migration-seeded only (13 MVP types); a write endpoint is real M4+ scope, not attempted
here (see open seams).

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

**`content.manage` = `religionorg.manage`, live-verified — with one real consequence of reusing it.**
The target-scoped check (`internal/content/application/authorize.go`'s `requireManage`) calls
go-oikumenea's `Authorize` with `Action: "religionorg.manage"` and `UnitId` set to the *specific
site's* congregation unit — same call shape as `registration`'s `IsOperator`, proven live against a
real stack: a `congregation-admin`-held grant on its own unit passes, and (as a byproduct of reusing
the exact permission `registration-operator`'s subtree grant already carries on the shared root) a
registration operator also passes `content.manage` for *any* unit within that subtree — i.e., every
congregation, not just ones they submitted or approved. Not tested here (M2.3's own acceptance
criteria already name this as the one thing not achievable headlessly: a true cross-tenant denial
needs a second real identity, not just a second role on the same bootstrap-admin token), but it
follows directly from the reuse decision and is worth naming: an operator can currently edit any
congregation's website, not only manage its registration. Acceptable for M3 (operators are a small,
trusted set — D-PlatformModerator), but revisit if that set ever grows past "small and trusted."

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
- **`content.catalog.manage` has no endpoint yet.** The block-type catalog is migration-seeded only
  (13 MVP types, `migrations/0004_content.sql`); a real write endpoint (platform-moderator-gated)
  is M4+ scope.
- **A cross-tenant `content.manage` denial is untested, and one real consequence of the
  `religionorg.manage` reuse decision is unverified in practice** — see the authorization-touchpoints
  section: a registration operator's subtree grant currently also satisfies `content.manage` for
  every congregation, not just the ones tied to their own submissions. Needs a second real identity
  to test properly (same limitation M2.3 already documented); revisit if the operator roster ever
  grows.
- **go-oikumenea's `Authorize` appears to fail open on a nonexistent target unit under a subtree
  grant** — live-verified (see authorization-touchpoints), not exploitable through this module
  today, but worth a note to whoever next builds a target-scoped check against an unverified id.
- **Full-text content search** (searching page/post bodies, not just location) has no owner yet —
  a candidate for a dedicated search index once content volume justifies one; not needed at MVP
  scale.
- **Media beyond images** (audio/video hosting, livestreaming) is a deliberate non-goal, inherited
  from the original FaithMap scope — see [D-Scope](../architecture/decisions.md). The
  `youtube_embed`/`social_embed` block types cover the embed case.
