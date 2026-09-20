# CLAUDE.md

Guidance for Claude Code (and any other agent or engineer) working in this repository.

## What this is

OpenFaithMap — an open-source, Christian church-discovery-and-presence platform: a **map**
(discovery) plus a per-congregation **site builder** (presence). One Go API
(`openfaithmap-api`) with its own Postgres schema, fronted by two independent Next.js apps:
anonymous `openfaithmap-web` (the public map + tenant sites) and credentialed `openfaithmap-admin`
(the only surface that ever holds a session — Google OAuth is the sole IdP). Full architecture:
[`docs/README.md`](docs/README.md) (start there, it names the reading order for everything else).

**Current status:** M0–M15 are done. The active roadmap and stage board live in
[`docs/milestones-2026-09-20-now.md`](docs/milestones-2026-09-20-now.md) — read its
"Unresolved unknowns" section before starting any build work. Closed history is archived in
[`docs/milestones-2026-08-20-2026-09-20.md`](docs/milestones-2026-08-20-2026-09-20.md) (M14–M15)
and [`docs/milestones-2026-08-07-2026-08-26.md`](docs/milestones-2026-08-07-2026-08-26.md)
(M0–M13.6).

Scope or architecture decisions are binding only once written as a `D-<Name>` block in
[`docs/architecture/decisions.md`](docs/architecture/decisions.md). The feature pipeline (idea →
decided → designed → backend → migrated → ui → verified) is in
[`docs/development-process.md`](docs/development-process.md) — read it before starting, advancing,
or reporting on any milestone-sized piece of work.

## Quickstart

```
./godelw verify              # Go: format + lint + test (the CI gate)
go run ./cmd/openfaithmap-api serve   # health check on :3001

cd web/apps/admin && npm install && npm run dev   # :3004, needs AUTH_SECRET/AUTH_GOOGLE_ID
cd web/apps/web && npm install && npm run dev     # :3002

docker compose up -d --build openfaithmap-api openfaithmap-web openfaithmap-admin
```

Full stack ports: `3001` openfaithmap-api management/health only (its app port `3000` is
compose-internal, reached only by the two web apps — D-HeadlessTopology) · `3002` openfaithmap-web
· `3004` openfaithmap-admin.

## Go backend conventions (mandatory, not just observed)

- **Errors.** Domain errors are either package-level sentinels (`var ErrXxx = errors.New(...)`) or
  typed structs (`type XxxError struct { Field ... }`) defined in `domain/`, matched at the
  transport boundary with `errors.Is`/`errors.As`. Every module's `transport/errors.go` has one
  `mapErr(err error, c errCtx) error` that switches over these and returns the matching generated
  Conjure error constructor — HTTP status comes from the Conjure contract's declared `code:`, never
  hand-mapped in Go. Wrap non-domain (infra/parsing) errors with `fmt.Errorf("...: %w", err)`;
  return domain errors directly, unwrapped. Canonical example:
  `internal/content/domain/content.go` + `internal/content/transport/errors.go`.
- **Layout.** Every module is `transport/ → application/ → domain/ → adapters/`, domain owns its
  interfaces and imports no framework. Inside `application/`, one file per use-case cluster
  (`blockvalidation.go`, `slugvalidation.go`, ...), not one giant file and not one-file-per-entity.
  `adapters/` holds `repository.go` plus a generated `<module>sql/` sqlc subpackage.
- **Wiring.** One `cmd/openfaithmap-api/register_<module>.go` per module:
  `func register<Module>(ctx, info, deps *Deps) error` builds adapters → application service →
  transport service, registers routes, writes the service back onto `deps.XxxAppSvc` for
  later-registered modules to consume directly (no interface/event bus for cross-module reads —
  direct struct-pointer injection is the convention). Registration order matters and is documented
  inline where a module depends on an earlier one.
- **Naming.** `ID`, never `Id`. Receivers are short and consistent per type (`s *Service`,
  `r *Repository`). Interfaces are named by role (`GrantStore`, `ClosurePort`), not `-er` suffix.
- **Comments.** Dense and reasoning-first, explaining *why* a check exists or a field can be nil —
  not restating what the code already says. Cite `D-<Name>` decisions
  (`docs/architecture/decisions.md`) when a comment's reasoning traces to one — those blocks are
  permanent. **Do not cite milestone IDs (`M14.13`) in new comments** — milestone files get
  archived/renamed as the roadmap moves, so an `M#` reference rots; put the reasoning in prose
  instead.
- **Linting.** `./godelw verify` runs gofmt + golangci-lint (currently default rule set, no custom
  list in `godel/config/golangci-lint-plugin.yml`) + `go test`. The default set is thin for a
  "verify quality before commit" bar — enabling `errcheck`/`gosec`/`revive` is a worthwhile
  follow-up, not done in this pass.

## Frontend conventions (mandatory, both `web/apps/admin` and `web/apps/web`)

- **Props.** Type inline: `function Foo({ x, y }: { x: string; y: number })`. No separate `Props`
  type/interface — this is 100% consistent across both apps today.
- **Forms.** Every form uses `useActionState` with an inline `"use server"` action defined in the
  hosting `page.tsx`, returning a discriminated-union state on failure
  (`{ error: "errorSlug", field: "slug" }`) that the client component surfaces via
  `aria-invalid`/an inline message next to the field — never a `?error=` redirect round trip.
  Canonical example: `web/apps/admin/app/[locale]/admin/sites/[unitId]/documents/new/`.
