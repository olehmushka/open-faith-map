# Milestones (2026-08-26 – now)

The architecture sequenced into buildable, dependency-ordered milestones. A roadmap, not binding —
[`architecture/decisions.md`](architecture/decisions.md) governs *what*, this governs *in what
order*. Gate definitions are in [`development-process.md`](development-process.md).

## Status

**M0–M13.6 are done** (no row had unbuilt Backend/Migrated/UI work as of 2026-08-26) — see
[`milestones-2026-08-07-2026-08-26.md`](milestones-2026-08-07-2026-08-26.md) for that full history.
This file starts empty and is where the next milestone gets planned and built.

## Unresolved unknowns — read this before building anything

Every place the doc set currently says "we don't actually know," carried forward from the archive
(only the still-open items at the time — already-resolved ones stayed behind there). Detail lives
where the third column points; this table exists so nothing is hidden, not to duplicate it.

**Empty as of 2026-08-26 — every item carried into this file has since been resolved:**

- **Group 1** (U2, U3 — must be measured against a real instance): both were actually resolved back
  in 2026-08-09/2026-08-10 (M2.5 and M4.1 each measured their own question and built on the answer)
  but were never struck in the original table — caught and corrected in the archive while splitting
  this file.
- **Group 2** (U7 — deferred decisions): resolved 2026-08-26. Cross-module FKs formalized as
  permitted in `architecture/conventions.md` (every module already shares one Postgres
  instance/schema since D-SharedDatabase/M10; the precedent already existed in the schema).
- **Group 3** (U11, U12 — contradictions/orphans): both resolved 2026-08-26. U11:
  `churchSiteTypeID` (both the `registration` and `congregationimport` copies) now fails loudly
  instead of silently falling back to the first available site type. U12: the two settings still
  bypassing `config.Install` (`DATABASE_URL`, `GOOGLE_OAUTH_CLIENT_ID`) are now real, schema-
  validated fields, matching `Environment`'s own M10.2 precedent.

Full detail for all five, including what was actually changed, is in the archive's own
[unresolved-unknowns table](milestones-2026-08-07-2026-08-26.md#unresolved-unknowns--read-this-before-building-anything)
(struck entries with a **Resolved (date)** note) — this file states none as currently open, and adds
new ones here as they surface.

## Stage board

**Gate legend.** ✅ passed · ⬜ not started · ➖ not applicable · 🔶 **passed once, now blocked on a
named dependency** — always named in that milestone's prose; 🔶 without a named blocker is just ⬜.
`Verified` additionally requires CI green on `main` — see
[development-process.md](development-process.md).

| # | Decided | Designed | Backend | Migrated | UI | Verified | Stage |
|---|---|---|---|---|---|---|---|

*(empty — no milestone planned yet)*

## Per-milestone detail

*(empty — the first entry lands here once the next milestone is scoped and built)*
