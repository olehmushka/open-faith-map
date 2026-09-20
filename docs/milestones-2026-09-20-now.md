# Milestones (2026-09-20 – now)

The architecture sequenced into buildable, dependency-ordered milestones. A roadmap, not binding —
[`architecture/decisions.md`](architecture/decisions.md) governs *what*, this governs *in what
order*. Gate definitions are in [`development-process.md`](development-process.md).

This is the **current, authoritative** milestones file. Everything through M15 is done and archived
in [`milestones-2026-08-20-2026-09-20.md`](milestones-2026-08-20-2026-09-20.md) (which itself
supersedes the earlier [`milestones-2026-08-07-2026-08-26.md`](milestones-2026-08-07-2026-08-26.md)).
This file starts from what was still open at the close of that window.

## Status

**M0–M15 are done** (Stage = built, committed, live-verified) — see the two archive files above for
that full history. Only two things carried forward from that window:

- **M14.18 · Deployment wiring** stays `🔶` — its config was written and locally validated
  2026-09-04, but it needs a real registered apex domain and DNS-provider API token to actually go
  live (**U14**). Nothing to build; it clears the moment U14 is resolved.
- **U14** and **U15** (below) remain open, carried forward unchanged from the archive.

Two new milestones are scoped as of 2026-09-20:

- **M16 · CLAUDE.md and formalized code-style conventions** — built (2026-09-20). A repo-onboarding
  doc, built interactively with the owner rather than inferred from docs alone, codifying Go and
  frontend conventions already observed in the codebase plus two real gaps it closed along the way:
  Prettier now formats both frontend apps, and `vitest run` is now a required CI step (previously a
  script that existed but nothing ever invoked in CI). Not docs-only, in the end.
- **M17 · Admin entity pickers, replacing raw-UUID inputs** — every admin screen that currently asks
  an operator to hand-type a UUID (role grants, person merge, unit reparenting, vouching,
  explain-access, audit-log filters, congregation-import aliases — ~13 fields across 11 files) gets a
  real search-and-pick control instead. Decided/Designed this session; Backend/Migrated/UI are a
  follow-up build pass.

A **Candidate milestones** section below lists items already tracked in the doc set
(`open-questions.md`, the unresolved-unknowns table) that aren't yet formally scheduled — surfaced
here for the owner to prioritize, not started.

## Unresolved unknowns — read this before building anything

