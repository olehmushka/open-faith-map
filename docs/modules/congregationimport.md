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

**Extended with a Spanish-language keyword block (2026-08-14, `ar-rnc`)** — a single merged list,
not a per-source dispatch mechanism: the Ukrainian (Cyrillic) and Spanish (Latin) stems occupy
disjoint Unicode ranges, so cross-language false positives are structurally impossible, and
`isLikelyChristian`'s signature is unchanged. See `ar-rnc`'s own Sources entry below for the
real diacritics finding that shaped it.

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
- **`ar-rnc`** — Argentina's Registro Nacional de Cultos (Ministerio de Relaciones Exteriores,
  Comercio Internacional y Culto). Real, verified live (2026-08-14): listed on `datos.gob.ar`
  (CKAN id `registro-nacional-de-cultos`, CC-BY 4.0), but **the CKAN-declared resource URL is
  dead** (`https://cancilleria.gob.ar/userfiles/datos/registro-nacional-cultos.csv` → `404`,
  confirmed live). The ministry's own current landing page
  (`https://cancilleria.gob.ar/iniciativas/datos-abiertos/set-de-datos-de-culto`) links the real,
  working export instead: `https://cancilleria.gob.ar/userfiles/datos/registro-culto-export.csv`
  — confirmed `200`, 3,608,415 bytes, `Last-Modified: 2025-08-13`. `robots.txt` checked directly:
  `/userfiles/` is not disallowed; `Crawl-delay: 10` honored (fetched once per run).
  - **Real schema, downloaded and inspected directly**: a plain 5-column CSV, no header row,
    UTF-8, 30,178 well-formed rows. Positional columns (the ministry's own landing-page prose
    names them in a different order than the file actually uses): name, address, locality,
    province, `CI`. **Unlike `ua-edr`, this source DOES carry real address text** (street,
    locality, province) — `Street`/`Locality`/`AdminArea1` are populated for real, a genuinely
    better operator starting point than `ua-edr`'s blank-everything case; `Latitude`/`Longitude`
    still stay nil (no coordinates in the source), so every candidate still lands in
    `NEEDS_GEOCODE` — the operator now has a real "Suggest coordinates" action for exactly this
    (2026-08-14, `application.Geocoder`/`nominatim`, see below), not a blank field to fill by hand.
  - **Real, consequential finding: `CI` is not a per-row unique key** — it's the registered
    institute's own registration number, shared across every `"- FILIAL N"` (branch) row of that
    institute (e.g. one institute has 100+ branch rows sharing one `CI`). `SourceRecordID` is
    therefore a SHA-256 hash of the normalized name+address+locality+province tuple, not `CI`
    directly — idempotent across re-runs, unique per real physical location.
  - **Real data-quality finding**: 503 of the 30,178 rows are byte-for-byte duplicates of another
    row (identical name+address+locality+province) — a genuine artifact in the source itself, not
    a parsing bug here. These correctly collapse onto one candidate under the hash-based
    `SourceRecordID` above (two visually-identical rows would be indistinguishable to an operator
    anyway): a clean first run yields `recordsFetched≈30178`, `candidatesCreated≈29675`.
  - **Much simpler than `ua-edr` by design**: at 3.6MB (vs. `ua-edr`'s ~3.15GB), the whole export
    loads into memory once per run — no stateful streaming, no `ConnectorCloser`, no
    reopen-and-reskip cursor arithmetic (and so none of `ua-edr`'s own real cursor-doubling bug
    class is even possible here). `Fetch` batches via plain integer-offset slicing over an
    already-materialized, already-correct slice.
  - **The D-Scope Christian-keyword pre-filter is now Spanish-aware too** (`application/
    christianfilter.go`, extended, not a new dispatch mechanism) — every stem checked against the
    real live export before being added. **Real, consequential diacritics finding**: unaccented
    `"evangelica"` outnumbers accented `"evangélica"` 10,055-to-781 in the live data, so matching
    is diacritic-insensitive (a small, fixed Spanish accent-stripping table, same treatment
    Ukrainian apostrophe variants already got).
- **`osm`** — OpenStreetMap, via the Overpass API (`amenity=place_of_worship` +
  `religion=christian`), scoped to Uruguay/Paraguay/Colombia/Chile by default — the D-Scope rollout
  countries with no confirmed dedicated government registry (Ukraine/Argentina already have
  dedicated connectors; Brazil's CNPJ is confirmed real but blocked, see below). Real, verified
  live (2026-08-14): two Overpass mirrors were robots.txt-checked before either was used —
  `overpass-api.de` (the main OSM-Foundation-run instance) disallows `/api/` in its own robots.txt,
  the exact path the query endpoint lives at, so it is **not** used; `overpass.kumi.systems`
  (Private.coffee) has **no robots.txt at all** (404, confirmed live) and its own published policy
  welcomes reasonable use, asking only that large-scale projects notify the operator first —
  `overpass.osm.ch` independently confirmed the same 404-no-robots.txt absence, as a second data
  point. This is the default endpoint (`OSM_OVERPASS_BASE_URL`).
  - **Real end-to-end query, all of Uruguay** (`area["ISO3166-1"="UY"][admin_level=2]`): 200 OK in
    ~20s, 566 real elements (290 node, 274 way, 2 relation — every way/relation carried a real
    `center` object from `out center`, 0 missing). Real tag shapes confirmed every field mapping:
    `name` (78 of 566 — ~14% — had none at all, a genuinely common case, not a rare edge case —
    these are filtered out before ever becoming a candidate, a deliberate data-quality floor),
    `denomination` (`catholic` **and** `roman_catholic` both appear for the same real denomination —
    a real vocabulary inconsistency an operator's alias table must account for with two separate
    aliases, not a parsing bug), `religion=christian`, `addr:street`/`addr:housenumber`/`addr:city`/
    `addr:country`, and a real `diocese` tag on at least one element.
  - **Unlike either other connector, OSM commonly carries both real address text AND real
    coordinates** — a node's own `lat`/`lon`, or a way/relation's `center`. A fresh `osm` candidate
    therefore lands in `STAGED` directly, bypassing `NEEDS_GEOCODE` entirely (the nil-check on
    `Latitude`/`Longitude` in `processRawRecord` already supported this; `osm` is simply the first
    connector to exercise that path for real).
  - **No pre-seeded taxon aliases exist for OSM's `denomination=*` vocabulary** (confirmed by reading
    the migrations directly — the alias tables have zero seed rows for any source) — an operator
    must create them, same as for `ua-edr`/`ar-rnc`. Starter list from real live data above:
    `catholic`, `roman_catholic` (separate alias, same taxon — see the vocabulary-inconsistency
    finding), `baptist`, `mormon`, `protestant`, `jehovahs_witness`, `evangelical`, `methodist`,
    `mennonite`, `pentecostal`, `evangelical_lutheran`, `seventh_day_adventist`, `anglican`,
    `waldensian`, `apostolic`, `greek_orthodox`.
  - **`SourceRecordID`** is OSM's own stable element identity (`"{type}/{id}"`, e.g.
    `"node/671836133"`) — genuinely per-element unique, no hashing workaround needed (unlike
    `ar-rnc`'s `CI`).
  - **Real operational reason for the default country scope, not just "match the docs' country
    list"**: `application/dedup.go`'s `findPossibleDuplicate` only checks a new candidate against
    **already-provisioned** go-oikumenea sites (`Religion.SearchSites`), never against sibling
    `STAGED` candidates still sitting in another connector's own review queue — confirmed by reading
    `dedup.go` directly. Running `osm` over Argentina today, while `ar-rnc`'s ~29,675 real candidates
    are still largely unreviewed, would flood the queue with near-duplicates dedup can't yet catch —
    scoping to the genuine coverage gap avoids this and is also the more valuable use of the
    connector. `OSM_COUNTRY_CODES` is fully operator-configurable if this changes.
- Brazil (CNPJ/Receita Federal open data, legal nature code `322-0` = "Organização Religiosa") is a
  **confirmed real registry, fully designed, but halted before any code was written** (2026-08-14):
  `arquivos.receitafederal.gov.br/robots.txt` (the host RFB migrated the whole CNPJ open-data dump
  to, a Nextcloud WebDAV share, since a January-2026 layout change) returns `Disallow: /` for all
  user agents. The owner's explicit call was to stop rather than proceed with a documented caveat
  (unlike `osm`'s two-mirror situation above, no alternative official host exists for this dataset).
  Full design + live-verification record kept in this project's session memory, not repeated here —
  if revisited, re-check that exact `robots.txt` line first, since an explicit written allowance
  from RFB or a policy change could resolve it. Uruguay/Paraguay/Colombia/Chile — no equivalent
  dedicated government registry confirmed yet (now substantially covered by `osm` instead).

## Conjure API surface

`api/congregationimport.conjure.yml` — one authenticated service (`CongregationImportService`,
`default-auth: header`), like `vouching`: no genuinely-anonymous endpoint, since a connector run is
always operator-triggered. `runConnector`/`listRuns`/`getRun`, `listCandidates`/`getCandidate`,
`editCandidate`/`approveCandidate`/`rejectCandidate`, plus (production-hardening pass)
`listTaxonAliases`/`createTaxonAlias`/`listJurisdictionAliases`/`createJurisdictionAlias`, plus
(2026-08-14) `suggestCoordinates` — advisory only, see Known limitations above. See the file for
full docs per endpoint.

`listRuns`/`listCandidates` have real keyset `(createdAt, id)` pagination as of the
production-hardening pass — mirrors `moderation`'s own M7 fix
(`internal/moderation/transport/cursor.go`) byte-for-byte: a malformed/tampered `pageToken` returns
`CongregationImport:InvalidPageToken`, never silently reinterpreted as page 1.
`listTaxonAliases`/`listJurisdictionAliases` are deliberately **not** paginated — both lists stay
small and operator-curated (`ListAliasesForMatching`'s own reasoning), loaded in full.

**`runConnector`'s `parameters` field (2026-08-14).** `RunConnectorRequest`/`ImportRun` both gained
an `optional<map<string, string>>` `parameters` field — `map<string, string>` is unprecedented in
this repo's own Conjure files but is an established, already-generated-from pattern in
go-oikumenea's own contracts (`company.conjure.yml`, `document.conjure.yml`, etc., same toolchain),
confirmed by reading them directly before using it here. Only a connector implementing
`domain.ConnectorConfigurable` accepts a non-empty map — supplying one for a connector that doesn't
returns `CongregationImport:RunParametersNotSupported` rather than being silently ignored. Today
only `osm` implements it (one key, `countryCodes`, a comma-separated ISO 3166-1 alpha-2 list — see
Sources above). Persisted on the run row (`congregationimport_runs.parameters`,
`migrations/0012_congregationimport_run_parameters.sql`) and echoed back by `listRuns`/`getRun`, so
run history shows what a past run actually used — though no admin-UI page renders run history today
(only the review-queue itself); the data is there for a future UI to use.

**Real, adjacent bug fixed the same day, not just the parameters feature itself**: `domain.Connector`
gained a required `Clone() Connector` method. Before this, `RunConnector` ran directly against the
long-lived, boot-registered connector instance every time — `arrnc`/`osm`'s `sync.Once`-cached
in-memory rows meant a SECOND `RunConnector` call for the same source silently replayed the FIRST
run's data forever, never re-querying the real source (`uaedr`'s HTTP-streaming design happened to
avoid this on its own, via its own per-run lock/stream reset). `RunConnector` now always runs
against a fresh, run-scoped connector value — `base.Clone()` when no parameters are supplied, or
`configurable.WithParameters(parameters)` when they are — so a manual re-run (with or without new
parameters) always gets real, current data.

**Admin UI (`web/apps/admin/app/[locale]/admin/congregation-import`).** The run-connector `<select>`
+ button (`page.tsx`) now includes all three registered sources (`ua-edr`/`ar-rnc`/`osm` — `osm` was
missing from the static `SOURCE_CODES` list until this pass, a real small gap found while wiring
this up). A small client component (`run-connector-form.tsx`) conditionally renders a `countryCodes`
text input when `osm` is selected (`PARAMETERIZED_SOURCES`, manually kept in sync with the backend —
no "list registered connectors + their parameter shape" endpoint exists, not worth building for one
parameterized source today). A blank field means "use the connector's own deploy-time default," never
an explicit empty-list override.

**Save no longer collapses the expanded candidate row (2026-08-14).** `page.tsx`'s `edit` action used
to `redirect()` after every save, same as `approve`/`reject` — a real UX defect an operator hit live:
correcting a candidate is often iterative (adjust, suggest coordinates, adjust again), and a full-page
redirect unmounts the whole tree, wiping `DataTable`'s own `expanded` row state
(`components/data-table.tsx`, plain `useState`) every single time. `edit` now returns the updated
`Candidate` instead of redirecting; `candidate-list.tsx` submits it via a plain `onSubmit` handler
(not `<form action>`) and patches the row into local state in place, so the row stays open. `approve`/
`reject` still `redirect()` — those are terminal (the candidate leaves the pending view or changes
status significantly), so collapsing is correct there.

**`listCandidates` gained a `sourceCode` filter (2026-08-14)**, mirroring `listRuns`'s own existing
`sourceCode` parameter byte-for-byte (same optional-query-arg shape, same `WHERE source_code = $n`
predicate-list pattern in the store). Requested by the owner once three sources
(`ua-edr`/`ar-rnc`/`osm`) made an unfiltered queue hard to scan. The admin UI's status-filter form
gained a second `<Select>` for source, submitted together as one GET request (`?status=...&source=...`)
— no new client component needed, unlike the run-connector parameters field, since this is a plain
uncontrolled filter with no conditional-field logic.

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
- **Real geocoder, added 2026-08-14** — this bullet previously claimed `application.Geocoder`
  already existed as an interface with zero implementations; that was **stale/aspirational, not
  real code** (`grep -rn "Geocoder" internal/congregationimport/` returned nothing before this
  date), found while diagnosing a real `NotApprovable` failure an operator hit live. Now real:
  `domain.Geocoder` (`domain/geocoder.go`) is the Strategy interface — mirrors `domain.Connector`
  exactly, same reasoning — with one implementation, `adapters/geocoders/nominatim` (OpenStreetMap,
  free, keyless, the same data source already behind this project's own Leaflet/OSM public map,
  M4). **Deliberately built pluggable from day one**, not a one-off: the owner's own stated goal is
  adding congregations globally over time, at a scale where Nominatim's real usage policy (1
  req/sec, no bulk/systematic querying —
  https://operations.osmfoundation.org/policies/nominatim/) will eventually need a second,
  paid provider (LocationIQ, Google) registered alongside or instead of it — adding one is a new
  adapter package plus one line in `cmd/openfaithmap-api/main.go`'s `geocoders` slice, zero
  interface or Conjure changes. Which registered provider is active is `CONGREGATIONIMPORT_GEOCODER`
  (env, defaults to `"nominatim"`), not a code change either.

  New endpoint `suggestCoordinates` (`POST /candidates/{candidateId}/suggest-coordinates`) —
  **ADVISORY ONLY**, same invariant as `suggestedJurisdictionUnitId`: it only returns a suggestion,
  never writes to the store; the operator must still call `editCandidate` to persist. Structured
  query (`street`/`city`/`state`/`country`), not a single concatenated string — resolves more
  reliably against exactly the fields a candidate already carries. `countryId` (a go-oikumenea RID)
  is best-effort resolved to a real country name via the service principal's own `Geo.ListCountries`
  before querying (never blocks the lookup if resolution fails). A real, mechanically-enforced
  `rate.Limiter` (1 req/sec, `golang.org/x/time/rate` — already a dependency, moderation's own
  inbound rate limiter uses it too) — not just a policy comment — since this module must never call
  Nominatim from `runConnector`'s bulk pipeline, only from an operator-triggered single lookup.

  **Live-verified against the real public Nominatim endpoint and the real running stack**
  (2026-08-14) — closed the loop on the exact candidate whose `NotApprovable` failure motivated
  this work (`Ministerio Evangelístico Mi Amigo Jesús A Las Naciones`, `ar-rnc`): its own messy,
  cadastral-code-laden street text correctly returned `GeocodeNoMatch` (not a crash, not a wrong
  guess) — approved instead with manually-supplied coordinates, `status: PROVISIONED`, a real
  `createdUnitId` confirmed. A second, cleaner-addressed real candidate (`IGLESIA EVANGELICA EL
  ALFARERO CRISTO JESUS`, `CASADO 1173, CASILDA, Santa Fe`) round-tripped the full happy path:
  `suggestCoordinates` → a real match (`-33.0513756, -61.1575169`, `displayName` confirming the
  real street) → `editCandidate` → `approveCandidate` → `PROVISIONED`. Also hit real connection
  resets/timeouts calling the public endpoint directly from this dev sandbox seconds apart — a real
  reminder this is a best-effort free community service, not a guaranteed-uptime API, which is
  exactly why a non-nil, non-`GeocodeNoMatch` error passes straight through to the operator as a
  clear "lookup failed" rather than being silently swallowed.
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

Added since (2026-08-14): `christianfilter_test.go`'s Spanish cases (real names from the `ar-rnc`
export — accented/unaccented "evangélica"/"evangelica", the "evangelístico" short-stem miss,
Assemblies of God, JW/Bahá'í "correctly/expectedly not caught here" cases). `adapters/connectors/
arrnc/connector_test.go` — batch-boundary correctness (this design's own real failure mode, not
`ua-edr`'s double-counting, which is structurally impossible here) and a `SourceRecordID` test
regression-testing the real `CI`-is-not-unique finding. `adapters/geocoders/nominatim/
geocoder_test.go` — an `httptest`-backed real-match case using the exact response shape captured
live against the real endpoint (string `lat`/`lon`, not numbers — a real gotcha), a real-empty-match
case, an upstream-failure-passes-through case, and a structured-query-params assertion (`street`/
`city`/`state`/`country`, not a concatenated `q=`).

Added since (2026-08-14): `adapters/connectors/osm/connector_test.go` — an `httptest`-backed
multi-country load test (each configured country queried exactly once, `CountryHint` assigned
correctly per country, a nameless element filtered out, a `name:es`-only element kept via the locale
fallback), a request-shape test (the real query string contains the ISO code and
`amenity`/`religion` filters), batch-boundary and single-batch-exhaustion tests mirroring `ar-rnc`'s
own, `coordinatesOf`'s node-vs-way/relation-center table test, `elementName`'s fallback-chain table
test, and `Normalize`'s full field mapping including the `denomination`/`diocese` real-vocabulary
cases and their fallback-to-`Name` paths.

Added since (2026-08-14): each connector's own `TestClone` — `arrnc`'s and `osm`'s are real
regression tests for the staleness bug (rewriting/re-serving different fixture content between the
original's first load and the clone's own first `Fetch`, confirming the clone sees the CURRENT
content, not a cached one); `uaedr`'s confirms the interface contract (a distinct value, zero
`httpModeState`) since `FilePath` mode has no cache to go stale in the first place. `osm`'s
`TestWithParameters` covers the `countryCodes` override, an unrecognized parameter key, an unknown
country code, a blank value, and the empty-map/no-override case. **Not added**: an
`application.Service.RunConnector`-level test — this module has no DB/go-oikumenea mocking
infrastructure anywhere (`docs/modules/congregationimport.md`'s own testing philosophy, "every test
operates on plain data in, plain data out"), and `RunConnector` needs both; the dispatch logic itself
was verified by full build/vet/test plus reading the code directly, matching how `RunConnector`'s
correctness has always been verified in this module — a real live run, not a unit test.

## Open seams

- **No run-history UI exists.** `listRuns`/`getRun` (including the new `parameters` field) are
  wired in `lib/congregation-import.ts` but nothing in `web/apps/admin` ever calls or renders them —
  the review-queue page only shows candidates, not past runs. An operator triggering `osm` with
  `countryCodes` today sees the resulting candidates land in the queue, but has no page showing
  "this run used {countryCodes: 'CL'}, fetched N, created M." Real, deliberately out of scope for
  the 2026-08-14 manual-run-parameters pass (not asked for, and would be a new page built from
  scratch, not an extension of an existing one) — the backend/data model is ready whenever this is
  built.
- Additional country sources — see "Sources" above. Argentina (`ar-rnc`) and OpenStreetMap (`osm`,
  scoped to Uruguay/Paraguay/Colombia/Chile) are now built (2026-08-14). Brazil's CNPJ is designed
  and live-verified but halted on a `robots.txt` finding, not built. USA remains unscoped. Explicitly
  out of scope beyond this for now (owner's own call: one connector at a time, done well).
- **Real, live finding (2026-08-14): Colombia's whole-country query genuinely times out on
  `overpass.kumi.systems`** — a real operator run through the admin UI failed, reproduced directly
  moments later (`504`, "the server is probably too busy"); an identical Uruguay query at the same
  time completed in 6.5s. Fixed: Colombia is now the one country configured with a `regionGrid`
  (`osm/connector.go`'s `countries["CO"]`) — its query is split into 3×2=6 smaller bbox-bounded
  requests (still intersected with the real country polygon via the same `area["ISO3166-1"=...]`
  filter, so results stay geographically accurate — bbox only limits how much of that polygon one
  request has to search). A second real finding from the same investigation: the mirror doesn't
  always fail with a clean `504` — one Colombia attempt came back `200 OK` with an HTML error page in
  the body instead of JSON, which a bare status-code check doesn't catch; `queryRegion` now detects a
  non-JSON body explicitly and says so, rather than surfacing a confusing raw
  `invalid character '<'` decode error. Only Colombia has a `Grid` — Uruguay/Paraguay/Chile keep
  their original single whole-country query, since only Colombia has actually been observed to need
  splitting (measured, not guessed); if another country is later found to time out the same way, give
  it a `Grid` too rather than pre-emptively splitting every country.
- `osm`'s real per-country totals beyond Uruguay (Paraguay/Chile, and now Colombia's post-split
  totals), and whether the `area["ISO3166-1"=...]` query shape stays reliable at those countries'
  real scale on `overpass.kumi.systems` — only Uruguay was live-verified end to end at full scale.
  `Country.Name`'s real content for all four target countries against a live go-oikumenea instance is
  also still unconfirmed (the literal strings in `osm`'s `countries` map are English ISO short names,
  same convention `ar-rnc`'s `"Argentina"` uses, but not live-checked the way `ar-rnc`'s was).
- HTML connector scaffolding + the first real HTML-scraped site.
- No delete/deactivate endpoint for taxon/jurisdiction aliases — `create`+`list` only. An alias is
  normalized/validated before insert, so a wrong one is rare, not a daily operator task; deferred
  rather than built speculatively.
- ~~A real geocoder plug-in.~~ Built 2026-08-14 (`domain.Geocoder`/`adapters/geocoders/nominatim`,
  see Known limitations above). A second provider (LocationIQ/Google, for volume beyond Nominatim's
  own ToS) remains open, not yet needed.
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
