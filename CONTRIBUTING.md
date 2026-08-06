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

For the web tier:

```sh
cd web
npm install
npm run dev
```

## Making a change

1. Fork the repo and create a branch off `main`.
2. Follow the layering already in `internal/<module>/{transport,application,domain,adapters}` —
   domain owns its interfaces and imports no framework; cross-module calls inside
   `openfaithmap-api` are direct interface calls, cross-module mutations are domain events (same
   rule go-oikumenea applies inside its own monolith).
3. Run `./godelw verify` before pushing — it runs format, lint, and test in one gate.
4. Migrations (once any exist) are expand-only via Atlas, under `migrations/`, one repo-root
   directory — never a destructive change without a documented contract-phase migration.
5. Conjure contracts (`api/<module>.conjure.yml`) are the source of truth; generated Go/TypeScript
   code is never hand-edited.
6. Open a PR describing what gate(s) it advances. Link the `D-<Name>` / module doc / stage-board
   row it corresponds to.

## Commit style

Short, imperative subject line (`Add content module domain types`, not `Added` or `Adding`).
Explain *why* in the body when the diff alone doesn't make it obvious.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you're
expected to uphold it.

## Reporting a security issue

See [SECURITY.md](SECURITY.md) — please do not open a public issue for vulnerabilities.
