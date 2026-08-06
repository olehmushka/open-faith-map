# web/ — openfaithmap-web (placeholder)

This is a bare Next.js (App Router) skeleton for the future `openfaithmap-web` service described in
[`docs/architecture/overview.md`](../docs/architecture/overview.md) and
[`docs/modules/web-facade.md`](../docs/modules/web-facade.md): the single public-facing app serving
the public site, the congregation-admin console, and the moderator console.

**Nothing here is real yet** — no session layer, no Auth.js/Keycloak wiring, no calls to
`openfaithmap-api` or go-oikumenea. That work starts once `web-facade` reaches its first "ui" gate
(see [`docs/development-process.md`](../docs/development-process.md)), which depends on M1
(go-oikumenea integration) landing first.

## Local dev

```sh
npm install
npm run dev
```

## Stack (D-Stack)

Next.js (App Router), React 19, Tailwind, Auth.js v5 (not yet wired) — see
[`docs/architecture/decisions.md#d-stack--the-same-toolchain-as-go-oikumenea`](../docs/architecture/decisions.md).