Carried forward unchanged from
[the archive's own table](milestones-2026-08-20-2026-09-20.md#unresolved-unknowns--read-this-before-building-anything).

| # | The unknown | Where it bites | Who resolves it |
|---|---|---|---|
| **U14** | **No apex domain is registered and no DNS-provider API token exists.** A wildcard certificate for `*.<apex>` can only be issued over the ACME **DNS-01** challenge — HTTP-01 cannot issue wildcards. D-ProductionDeployment deliberately left the VM/DNS provider undecided; `D-TenantSubdomains` now constrains that choice for the first time (the provider must expose a DNS API Caddy has a module for). | M14.18 only. Every other M14 milestone is verifiable locally against `*.localhost`, which browsers resolve to loopback with no DNS at all. | The owner, by registering a domain and picking a DNS provider. M14.18 carries `🔶` until then — the same honest gate M1.2/M2 already use for the Google OAuth redirect URI. |
| **U15** | **Google Drive hotlink reliability at volume is unmeasured.** `D-ExternalMediaOnly` makes congregations host their own images on Drive/Dropbox/OneDrive. Direct-content URLs for these hosts are undocumented, have been changed by their vendors before, and are throttled under load — none of which we can measure before real congregations use it. | M14.3's normalizer, and every `image`/`gallery` block on every public site thereafter. A vendor-side change breaks images platform-wide at once. | Only real traffic. M14.3 mitigates rather than resolves: the original URL is preserved alongside the normalized one, so a normalizer fix is a re-derivation, not a data-loss event. Escalation path is the first-party `media` module (`DS-OFM-17`). |

## Stage board

**Gate legend.** ✅ passed · ⬜ not started · ➖ not applicable · 🔶 **passed once, now blocked on a
named dependency** — always named in that milestone's prose; 🔶 without a named blocker is just ⬜.
`Verified` additionally requires CI green on `main` — see
[development-process.md](development-process.md).

| # | Decided | Designed | Backend | Migrated | UI | Verified | Stage |
|---|---|---|---|---|---|---|---|
| M14.18 · Deployment wiring | ✅ | ✅ | ➖ | ➖ | ➖ | 🔶 | **Config written (2026-09-04); still blocked on U14: a registered apex domain + a DNS-provider API token.** `deploy/caddy/Caddyfile` with the DNS-01 wildcard block (wildcards cannot be issued over HTTP-01 — a new, real constraint on the provider choice D-ProductionDeployment deliberately left open), HSTS with `includeSubDomains`, per-tenant read rate limiting, plus `docker-compose.prod.yml` and `.env.prod.example` — all validated locally (custom Caddy image builds, both third-party modules load, `caddy validate` accepts the config). Confirms the backup story is unchanged: no blobs to back up, because there are no uploads. Records the `openfaithmap-sites` extraction as the named Phase 2 trigger. `🔶` stays until a real domain actually serves — see `deploy/README.md`. |
| M16 · CLAUDE.md and formalized code-style conventions | ✅ | ✅ | ➖ | ➖ | ➖ | ⬜ | **Built (2026-09-20).** Root `CLAUDE.md`, built through an interactive Q&A with the owner rather than inferred — Go and frontend code-style conventions formalized from what the codebase already does (error handling, file layout, DI wiring, naming, comments; React prop typing, form pattern, data-fetching, TypeScript conventions), plus explicit quality-gate and business-logic-test-coverage rules. Two real gaps closed along the way, decided with the owner rather than assumed: Prettier added to both frontend apps (`.prettierrc.json`, `format`/`format:check` scripts, existing code reformatted) and `npm run test` (vitest) wired into `.github/workflows/ci.yml`'s `web` job, which previously never ran it. `Verified` awaits CI green on `main`. |
| M17 · Admin entity pickers | ✅ | ✅ | ⬜ | ➖ | ⬜ | ⬜ | **Decided/Designed (2026-09-20).** See its own detail section below for the full target list and component design. Backend/UI build is a follow-up pass. |

## Per-milestone detail

### M14.18 · Deployment wiring

**Config written (2026-09-04); still 🔶 blocked on U14: a registered apex domain and a
DNS-provider API token** — neither exists, and no VM is provisioned. `deploy/caddy/Caddyfile`
(DNS-01 wildcard block, HSTS with `includeSubDomains`, a per-tenant-host `rate_limit` zone),
`deploy/caddy/Dockerfile` (an `xcaddy` build carrying `caddy-dns/cloudflare` — a concrete,
swappable example provider, not a decision — and `caddy-ratelimit`, neither shipped by stock
Caddy), `docker-compose.prod.yml` (the override D-ProductionDeployment names, layered on top of
the unmodified `docker-compose.yml`), and `.env.prod.example` all exist and validate locally: the
custom image builds, `caddy list-modules` shows both third-party modules registered, and `caddy
validate` accepts the Caddyfile against dummy env values. See `deploy/README.md`. None of this
constitutes going live — no DNS-01 challenge has ever run for real, because there is no real
domain to run one against. Everything else in M14 is verifiable locally against `*.localhost`
with no DNS at all, which is why this milestone is last and alone.

A Caddyfile with the DNS-01 wildcard block for `*.<apex>`. This is the one genuinely new
infrastructure constraint the arc introduces: **ACME cannot issue a wildcard certificate over
HTTP-01**, so the DNS provider must expose an API Caddy has a module for — narrowing a provider
choice D-ProductionDeployment deliberately left open. The wildcard is what avoids Let's Encrypt's
per-account new-order ceiling entirely, which per-subdomain issuance would run into as soon as
`congregationimport` provisions congregations in bulk.

Also: the wildcard DNS record, HSTS with `includeSubDomains`, per-tenant read rate limiting, and a
restatement that the backup story is unchanged — there are no blobs to back up, because
`D-ExternalMediaOnly` means there are no uploads. Records the `openfaithmap-sites` extraction as the
named trigger for the owner's Phase 2.

**Acceptance criteria.** A real congregation subdomain serves over HTTPS with a wildcard
certificate. HSTS present. The reserved-slug blocklist holds against a real registration attempt.
`🔶` clears only when a real domain is serving — not when the Caddyfile is written.

### M16 · CLAUDE.md and formalized code-style conventions

**Built (2026-09-20).** Depends on nothing. A root `CLAUDE.md` is the file this and every future
Claude Code session reads automatically — the highest-leverage place to record what this repo's
own docs (`README.md`, `docs/architecture/*`, `docs/modules/*`, `development-process.md`) don't
state on their own: day-to-day code style, quality-gate rules, and gotchas worth repeating so they
aren't rediscovered by trial and error each session.

**docs/architecture/conventions.md covers schema/layering/authorization conventions; it does not
cover day-to-day code style** (error-handling shape, file layout, naming, form patterns, ...) —
that was the actual gap this milestone closes.

**Built by interactive Q&A with the owner**, not inferred, precisely because the value of a
`CLAUDE.md` is the stuff that isn't already written down anywhere else. Three rounds of questions,
each seeded by a factual survey of the actual codebase (not assumptions) across Go backend,
frontend, and testing:

- **Go backend** — codified as *mandatory*, not just observed: the sentinel/typed-error +
  per-module `mapErr` pattern; one-file-per-use-case `application/` layout plus the
  `register_<module>.go` DI wiring pattern; `ID` never `Id`, role-named interfaces; comments may
  cite `D-<Name>` decisions (permanent) but must not cite milestone IDs (`M14.13`) going forward,
  since milestone files get archived/renamed as the roadmap moves (as this very session just did).
  Flagged, not changed: golangci-lint's config is currently defaults-only — `errcheck`/`gosec`/
  `revile` are worth adding in a follow-up.
- **Frontend** — codified: inline prop typing (no separate `Props` type), `useActionState` +
  inline server action mandatory for every form (not just secret-bearing ones), `type` over
  `interface` in hand-written code. Closed a real gap rather than just documenting one: Prettier
  didn't exist in either app before this milestone — added now (`.prettierrc.json`,
  `.prettierignore` mirroring each app's ESLint generated-code exclusion, `format`/`format:check`
  scripts), and both apps' existing code was reformatted to match (lint/test/build re-verified
  clean after).
