# M10.9 pre-teardown baselines

Captured 2026-08-18, directly against the live `oikumenea` Postgres schema (same shared instance,
before M10.8's teardown migration drops it) — not through go-oikumenea's own HTTP API, since no
service-principal/person credential capable of calling `GeoService.ListCountries()` remained in this
repo by M10.7 (D-DirectTokenVerification deleted the service-principal concept; the human bootstrap
scripts were deleted at M10.6). A direct query of the same tables `ListCountries()` itself reads is
an equivalent, equally-faithful capture — arguably more so, since it bypasses any HTTP-layer
transformation.

No pre-M10.6 baseline was ever captured by any earlier session (confirmed: no `*baseline*` file
existed anywhere in the repo before this one) — this is the first and only one, captured live,
immediately before teardown, not "before M10.6" as the original milestone framing assumed.

## `oikumenea_baseline_countries.json`

All 250 active rows of `oikumenea.geo_countries` joined against `oikumenea.i18n_translations`
(`entity_type='country'`, `field='name'`) for the four locales this repo uses (`eng`/`spa`/`por`/
`ukr`). Sanity-checked at capture time: byte-for-byte identical (all 250 codes, all 4 locale names
each) to `openfaithmap.refdata_countries`/`refdata_country_names` as they stood right after M10.1's
extraction — confirming that extraction was correct. M10.9's country-parity test diffs this fixture
against `internal/refdata.Service.ListCountries()` post-teardown, matched by `code` (RIDs differ
between the two schemas by design — M10.1 minted OpenFaithMap's own RIDs, never reused oikumenea's).

## `oikumenea_baseline_discovery.json`

**Not production data, and not a scale baseline** — `oikumenea.religion_sites` held exactly 18 rows
at capture time, all real-only in the sense that they were created by hand during M2–M8's own live
verification passes (registration/congregation-import testing across those milestones), not migrated
production congregations. `openfaithmap.religion_sites` is genuinely empty right now (0 rows) — no
data migration ever copied congregation rows from `oikumenea` into `openfaithmap` (D-CongregationImport:
real congregation data is reproduced post-cutover via a fresh `ua-edr` run, not migrated). This
fixture exists to prove the count/shape comparison *mechanism* works (grouped by country code +
`public_precision`), not to assert scale parity — the real scale proof is the `ua-edr` re-run's
30,721 count, which has no oikumenea-side analog to diff against at all.
