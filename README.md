# OpenFaithMap

[![CI](https://github.com/olehmushka/open-faith-map/actions/workflows/ci.yml/badge.svg)](https://github.com/olehmushka/open-faith-map/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A free, open-source, **Christian** church-discovery-and-presence platform — a **map** (discovery)
and a per-congregation **site builder** (presence) — built as a facade on top of
[go-oikumenea](https://github.com/olehmushka/go-oikumenea), consumed as a headless internal core
via its docker image. go-oikumenea supplies identity, authorization, the organizational graph,
location, and the multi-faith religion taxonomy; OpenFaithMap supplies everything a general
directory/authz core has no reason to own: site content, public discovery UX, moderation, and a
web-of-trust vouching layer.

Two audiences: **visitors** (anonymous, use the map and read congregation sites) and
**congregation admins** (verified, manage one or more congregations' presence and roster). A small
platform-wide **moderator** roster handles reports, appeals, and the denomination-exclusion policy.
Geographic rollout: Ukraine + USA first, then Poland/UK, then the rest of EU/LATAM/Africa/Asia.

## Status

**M1 in progress** (go-oikumenea integration wiring — see the
[stage board](docs/milestones.md#stage-board)). The architecture is design-complete for milestones
M0–M6 (see [docs/milestones.md](docs/milestones.md)) — every binding decision, module, and entity
model is written down before any of it is built, on purpose (see
[docs/development-process.md](docs/development-process.md)). This repository currently holds:

- The full design doc set under [docs/](docs/) — **read `docs/README.md` first**, it is the source
  of truth code is held to.
- A minimal, runnable `openfaithmap-api` skeleton (boots, serves health checks).
- `docker-compose.yml`: a real, working go-oikumenea instance (its published image, migrated, on a
  shared Postgres with OpenFaithMap — see `oikumenea` vs `openfaithmap` schemas in the compose file)
  plus `openfaithmap-api`. **No Keycloak** — OpenFaithMap authenticates directly against Google (see
  `deploy/oikumenea-install.yml`'s header comment); this is a real deviation from
  `architecture/decisions.md`'s original shared-Keycloak sketch that still needs its own `D-<Name>`
  write-up.
- `internal/coreintegration`: a real, proven service-principal client — a GCP service account mints
  its own Google ID token per call (no OAuth2 client-credentials, no Keycloak), which go-oikumenea
  validates and resolves by `(issuer, subject)`. `scripts/bootstrap-service-principal` registers it
  against a fresh instance. Proven end-to-end (`internal/coreintegration`'s integration test) against
  `connector.read`, not the `religion.read` grant OpenFaithMap's own docs describe needing — every
  religion-module read endpoint currently denies machine subjects outright (`RequireAnywhere`, a
  person-only PEP path); only the `connector`/`wiring` modules are machine-reachable
  (`RequireService`) today. That's a real go-oikumenea gap worth raising upstream, not something
  papered over here. `core-integration.md`'s `audit.write` grant also doesn't exist (go-oikumenea's
  audit module is read-only from the API) — both are documented in the script's comments, not yet
  corrected in `docs/modules/core-integration.md`.
- A placeholder `web/` for the future `openfaithmap-web` Next.js app — the user-token-passthrough /
  login path is still deferred to a follow-up session, so M1's "login working" exit criterion isn't
  met yet.

## Repository layout

```
cmd/openfaithmap-api/   composition root — the openfaithmap-api binary
internal/               hexagonal modules (transport → application → domain → adapters),
                         one directory per docs/modules/*.md
docs/                   the binding design doc set — read this first
var/conf/               local-dev install.yml / runtime.yml
web/                    openfaithmap-web (Next.js) — placeholder, see web/README.md
```

`api/` (Conjure IDL contracts) and `internal/conjure/` (generated server code) don't exist yet —
they're added together, in the same PR as the first module's first `<module>.conjure.yml`, once a
module reaches its "backend" gate (see [docs/development-process.md](docs/development-process.md)).

## Toolchain (D-Stack)

Same stack as go-oikumenea: Go + [gödel](https://github.com/palantir/godel) +
[Conjure](https://github.com/palantir/conjure) +
[witchcraft-go-server](https://github.com/palantir/witchcraft-go-server) + pgx/sqlc + Atlas
migrations, Next.js (App Router) for the web tier. See
[docs/architecture/decisions.md#d-stack--the-same-toolchain-as-go-oikumenea](docs/architecture/decisions.md).

## Quickstart

```sh
./godelw verify        # format + lint + test, the same gate CI runs
go run ./cmd/openfaithmap-api serve
curl -sk https://localhost:3001/status/liveness   # 3000 = app API, 3001 = management (health/readiness)
```

To also bring up a real go-oikumenea instance (M1) — needs a sibling checkout of
[go-oikumenea](https://github.com/olehmushka/go-oikumenea) for its migrations (not published as a
standalone artifact yet), and a GCP service-account key for the service principal:

```sh
docker compose up -d postgres oikumenea-migrate oikumenea-init-role oikumenea-app
docker run --rm --network open-faith-map_default -v "$PWD":/src -w /src golang:1.26-bookworm \
  go run ./scripts/bootstrap-service-principal -subject <google-service-account-numeric-sub>
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/development-process.md](docs/development-process.md)
for the feature pipeline (idea → decided → designed → backend → migrated → ui → verified) every
change moves through.

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
