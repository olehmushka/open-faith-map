// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const candidateSelectColumns = `
	id, import_run_id, source_code, source_record_id, name, taxon_hint, taxon_id, jurisdiction_hint,
	suggested_jurisdiction_unit_id, country_id, admin_area1, locality, street, house_number,
	postal_code, latitude, longitude, geocode_precision, raw_payload, status,
	possible_duplicate_of_candidate_id, possible_duplicate_of_unit_id, rejection_reason,
	reviewed_by_person_rid, reviewed_at, created_unit_id, created_at, updated_at`

// UpsertCandidate inserts a new candidate or, on a (source_code, source_record_id) conflict,
// updates the scraped fields of an existing STAGED/NEEDS_*/POSSIBLE_DUPLICATE one — never touches a
// row already past operator review (APPROVED/PROVISIONING/PROVISIONED/REJECTED/REJECTED_EXCLUDED),
// so a re-scrape can never silently undo a decision. Returns (candidate, created bool, error).
func (s *Store) UpsertCandidate(ctx context.Context, importRunID *string, sourceCode, sourceRecordID string, in domain.NormalizedCandidate, rawPayload []byte, status domain.Status) (domain.Candidate, bool, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.congregationimport_candidates
			(import_run_id, source_code, source_record_id, name, taxon_hint, jurisdiction_hint,
			 admin_area1, locality, street, house_number, postal_code, latitude, longitude,
			 raw_payload, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (source_code, source_record_id) DO UPDATE SET
			name = EXCLUDED.name, taxon_hint = EXCLUDED.taxon_hint,
			jurisdiction_hint = EXCLUDED.jurisdiction_hint, admin_area1 = EXCLUDED.admin_area1,
			locality = EXCLUDED.locality, street = EXCLUDED.street, house_number = EXCLUDED.house_number,
			postal_code = EXCLUDED.postal_code, latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
			raw_payload = EXCLUDED.raw_payload, import_run_id = EXCLUDED.import_run_id
			WHERE openfaithmap.congregationimport_candidates.status IN
				('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE')
		RETURNING `+candidateSelectColumns+`, (xmax = 0) AS inserted`,
		importRunID, sourceCode, sourceRecordID, in.Name, in.TaxonHint, in.JurisdictionHint,
		in.AdminArea1, in.Locality, in.Street, in.HouseNumber, in.PostalCode, in.Latitude,
		in.Longitude, rawPayload, string(status),
	)
	c, inserted, err := scanCandidateWithInserted(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// The WHERE clause on the DO UPDATE arm didn't match (row exists but already past review) —
		// fetch it as-is so the caller still gets a real candidate back, just unmodified.
		existing, getErr := s.getCandidateBySource(ctx, sourceCode, sourceRecordID)
		return existing, false, getErr
	}
	return c, inserted, err
}

func (s *Store) getCandidateBySource(ctx context.Context, sourceCode, sourceRecordID string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+candidateSelectColumns+` FROM openfaithmap.congregationimport_candidates
		WHERE source_code = $1 AND source_record_id = $2`, sourceCode, sourceRecordID)
	return scanCandidate(row)
}

func (s *Store) GetCandidate(ctx context.Context, id string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+candidateSelectColumns+` FROM openfaithmap.congregationimport_candidates WHERE id = $1`, id)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	return c, err
}

