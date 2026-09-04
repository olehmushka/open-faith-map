# Deployment (M14.18, D-ProductionDeployment, D-TenantSubdomains)

🔶 **Gated on U14** (`docs/milestones-2026-08-26-now.md`): no apex domain is registered and no
DNS-provider API token exists yet. Everything in this directory is real, validated config — it
just hasn't served real traffic, and the milestone stays 🔶 until it has. Every other M14
milestone is verified locally against `*.localhost`, which needs none of this.

## What's here

- `caddy/Caddyfile` — TLS termination and reverse-proxying for `openfaithmap-web` and
  `openfaithmap-admin` only. `openfaithmap-api` is never fronted by Caddy — it stays internal,
  reached over the compose network exactly as in local dev.
- `caddy/Dockerfile` — builds a custom Caddy binary (via `xcaddy`) with the two third-party
  modules the Caddyfile needs: a DNS-01 provider and a rate-limiting handler. Stock `caddy:2` has
  neither.
- `../docker-compose.prod.yml` — the production override, layered on top of the existing
  `docker-compose.yml` (which is not itself modified — see D-ProductionDeployment).
- `../.env.prod.example` — the variables the override reads.

## Going live, once U14 resolves

1. Register the apex domain and point its nameservers at whichever DNS provider you choose.
2. Copy `.env.prod.example` to `.env` on the VM (`chmod 600`), fill in `APEX_DOMAIN`, `ACME_EMAIL`,
   and the DNS provider's API token, plus the existing secrets `docker-compose.yml` already reads
   (`OPENFAITHMAP_ADMIN_AUTH_SECRET`, `AUTH_GOOGLE_SECRET`, etc.).
3. `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build`
4. Confirm: `https://<apex>/` serves discovery, `https://admin.<apex>/` serves the admin app, a
   real `content_sites` row's slug serves at `https://<slug>.<apex>/`, and the response carries
   `Strict-Transport-Security: max-age=31536000; includeSubDomains`. Only then does M14.18's 🔶
   clear — per the milestone's own acceptance criteria, writing this config was never the bar.

## Using a different DNS provider

Cloudflare is this config's concrete example, not a decision — U14 only requires "a DNS API a
Caddy module supports". To swap:

1. Pick the matching module from [Caddy's DNS provider list](https://caddyserver.com/download)
   (module path looks like `github.com/caddy-dns/<provider>`).
2. In `caddy/Dockerfile`, replace the `--with github.com/caddy-dns/cloudflare` line with the new
   module.
3. In `caddy/Caddyfile`, replace both `dns cloudflare {$CLOUDFLARE_API_TOKEN}` lines with that
   module's own directive and credential shape (each provider module's own docs cover this — they
   differ, e.g. some take a token, some take key+secret pairs).
4. Rename the env var in `.env.prod.example` and `docker-compose.prod.yml`'s `caddy.environment`
   block to match.

## Why not more than this

`D-ProductionDeployment` also names per-surface OAuth clients, WireGuard for `oikumenea-console`,
`.env` secrets rotation, a `pg_dump` backup timer, and a weekly `ua-edr` re-run timer as scheduled
work. None of that is this milestone: `oikumenea-console` was deleted (D-OwnCore), which
explicitly retires the WireGuard item; the rest is real VM-provisioning work that needs an actual
VM to attach to, not config that can be written and validated ahead of one. M14.18 itself scopes
to exactly what's here: the Caddyfile, the wildcard DNS-01 cert, HSTS, and per-tenant read rate
limiting.
