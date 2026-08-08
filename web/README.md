# web/ — OpenFaithMap's two Next.js apps

Per `docs/architecture/decisions.md`'s D-AdminSurface, this directory holds two fully independent
Next.js apps — no shared workspace, no shared code:

- **[`apps/web`](apps/web/README.md)** — `openfaithmap-web`, the anonymous public site. Never holds
  a session. See `docs/modules/web-facade.md`.
- **[`apps/admin`](apps/admin/README.md)** — `openfaithmap-admin`, the only surface that ever holds
  a credential: registration wizard, operator-approval console, congregation-admin console,
  moderator console. See `docs/modules/web-admin.md`.

`web/go.mod` walls this whole directory off from the root Go module (see its header comment) — it
covers both apps, no real Go code lives in it.