// ListCandidates uses real keyset pagination, mirroring moderation's own M7 fix
// (internal/moderation/adapters/report_store.go's ListReports): the caller passes after, the
// decoded cursor of the last row of the previous page, and this queries pageSize+1 rows ordered
// (created_at DESC, id DESC) so the caller can tell whether a next page exists without a second
// round trip. The WHERE clause is assembled from a predicate list rather than branching on every
// (status, after) combination.
func (s *Store) ListCandidates(ctx context.Context, status *domain.Status, sourceCode *string, pageSize int, after *domain.PageCursor) ([]domain.Candidate, error) {
	var where []string
	var args []any
	if status != nil {
		args = append(args, string(*status))
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if sourceCode != nil {
		args = append(args, *sourceCode)
		where = append(where, fmt.Sprintf("source_code = $%d", len(args)))
	}
	if after != nil {
		args = append(args, after.CreatedAt, after.ID)
		where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, pageSize)
	query := `SELECT ` + candidateSelectColumns + ` FROM openfaithmap.congregationimport_candidates`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("congregationimport: list candidates: %w", err)
	}
	defer rows.Close()

	var out []domain.Candidate
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetTaxonMatch records an automated or alias-resolved taxon match, without changing status —
// callers apply the resulting D-Exclusions decision separately (RejectExcluded or clearing
// NEEDS_TAXON_REVIEW).
func (s *Store) SetTaxonMatch(ctx context.Context, id, taxonID string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates SET taxon_id = $2 WHERE id = $1
		RETURNING `+candidateSelectColumns,
		id, taxonID,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	return c, err
}

// SetJurisdictionMatch records an alias-matched jurisdiction suggestion, without changing status —
// purely advisory (D-JurisdictionUnits: jurisdiction is operator-assigned, never inferred). Distinct
// from SetTaxonMatch's caller-driven follow-up decisions: a jurisdiction match never gates
// D-Exclusions or dedup, so this has no further branching in processRawRecord.
func (s *Store) SetJurisdictionMatch(ctx context.Context, id, suggestedJurisdictionUnitID string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates SET suggested_jurisdiction_unit_id = $2 WHERE id = $1
		RETURNING `+candidateSelectColumns,
		id, suggestedJurisdictionUnitID,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	return c, err
}

// SetCountryMatch records a connector-resolved country (matchCountry) directly on CountryID, not a
// separate "suggested" column like SetJurisdictionMatch — unlike jurisdiction, a country hint is an
// unambiguous, deterministic fact about the source (e.g. ar-rnc is Argentina-only), not a fuzzy
// guess the operator must still confirm. Still never overwrites an operator's own already-set value
// (COALESCE, same as every other UpsertCandidate column) — see the WHERE $2 IS NOT NULL guard here
// achieved via COALESCE against the existing value.
func (s *Store) SetCountryMatch(ctx context.Context, id, countryID string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates SET country_id = COALESCE(country_id, $2) WHERE id = $1
		RETURNING `+candidateSelectColumns,
		id, countryID,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	return c, err
}

// SetStatus is the plain status transition used for the automated STAGED/NEEDS_TAXON_REVIEW/
// NEEDS_GEOCODE/POSSIBLE_DUPLICATE moves that don't carry extra fields.
func (s *Store) SetStatus(ctx context.Context, id string, status domain.Status, dupCandidateID, dupUnitID *string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates
		SET status = $2, possible_duplicate_of_candidate_id = $3, possible_duplicate_of_unit_id = $4
		WHERE id = $1
		RETURNING `+candidateSelectColumns,
		id, string(status), dupCandidateID, dupUnitID,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	return c, err
}

// RejectExcluded is the automatic, pre-review D-Exclusions rejection — never reachable through the
// operator-facing RejectCandidate path, distinguished by its own status value
// (REJECTED_EXCLUDED, not REJECTED).
func (s *Store) RejectExcluded(ctx context.Context, id, reason string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates
		SET status = 'REJECTED_EXCLUDED', rejection_reason = $2
		WHERE id = $1
		RETURNING `+candidateSelectColumns,
		id, reason,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	return c, err
}

// Edit applies an operator's corrections to a candidate still in an editable (pre-approval) status.
// Only non-nil fields in in are applied (COALESCE against the existing value).
func (s *Store) Edit(ctx context.Context, id string, in domain.EditInput) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates SET
			name = COALESCE($2, name), taxon_id = COALESCE($3, taxon_id),
			country_id = COALESCE($4, country_id), admin_area1 = COALESCE($5, admin_area1),
			locality = COALESCE($6, locality), street = COALESCE($7, street),
			house_number = COALESCE($8, house_number), postal_code = COALESCE($9, postal_code),
			latitude = COALESCE($10, latitude), longitude = COALESCE($11, longitude)
		WHERE id = $1 AND status IN ('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE')
		RETURNING `+candidateSelectColumns,
		id, in.Name, in.TaxonID, in.CountryID, in.AdminArea1, in.Locality, in.Street, in.HouseNumber,
		in.PostalCode, in.Latitude, in.Longitude,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotEditable
	}
	return c, err
}

