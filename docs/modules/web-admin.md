# Module: web-admin

> **Partially superseded (M10.7–M10.8, 2026-08-18) by
> [D-OwnCore](../architecture/decisions.md#d-owncore--openfaithmap-owns-its-core-go-oikumenea-is-removed)
> and [D-SuperAdminFold](../architecture/decisions.md#d-superadminfold--super-admin-folds-into-openfaithmap-admin-behind-a-role).**
> Every "go-oikumenea" reference below (the SDK, `deploy/oikumenea-install.yml`, calls to a separate
> core) describes the pre-M10.7 architecture — `openfaithmap-admin` now calls only
> `openfaithmap-api`'s own `lib/core.ts`/generated SDK (M10.7), and gained a fifth surface this
> section doesn't yet list: the instance-admin console (people/role-grants/units/taxa,
> D-SuperAdminFold, M10.8), gated by `api/core.conjure.yml`'s `CoreSuperAdminService` route-group
> middleware plus a cosmetic redirect in the admin app's own `(super-admin)` route group. Kept below
> largely as written rather than rewritten in full — the session/identity mechanics (Auth.js, Google
> ID token forwarding, no anonymous paths) are still accurate, only their destination changed.
>
> Reads: [glossary](../glossary.md) · [architecture/overview](../architecture/overview.md) ·
> [core-integration](core-integration.md) · [web-facade](web-facade.md)
> Owns no schema — a consumer, not a backend module. New at D-AdminSurface, split out of
> `web-facade` once that module's original "two audiences, one app" framing stopped holding (the
> public app never authenticated anyone, so there was never a session to share).

## Purpose

`openfaithmap-admin` — the **verified surface** (D-AdminSurface): a separate Next.js App Router
application, its own deployment and its own origin, holding the *only* session that exists anywhere
in OpenFaithMap. Everything that requires being logged in lives here: the registration wizard, the
operator-approval console, the congregation-admin console, and the moderator console. It asserts no
authority of its own (D-Facade) — it forwards the logged-in user's token everywhere, exactly as the
single-app `openfaithmap-web` used to before the split.

## Surfaces

- **Registration wizard** — any authenticated person can submit a congregation-registration request
  (see [registration.md](registration.md)); being logged in is required to submit at all, which is
  why this lives here and not on the anonymous public site.
- **Operator-approval console** — a registration operator reviews pending requests and
  approves/rejects them, performing the real go-oikumenea writes with their own token
  ([registration.md](registration.md)).
- **Congregation-admin console** — site editor (block-based page builder), staff/clergy roster (thin
  views over go-oikumenea's `person`/`membership`/`religion` data), vouching request UI,
  report/appeal filing for an admin's own congregation.
- **Moderator console** — report queue, action history, appeal decisions, guarantor management.
  Small audience (platform moderators only); gated by the `moderation.read`/`moderation.act` checks
  documented in [moderation.md](moderation.md).

## Session & identity

- **Login** — Auth.js v5, Google as the sole OIDC provider (D-CoreDependency's as-built note — no
  Keycloak, no shared realm). go-oikumenea's own console-bff supports registering multiple IdPs;
  OpenFaithMap uses only the one provider go-oikumenea's `deploy/oikumenea-install.yml` is
  configured to trust for human login.
- **Session storage** — httpOnly cookie, server-side only; the ID token is never exposed to
  client-side JavaScript. This is the only app in OpenFaithMap that has a cookie-based session at
  all — `openfaithmap-web` has none (D-AdminSurface).
- **Token forwarding** — every server-side call `openfaithmap-admin` makes, to either go-oikumenea or
  `openfaithmap-api`, forwards the logged-in user's Google **ID token** unchanged — not the access
  token, which is an opaque string go-oikumenea cannot verify; the ID token is the JWT whose `aud`
  go-oikumenea's Google issuer entry pins (`deploy/oikumenea-install.yml`). `openfaithmap-admin`
  holds no credential of its own that widens what a caller can do (D-HeadlessTopology's
  no-confused-deputy guarantee, inherited).
- **No anonymous paths.** Everything behind this app assumes a logged-in person; a visitor who isn't
  logged in is redirected to Google login, not served a degraded anonymous view — that's what
  `openfaithmap-web` is for.

## Dependencies

- **Calls:** go-oikumenea's generated TypeScript SDK directly for anything
  tenant/person/religion-shaped (congregation roster, claim provisioning); `openfaithmap-api`'s own
  generated TypeScript SDK for content/moderation/vouching writes. Never a raw `fetch` against either
  service's REST surface — always the typed client.
  > **True as of M2.6 (2026-08-10).** `web/apps/admin/lib/registration.ts` now delegates to a
  > generated client (`web/apps/admin/lib/openfaithmap`, generated in place from
  > `api/registration.conjure.yml` — not a separate package, see D-Stack in
  > `architecture/decisions.md`), the same shape `lib/oikumenea.ts` already used for go-oikumenea's
  > SDK. No hand-written HTTP client remains for either service. Full detail:
  > [milestones-2026-08-07-2026-08-26.md](../milestones-2026-08-07-2026-08-26.md)'s M2.6.
- **Called by:** nothing — it is one of two public ingress points (the other is
  [`openfaithmap-web`](web-facade.md#dependencies)). It publishes its own host port and, once built,
  is expected to sit at its own subdomain (e.g. `admin.openfaithmap.org`), separate from the public
  site's origin.

## Patterns

- **No client-side authorization.** The console renders based on what the server-rendered payload
  already decided to include — it does not fetch permissively and hide UI client-side. A disallowed
  action returns an error from go-oikumenea/`openfaithmap-api`, surfaced as a normal error state, not
  silently prevented by a hidden button.
- **The exclusion check runs before any go-oikumenea write.** *As designed*, the wizard would call
  `POST /moderation/v1/exclusion-check` first. **As built (M2), there is no separate pre-check
  call:** `moderation` doesn't exist yet, and `POST /registration/v1/requests` runs the D-Exclusions
  taxon-ancestor walk server-side before persisting anything, returning
  `Registration:TaxonExcluded` ([registration.md](registration.md)). The invariant that matters —
  an ineligible tradition never reaches org creation — holds either way. Revisit the split when M5
  lands and the check consolidates into `moderation`.

## Invariants

- **One session, forwarded everywhere.** `openfaithmap-admin` never mints its own service-to-service
  credential for a request made on behalf of a logged-in user — only its background jobs (owned by
  `openfaithmap-api`, not this module) use the service-principal path.
- **The only credential-holding surface.** No other OpenFaithMap-owned app is allowed to gain a
  session — if a feature seems to need one outside this app, that's a sign it belongs in
  `openfaithmap-admin`, not a reason to add a session elsewhere (D-AdminSurface).

## Open seams

- **Shared-code boundary with `openfaithmap-web`** is resolved as "none" (M2.1): the two apps are
  fully independent, no `web/packages/*`. Revisit only if config/UI-primitive duplication between
  them becomes a real cost.
- **`admin.openfaithmap.org`-style dedicated subdomain** isn't provisioned yet — this app runs on a
  plain host port (`3004`) in local dev, same as every other compose service.
