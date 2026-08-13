// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const congregationStatusSelectColumns = `
	congregation_unit_rid, source_code, import_candidate_id, verified_by_person_rid, verified_at,
	claimed_by_person_rid, claimed_at, created_at, updated_at`

// CreateCongregationStatus writes the verified overlay row at provisioning time — see
// D-CongregationImport for why this exists (a proposal, not a settled design).
func (s *Store) CreateCongregationStatus(ctx context.Context, congregationUnitRID, sourceCode string, importCandidateID *string, verifiedByPersonRID string) (domain.CongregationStatus, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.congregationimport_congregation_status
			(congregation_unit_rid, source_code, import_candidate_id, verified_by_person_rid, verified_at)
		VALUES ($1, $2, $3, $4, now())
		RETURNING `+congregationStatusSelectColumns,
		congregationUnitRID, sourceCode, importCandidateID, verifiedByPersonRID,
	)
	return scanCongregationStatus(row)
}

func scanCongregationStatus(row rowScanner) (domain.CongregationStatus, error) {
	var cs domain.CongregationStatus
	if err := row.Scan(
		&cs.CongregationUnitRID, &cs.SourceCode, &cs.ImportCandidateID, &cs.VerifiedByPersonRID,
		&cs.VerifiedAt, &cs.ClaimedByPersonRID, &cs.ClaimedAt, &cs.CreatedAt, &cs.UpdatedAt,
	); err != nil {
		return domain.CongregationStatus{}, err
	}
	return cs, nil
}
