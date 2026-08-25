-- name: UpsertCandidate :one
INSERT INTO openfaithmap.congregationimport_candidates
	(import_run_id, source_code, source_record_id, name, taxon_hint, jurisdiction_hint,
	 admin_area1, locality, street, house_number, postal_code, latitude, longitude,
	 raw_payload, status)
VALUES (sqlc.narg('import_run_id'), sqlc.arg('source_code'), sqlc.arg('source_record_id'), sqlc.arg('name'),
	sqlc.narg('taxon_hint'), sqlc.narg('jurisdiction_hint'), sqlc.narg('admin_area1'), sqlc.narg('locality'),
	sqlc.narg('street'), sqlc.narg('house_number'), sqlc.narg('postal_code'), sqlc.narg('latitude'), sqlc.narg('longitude'),
	sqlc.arg('raw_payload'), sqlc.arg('status'))
ON CONFLICT (source_code, source_record_id) DO UPDATE SET
	name = EXCLUDED.name, taxon_hint = EXCLUDED.taxon_hint,
	jurisdiction_hint = EXCLUDED.jurisdiction_hint, admin_area1 = EXCLUDED.admin_area1,
	locality = EXCLUDED.locality, street = EXCLUDED.street, house_number = EXCLUDED.house_number,
	postal_code = EXCLUDED.postal_code, latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
	raw_payload = EXCLUDED.raw_payload, import_run_id = EXCLUDED.import_run_id
	WHERE openfaithmap.congregationimport_candidates.status IN
		('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at, (xmax = 0) AS inserted;

-- name: GetCandidateBySource :one
SELECT id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at
FROM openfaithmap.congregationimport_candidates
WHERE source_code = sqlc.arg('source_code') AND source_record_id = sqlc.arg('source_record_id');

-- name: GetCandidate :one
SELECT id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at
FROM openfaithmap.congregationimport_candidates WHERE id = sqlc.arg('id');

-- name: ListCandidates :many
SELECT id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at
FROM openfaithmap.congregationimport_candidates
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
	AND (sqlc.narg('source_code')::text IS NULL OR source_code = sqlc.narg('source_code')::text)
	AND (
		sqlc.narg('after_created_at')::timestamptz IS NULL
		OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
	)
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg('page_size');

-- name: SetTaxonMatch :one
UPDATE openfaithmap.congregationimport_candidates SET taxon_id = sqlc.arg('taxon_id') WHERE id = sqlc.arg('id')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: SetJurisdictionMatch :one
UPDATE openfaithmap.congregationimport_candidates SET suggested_jurisdiction_unit_id = sqlc.arg('suggested_jurisdiction_unit_id') WHERE id = sqlc.arg('id')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: SetCountryMatch :one
UPDATE openfaithmap.congregationimport_candidates SET country_id = COALESCE(country_id, sqlc.arg('country_id')) WHERE id = sqlc.arg('id')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: SetCandidateStatus :one
UPDATE openfaithmap.congregationimport_candidates
SET status = sqlc.arg('status'), possible_duplicate_of_candidate_id = sqlc.narg('possible_duplicate_of_candidate_id'),
	possible_duplicate_of_unit_id = sqlc.narg('possible_duplicate_of_unit_id')
WHERE id = sqlc.arg('id')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: RejectExcludedCandidate :one
UPDATE openfaithmap.congregationimport_candidates
SET status = 'REJECTED_EXCLUDED', rejection_reason = sqlc.arg('rejection_reason')
WHERE id = sqlc.arg('id')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: EditCandidate :one
UPDATE openfaithmap.congregationimport_candidates SET
	name = COALESCE(sqlc.narg('name'), name), taxon_id = COALESCE(sqlc.narg('taxon_id'), taxon_id),
	country_id = COALESCE(sqlc.narg('country_id'), country_id), admin_area1 = COALESCE(sqlc.narg('admin_area1'), admin_area1),
	locality = COALESCE(sqlc.narg('locality'), locality), street = COALESCE(sqlc.narg('street'), street),
	house_number = COALESCE(sqlc.narg('house_number'), house_number), postal_code = COALESCE(sqlc.narg('postal_code'), postal_code),
	latitude = COALESCE(sqlc.narg('latitude'), latitude), longitude = COALESCE(sqlc.narg('longitude'), longitude)
WHERE id = sqlc.arg('id') AND status IN ('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: RejectCandidate :one
UPDATE openfaithmap.congregationimport_candidates
SET status = 'REJECTED', rejection_reason = sqlc.arg('rejection_reason'), reviewed_by_person_rid = sqlc.arg('reviewed_by_person_rid'), reviewed_at = sqlc.arg('reviewed_at')
WHERE id = sqlc.arg('id') AND status IN ('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: MarkCandidateProvisioning :one
UPDATE openfaithmap.congregationimport_candidates
SET status = 'PROVISIONING', reviewed_by_person_rid = sqlc.arg('reviewed_by_person_rid'), reviewed_at = sqlc.arg('reviewed_at'), created_unit_id = sqlc.arg('created_unit_id')
WHERE id = sqlc.arg('id') AND status IN ('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE',
                              'APPROVED', 'PROVISIONING')
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: MarkCandidateProvisioned :one
UPDATE openfaithmap.congregationimport_candidates SET status = 'PROVISIONED'
WHERE id = sqlc.arg('id') AND status = 'PROVISIONING'
RETURNING id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at;

-- name: CreateCongregationStatus :one
INSERT INTO openfaithmap.congregationimport_congregation_status
	(congregation_unit_rid, source_code, import_candidate_id, verified_by_person_rid, verified_at)
VALUES (sqlc.arg('congregation_unit_rid'), sqlc.arg('source_code'), sqlc.narg('import_candidate_id'), sqlc.arg('verified_by_person_rid'), now())
RETURNING congregation_unit_rid, source_code, import_candidate_id, verified_by_person_rid, verified_at,
	claimed_by_person_rid, claimed_at, created_at, updated_at;

-- name: GetCitation :one
SELECT robots_txt_url, robots_checked_at, terms_url, terms_checked_at, user_agent,
       rate_limit_notes, citation_notes
FROM openfaithmap.congregationimport_connector_citations WHERE connector_code = sqlc.arg('connector_code');

-- name: UpsertCitation :exec
INSERT INTO openfaithmap.congregationimport_connector_citations
	(connector_code, robots_txt_url, robots_checked_at, terms_url, terms_checked_at,
	 user_agent, rate_limit_notes, citation_notes)
VALUES (sqlc.arg('connector_code'), sqlc.narg('robots_txt_url'), sqlc.narg('robots_checked_at'), sqlc.narg('terms_url'),
	sqlc.narg('terms_checked_at'), sqlc.arg('user_agent'), sqlc.narg('rate_limit_notes'), sqlc.arg('citation_notes'))
ON CONFLICT (connector_code) DO UPDATE SET
	robots_txt_url = EXCLUDED.robots_txt_url, robots_checked_at = EXCLUDED.robots_checked_at,
	terms_url = EXCLUDED.terms_url, terms_checked_at = EXCLUDED.terms_checked_at,
	user_agent = EXCLUDED.user_agent, rate_limit_notes = EXCLUDED.rate_limit_notes,
	citation_notes = EXCLUDED.citation_notes;

-- name: CreateJurisdictionAlias :one
INSERT INTO openfaithmap.congregationimport_jurisdiction_aliases (source_code, alias_text, jurisdiction_unit_id, created_by_person_rid)
VALUES (sqlc.narg('source_code'), sqlc.arg('alias_text'), sqlc.arg('jurisdiction_unit_id'), sqlc.arg('created_by_person_rid'))
RETURNING id, source_code, alias_text, jurisdiction_unit_id, created_by_person_rid, created_at, updated_at;

-- name: ListJurisdictionAliasesForMatching :many
SELECT id, source_code, alias_text, jurisdiction_unit_id, created_by_person_rid, created_at, updated_at
FROM openfaithmap.congregationimport_jurisdiction_aliases
WHERE source_code = sqlc.arg('source_code') OR source_code IS NULL
ORDER BY source_code NULLS LAST;

-- name: ListAllJurisdictionAliases :many
SELECT id, source_code, alias_text, jurisdiction_unit_id, created_by_person_rid, created_at, updated_at
FROM openfaithmap.congregationimport_jurisdiction_aliases
ORDER BY source_code NULLS LAST, alias_text;

-- name: CreateTaxonAlias :one
INSERT INTO openfaithmap.congregationimport_taxon_aliases (source_code, alias_text, taxon_id, created_by_person_rid)
VALUES (sqlc.narg('source_code'), sqlc.arg('alias_text'), sqlc.arg('taxon_id'), sqlc.arg('created_by_person_rid'))
RETURNING id, source_code, alias_text, taxon_id, created_by_person_rid, created_at, updated_at;

-- name: ListAliasesForMatching :many
SELECT id, source_code, alias_text, taxon_id, created_by_person_rid, created_at, updated_at
FROM openfaithmap.congregationimport_taxon_aliases
WHERE source_code = sqlc.arg('source_code') OR source_code IS NULL
ORDER BY source_code NULLS LAST;

-- name: ListAllTaxonAliases :many
SELECT id, source_code, alias_text, taxon_id, created_by_person_rid, created_at, updated_at
FROM openfaithmap.congregationimport_taxon_aliases
ORDER BY source_code NULLS LAST, alias_text;

-- name: CreateRun :one
INSERT INTO openfaithmap.congregationimport_runs (source_code, triggered_by_person_rid, parameters, cursor_at_start)
VALUES (sqlc.arg('source_code'), sqlc.arg('triggered_by_person_rid'), sqlc.narg('parameters'), sqlc.narg('cursor_at_start'))
RETURNING id, source_code, status, triggered_by_person_rid, parameters, cursor_at_start, cursor_at_end,
	records_fetched, candidates_created, candidates_updated, candidates_auto_rejected, error,
	started_at, finished_at;

-- name: GetRun :one
SELECT id, source_code, status, triggered_by_person_rid, parameters, cursor_at_start, cursor_at_end,
	records_fetched, candidates_created, candidates_updated, candidates_auto_rejected, error,
	started_at, finished_at
FROM openfaithmap.congregationimport_runs WHERE id = sqlc.arg('id');

-- name: ListRuns :many
SELECT id, source_code, status, triggered_by_person_rid, parameters, cursor_at_start, cursor_at_end,
	records_fetched, candidates_created, candidates_updated, candidates_auto_rejected, error,
	started_at, finished_at
FROM openfaithmap.congregationimport_runs
WHERE (sqlc.narg('source_code')::text IS NULL OR source_code = sqlc.narg('source_code')::text)
	AND (
		sqlc.narg('after_started_at')::timestamptz IS NULL
		OR (started_at, id) < (sqlc.narg('after_started_at')::timestamptz, sqlc.narg('after_id')::uuid)
	)
ORDER BY started_at DESC, id DESC LIMIT sqlc.arg('page_size');

-- name: FinishRun :one
UPDATE openfaithmap.congregationimport_runs
SET status = sqlc.arg('status'), cursor_at_end = sqlc.narg('cursor_at_end'), records_fetched = sqlc.arg('records_fetched'),
    candidates_created = sqlc.arg('candidates_created'), candidates_updated = sqlc.arg('candidates_updated'),
    candidates_auto_rejected = sqlc.arg('candidates_auto_rejected'), error = sqlc.narg('error'), finished_at = sqlc.arg('finished_at')
WHERE id = sqlc.arg('id')
RETURNING id, source_code, status, triggered_by_person_rid, parameters, cursor_at_start, cursor_at_end,
	records_fetched, candidates_created, candidates_updated, candidates_auto_rejected, error,
	started_at, finished_at;

-- name: GetJurisdictionUnitByNaturalKey :one
SELECT id, source_code, external_id, parent_external_id, name, org_kind_id, status, created_unit_id,
	failure_reason, created_at, updated_at
FROM openfaithmap.congregationimport_jurisdiction_units
WHERE source_code = sqlc.arg('source_code') AND external_id = sqlc.arg('external_id');

-- name: CreatePendingJurisdictionUnit :one
INSERT INTO openfaithmap.congregationimport_jurisdiction_units
	(source_code, external_id, parent_external_id, name, org_kind_id)
VALUES (sqlc.arg('source_code'), sqlc.arg('external_id'), sqlc.narg('parent_external_id'), sqlc.arg('name'), sqlc.arg('org_kind_id'))
RETURNING id, source_code, external_id, parent_external_id, name, org_kind_id, status, created_unit_id,
	failure_reason, created_at, updated_at;

-- name: MarkJurisdictionUnitCreated :one
UPDATE openfaithmap.congregationimport_jurisdiction_units
SET status = 'CREATED', created_unit_id = sqlc.arg('created_unit_id'), failure_reason = NULL
WHERE id = sqlc.arg('id')
RETURNING id, source_code, external_id, parent_external_id, name, org_kind_id, status, created_unit_id,
	failure_reason, created_at, updated_at;

-- name: MarkJurisdictionUnitFailed :one
UPDATE openfaithmap.congregationimport_jurisdiction_units
SET status = 'FAILED', failure_reason = sqlc.arg('failure_reason')
WHERE id = sqlc.arg('id')
RETURNING id, source_code, external_id, parent_external_id, name, org_kind_id, status, created_unit_id,
	failure_reason, created_at, updated_at;
