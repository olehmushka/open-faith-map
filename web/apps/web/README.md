# web/apps/web — openfaithmap-web

The `openfaithmap-web` service described in
[`docs/architecture/overview.md`](../../../docs/architecture/overview.md) and
[`docs/modules/web-facade.md`](../../../docs/modules/web-facade.md): the anonymous public site —
home, discovery map/search, per-congregation pages. Holds no session, no credential, no Auth.js
dependency at all (D-AdminSurface) — that capability lives entirely in the sibling
[`apps/admin`](../admin/README.md) app.

Today this is still the M1-era placeholder home page only; the real public site (map, search,
congregation pages) lands with `web-facade`'s later milestones (M4).

## Local dev

```sh
npm install
npm run dev
```

## Stack (D-Stack)

Next.js (App Router), React 19, Tailwind — no Auth.js. See
[`docs/architecture/decisions.md#d-stack--the-same-toolchain-as-go-oikumenea`](../../../docs/architecture/decisions.md).
