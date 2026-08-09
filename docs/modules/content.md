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

Conventions per [conventions.md](../architecture/conventions.md): RID PKs, `TIMESTAMPTZ`,
`set_updated_at`, soft-delete, `TEXT` + `CHECK` for fixed states.

**`content_sites`**
- `id` PK (RID)
- `congregation_unit_rid TEXT NOT NULL` — the go-oikumenea `Unit` RID this site belongs to
  (opaque foreign value, no local FK — see conventions.md)
- `slug TEXT NOT NULL UNIQUE` — the site's public path/subdomain segment
- `theme JSONB NOT NULL DEFAULT '{}'` — accent color, font pairing, header layout; data, never a
  per-tenant code fork
- `created_at`, `updated_at`, `deleted_at`

**`content_block_types`** (catalog)
- `id` PK · `code TEXT UNIQUE` · `name TEXT` (translatable, OpenFaithMap's own admin-UI label
  store) · `json_schema JSONB NOT NULL` · `status TEXT CHECK (status IN ('active','retired'))` ·
  `sort_order` · timestamps + soft-delete

**`content_documents`** (a Page, Post, or Event — one row per locale variant)
- `id` PK (RID)
- `site_id` FK → `content_sites`
- `kind TEXT NOT NULL CHECK (kind IN ('page','post','event'))`
- `translation_group_id UUID NOT NULL` — shared across locale variants of the same document
- `locale TEXT NOT NULL` — this variant's locale
- `parent_document_id` FK → `content_documents` (nullable; pages only, ≤3 levels deep, enforced in
  the application)
- `slug TEXT NOT NULL` — unique within `(site_id, kind, locale)`
- `state TEXT NOT NULL DEFAULT 'draft' CHECK (state IN ('draft','published','unlisted'))`
- `published_at TIMESTAMPTZ` — set on first transition to `published`
- `event_starts_at`, `event_ends_at`, `event_recurrence_rrule TEXT` — populated only when
  `kind = 'event'`
- `created_at`, `updated_at`, `deleted_at`

**`content_blocks`**
- `id` PK (RID)
- `document_id` FK → `content_documents` **ON DELETE CASCADE**
- `block_type_id` FK → `content_block_types`
- `position INT NOT NULL` — ordering within the document, unique per `document_id`
- `data JSONB NOT NULL` — validated against `content_block_types.json_schema` at write time
- `created_at`, `updated_at`, `deleted_at`

## Conjure API surface

`ContentService` (`/content/v1`), `api/content.conjure.yml`:

| Op | Intent | Perm |
|---|---|---|
| `GET /sites/{congregationUnitRid}` | Read a congregation's site (public, published documents only) | none (public) |
| `POST /sites` | Create a site for a congregation the caller administers | `content.manage` (on the unit) |
| `PUT /sites/{id}/theme` | Update theme choices | `content.manage` (on the unit) |
| `GET /sites/{id}/documents?kind=&locale=&state=` | List a site's documents (admin: all states; public: published/unlisted only) | `content.manage` for draft state / none for public state |
| `POST /sites/{id}/documents` | Create a document (starts a new translation group, or joins one via `translationGroupId`) | `content.manage` (on the unit) |
| `PUT /documents/{id}` | Update slug/parent/event fields | `content.manage` (on the unit) |
| `POST /documents/{id}/transition` | `draft → published → unlisted` and back | `content.manage` (on the unit) |
| `GET·PUT /documents/{id}/blocks` | Read / replace the ordered block list | `content.manage` (on the unit) for write / gated by document state for read |
| `GET·PUT /block-types` | Read / manage the block-type catalog | none (public read) / `content.catalog.manage` (platform) |

Public reads never expose `draft` documents or blocks belonging to them. `content.manage` is
**not a go-oikumenea permission code** — it is OpenFaithMap's name for a **target-scoped capability
check** against go-oikumenea's PDP: "does this caller hold write authority over *this specific*
congregation unit?" See [D-PlatformModerator](../architecture/decisions.md) for the pattern, which
every OpenFaithMap-owned module follows.

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
[D-PlatformModerator](../architecture/decisions.md), not a local roster).

Neither name is ever consulted by go-oikumenea's PDP — it has no knowledge of the `content` schema.
What the PDP answers is the underlying capability question; these names are how this module refers
to that answer. That distinction matters for the same reason it does in
[registration.md](registration.md)'s known defects: when the rows being protected are
OpenFaithMap's own, the local check is the *entire* access-control decision, with no PDP behind it
to catch a wrong answer. Get the target and the verb right.

## Invariants

- **A document's blocks are always schema-valid.** `content_blocks.data` is validated against its
  `block_type`'s `json_schema` at write time — never left to render-time discovery of a malformed
  block.
- **Draft is never public.** The public read path filters on `state IN ('published','unlisted')`
  at the query level, not as an application-layer afterthought.
- **A translation group's documents share nothing but the group id.** Each locale variant is
  independently editable, independently publishable — a Ukrainian page can be published while its
  English translation is still a draft.
- **Page nesting is capped at 3 levels**, enforced in the application on `parent_document_id`
  assignment (not a DB constraint — a shallow product rule, not a schema invariant).
- **Congregation-unit authority is always re-checked live**, never cached across requests — see
  [core-integration.md](core-integration.md#invariants).

## Open seams

- **Post and Event are designed but not MVP.** M3 ships `page` only; `post`/`event` land at **M4**,
  once the public site has something to link them from. *(Audit 2026-08-09: this doc and
  milestones.md's M3 previously disagreed — M3's entry said "this milestone's own later iteration,"
  this doc said the discovery milestone. M4 is the answer, and both now say so.)*
- **`content_sites.slug` collisions have no design.** The column is globally `UNIQUE`, but under
  [D-FlatRoot](../architecture/decisions.md) congregations self-register with no namespace between
  them, so two "St. Mary's" collide on first publish. `registration`'s `slugCode` solves the
  equivalent problem for go-oikumenea unit codes by appending a random suffix; nothing here does.
  Decide when M3's migration is written — random suffix, per-country namespacing, or admin-chosen
  with a uniqueness probe and a real error message. The last is the only one that produces a URL a
  congregation would want to print on a sign.
- **Full-text content search** (searching page/post bodies, not just location) has no owner yet —
  a candidate for a dedicated search index once content volume justifies one; not needed at MVP
  scale.
- **Media beyond images** (audio/video hosting, livestreaming) is a deliberate non-goal, inherited
  from the original FaithMap scope — see [D-Scope](../architecture/decisions.md). The
  `youtube_embed`/`social_embed` block types cover the embed case.
