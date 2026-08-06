# Development process

OpenFaithMap adopts go-oikumenea's feature pipeline verbatim (same gate names, same discipline),
scoped to this repo's own doc paths. Read this before starting, advancing, or reporting on any
feature. The [stage board](milestones.md#stage-board) records where each milestone currently sits;
this doc defines the gates it moves through.

## The seven states, six gates

| State | Exit artifact | Docs touched |
|---|---|---|
| idea | A `## TODO-N · Title [status: idea]` entry in a project `todo.md` (not yet created — added at the first idea past this initial doc set) | `todo.md` |
| decided | A binding `D-<Name>` block in [`architecture/decisions.md`](architecture/decisions.md) | `architecture/decisions.md` |
| designed | A module doc written to the fixed template (purpose → entities → data model → Conjure sketch → dependencies → authorization touchpoints → invariants → open seams) + an `M#` row added to [`milestones.md`](milestones.md) | `milestones.md`, `modules/`, `glossary.md` |
| backend | A Go module in `openfaithmap-api` + an `api/<module>.conjure.yml` (generated, never hand-edited) | code |
| migrated | One versioned Atlas migration under `migrations/`, expand-only, lint-gated | code |
| ui | Page(s) under `openfaithmap-web/`, or `➖` for a backend-only milestone | code |
| verified | The milestone's exit criteria are met, tests pass, the slice boots/migrates/demos end-to-end | stage board → ✅ |

Two entry points converge at "decided": a raw idea logged in `todo.md`, or a seam parked in
[`open-questions.md`](open-questions.md). An idea-stage feature lives only in `todo.md` until it
earns an `M#` at "designed" — the stage board holds only `M#` rows, same rule go-oikumenea follows.

## Runbook

**Advancing to decided.** Write the `D-<Name>` block: decision, why, why-not (the alternatives
seriously considered), consequences. If it touches [D-Facade](architecture/decisions.md) (i.e.,
does this new feature belong in OpenFaithMap or does go-oikumenea already own it?), that question
must be answered explicitly in the block — the default assumption is always "check
[core-integration.md](modules/core-integration.md) first," never "build it here because it's
convenient."

**Advancing to designed.** Write or extend the module doc to the fixed template. Every entity gets
its RID-space note (an OpenFaithMap-local RID, or an explicit "opaque go-oikumenea RID foreign
value" — see [conventions.md](architecture/conventions.md)). Add the `M#` row to the stage board
with all six gate columns starting `⬜`.

**Advancing to backend/migrated/ui.** Standard go-oikumenea discipline: migrations are
expand-only (never a destructive change without a documented contract-phase migration), Conjure
contracts are regenerated not hand-edited, and every go-oikumenea call goes through the generated
SDK (never a raw HTTP client — see [D-CoreDependency](architecture/decisions.md)).

**Advancing to verified.** Ground every ✅ in a real artifact — a migration file, a rendered page, a
passing end-to-end test — never from memory. This is the same rule go-oikumenea's own
`CLAUDE.md` states for its stage board, carried over unchanged because it's the discipline that
actually keeps a stage board trustworthy.

## Stage-board honesty

The board is authoritative for **stage**; each milestone's own prose in
[`milestones.md`](milestones.md) is authoritative for **detail**. Discrepancies resolve in the
board's favor. Update the stage board in the same commit/PR that passes a gate — not as a
follow-up.