- **Quality gates** — `./godelw verify` (Go) and `npm run lint && npm run format:check && npm run
  test && npm run build` (frontend) are the required pre-commit bar, now enforced in CI in full for
  the first time: `.github/workflows/ci.yml`'s `web` job previously ran `lint`/`build` only —
  `vitest run` existed as a `package.json` script but nothing ever invoked it in CI, so a broken
  frontend test suite could land on `main` unnoticed. Added `format:check` and `test` steps to
  close that gap. A concrete, checkable business-logic-test-coverage rule was written (positive +
  negative + boundary cases required; mock-call-only/render-only/snapshot assertions don't count
  alone), matching patterns already exemplified by `content_integration_test.go`'s
  authorization triples and `moderation/domain/rules_test.go`'s exact-boundary grace-period case.

**Acceptance criteria — met.** `CLAUDE.md` exists at the repo root and reflects the owner's own
answers, not just a summary of existing docs. `npm run format:check` passes clean in both apps.
`npm run lint`, `npm run test`, and `npm run build` re-verified passing in both apps after the
Prettier reformat. `.github/workflows/ci.yml`'s `web` job runs `format:check` and `test` for both
`admin` and `web`.

### M17 · Admin entity pickers, replacing raw-UUID inputs

**Decided/Designed (2026-09-20).** Depends on nothing structurally; touches only
`web/apps/admin`. No admin screen should ask an operator to hand-type a UUID for an entity
reference — a discovery pass across the admin app found ~13 such fields across 11 files, all
plain `<Input>` text fields bound to a person/unit/taxon id, with no search or picker anywhere
except one bespoke case.

