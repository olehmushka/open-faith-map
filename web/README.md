# web/ — openfaithmap-web

The `openfaithmap-web` service described in
[`docs/architecture/overview.md`](../docs/architecture/overview.md) and
[`docs/modules/web-facade.md`](../docs/modules/web-facade.md): the single public-facing app serving
the public site, the congregation-admin console, and the moderator console.

**M1's session layer is wired** (`auth.ts`, `app/api/auth/[...nextauth]/route.ts`,
`lib/oikumenea.ts`): Auth.js v5 with Google as the sole OIDC provider (no Keycloak — see
`docs/architecture/decisions.md`'s D-CoreDependency), forwarding the Google ID token as the bearer
on go-oikumenea calls. `/login` starts the flow; `/whoami` calls
`identityFederation.whoami()` through the forwarded token as the end-to-end proof. Everything else
(public site, admin console, moderator console) is still unbuilt — that lands with `web-facade`'s
later milestones.

## Local dev

```sh
npm install
npm run dev
```

`npm run dev` alone is only useful for iterating on things that don't call go-oikumenea (build,
typecheck, static pages) — `oikumenea-app` publishes no host port, so a host-run dev server can't
reach it. Testing the real login flow requires the compose stack:

```sh
docker compose up --build
```

with `AUTH_SECRET` and `AUTH_GOOGLE_SECRET` set in the repo-root `.env` (see `.env.example`). See
`web/.env.example` for the full set of variables this service reads.

## Stack (D-Stack)

Next.js (App Router), React 19, Tailwind, Auth.js v5 (Google provider) — see
[`docs/architecture/decisions.md#d-stack--the-same-toolchain-as-go-oikumenea`](../docs/architecture/decisions.md).
