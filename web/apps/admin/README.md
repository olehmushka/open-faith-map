# web/apps/admin — openfaithmap-admin

The `openfaithmap-admin` service described in
[`docs/modules/web-admin.md`](../../../docs/modules/web-admin.md): the **only** app in OpenFaithMap
that ever holds a session — registration wizard, operator-approval console, congregation-admin
console, moderator console (D-AdminSurface). Split out of the original single `web/` app at M2.1;
see `docs/milestones-2026-08-07-2026-08-26.md`'s M2.1 entry for the history.

**Session layer** (`auth.ts`, `app/api/auth/[...nextauth]/route.ts`, `lib/oikumenea.ts`): Auth.js v5
with Google as the sole OIDC provider (no Keycloak — see
`docs/architecture/decisions.md`'s D-CoreDependency), forwarding the Google ID token as the bearer
on go-oikumenea calls. `/login` starts the flow; `/whoami` calls `identityFederation.whoami()`
through the forwarded token as the end-to-end proof.

**Registration** (`lib/registration.ts`, `/register`, `/register/submitted`,
`/admin/registrations`, `/my-congregation`) — see `docs/modules/registration.md`.

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

with `OPENFAITHMAP_ADMIN_AUTH_SECRET` and `AUTH_GOOGLE_SECRET` set in the repo-root `.env` (see
`.env.example`). See `web/apps/admin/.env.example` for the full set of variables this service reads.

## Stack (D-Stack)

Next.js (App Router), React 19, Tailwind, Auth.js v5 (Google provider) — see
[`docs/architecture/decisions.md#d-stack--the-same-toolchain-as-go-oikumenea`](../../../docs/architecture/decisions.md).
