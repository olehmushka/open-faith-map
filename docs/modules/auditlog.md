# Module: auditlog

> Reads: [glossary](../glossary.md) · [architecture/conventions](../architecture/conventions.md)
> Table prefix: `openfaithmap.identity_audit_log`

## Purpose

The shared, append-only mutation ledger every module with a real write path logs into. Built at
M11.2 for `internal/core`'s own super-admin screens (units, role assignments, sessions, accounts,
invites, API keys); M15 (`DS-OFM-15`) extended its consumer list to `internal/content`
(site/nav/catalog writes), `internal/registration` (congregation-approval unit creation and role
grant), and `internal/congregationimport` (taxon/jurisdiction alias creation and
candidate-approval unit creation). It is a primitive layer — no permission logic of its own, no
Conjure transport — the same tier as `internal/directory`/`internal/authz`/`internal/religion`,
consumed directly by whichever module performs a write worth recording.

`moderation_actions` (`migrations/0004_moderation.sql`) is this platform's other append-only
ledger, older and narrower in scope (moderator decisions only) — the two are not merged. See
[moderation.md](moderation.md)'s own Purpose section for why: `moderation_actions` predates this
module and has its own `report_id`/`reverses_action_id` shape a general mutation ledger doesn't
need.

## Entities & aggregates

- **Entry** — one row per mutation: who did it (`actor_person_id`), what (`action`), to what
  (`target_kind`/`target_id`), and a curated before/after snapshot. Immutable once written.

## Data model

Conventions per [conventions.md](../architecture/conventions.md).

**`identity_audit_log`** (`migrations/0016_core_audit.sql`)
- `id` PK (RID, service 1 / kind 1 / type 4)
- `actor_person_id UUID REFERENCES identity_persons(id) ON DELETE SET NULL` — nullable so a later
  person deletion never cascades away the record of what they did.
- `action TEXT NOT NULL` — free text, no DB `CHECK`. Each consumer module owns a private Go
  `const` block naming its own actions (e.g. `internal/content/application/audit.go`); two modules
  performing the *same* underlying primitive call (e.g. `internal/registration` and
  `internal/congregationimport` both calling `religion.CreateChildOrg`) deliberately use the same
  literal string (`"CREATE_CHILD_ORG"`) so the audit viewer can filter across modules.
- `target_kind TEXT NOT NULL`, `target_id TEXT NOT NULL` — an opaque discriminator+ref pair, same
  shape `moderation_reports.target_kind`/`target_ref` already established, since targets span
  units, role assignments, sites, block types, patterns, taxon/jurisdiction aliases, and more.
- `before JSONB`, `after JSONB` — a **curated map per call site, never a full-row diff**. `nil`
  marshals to SQL `NULL` (a legitimate value: `before` is `nil` on a create, `after` is `nil` on a
  delete).
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Append-only: `identity_audit_log_reject_mutation` (`BEFORE UPDATE OR DELETE`, reusing
  `openfaithmap.reject_mutation()` from `0004_moderation.sql`) rejects both.
- Indexes: `(created_at DESC, id DESC)`, `(actor_person_id, created_at DESC)`,
  `(target_kind, target_id, created_at DESC)`.

## Conjure API surface

None of its own. `internal/auditlog` has no `transport` package — it is a pure in-process
primitive, exposed to callers only through `CoreSuperAdminService.listAuditLog`
(`internal/core/transport`), which reads via `Service.List`.

## Dependencies

- **Calls:** nothing — self-contained, its own store only (`internal/auditlog/adapters`).
- **Called by:**
  - `internal/core/application` — the original consumer (M11.2): units, role assignments, instance-
    admin grants, sessions, accounts, invites, API keys, person merges.
  - `internal/content/application` (M15) — `CreateSite`, `UpdateSiteTheme`, `UpdateSiteChrome`,
    `PutNavItems`, and the M14.13 catalog CRUD (`CreateBlockType`, `UpdateBlockType`,
    `CreatePattern`, `UpdatePattern`, `DeletePattern`). Document/block content itself
    (`PutBlocks`, `TransitionDocument`) is **not** covered here — it already has its own trail via
    M14.6's `content_document_revisions`.
  - `internal/registration/application` (M15) — `ensureUnit` (the congregation unit a registration
    approval creates) and `ensureGrant` (the congregation-admin role grant an approval performs).
  - `internal/congregationimport/application` (M15) — `CreateTaxonAlias`, `CreateJurisdictionAlias`,
    and `ensureUnit` (the congregation unit a candidate approval creates). **Not** covered:
    `jurisdictionsync.go`'s `RunJurisdictionSync`/`ensureJurisdictionUnit` — see Open seams.

## Authorization touchpoints

`Record` enforces exactly one thing: `ctx` must carry a resolved subject
(`authz.SubjectFromContext`), hard-failing with `authzdomain.ErrPermissionDenied` otherwise — a
missing subject means a bug in the calling mutation path (every caller above is already reached
only through its own write-authority check), and a log entry with no actor would be worse than no
log entry at all. The write authorization itself (`content.manage`, `religionorg.manage`, the
registration/congregationimport operator gate, …) is each caller's own job, already checked before
`Record` is ever reached.

## Invariants

- Append-only, DB-enforced (`reject_mutation()`), not just an application convention.
- `before`/`after` are curated, hand-picked field sets per call site — never a full-row diff of the
  underlying table. Keeps the ledger readable and avoids leaking columns nothing should care about.
- A mutation that already has its own dedicated audit trail (content's document revisions) is not
  double-logged here.
- `Record` never silently drops a write it was asked to log — a missing subject is a hard failure
  for the caller, not a swallowed no-op.

## Open seams

- **`GrantUnitRole`'s conflict-as-success path gives no created-vs-resumed signal.**
  `internal/authz/adapters/repository.go`'s `InsertRoleAssignment` catches a `23505` conflict on
  `authz_role_assignments_active_idx` and returns the *pre-existing* row's id on both a genuine
  create and a resumed retry — so both `internal/core`'s and `internal/registration`'s
  `auditLog.Record` calls after `GrantUnitRole` may write a second, redundant `GRANT_UNIT_ROLE` row
  on a resumed retry. Low severity (append-only, no state corruption, the duplicate row is
  truthful) but real. Fixing it properly means giving `InsertRoleAssignment` a real `created bool`
  — worth doing once, fixing both call sites at once, not worth a one-off workaround in either.
- **`congregationimport.RunJurisdictionSync` stays unattributed, coupled to `DS-OFM-16`.** Its
  jurisdiction-tier unit creation runs under `authz.SystemContext`, deliberately, to keep automated
  sync unattributed to whichever operator triggered it (see that file's own doc comment and
  [congregationimport.md](congregationimport.md)). Wiring `Record` into that path is exactly what
  `DS-OFM-16` ("background writes are unattributable") already names as future work, not solved by
  M15.
