# Module: web-facade

> Reads: [glossary](../glossary.md) · [architecture/overview](../architecture/overview.md) ·
> [core-integration](core-integration.md) · [web-admin](web-admin.md)
> Owns no schema — a consumer, not a backend module (mirrors how go-oikumenea documents its own
> `web-ui`).

## Purpose

`openfaithmap-web` — the **anonymous public surface** (D-AdminSurface): a Next.js App Router
application serving the public site only — the map, search, and per-congregation pages. It holds no
session, no credential, and no identity of any kind: it never authenticates a visitor and never
forwards a bearer token, because it never has one. Anything that requires being logged in — the
congregation-admin console, the registration wizard, the moderator console — lives in the separate
[`openfaithmap-admin`](web-admin.md) app instead.

## Surfaces

- **Public site** — home, discovery map/search, per-congregation pages (rendering
  `content_documents` in `published`/`unlisted` state), and the registration *entry point* (a page
  explaining how to register and linking out to `openfaithmap-admin`, where the actual wizard runs
  since submitting requires being logged in). Fully unauthenticated; SEO-relevant, server-rendered.
  - **Discovery map/search (M4)** — the home page: a Leaflet/OpenStreetMap map plus filter controls
    (tradition/language/day-of-week), backed entirely by `discovery`'s `GET /discovery/v1/search`
    (see [discovery.md](discovery.md)'s redesign). Never calls go-oikumenea directly.
  - **Per-congregation page (M4)** — one route per published `content_sites` row, rendering its
    pages/posts/events via `content`'s public reads (`getSite`/`listPublicDocuments`/
    `getPublicBlocks`) — the same generated client, same no-token pattern as discovery.
- **Public report filing** — anyone can file a moderation report or run the exclusion pre-check
  without logging in (see [moderation.md](moderation.md)) — these stay here, not in
  `openfaithmap-admin`, because they don't require a session either.

## Session & identity

`openfaithmap-web` holds no session. No Auth.js, no OAuth client, no cookie beyond whatever
stateless preferences (e.g. a locale choice) it may set directly, unrelated to identity. This is
deliberate (D-AdminSurface): the one thing this app is not allowed to become is a second place that
can hold a credential. Anything that needs to know who's logged in belongs in
[`openfaithmap-admin`](web-admin.md).

## Dependencies

- **Calls:** `openfaithmap-api`'s own generated TypeScript client only — `discovery`'s
  `DiscoveryPublicService` (`search`) and `content`'s `ContentPublicService`
  (`getSite`/`listPublicDocuments`/`getPublicBlocks`). **Never go-oikumenea, directly or
  indirectly** — this app has no token to forward (D-AdminSurface) and every `religion` read
  genuinely 401s an anonymous caller (M2.5). Every call is unauthenticated on the wire — no bearer
  token is ever attached, since this app never holds one — via its own token-free
  `createOpenFaithMapClient` wiring (`web/apps/web/lib/openfaithmap/index.ts`, generated the same
  way as `openfaithmap-admin`'s but with no `auth()` call anywhere). Never a raw `fetch` against
  either service's REST surface — always the typed client.
  > **History.** M2.5 (2026-08-09/10) verified false the original assumption that this app could
  > call go-oikumenea's discovery endpoint directly, even unauthenticated: every `religion` read
  > returns `401 IdentityFederation:Unauthorized` to a caller with no token, deliberately and
  > permanently (go-oikumenea#33 scoped genuine anonymous access out of its fix). M2.6 had already
  > stood up the TypeScript codegen *pipeline* (for `registration`, `openfaithmap-admin`'s module,
  > never called from here) without yet generating this app's own client. **M4 (2026-08-10)**
  > resolved both: `discovery`'s redesign (cache-only, [discovery.md](discovery.md)) gives this app
  > something safe to call, and this is the first milestone where `openfaithmap-web` makes any
  > backend call at all.
- **Called by:** nothing — it is one of two public ingress points (the other is
  [`openfaithmap-admin`](web-admin.md#dependencies)). It publishes its own host port, independent of
  `openfaithmap-admin`'s.

## Patterns

- **No client-side authorization.** There is nothing to authorize — every page this app renders is
  public by construction. A page that would need a permission check does not belong in this app at
  all; it belongs in `openfaithmap-admin`.

## Invariants

- **Never holds a credential.** `openfaithmap-web` must never gain a session, a cookie-based login,
  or any mechanism that would let it forward a bearer token — that capability is
  `openfaithmap-admin`'s alone (D-AdminSurface). If a feature needs to know who the visitor is, it is
  not a `openfaithmap-web` feature.
- **The public site never renders draft content or coordinates finer than go-oikumenea already
  coarsened** — inherited directly from [content.md](content.md) and
  [discovery.md](discovery.md)'s own invariants; this module adds no new privacy surface, it only
  renders what those modules already decided was safe to return.

## Open seams

- **Locale switching UX** for the public site (which locales are offered, how a visitor's
  preference is detected) is not designed yet — content translation groups
  ([content.md](content.md)) support it structurally; the UI decision is open.
- **Mobile app / native client** is out of scope for the current milestone set entirely — the
  public site is responsive web only at MVP.
