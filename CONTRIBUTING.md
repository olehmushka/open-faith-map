# Contributing to OpenFaithMap

Thanks for your interest in contributing. This is a young, design-first project — please read
[`docs/README.md`](docs/README.md) before writing any code. It explains the reading order for the
rest of the doc set and states plainly: **if code and a decision in
[`docs/architecture/decisions.md`](docs/architecture/decisions.md) disagree, the code is wrong.**

## The feature pipeline

Every change moves through the same six gates, defined in
[`docs/development-process.md`](docs/development-process.md): idea → decided → designed → backend →
migrated → ui → verified. In short:

- A new idea starts as a `TODO-N` entry (or a parked seam in
  [`docs/open-questions.md`](docs/open-questions.md)), not a pull request.
- Anything touching scope or architecture needs a binding `D-<Name>` decision block in
  `docs/architecture/decisions.md` **before** the code lands.
- A new module needs its doc under `docs/modules/` written to the fixed template before its
  `backend` gate.
- [`docs/milestones.md`](docs/milestones.md)'s stage board is updated in the **same** PR that
  passes a gate — not as a follow-up.

If you're unsure whether something is decided or designed yet, check the stage board first.

## Local setup

```sh
git clone git@github.com:olehmushka/open-faith-map.git
cd open-faith-map
./godelw verify        # format + lint + test — the same gate CI runs
go run ./cmd/openfaithmap-api serve
```

`./godelw` is a self-bootstrapping wrapper (like Gradle's `gradlew`) — it downloads the pinned
gödel version on first run, no separate install needed. See
[`docs/architecture/decisions.md#d-stack--the-same-toolchain-as-go-oikumenea`](docs/architecture/decisions.md)
for why this toolchain (gödel + Conjure + witchcraft-go-server + pgx/sqlc + Atlas) rather than
something more generic.

For the web tier — two fully independent apps, no npm workspace (D-AdminSurface):

```sh
cd web/apps/web   && npm install && npm run dev   # openfaithmap-web  — anonymous public site
cd web/apps/admin && npm install && npm run dev   # openfaithmap-admin — the surface with a session
```

Each has its own `package.json`, `package-lock.json`, and `Dockerfile`; there is nothing to install
at `web/` itself.

## Branching

Every change — including solo, self-reviewed ones — goes through a branch and a PR against `main`.
No direct pushes to `main`: this formalizes what was already scaffolded
(`.github/PULL_REQUEST_TEMPLATE.md`) but never actually required.

Branch names: `<type>/<kebab-case-slug>`, one of four types:

- `feature/` — new capability: a milestone's backend, UI, or both (`feature/split-admin-ui`).
- `fix/` — a bug fix, a wrong assumption corrected, a doc-vs-code drift closed
  (`fix/registration-idempotency`).
- `docs/` — doc-only changes: a new `D-<Name>`, a module doc, a milestones/stage-board update
  (`docs/git-strategy`).
- `chore/` — everything else: tooling, CI, refactors, dependency bumps, tests
  (`chore/upgrade-godel`).

Keep the slug short and descriptive, not just the milestone number — the PR description is where
the `D-<Name>` / module / stage-board row it advances gets linked explicitly (see below).

## Making a change

1. Branch off `main` (see naming above).
2. Follow the layering already in `internal/<module>/{transport,application,domain,adapters}` —
   domain owns its interfaces and imports no framework; cross-module calls inside
   `openfaithmap-api` are direct interface calls, cross-module mutations are domain events (same
   rule go-oikumenea applies inside its own monolith).
3. Run `./godelw verify` before pushing — it runs format, lint, and test for the Go tree. Note it
   does **not** cover `web/`: run each app's own `npm run lint && npm run build` when you touch one.
   CI runs both. ⚠️ **CI's `web` job has been failing since the M2.1 split** — it still expects the
   deleted `web/package.json`. Fixed by [M2.4](docs/milestones.md); until then, `main` being red is
   known, and no milestone may advance to Verified while it is.
4. Migrations are expand-only via Atlas, under `migrations/`, one repo-root directory — never a
   destructive change without a documented contract-phase migration. Re-hash with
   `atlas migrate hash --env local` after adding one, or `atlas.sum` fails the apply.
5. Conjure contracts (`api/<module>.conjure.yml`) are the source of truth; generated Go/TypeScript
   code is never hand-edited.
6. Open a PR using the template — describe what gate(s) it advances and link the `D-<Name>` /
   module doc / stage-board row it corresponds to. Self-review and self-merge are fine while there's
   no other maintainer; the PR exists for the record and the CI gate, not to wait on someone else.
   Squash-merge by default — most PRs are already one well-composed commit by the time they're
   ready (see Commit style below), so squashing is usually a no-op, and it keeps `main` linear
   either way. Delete the branch after merge.

## Commit style

`<type>(<scope>): <Imperative summary>` — Conventional Commits' type vocabulary, this project's own
capitalization:

- **Types:** `feat`, `fix`, `docs`, `chore` (also `refactor`/`test`/`ci` for a commit that's purely
  one of those, still under a `chore/` branch) — matches the branch prefixes above one-to-one,
  except a `feature/` branch carries `feat:` commits (the Conventional Commits keyword, not the
  branch word).
- **Scope is optional** — the module or doc area touched (`feat(registration): ...`,
  `docs(web-admin): ...`) when the change is narrow; omit it for anything cross-cutting, same as
  most of this repo's history so far.
- **Summary stays imperative and capitalized** after the colon (`Add content module domain types`,
  not `added` or `adding`) — this project's existing voice, not Angular's lowercase convention.

Keep composing commits the way this repo already does: one commit per gate advanced (or a coherent
slice of one), bundling backend + migration + UI + docs for that slice rather than splitting them
into separate commits. The body is where the real content lives — what was built, what was verified
end-to-end (cite the actual command/test, not "should work"), and which `D-<Name>` / module doc /
milestone row it touches. [`docs/milestones.md`](docs/milestones.md)'s stage board gets updated in
the *same* commit that passes a gate, never a follow-up — restating
[`docs/development-process.md`](docs/development-process.md)'s rule here because it's a
commit-composition rule as much as a process one.

End the commit with `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>` (or the relevant
model/tool trailer) whenever an AI assistant materially wrote or drove the change — every commit in
this repo's history so far does, and that stays true going forward.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you're
expected to uphold it.

## Reporting a security issue

See [SECURITY.md](SECURITY.md) — please do not open a public issue for vulnerabilities.