- **Data fetching.** Never call `fetch()` directly against the backend. Both apps generate a
  Conjure TypeScript SDK (`lib/openfaithmap/generated/**`, never hand-edited — `make sdk-verify`
  fails on drift). Domain wrapper modules (`lib/content.ts`, `lib/registration.ts`, ...) are
  `import "server-only"`, build a per-request client with the session's bearer token, and `unwrap()`
  Conjure errors into a typed `ApiError`. Server Components call these wrappers directly; there is
  no client-side data-fetching library (no SWR/React Query) — client interactivity loads via
  `next/dynamic({ ssr: false })` where needed (e.g. the Leaflet map).
- **TypeScript.** Prefer `type` over `interface` in all hand-written code; `interface` is reserved
  for cases needing declaration merging or generated/SDK code. `strict: true` in both `tsconfig.json`s
  — keep it that way.
- **Styling.** Tailwind v4 + shadcn/Radix (`components/ui/*`, CLI-managed, don't hand-edit
  stylistically). Use the `cn()` helper (`lib/utils.ts`) for conditional/merged class names.
- **Security.** `dangerouslySetInnerHTML` is banned outright (ESLint `no-restricted-syntax`,
  D-PublicSiteCSP) — render structured data as elements, never raw HTML strings. This is enforced,
  not a suggestion.
- **Formatting.** Prettier is configured in both apps (`.prettierrc.json`, added alongside this
  file) — `npm run format` to fix, `npm run format:check` to verify (now part of CI). Settings:
  double quotes, semicolons, trailing commas, 100-char print width.
- **File naming.** kebab-case filenames throughout; one component per file, filename matches the
  primary export.

## Quality gates — required before considering any change done

- **Go changes:** `./godelw verify` must pass (format + lint + test).
- **Frontend changes:** `npm run lint`, `npm run format:check`, and `npm run test` (vitest) must
  all pass in the affected app(s) — all three are now enforced in CI
  (`.github/workflows/ci.yml`'s `web` job), not just a local nicety.
- **Behavior-affecting changes** (not docs-only, not a pure refactor): live-verify against the real
  running stack (`docker compose up -d --build ...`), the same discipline every milestone in the
  archive used — a happy-path `curl`/browser check is not enough on its own; exercise the
  authorization/failure-mode paths specifically, per `docs/development-process.md`'s
  "Stage-board honesty" section.
- **Never claim a change works from reading the code alone.** Run the actual check. This project's
  own history has an incident (M2.1/M2.2, documented in `CONTRIBUTING.md`) where a milestone was
  marked `✅ Verified` on top of a red `main` — check CI/test output, don't assume.

### Tests must cover business logic, not just raise a coverage number

For any change touching business logic (validation, authorization, state transitions, anything
with a "who's allowed to do this" or "what happens at the edge" question), a test suite needs:

- **A positive case** — the expected-success path.
- **A negative case** — denied/rejected, and for anything authorization-shaped, both "no grant at
  all" and "wrong scope" (e.g. a role granted on a different unit) as separate cases. Mirror
  `internal/content/content_integration_test.go`'s pattern: same-tenant allowed, cross-tenant
  denied, wrong-role denied, as distinct assertions.
- **A boundary case, where one exists** — e.g. `internal/moderation/domain/rules_test.go`'s
  `CanReverse`: exactly at the grace-period cutoff is reversible, one unit past it is not.

**What does not count on its own:** a test that only asserts a mock was called
(`toHaveBeenCalled*` with no accompanying behavioral assertion), a component "renders without
crashing" with no interaction, or a snapshot test. These can accompany real assertions but never
substitute for them — this codebase has no snapshot tests today; keep it that way.

Go integration tests run against a **real Postgres** (no mocks) — `internal/*/​*_integration_test.go`,
skipped via `t.Skip` if `DATABASE_URL` is unset. Because `go test ./...` races sibling packages
against the same live database, scope any count-based assertion narrowly (by `actor_person_id`,
by `(action, targetKind, targetID)`) — **never** a whole-table `COUNT(*)`, which will flake against
concurrent test packages.

## Known gotchas

- **`docker compose up -d` alone does not rebuild.** After *any* code edit (Go or TypeScript),
  `--build` is mandatory — `up -d` silently keeps the previous image. This has cost real debugging
  time before.
- **Port 3000 (the API's app port) is not host-published.** Reach it via
  `docker exec open-faith-map-openfaithmap-admin-1 wget ... https://openfaithmap-api:3000/...`
  (BusyBox wget, GET only) or a throwaway `curlimages/curl` on the compose network. Port 3001 is
  the management port only (health/readiness), not a real API route.
- **`*.localhost` subdomains work with no DNS setup.** Browsers resolve them to loopback —
  `http://grace.localhost:3002/` reaches a real tenant site locally with no hosts-file edit.
- **The rate limiter shares one bucket per container IP.** `internal/platform/ratelimit` is ~5
  req/min per client IP per endpoint; every server-to-server call from `openfaithmap-web` shares
  that container's own IP, so a burst of page loads (including your own `curl` probes just before)
  exhausts it and looks like a generic error. Wait ~60s rather than treating it as a bug.
- **Headless browser verification:** `chromium-cli` isn't installed. Use `playwright-core` in a
  scratch directory with `executablePath: "/usr/bin/google-chrome"`, `headless: true`,
  `--no-sandbox --disable-dev-shm-usage`.
- **sqlc infers `to_char()` as nullable** — cast with `::text` to keep a real non-null string type
  where you know the value can't be null.
- **`proxy.ts` can duplicate the `Host` header** — split on comma defensively when reading it.

## Where to look next

`docs/README.md` names the full reading order (architecture overview → decisions → conventions →
module docs → milestones → development process). `docs/architecture/conventions.md` covers
schema/layering/authorization conventions inherited from go-oikumenea or specific to this repo —
read it alongside this file, it's the architectural complement to the code-style rules above.
