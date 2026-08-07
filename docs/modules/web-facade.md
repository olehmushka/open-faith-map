# Module: web-facade

> Reads: [glossary](../glossary.md) · [architecture/overview](../architecture/overview.md) ·
> [core-integration](core-integration.md)
> Owns no schema — a consumer, not a backend module (mirrors how go-oikumenea documents its own
> `web-ui`).

## Purpose

`openfaithmap-web` — the single public-facing surface (D-HeadlessTopology, reused): a Next.js App
Router application serving both the anonymous public site (map/search/congregation pages) and the
authenticated congregation-admin console, on one deployment. It owns the browser session; it
asserts no authority of its own (D-Facade). Two audiences, one app, because both share the same
identity provider and the same "forward the user's token" rule — splitting them into separate
deployments would duplicate session handling for no isolation benefit at this stage.

## Surfaces

- **Public site** — home, discovery map/search, per-congregation pages (rendering
  `content_documents` in `published`/`unlisted` state), the registration entry point. Mostly
  unauthenticated; SEO-relevant, server-rendered.
- **Congregation-admin console** — site editor (block-based page builder), congregation
  claim/registration wizard, staff/clergy roster (thin views over go-oikumenea's `person`/
  `membership`/`religion` data), vouching request UI, report/appeal filing.
- **Moderator console** — report queue, action history, appeal decisions, guarantor management.
  Small audience (platform moderators only); same app, gated by the `moderation.read`/
  `moderation.act` checks documented in [moderation.md](moderation.md).

## Session & identity

- **Login** — Auth.js v5, Google as the sole OIDC provider (D-CoreDependency's as-built note — no
  Keycloak, no shared realm). go-oikumenea's own console-bff supports registering multiple IdPs;
  OpenFaithMap uses only the one provider go-oikumenea's `deploy/oikumenea-install.yml` is
  configured to trust for human login.
- **Session storage** — httpOnly cookie, server-side only; the ID token is never exposed to
  client-side JavaScript.
- **Token forwarding** — every server-side call `openfaithmap-web` makes, to either go-oikumenea or
  `openfaithmap-api`, forwards the logged-in user's Google **ID token** unchanged — not the access
  token, which is an opaque string go-oikumenea cannot verify; the ID token is the JWT whose `aud`
  go-oikumenea's Google issuer entry pins (`deploy/oikumenea-install.yml`). `openfaithmap-web`
  holds no credential of its own that widens what a caller can do (D-HeadlessTopology's
  no-confused-deputy guarantee, inherited).
- **Anonymous paths** — the public site and discovery search require no session at all; report
  filing and the exclusion pre-check are explicitly public too (see
  [moderation.md](moderation.md)).

## Dependencies

- **Calls:** go-oikumenea's generated TypeScript SDK directly for anything
  tenant/person/religion-shaped (congregation roster, claim provisioning); `openfaithmap-api`'s own
  generated TypeScript SDK for content/moderation/vouching. Never a raw `fetch` against either
  service's REST surface — always the typed client.
- **Called by:** nothing — it is the ingress. It is the only container in
  [the deployment topology](../architecture/overview.md#deployment-topology) that publishes a host
  port.

## Patterns

- **No client-side authorization.** The console renders based on what the server-rendered payload
  already decided to include — it does not fetch permissively and hide UI client-side. A
  disallowed action returns an error from go-oikumenea/`openfaithmap-api`, surfaced as a normal
  error state, not silently prevented by a hidden button.
- **Registration wizard runs the exclusion pre-check first.** Before any go-oikumenea call, the
  wizard calls `POST /moderation/v1/exclusion-check` — see
  [core-integration.md](core-integration.md#provisioning-a-congregation-the-core-end-to-end-flow)
  step 1 — so an ineligible tradition never reaches the point of attempting org creation.

## Invariants

- **One session, forwarded everywhere.** `openfaithmap-web` never mints its own service-to-service
  credential for a request made on behalf of a logged-in user — only its background jobs (owned by
  `openfaithmap-api`, not this module) use the service-principal path.
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
