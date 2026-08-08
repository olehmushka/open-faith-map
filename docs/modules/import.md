# Module: import

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [registration](registration.md) · [architecture/decisions](../architecture/decisions.md)
> Owns no schema — a consumer, not a backend module (same shape as
> [web-facade](web-facade.md)/[web-admin](web-admin.md)). New at D-BulkImport.

## Purpose

`hermenea` — a small Go CLI, published as its own image (`docker.io/olegamysk/hermenea`), that
lets a **registration operator** bulk-onboard many congregations in one run instead of one
submission at a time through `openfaithmap-admin`'s registration wizard. It exists for the
scenario M2's registration flow was never designed for: importing an existing directory of
churches (a legacy-system migration, a partner-diocese bulk signup), where hundreds of rows need
the same submit → D-Exclusions-check → approve sequence a single prospective admin would otherwise
walk through by hand.

`hermenea` adds **no new mechanism**. For each row in its input file, it calls the exact same two
[registration](registration.md) Conjure endpoints the wizard and the operator-approval console
already call: `POST /requests` (submit, as if the row's designated congregation-contact person
were submitting it themselves), then `POST /requests/{id}/approve` (the real go-oikumenea writes —
`createChildOrg`, location, site, position, `unit`-scoped grant). Every invariant `registration.md`
already guarantees — the D-Exclusions taxon check per row, the submitter's person RID always
resolved from a real token rather than client-supplied, grants always `unit`-scoped — holds
identically for a `hermenea` run because it is the same code path, just called in a loop.

## Mechanism

- **Input.** A structured file (CSV or JSON — exact schema is a build-time decision for M2.2, not
  fixed here) with one row per congregation: tradition (a `religion_taxa` identifier or
  human-readable name resolved against the taxonomy), name, address, coordinates, and the
  congregation-contact person's identity (however submission attribution is resolved for an
  imported row — a build-time decision, see Open seams).
- **Authentication.** `hermenea` holds no credential of its own. A registration operator supplies
  their own real forwarded token to the tool for the duration of one run (an env var or a flag,
  decided at build time) — the same "operator-owned" trust level
  [registration.md](registration.md)'s `scripts/bootstrap-registration-org` already assumes. Every
  `POST /requests/{id}/approve` call `hermenea` makes uses that operator's token, exactly as the
  operator-approval console's UI does today.
- **Execution.** Submit-then-approve, row by row, against `openfaithmap-api`'s existing
  `RegistrationService`. A row whose D-Exclusions check rejects it, or whose approve call fails PDP
  authorization, is reported and skipped — it does not abort the run (partial success is expected
  for a large import).
- **Output.** A per-row result summary (created unit RID on success; rejection/failure reason
  otherwise) — exact reporting shape (stdout table, a written report file) is a build-time
  decision.

## Dependencies

- **Calls:** `openfaithmap-api`'s `RegistrationService` (`POST /requests`,
  `POST /requests/{id}/approve`) — the same Conjure-generated Go client
  `internal/coreintegration` already uses server-side, bound to the operator's forwarded token for
  each call. Never go-oikumenea directly, and never a service-principal token
  ([D-BulkImport](../architecture/decisions.md)).
- **Called by:** nothing — it is a CLI, run manually by a registration operator, not a service any
  other OpenFaithMap component calls into.

## Invariants

- **No new write path.** Every go-oikumenea write `hermenea` triggers happens through
  `registration.md`'s existing `approveRequest` application logic — nothing in this module
  performs a go-oikumenea call directly.
- **No new credential.** `hermenea` never holds, mints, or is issued a token of its own; it only
  ever forwards a real operator's own token, handed to it explicitly for one run
  ([core-integration.md](core-integration.md)'s no-on-behalf-of invariant, inherited unchanged).
- **Unattended runs are out of scope.** Because a real operator's token is required, `hermenea`
  cannot be wired into a scheduled/background job — every run is a deliberate, attended action by
  a real person, matching the same human-in-the-loop guarantee a one-by-one approval already gives
  the D-Exclusions check.

## Open seams

- **Not built yet.** This module doc describes the target shape (D-BulkImport); no code exists —
  sequenced as M2.2 in [milestones.md](../milestones.md), after M2's registration endpoints.
- **Input file schema** (CSV vs. JSON, exact column/field set, how a row's congregation-contact
  person is identified/resolved to a go-oikumenea person RID for attribution) is undecided —
  pick this when M2.2 is actually built.
- **Partial-failure reporting shape** (stdout, a written report file, exit-code semantics for
  scripting) is undecided.
- **Row-level throughput.** Submit-then-approve is two sequential HTTP round-trips per row;
  fine at expected import volumes (hundreds, not millions, of congregations), but if that changes,
  [D-BulkImport](../architecture/decisions.md) already records the fallback considered (a batch
  endpoint) and why it isn't built now.
