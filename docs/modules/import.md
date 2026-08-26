# Module: import

> **Superseded (M10.8, 2026-08-18) by
> [D-StaticRefData](../architecture/decisions.md#d-staticrefdata--reference-data-is-a-static-seed-hermenea-is-removed).**
> `hermenea` — the service this entire document is about deploying — is deleted. Country reference
> data is now a static seed (`internal/refdata`, `migrations/0012_core_refdata.sql`), read
> byte-for-byte from what `hermenea` itself had synced before teardown (M10.1, sanity-checked again
> at M10.8's baseline capture — see `internal/refdata/testdata/README.md`). Kept below as historical
> record of why this module existed and what it did, not as current deployment guidance.
>
> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [architecture/decisions](../architecture/decisions.md)
> Owns no schema, no code, no service of its own — this doc describes **deploying** a service
> OpenFaithMap doesn't build. Corrects D-BulkImport's original premise — see its Correction
> subsection in [architecture/decisions.md](../architecture/decisions.md).

## Purpose

`hermenea` is **go-oikumenea's own, pre-existing companion service** (sibling repo, `cmd/hermenea` +
`internal/hermenea/*`) — not something OpenFaithMap builds. It seeds and enriches go-oikumenea's
reference/catalog data: countries, languages, external organizations, geo places. go-oikumenea's
core already pre-populates a base skeleton of this data itself at boot (its own embedded `pinax`
presets — code/name-only countries, a handful of top languages, a minimal religion taxonomy); what
`hermenea` adds is everything beyond that baseline: country geometries (Who's On First), a fuller
language catalog (Glottolog/CLDR), external organizations (Wikidata), and any future source a
`sources:` entry declares.

OpenFaithMap's only job for this module is **deploying** `hermenea` — compose wiring and an install
config — in its own `docker-compose.yml`, the same way M1.2 deployed go-oikumenea's own
`oikumenea-console` unmodified. No CLI, no Go code, no schema, no credential of OpenFaithMap's own.

This corrects M2.2's original scope, which described a CLI OpenFaithMap would build at
`cmd/hermenea` to bulk-onboard *congregations* by replaying `registration`'s submit/approve
endpoints. That design never worked mechanically (see D-BulkImport's Correction) and collided with
a name go-oikumenea already owns for something unrelated. The congregation/scraped-church-data
import scenario that original design was reaching for is real but is the project owner's own
**future**, separately-scoped, separately-named work — see Open seams.

## Mechanism

- **Sources, declared not coded.** A mounted YAML install config (`hermenea-install.yml`) lists
  `sources:` entries, each with a `code`, a `connector-type` (`file`, `http`, `http-files`,
  `wof-sqlite`, `factbook`), an `object-type` it produces, a `locator` (a bundled file path, a URL,
  or a `owner/repo@ref`), a `cron` schedule, and `enabled`. Adding a new import source is a config
  change, not a code change.
- **Its own database.** `hermenea` owns a separate Postgres database (named `hermenea`, not a schema
  inside `oikumenea`/`openfaithmap`) with its own Atlas migration set — staging tables for fetched
  batches, run history, a worker job queue, a resume cursor. Fully isolated from both of
  OpenFaithMap's own schemas.
- **Fetch → stage → post.** For an enabled source, a connector fetches the external data, stages it,
  maps it into a `CanonicalEnvelope` (`{objectType, source, sourceVersion, records}`), and posts it
  to go-oikumenea's core at `POST /import/{objectType}` (`api/import.conjure.yml`) — idempotent,
  non-destructive upserts, chunked for large sources.
- **Its own credential.** `hermenea` authenticates outbound as the `hermenea-importer` service
  principal, holding exactly instance-scope `import.manage` — boot-seeded automatically by
  go-oikumenea's own migration (`migrations/0011_infra.sql`), nothing OpenFaithMap registers or
  bootstraps. Two shared-secret env vars carry the trust in both directions: `HERMENEA_OIKUMENEA_TOKEN`
  (hermenea → core, inbound to core) and `OIKUMENEA_HERMENEA_TOKEN` (core → hermenea, for core to
  proxy an operator-triggered sync).
- **Trigger.** A source runs on its own `cron` schedule, or on demand via
  `POST /sync/{source-code}` against `hermenea`'s own API (bearer = `OIKUMENEA_HERMENEA_TOKEN`).
  There is no CLI invocation, no flag-driven one-shot run — `hermenea` is a persistent server
  (`cmd/hermenea`'s `main.go` boots a witchcraft server unconditionally), not a job binary.

## Dependencies

- **Calls:** go-oikumenea's core, over HTTP only (`POST /import/{objectType}`), authenticated as the
  `hermenea-importer` service principal. Never calls `openfaithmap-api`, never touches
  `registration`'s endpoints — this module has no relationship to congregation registration at all.
- **Called by:** nothing in OpenFaithMap. Triggered by its own cron, or by a human operator's direct
  `POST /sync/{source-code}` call (or, on go-oikumenea's side, a UI-triggered call proxied through
  core using `OIKUMENEA_HERMENEA_TOKEN` — not built/exposed anywhere in OpenFaithMap's own UI).

## Invariants

- **No OpenFaithMap code touches this path.** Everything above lives in go-oikumenea's own repo and
  runs as its own service; this repo contributes only deploy configuration.
- **`hermenea`'s credential is scoped to `import.manage` only** — minted and enforced entirely by
  go-oikumenea's own bootstrap migration and PDP, not by anything OpenFaithMap issues or manages.
- **Unattended operation is the normal mode**, not an exception — sources run on cron by default;
  manual triggering via `POST /sync/{source-code}` is the exceptional case, not the rule (this
  corrects the original CLI design's opposite assumption).

## Operating hermenea

Bring-up (`hermenea` itself now pulls a published image, `docker.io/olegamysk/hermenea`, same as
`oikumenea-app` — but its migrations still aren't published as a standalone artifact, so a sibling
checkout of go-oikumenea is still needed for `migrations/hermenea/`, same `OIKUMENEA_SRC` convention
the top of `docker-compose.yml` already documents for go-oikumenea's main migrations):

```sh
OIKUMENEA_SRC=/path/to/go-oikumenea docker compose up --build \
  postgres oikumenea-migrate oikumenea-init-role oikumenea-app \
  init-hermenea-db migrate-hermenea hermenea
```

The bundled, network-free countries source runs automatically on first boot (its `@weekly` cron has
never fired before, so the scheduler treats it as due immediately) — no trigger needed for a fresh
stack. To re-trigger any source on demand, `POST` to `hermenea`'s own API (base-path `/hermenea/v1`,
per `api/hermenea.conjure.yml` in go-oikumenea — **not** the bare `/sync/{source}` path go-oikumenea's
own compose-file comment shows, which 404s against a real instance):

```sh
curl -k -X POST https://localhost:9443/hermenea/v1/sync/geo-countries-iso3166 \
  -H "Authorization: Bearer dev-oikumenea-trigger-token-change-me"
# -> {"jobId":"...","status":"queued"}
```

Verify the data landed — either via `oikumenea-console` (M1.2)'s geo/religion catalog view, or
directly against go-oikumenea's `oikumenea` schema's `geo_countries` table, or against hermenea's
own `hermenea.import_runs` (joined to `hermenea.import_sources` on `source_id`) for per-run
created/updated/skipped counts. Confirmed end-to-end against a live stack while building this
milestone: the boot-time run for `geo-countries-iso3166` reached `status = succeeded`
(0 created / 2 updated / 30 skipped) and `oikumenea.geo_countries` held 250 rows. Note: with the
default single-worker concurrency (`worker.concurrency` unset), a slow live-network source
(`glottolog-languoids`, fetching from raw.githubusercontent.com) can occupy the queue for minutes
and delay a subsequently triggered source — raise `worker.concurrency` if that matters for a real
run, or trigger the network-free source first on a fresh stack.

**Operational model — a persistent service, not a one-shot job.** `hermenea` does not exit when a
sync finishes; it's meant to keep running (cron-scheduled, or triggerable) the way go-oikumenea's
own `docker-compose.yml` runs it. That said, it's safe to bring the whole deployment down once the
imports you need have landed: `hermenea` owns no state go-oikumenea's core still needs after
`POST /import/{objectType}` has been called — deleting its database, migrate step, and service
together loses nothing core depends on. Translating "spin up, seed, tear down" into a real ArgoCD
`Application` lifecycle (sync-then-prune, or scale-to-zero) is this project's own infra-repo
decision, outside anything this repo's `docker-compose.yml` can encode on its own — this doc names
the pattern, not a specific manifest.

## Open seams

- **Resolved (2026-08-12): the future scraped-church-data importer is now the `congregationimport`
  module** — see [congregationimport.md](congregationimport.md) and
  [D-CongregationImport](../architecture/decisions.md). Not `hermenea`, avoiding the name
  collision this doc originally flagged; `DS-OFM-10` is closed.
- **Default-enabled live-network sources.** The reference install config this repo copies from
  go-oikumenea enables `glottolog-languoids`, `cldr-language-scripts`, and `wikidata-orgs-ua` by
  default (`@weekly` cron, live calls to raw.githubusercontent.com / iso639-3.sil.org /
  query.wikidata.org). Reasonable defaults to carry over unchanged, but worth a deliberate choice
  later if a quieter local-dev default is preferred.
- ~~**Exact verified table name for a post-import check.**~~ **Resolved by M2.2's own verification:**
  `oikumenea.geo_countries`, confirmed with 250 real rows after the boot-time
  `geo-countries-iso3166` run. Already stated in the runbook above; this seam predated that
  confirmation.
- **Shared secrets are hardcoded in `docker-compose.yml`.** `HERMENEA_OIKUMENEA_TOKEN` and
  `OIKUMENEA_HERMENEA_TOKEN` appear as literals in two services each, unlike every other secret in
  this repo, which goes through `${...}` + `.env.example`. Combined with `hermenea` publishing host
  ports 9443/9444, that token is the only gate on `POST /hermenea/v1/sync/{source}` from the host.
  Fine for local dev; a real deployment has to remember to edit the compose file itself, which is
  exactly the kind of thing that gets forgotten. Moved to `.env` by
  [milestones.md](../milestones.md)'s **M2.4**.
