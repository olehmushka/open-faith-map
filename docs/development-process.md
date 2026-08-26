# Development process

OpenFaithMap adopts go-oikumenea's feature pipeline verbatim (same gate names, same discipline),
scoped to this repo's own doc paths. Read this before starting, advancing, or reporting on any
feature. The stage board records where each milestone currently sits — the
[active one](milestones-2026-08-26-now.md#stage-board) for anything after 2026-08-26, the
[closed record](milestones-2026-08-07-2026-08-26.md#stage-board) for M0–M13.6 — this doc defines
the gates it moves through.

## The seven states, six gates

| State | Exit artifact | Docs touched |
|---|---|---|
| idea | A `## TODO-N · Title [status: idea]` entry in a project `todo.md` (not yet created — added at the first idea past this initial doc set) | `todo.md` |
| decided | A binding `D-<Name>` block in [`architecture/decisions.md`](architecture/decisions.md) | `architecture/decisions.md` |
| designed | A module doc written to the fixed template (purpose → entities → data model → Conjure sketch → dependencies → authorization touchpoints → invariants → open seams) + an `M#` row added to [`milestones-2026-08-26-now.md`](milestones-2026-08-26-now.md) | `milestones-2026-08-26-now.md`, `modules/`, `glossary.md` |
| backend | A Go module in `openfaithmap-api` + an `api/<module>.conjure.yml` (generated, never hand-edited) | code |
| migrated | One versioned Atlas migration under `migrations/`, expand-only, lint-gated | code |
| ui | Page(s) under `web/apps/web/` (anonymous) or `web/apps/admin/` (verified) — D-AdminSurface — or `➖` for a backend-only milestone | code |
| verified | The milestone's exit criteria are met, **CI is green on `main`**, and the slice boots/migrates/demos end-to-end | stage board → ✅ |

**A sixth gate symbol: 🔶.** A milestone that is fully built but blocked on a named external action
(an OAuth redirect URI added in a console this repo can't automate, an upstream feature request)
shows `🔶` in `Verified`, not `⬜` — "done, waiting on one specific thing someone must do" and
"not started" are different states, and collapsing them hides work that is one step from finished.
Name the blocking action in the milestone's prose; `🔶` without a named action is just `⬜`.

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

**Check CI before you write ✅.** Added after the 2026-08-09 audit found `main` had been red since
M2.1 — the split deleted `web/package.json` and left CI's `web` job pointing at it — while three
subsequent milestones passed their gates and `CONTRIBUTING.md` kept promising "the same gate CI
runs." A green run on `main` at the merge commit is now part of the definition above. A red `main`
means no milestone advances to Verified until it's green, whatever else was proven by hand.

**A happy-path proof is not a Verified proof.** M2 was proven end-to-end by curl and still shipped
three defects — an authorization gate that discloses every submitter's PII, an endpoint with no
authorization at all, and a distributed write with no atomicity — because none of them appear on the
happy path. When the exit criteria are about *authorization* or *failure modes*, the artifact has to
exercise those specifically: a second token that should be refused, a process killed mid-write.

## Stage-board honesty

The board is authoritative for **stage**; each milestone's own prose — in
[`milestones-2026-08-26-now.md`](milestones-2026-08-26-now.md) for anything after 2026-08-26, in
[`milestones-2026-08-07-2026-08-26.md`](milestones-2026-08-07-2026-08-26.md) for M0–M13.6 — is
authoritative for **detail**. Discrepancies resolve in the board's favor. Update the (active) stage
board in the same commit/PR that passes a gate — not as a follow-up.

This is not a hypothetical rule. M2.1 deleted `web/package.json`, leaving CI's `web` job pointing at
a path that no longer existed — every run since had been failing. M2.2 was still marked `✅` Verified
on top of that red `main`, and M2.1 itself sat at 🔶 rather than being caught as the actual cause.
**Check CI before you write `✅`** above exists because of this incident, fixed by M2.4. If the board
and a red `main` ever disagree again, the board is wrong until CI says otherwise.