// Reject is the operator-decided rejection (e.g. duplicate, not a real church) — distinct from the
// automatic RejectExcluded.
func (s *Store) Reject(ctx context.Context, id, reviewedByPersonRID, reason string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates
		SET status = 'REJECTED', rejection_reason = $2, reviewed_by_person_rid = $3, reviewed_at = $4
		WHERE id = $1 AND status IN ('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE')
		RETURNING `+candidateSelectColumns,
		id, reason, reviewedByPersonRID, time.Now(),
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotEditable
	}
	return c, err
}

// MarkProvisioning transitions an APPROVED (or already-PROVISIONING, on a resumed retry) candidate
// to PROVISIONING, persisting createdUnitID as soon as the first, unrecoverable go-oikumenea write
// succeeds — registration.MarkProvisioning's exact crash-resume pattern (M2.3).
func (s *Store) MarkProvisioning(ctx context.Context, id, reviewedByPersonRID, createdUnitID string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates
		SET status = 'PROVISIONING', reviewed_by_person_rid = $2, reviewed_at = $3, created_unit_id = $4
		WHERE id = $1 AND status IN ('STAGED', 'NEEDS_TAXON_REVIEW', 'NEEDS_GEOCODE', 'POSSIBLE_DUPLICATE',
		                              'APPROVED', 'PROVISIONING')
		RETURNING `+candidateSelectColumns,
		id, reviewedByPersonRID, time.Now(), createdUnitID,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotApprovable
	}
	return c, err
}

// MarkProvisioned completes provisioning — the terminal, successful state.
func (s *Store) MarkProvisioned(ctx context.Context, id string) (domain.Candidate, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_candidates SET status = 'PROVISIONED'
		WHERE id = $1 AND status = 'PROVISIONING'
		RETURNING `+candidateSelectColumns,
		id,
	)
	c, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotApprovable
	}
	return c, err
}

func scanCandidate(row rowScanner) (domain.Candidate, error) {
	var c domain.Candidate
	var status string
	if err := row.Scan(candidateScanDest(&c, &status)...); err != nil {
		return domain.Candidate{}, err
	}
	c.Status = domain.Status(status)
	return c, nil
}

// scanCandidateWithInserted scans a row whose RETURNING clause appends one extra
// (xmax = 0) AS inserted boolean column after candidateSelectColumns — only
// UpsertCandidate's query shape carries it.
func scanCandidateWithInserted(row rowScanner) (domain.Candidate, bool, error) {
	var c domain.Candidate
	var status string
	var inserted bool
	dest := append(candidateScanDest(&c, &status), &inserted)
	if err := row.Scan(dest...); err != nil {
		return domain.Candidate{}, false, err
	}
	c.Status = domain.Status(status)
	return c, inserted, nil
}

func candidateScanDest(c *domain.Candidate, status *string) []any {
	return []any{
		&c.ID, &c.ImportRunID, &c.SourceCode, &c.SourceRecordID, &c.Name, &c.TaxonHint, &c.TaxonID,
		&c.JurisdictionHint, &c.SuggestedJurisdictionUnitID, &c.CountryID, &c.AdminArea1, &c.Locality,
		&c.Street, &c.HouseNumber, &c.PostalCode, &c.Latitude, &c.Longitude, &c.GeocodePrecision,
		&c.RawPayload, status, &c.PossibleDuplicateCandidateID, &c.PossibleDuplicateUnitID,
		&c.RejectionReason, &c.ReviewedByPersonRID, &c.ReviewedAt, &c.CreatedUnitID, &c.CreatedAt,
		&c.UpdatedAt,
	}
}
