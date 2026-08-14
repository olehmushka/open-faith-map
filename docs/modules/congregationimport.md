# Module: congregationimport

## Purpose

Stages congregations from external sources — government open data, OpenStreetMap, denomination
locator sites — for a registration operator to review before any of them becomes a real,
publicly-discoverable go-oikumenea Unit. Resolves `DS-OFM-10` (`docs/open-questions.md`), the
long-deferred "scraped-church-data importer" gap: `registration`'s own submit/approve flow cannot
be reused for this (see [D-CongregationImport](../architecture/decisions.md) and M2.2's own
rejected bulk-import CLI design) — there is no real human submitter behind a scraped row, and
`registration`'s entire contract assumes one.

## The row-attribution finding

`registration.SubmitRegistrationRequest` has no contact-person field; `submittedByPersonId` is
always resolved server-side from the caller's own bearer token. A connector has no such token to
forward on any individual candidate's behalf. Replaying `registration`'s endpoints from a bulk
process (M2.2's original design) would attribute — and grant `congregation-admin` on — every
imported row to whichever operator ran the import, not the congregation's real contact. That
design was built once and rejected; see `D-BulkImport`'s Correction.

This module takes a different path: an operator reviews and approves a candidate, and the
resulting go-oikumenea Unit is provisioned under the **approving operator's own token** — but
**no `congregation-admin` grant happens at provisioning time.** The congregation is real and
publicly discoverable; it is genuinely admin-less until a real contact claims it. See
D-CongregationImport for the full "why this isn't an on-behalf-of write" reasoning, and its
explicit note that the resulting verified/claimed status model is a **proposal**, not settled.

## Entities & aggregates

- **Connector** (`domain.Connector`) — the Strategy-pattern interface every source implements:
  `Code()`, `Citation()`, `Fetch(ctx, cursor)`, `Normalize(raw)`. One Go package per source under
  `adapters/connectors/`.
- **Run** (`congregationimport_runs`) — one row per triggered connector execution.
- **Candidate** (`congregationimport_candidates`) — one staged row per upstream record, keyed by
  `(source_code, source_record_id)` for idempotent re-scrapes.
- **TaxonAlias** (`congregationimport_taxon_aliases`) — operator-maintained free-text-hint →
  `religion_taxa` RID mapping, optionally scoped per source.
- **JurisdictionAlias** (`congregationimport_jurisdiction_aliases`) — operator-maintained
  free-text-hint → go-oikumenea jurisdiction Unit RID mapping (D-JurisdictionUnits), same shape and
  matching discipline as TaxonAlias, deliberately not merged with it: a taxon alias resolves *what
  denomination*, a jurisdiction alias resolves *which specific diocese/eparchy/synod unit* —
  unrelated ID spaces, and a candidate may need one without the other.
- **CongregationStatus** (`congregationimport_congregation_status`) — the verified/claimed
  overlay, written once a candidate is approved and provisioned. Same shape as
  `vouching_guarantor_status`: a mutable projection keyed by an immutable go-oikumenea entity.

## Data model

See `migrations/0010_congregationimport.sql` for the full DDL. Plain `uuid` PKs; go-oikumenea
references (`taxon_id`, `country_id`, `created_unit_id`, `*_person_rid`) are opaque `TEXT` foreign
values, no cross-schema FK (`conventions.md`). Real in-schema FKs only between this module's own
tables (`candidates.import_run_id → runs`, `congregation_status.import_candidate_id →
candidates`). `migrations/0011_congregationimport_hardening.sql` (production-hardening pass) adds
two composite indexes — `(status, created_at DESC, id DESC)` on `candidates`,
`(source_code, started_at DESC, id DESC)` on `runs` — backing the real keyset pagination fix, the
identical shape `0009_hardening.sql` added for `moderation_reports`/`moderation_appeals` at M7.

`congregationimport_candidates.status` lifecycle:

```
STAGED / NEEDS_TAXON_REVIEW / NEEDS_GEOCODE / POSSIBLE_DUPLICATE
   → (operator edits, then) →  APPROVED  → PROVISIONING → PROVISIONED
   → REJECTED  (operator decision)
   → REJECTED_EXCLUDED  (automatic, D-Exclusions match — never operator-reachable as pending)
```

`PROVISIONING`/`PROVISIONED` require `created_unit_id` set; `REJECTED`/`REJECTED_EXCLUDED` require
`rejection_reason` set — enforced by a `CHECK` constraint, the same decision-shape discipline
`registration_requests` established.

## Pipeline

```
Connector.Fetch → Connector.Normalize → jurisdiction-hint match (alias table, substring, advisory
  only) → taxon-match (alias table, substring, not exact) → [no taxon match: D-Scope Christian-name
  pre-filter, auto-reject on no keyword hit] → [taxon matched: D-Exclusions check, service-principal
  identity] → dedup check (geo-radius against live go-oikumenea sites, service-principal identity)
  → stage → operator review (edit/approve/reject) → on approve: resumable go-oikumenea provisioning
  (operator's own token, no congregation-admin grant, jurisdiction parent only if the operator
  explicitly chose one) → congregation_status overlay row written
```

`application.Service.RunConnector` writes each batch to the staging table as it goes — never
buffers a whole run in memory, since a real source (e.g. ЄДР) is hundreds of thousands of records.

**Taxon matching is substring-based, not exact-match**, deliberately: a connector's `TaxonHint` is
typically a full scraped name (e.g. a legal entity's registered name), not a short keyword, so an
exact match against an alias would essentially never fire. `matchTaxon` checks whether any known
alias (source-scoped first, then global) appears as a substring of the normalized hint.

**Institutional hierarchy (Catholic dioceses/eparchies, Orthodox eparchies/exarchates, Lutheran
synods, Anglican/Episcopal dioceses) is handled as an advisory suggestion, never an inference.**
[D-JurisdictionUnits](../architecture/decisions.md) already decided jurisdiction is an
operator-assigned, non-uniform layer — real for the traditions where it structurally exists,
genuinely absent for independent-polity congregations (Baptist, Pentecostal, many
non-denominational bodies), and never a single canonical tree encoded in schema (Orthodox
jurisdiction in particular is often multiple and parallel even within one country). This module
follows that decision exactly, one layer earlier in the pipeline: a connector may set
`NormalizedCandidate.JurisdictionHint` (free text naming the parish's superior jurisdiction, when
the source carries one), `matchJurisdiction` resolves it against
`congregationimport_jurisdiction_aliases` the same substring-match way `matchTaxon` resolves a
taxon, and a match populates `Candidate.SuggestedJurisdictionUnitID` — **purely informational,
surfaced to the operator, never applied automatically.** `ApproveCandidateRequest.jurisdictionUnitId`
still requires the operator's own explicit choice, exactly as before this feature existed; a
candidate with a matched suggestion the operator ignores provisions under the configured root unit,
identically to a candidate with no suggestion at all. **Live-verified, not just reasoned about**
(`docs/milestones.md`'s M8 detail): a real early-return bug meant jurisdiction matching never ran at
all for a candidate whose *taxon* hint didn't resolve — exactly the candidates most likely to carry
a useful jurisdiction hint (an unaliased denomination keyword next to a resolvable diocese name in
the same legal-entity name) — found by seeding a real jurisdiction alias and observing the
suggestion silently never populate; fixed by moving the jurisdiction-match call ahead of the
taxon-match early return, independent of its outcome. Separately live-confirmed the suggestion is
never auto-applied: approving a candidate with a real (but deliberately fake, unresolvable) matched
jurisdiction-unit suggestion, without passing `jurisdictionUnitId`, produced a unit parented under
the configured root — not the suggested unit — confirmed by a direct `tenant_unit_edges` query
against the real created edge.

**D-Scope pre-filter (source-agnostic, application layer, not per-connector) auto-rejects a
candidate whose taxon hint resolved to nothing at all**, added 2026-08-14 in response to a real,
live problem: `docs/architecture/decisions.md`'s D-Scope declares OpenFaithMap Christian-only, but
a source's own institutional-form filter (e.g. `ua-edr`'s `OPF == "religious organization"`) has no
way to distinguish *which* religion — Muslim and Jewish congregations were being staged into the
review queue identically to Christian ones, confirmed live by direct sampling of a real ЄДР run.
`application/christianfilter.go`'s `isLikelyChristian` is a **positive** keyword match (does the
name look Christian?), deliberately not a blacklist of every non-Christian religion/sect name
variant — open-ended, easy to miss entries for. Placement matters: it only runs inside the
`!matched` branch (taxon hint resolved to nothing), never before — an excluded-denomination record
(e.g. Jehovah's Witnesses, matched via a real taxon alias) is caught downstream by the existing
D-Exclusions check with its own, more specific reason, never reaching this filter at all; running
the Christian filter unconditionally-early would have overwritten that specific reason with the
generic one for no benefit. Reuses `StatusRejectedExcluded` with a distinct `rejection_reason`
string (`"D-Scope: ..."`) — no migration, matching `checkExcluded`'s own existing precedent exactly
(same status, the specific reason lives in the free-text column). ~99% recall is the explicit bar,
not 100%: a keyword miss on a real Christian name is a no-op (falls through to manual review,
unchanged from before this filter existed); a keyword hit on a non-Christian name is also harmless
(under-filtering, today's status quo). **Two real keyword-list bugs found live** sampling a real
30,721-record run, both fixed: the original `"парафія"` entry (parish) didn't match its own
genitive form `"парафії"` — the real form most registered names use — the exact declension trap
the file's own doc comment already warned about for `"церква"`, just not applied consistently on
the first pass; and two near-ubiquitous Orthodox abbreviations (`УПЦ`, `ПЦУ`) plus `єпархія`
(eparchy/diocese) were missing entirely. `christianfilter_test.go` regression-tests both.

**Dedup is geo-radius only, not name+radius** — go-oikumenea's `DiscoverySite` (the `SearchSites`
response shape) carries no name field to compare against, checked directly against the Conjure
struct, not assumed. A candidate within 250m of an already-provisioned site is flagged
`POSSIBLE_DUPLICATE` for the operator's own judgment; this never auto-merges or auto-rejects, so a
same-building-different-congregation false positive costs one extra human decision, not a wrong
automated one.

## Sources (v1)

- **`ua-edr`** — Ukraine's ЄДР (Unified State Register of Legal Entities). Real, verified: the
  Ministry of Justice publishes this as open data at data.gov.ua (dataset
  `03cc1239-3988-4451-aa0d-aadb77448714`, resource `uo.zip`), weekly, free. Filtered by
  `OPF = "Релігійна організація"` (КОПФГ classifier code `825`), matched **case-insensitively** —
  a real correction, found by downloading and scanning the actual export: the live data stores OPF
  as `"РЕЛІГІЙНА ОРГАНІЗАЦІЯ"` (all uppercase), not the classifier resource's own title-case text;
  an exact match matched zero real rows before this fix. **Real constraint: this export has no
  address field at all** (checked against the dataset's own published `uo_schema.zip` XSD — an
  older, now-superseded schema version had one; the current one does not). Every `ua-edr` candidate
  lands in `NEEDS_GEOCODE`; an operator fills location in manually during review. **Verified
  encoding: windows-1251**, checked directly against the real downloaded export's XML prolog (the
  schema XSD's own "UTF-8" declares the XSD file, not the data file).
  Live-tested end-to-end (`docs/milestones.md`'s M8 detail) against a real subset of the actual
  downloaded export. `STAN` values distinguishing active/terminated are not filtered
  on in v1 — every `OPF`-matched record is staged regardless, `STAN`'s raw value visible to the
  operator in `raw_payload`. **`JurisdictionHint` reuses the same NAME string as `TaxonHint`, not a
  separate field** — re-checked against `uo_schema.zip`'s `<SUBJECT>` element: there is no dedicated
  parent-organization/jurisdiction field in this export. What the real data does carry, for
  hierarchical-polity bodies, is the eparchy/diocese/deanery named directly inside the legal `NAME`
  itself (a UGCC parish's registered name routinely reads "...ПАРАФІЯ ... ЛЬВІВСЬКОЇ АРХІЄПАРХІЇ
  УКРАЇНСЬКОЇ ГРЕКО-КАТОЛИЦЬКОЇ ЦЕРКВИ" — the archeparchy is textually present); independent-polity
  registrations simply produce no jurisdiction-alias substring match, correctly.
  - **Two ways to reach the export**, mutually exclusive (`uaedr.New`): `UAEDR_UO_FILE_PATH` (a
    local file, optionally `.zip`, `fetchFile` — stateless, reopens and reskips from scratch on
    every batch) or `UAEDR_SOURCE_URL` (a remote HTTP(S) URL, `fetchHTTP` — stateful, one held-open
    stream across the whole run). The HTTP mode exists for a cheap, memory-constrained cloud VM
    deployment that can't or won't stage the ~326MB compressed export on local disk: the response
    body streams straight through a hand-written single-entry zip-local-file-header parser (no
    `archive/zip`, no `io.ReaderAt` — DEFLATE decodes via `compress/flate`'s own forward-only
    end-of-stream marker, independent of the zip's declared sizes) into the same charset-aware
    `xml.Decoder` the file mode uses. Live-verified against the real data.gov.ua resource
    (2026-08-14): completed a full run with no local file ever written to disk.
  - **A real, serious bug was found and fixed in `fetchFile`'s cursor arithmetic (2026-08-14)**:
    it returned `cursorOf(skip + seen)`, double-counting the `skip` prefix on every call after the
    first (`seen` already includes it, since each reopened pass re-decodes from byte zero). The
    error compounds across calls — each returned cursor carries a growing extra copy of the
    previous `skip` — until the inflated value races past the file's true record count and
    `dec.Token()` hits a real `io.EOF` far too early, indistinguishable from a genuinely complete
    run (`SUCCEEDED`, not `FAILED`) unless independently cross-checked. **This is why M8's original
    "full-scale" verification reported only 3,000 matched candidates** (see `docs/milestones.md`'s
    corrected M8 entry) — the true figure, confirmed three independent ways (a plain
    `unzip | iconv | grep` count, and two separate live runs of the fixed HTTP-streaming path) is
    **30,721**. Fixed (`cursorOf(seen)`); regression-tested
    (`connector_test.go`'s `TestFetchFileMultiBatchResume`, a fixture large enough to force more
    than one batch — the bug was invisible at any scale under 500 matches, since `skip=0` on the
    first call hides it completely).
  - **A D-Scope pre-filter now runs for every source** (not just `ua-edr`) — see `application/
    christianfilter.go` below; `ua-edr`'s own OPF filter only means "any religion," not "Christian."
- Real candidates identified but **not yet built**: Brazil (CNPJ/Receita Federal open data, legal
  nature code `322-0` = "Organização Religiosa"), Argentina (Registro Nacional de Cultos,
  datos.gob.ar — excludes the Catholic Church by law), OpenStreetMap (Overpass API,
  `amenity=place_of_worship`, covers all target countries with zero additional per-country
  research). Uruguay/Paraguay/Colombia/Chile — no equivalent registry confirmed yet; check before
  building, don't assume one exists.

## Conjure API surface

`api/congregationimport.conjure.yml` — one authenticated service (`CongregationImportService`,
`default-auth: header`), like `vouching`: no genuinely-anonymous endpoint, since a connector run is
always operator-triggered. `runConnector`/`listRuns`/`getRun`, `listCandidates`/`getCandidate`,
`editCandidate`/`approveCandidate`/`rejectCandidate`, plus (production-hardening pass)
`listTaxonAliases`/`createTaxonAlias`/`listJurisdictionAliases`/`createJurisdictionAlias`. See the
file for full docs per endpoint.

`listRuns`/`listCandidates` have real keyset `(createdAt, id)` pagination as of the
production-hardening pass — mirrors `moderation`'s own M7 fix
(`internal/moderation/transport/cursor.go`) byte-for-byte: a malformed/tampered `pageToken` returns
`CongregationImport:InvalidPageToken`, never silently reinterpreted as page 1.
`listTaxonAliases`/`listJurisdictionAliases` are deliberately **not** paginated — both lists stay
small and operator-curated (`ListAliasesForMatching`'s own reasoning), loaded in full.

## Known limitations (as-built)

- ~~`ApproveCandidate` under a genuinely non-admin `registration-operator` identity fails on
  go-oikumenea's own RLS~~ — **fixed and live-verified (2026-08-13).**
  [go-oikumenea#36](https://github.com/olehmushka/go-oikumenea/issues/36) (root-caused in the
  production-hardening pass, 2026-08-12: `tenant_units_reach`'s person-shaped arm required
  `tenant_unit_closure` to already contain a brand-new unit at its own first INSERT — impossible by
  construction, since `CreateChildOrg` inserted the unit before `AddEdge` populated closure, in a
  separate, later transaction) was fixed upstream by seeding closure before the INSERT
  (go-oikumenea commit `02a1c6f`, released as image tag `0.0.4`, issue closed). Bumped
  `docker-compose.yml`'s `oikumenea-app` pin to `0.0.4` and re-ran the exact repro from a genuinely
  non-admin `registration-operator` identity (`scripts/mint-local-token`, not the instance-admin,
  which bypasses RLS and PDP checks entirely): `ApproveCandidate` now returns `PROVISIONED`, a real
  `tenant_units` row is created, and the admin-less-provisioning invariant still holds (zero rows in
  `authz_role_assignments` for the new unit) — confirmed directly in Postgres, not just by the
  200 response.
  - **The RLS fix unmasked two further, real, open-faith-map-side gaps**, found live in the same
    session, immediately behind it: `registration-operator`'s role (`scripts/bootstrap-registration-
    org`) was missing `religion.read` (needed by `ensureSite`'s `ListUnitSites` resumability check,
    unit-scoped `pep.Require`) and `location.create` (needed by `ensureSite`'s `CreateLocation` call,
    instance-wide `pep.RequireAnywhere`) — both fixed by adding the two permissions to the role
    definition and re-running the (idempotent) bootstrap script. Neither was ever reachable before
    this session: every prior live proof of approve used the instance-admin identity, which bypasses
    PDP checks entirely, so these gaps sat directly behind the RLS wall, invisible until it came down.
  - Confirmed **not specific to this module**: `registration.Approve`, which shares the identical
    `ensureSite` pattern (`registration.ensureSite` — this module's own is an intentional near-copy),
    was live-verified the same way under the same non-admin identity — submit → approve — real unit
    created, and (unlike this module) the real `congregation-admin` unit-scoped grant to the submitter
    confirmed in `authz_role_assignments`, the one place the two modules' provisioning intentionally
    differ.
- **No geocoder implementation** — `application.Geocoder` is an interface with zero real
  implementations in v1. Every candidate missing coordinates needs a human to fill them in during
  review. A real implementation (e.g. Nominatim, under the same per-host rate-limit discipline the
  HTML connectors use) is a plug-in to add later without a schema change.
- **No claim flow** — `congregation_status.claimed_by_person_rid`/`claimed_at` exist in the schema
  but nothing writes them. `vouching.md` already names this gap ("the eventual real caller ... the
  web-admin.md congregation-claim flow") for an unrelated reason; this module's row-attribution
  problem and that gap are the same missing piece, deliberately left for a real, separate decision
  rather than improvised here.
- **HTML connectors are not built yet** — only `ua-edr` (a structured open-data source) exists.
  `adapters/connectors/html/base` (robots.txt check, per-host rate limiting, citation-row
  enforcement) is designed but not implemented; no denomination-locator site has been individually
  robots.txt/ToS-checked.
- **No `ListTaxonAliases`/`CreateTaxonAlias` on the Conjure surface** — the store methods exist
  (`adapters.Store.CreateTaxonAlias`), the endpoint doesn't. Follow-up.

## Dependencies

- `internal/registration/domain.ExcludedTaxonCodes` — imported directly for the D-Exclusions check,
  never forked (same discipline `moderation`'s own copy already established).
- `internal/coreintegration` — the service-principal client, for the D-Exclusions taxon read and
  the dedup `SearchSites` call. Provisioning writes never use this — always the operator's own
  forwarded token.
- go-oikumenea: `Religion.GetTaxon`, `Religion.SearchSites`, `Religion.CreateChildOrg`,
  `Religion.ListUnitSites`, `Religion.ListSiteTypes`, `Religion.CreateSite`, `Location.CreateLocation`,
  `Authorization.Authorize`, `IdentityFederation.Whoami`.

## Authorization touchpoints

Every operator-facing endpoint (`editCandidate`/`approveCandidate`/`rejectCandidate`) uses the
same target-scoped capability check every other module hand-duplicates: does the caller hold
`religionorg.manage` on `Config.RootUnitID` specifically, verified live against go-oikumenea's
`Authorize`? `runConnector`/list/get endpoints resolve the caller's identity (`whoami`) but apply
no further gate at the application layer — they make no go-oikumenea write, only read-only
service-principal calls a connector run needs regardless of who triggered it.

## Invariants

- **No `congregation-admin` grant at provisioning time.** Ever. This is the load-bearing
  difference from `registration.Approve`, not an oversight — see D-CongregationImport.
- **The D-Exclusions list is imported, never forked.** `application.checkExcluded` uses
  `registrationdomain.ExcludedTaxonCodes` directly.
- **A `REJECTED_EXCLUDED` candidate is never operator-reachable as pending** — the exclusion check
  runs before a candidate's status is ever set to a reviewable one.
- **A re-scrape never overwrites a row already past review** — `UpsertCandidate`'s `ON CONFLICT ...
  WHERE status IN (...)` only touches `STAGED`/`NEEDS_*`/`POSSIBLE_DUPLICATE` rows.
- **`SuggestedJurisdictionUnitID` is never applied without the operator's own explicit choice.**
  D-JurisdictionUnits: jurisdiction is operator-assigned, never inferred. `matchJurisdiction`
  populates it as a hint only; `ApproveCandidate` only ever parents under
  `ApproveCandidateRequest.jurisdictionUnitId` (or the configured root unit if omitted) — live-
  confirmed via a direct `tenant_unit_edges` query, not just by reading `provision.go`.

## Automated test coverage (production-hardening pass)

Zero test files existed before this pass. The pure, DB-free logic behind three of the invariants
above is now unit-tested directly, matching this repo's own "pure function, split out so it's
testable" convention (`scripts/bootstrap-registration-org`'s `permissionsToAdd`):
`application/taxonmatch_test.go`/`jurisdictionmatch_test.go` (substring matching, case-
insensitivity, source-scoped-over-global override), `application/provision_test.go` (a table test
over every `domain.Status` — a direct regression test for the real duplicate-provisioning bug,
`docs/milestones.md`'s M8 detail), `application/dedup_test.go` (`haversineMeters` against known
coordinate pairs), and `transport/cursor_test.go`/`service_test.go` (pageToken round-trip/tamper
cases, `pageSizeOrDefault`'s clamp — copied directly from `moderation`'s own M7 test cases). No
DB/go-oikumenea mocking framework — every test operates on plain data in, plain data out.

Added since (2026-08-14): `application/christianfilter_test.go` (the D-Scope positive-keyword
filter — real Ukrainian names across every keyword family, the JW/LDS "correctly not caught here,
caught downstream" ordering case, all three real apostrophe spellings). `adapters/connectors/uaedr/
connector_test.go`'s `TestFetchFileMultiBatchResume` is a direct regression test for the real
cursor-doubling bug (a fixture forcing more than one `batchSize=500` batch — confirmed to fail
against the pre-fix code, not just pass trivially against the fixed one). `adapters/connectors/
uaedr/connector_http_test.go` covers the hand-written streaming zip parser (good/bad signature,
DEFLATE, STORE with and without a data-descriptor, the latter empirically what Go's own
`archive/zip.Writer` produces for a streamed STORE entry) and an `httptest`-backed end-to-end
`fetchHTTP` run plus a concurrency-guard test (two goroutines, one must be rejected).

## Open seams

- Additional country sources (Brazil, Argentina, OSM, and confirming Uruguay/Paraguay/Colombia/
  Chile each have — or don't have — an equivalent open registry) — see "Sources" above. Explicitly
  out of scope for the production-hardening pass (owner's own call: depth on `ua-edr` first, breadth
  later).
- HTML connector scaffolding + the first real HTML-scraped site.
- No delete/deactivate endpoint for taxon/jurisdiction aliases — `create`+`list` only. An alias is
  normalized/validated before insert, so a wrong one is rare, not a daily operator task; deferred
  rather than built speculatively.
- A real geocoder plug-in.
- The claim flow — a real, separate design decision, not this module's to make alone.
- The review-queue UI's admin browser click-through (page loads, "Load more", approve/reject/alias
  forms submit for real) — `tsc`/`eslint`/`next build` clean and a headless API-level proof exist;
  no Google OAuth session was available in this environment to click through the actual pages
  (same limitation M4.1/M7 both named and accepted as equivalent evidence for their own UI work).
- ~~The go-oikumenea RLS fix itself~~ — resolved; see Known limitations
  ([go-oikumenea#36](https://github.com/olehmushka/go-oikumenea/issues/36)).
- ~~Deploying `ua-edr` on a memory-constrained VM without a local file / cron / object storage~~ —
  resolved (`UAEDR_SOURCE_URL`, see Sources above).
- A bad/nonexistent taxon-alias RID can still abort an entire connector run rather than just
  failing that one candidate (named as a small open seam in the original production-hardening
  pass, unchanged).