**Design.** Generalize
`web/apps/admin/app/[locale]/admin/congregation-import/jurisdiction-field.tsx`
(`JurisdictionField` — today a one-off: a text query box + Search button + a result list of
buttons that set a hidden ID input, plus an inline "create new unit" dialog) into a reusable
`EntityPicker`/`SearchSelect` component, parameterized by entity type (person / unit / taxon) and
its search endpoint. Built on the `cmdk`-based primitives already in
`web/apps/admin/components/ui/command.tsx` (today used only by `command-palette.tsx`'s global
nav search) — no new UI dependency needed. Returns a hidden ID field the same way
`JurisdictionField` already does, so every consuming form's submit path is unchanged; this is a
drop-in replacement for the `<Input>`, not a form-shape change.

**Conversion targets** (every raw-UUID `<Input>` found; filter-only fields get the picker too, for
consistency):

- `(super-admin)/role-grants/page.tsx` — unit-lookup input, person-link input
- `(super-admin)/people/[personId]/page.tsx` — grant-role unit input
- `(super-admin)/people/[personId]/merge/page.tsx` — duplicate-person input
- `(super-admin)/units/[unitId]/page.tsx` — reparent's new-parent-unit input
- `registrations/page.tsx` — parent-unit input
- `registrations/reparent/page.tsx` and `registrations/reparent/reparent-list.tsx` —
  new-parent-unit inputs
- `registrations/request-list.tsx` — jurisdiction-unit filter input
- `congregation-import/aliases/page.tsx` — taxon input, jurisdiction-unit input
- `congregation-import/jurisdiction-field.tsx`'s own "create missing unit" dialog — parent-unit
  input (the one place already closest to the target UX, converted for consistency)
- `vouching/new/page.tsx` — claimant-person, congregation-unit, guarantor-congregation-unit inputs
- `vouching/page.tsx` — guarantor-person input
- `(super-admin)/explain-access/page.tsx` — subject-person input, unit input
- `(super-admin)/audit-log/page.tsx` — actor-person filter input

**Acceptance criteria.** No admin screen has a freehand text `<Input>` bound to a person/unit/taxon
id. Every target above uses the generalized picker. Search-by-name (not just exact id) works for
every entity type the picker supports. Existing form submit paths (hidden id field) are unchanged,
so no backend endpoint needs to change shape.

**Not yet built.** This session recorded the Decided/Designed columns only. Backend work is
`⬜` pending confirmation no new search endpoint is needed (existing list endpoints may already
support a name filter — to confirm during the build pass); UI work is the ~13-site conversion
above.

## Candidate milestones (not yet scheduled)

Mined from the existing doc set rather than newly invented — each already has an owner-facing
write-up; what's missing is a milestone number and a build slot. Listed for prioritization, not
started.

- **DS-OFM-18 — `GrantUnitRole` gives no created-vs-resumed signal.** Small, well-scoped fix:
  `internal/authz/adapters/repository.go`'s `InsertRoleAssignment` needs a real `created bool` (or
  equivalent) so `core`'s and `registration`'s `auditLog.Record` call sites can skip logging on a
  resumed no-op retry, instead of possibly double-logging `GRANT_UNIT_ROLE`. Low severity today
  (append-only, no state corruption) but cheap to close. See
  [open-questions.md](open-questions.md).
- **DS-OFM-5 — Full-text content search.** Searching page/post bodies (not just location) has no
  owner yet. Real scoping work — index choice, what's searchable, ranking — not started. See
  [content.md](modules/content.md#open-seams).
- **DS-OFM-17 — First-party media module.** The escalation path named by `D-ExternalMediaOnly`
  once Google Drive/Dropbox/OneDrive hotlinking (**U15**) proves unreliable at real volume — needs
  real traffic data before it's worth scoping, not a code gap today. See
  [content.md](modules/content.md#open-seams).
- **The owner's named Phase 2 — `openfaithmap-sites` extraction.** Splitting the tenant-site
  rendering path out of `web/apps/web` into its own app/process, for blast-radius isolation, once
  the VM budget allows it (D-ProductionDeployment's Phase 1/Phase 2 split, recorded at M14.0).
  Trigger is budget and scale, not a design gap — unscheduled by design.
- **U14 / U15** (above) — not code work; U14 needs the owner to register a domain and pick a DNS
  provider, U15 needs real traffic.
