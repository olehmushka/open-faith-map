# Architecture overview

## Shape

OpenFaithMap is **two new services sitting in front of a headless go-oikumenea core**:

```
                          ┌─────────────────────────────┐
  Visitor / congregation  │   openfaithmap-web (Next.js) │   owns the browser session
  admin (browser)  ─────► │   public site · admin console│   (Auth.js + Keycloak)
                          └───────────────┬──────────────┘
                                          │  user's bearer token, forwarded, never widened
                    ┌──────────────────────┼──────────────────────┐
                    ▼                                              ▼
        ┌───────────────────────┐                     ┌─────────────────────────┐
        │  openfaithmap-api     │                     │      go-oikumenea       │
        │  (Go, own Postgres)   │◄── SDK, user token ─►│  (internal-only, no     │
        │  content · moderation │    or service        │   public port)          │
        │  vouching             │    principal          │  tenant · person ·      │
        └───────────────────────┘                     │  authorization · location│
                    ▲                                  │  religion · search ·     │
                    │ service-principal token           │  audit                  │
                    │ (background jobs)                 └─────────────────────────┘
                    └───────────────────────────────────────────────┘
```

Two new binaries, same toolchain go-oikumenea uses (D-Stack):

- **`openfaithmap-web`** — a Next.js (App Router) server tier. It is the **facade** in the
  D-HeadlessTopology sense: it owns the httpOnly session, and every call it makes to either
  go-oikumenea or `openfaithmap-api` forwards the logged-in user's bearer token. It never asserts
  its own authority. Serves both the public discovery/site pages (mostly unauthenticated) and the
  congregation-admin console (authenticated).
- **`openfaithmap-api`** — a Go modular monolith, same hexagonal layering go-oikumenea uses
  (`transport → application → domain → adapters`), Conjure-contracted, its own Postgres. Owns
  `content`, `moderation`, and `vouching` (see their module docs). For everything else it is a
  **pure client** of go-oikumenea's generated SDK — it holds no tenant/person/authorization state
  of its own (D-Facade).

go-oikumenea itself is unmodified — run from its published docker image, headless
(D-CoreDependency), exactly as go-oikumenea's own `docker-compose.yml` runs its `app` container:
no host port published, reachable only over the compose-internal network.

## Request paths

**Anonymous discovery** (a visitor searching the map): browser → `openfaithmap-web` →
`openfaithmap-api` reads its own content cache **and** calls go-oikumenea's
`GET /religion/discovery/sites` directly (unauthenticated public read, `religion.read`) → merged
response rendered by `openfaithmap-web`. No user token exists in this path.

**Authenticated write** (a congregation admin editing their site): browser (with session cookie) →
`openfaithmap-web` extracts the bearer token from the session → calls `openfaithmap-api` for
content writes (its own PDP-free authorization check, delegated — see
[content.md](../modules/content.md#authorization-touchpoints)) and/or go-oikumenea directly for
anything tenant/person/religion-shaped (e.g. updating the congregation's clergy roster) — **always
forwarding the same user token**, never `openfaithmap-web`'s or `openfaithmap-api`'s own
credential.

**Background job** (nightly discovery-cache refresh, moderation-queue sweep, vouching-graph
integrity check): `openfaithmap-api`'s own scheduler → calls go-oikumenea using its
**service-principal** client-credentials token (D-ServiceIdentities) → narrow, named grants only
(see [core-integration.md](../modules/core-integration.md#authorization-touchpoints)).

## Deployment topology

`open-faith-map/docker-compose.yml` (planned shape, built at M1 — see
[milestones.md](../milestones.md)) extends go-oikumenea's own compose pattern rather than
reinventing it:

- `oikumenea-postgres`, `oikumenea-migrate`, `oikumenea-init-role`, `oikumenea-app` — go-oikumenea's
  own four-service bootstrap sequence, verbatim, image pulled from its published registry. `app`
  publishes no host port.
- `openfaithmap-postgres`, `openfaithmap-migrate` — OpenFaithMap's own Postgres + Atlas migration
  runner, for the `content`/`moderation`/`vouching` schema only.
- `openfaithmap-api` — reachable only internally by `openfaithmap-web`, no host port either.
- `openfaithmap-web` — the only container that publishes a host port; the single ingress for both
  human traffic and (indirectly) go-oikumenea traffic.
- `keycloak` (dev only, mirroring go-oikumenea's `docker-compose.dev.yml`) — the shared IdP both
  humans and OpenFaithMap's service principal authenticate against.

## What's unchanged from go-oikumenea, what's new

| | go-oikumenea | OpenFaithMap |
|---|---|---|
| Auth model | Delegates to external IdP, decides authorization (PDP) | Delegates *entirely* to go-oikumenea for anything tenant/person/religion-shaped; owns a narrow authorization check only for its own content/moderation/vouching tables |
| Toolchain | Go + gödel + Conjure + witchcraft + pgx/sqlc + Atlas | Identical |
| Schema | `oikumenea` schema, `oikumenea.<module>_*` tables | `openfaithmap` schema, `openfaithmap.<module>_*` tables — see [conventions.md](conventions.md) |
| UI | Optional Next.js admin console, BFF over the public API | Two audiences (public site + admin console) in one Next.js app, facade over both go-oikumenea and OpenFaithMap's own API |
| Exposure | Headless, internal-only (D-HeadlessTopology) | `openfaithmap-web` is the only public ingress |
