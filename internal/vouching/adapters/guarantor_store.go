// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

const guarantorStatusColumns = `guarantor_person_rid, status, revoked_at, revoked_reason, revoked_by_person_rid, updated_at`

// GetGuarantorStatus returns the guarantor's current status. The absence of any row means
// StatusTrusted (the column's own DEFAULT) — synthesized here rather than surfaced as a not-found
// error, so callers never need a separate "ensure row exists" write path.
func (s *Store) GetGuarantorStatus(ctx context.Context, guarantorPersonRID string) (domain.GuarantorStatus, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+guarantorStatusColumns+` FROM openfaithmap.vouching_guarantor_status WHERE guarantor_person_rid = $1`, guarantorPersonRID)
	status, err := scanGuarantorStatus(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GuarantorStatus{GuarantorPersonRID: guarantorPersonRID, Status: domain.StatusTrusted}, nil
	}
	return status, err
}

// UpsertRevoked marks a guarantor revoked, inserting a fresh row if none exists yet.
func (s *Store) UpsertRevoked(ctx context.Context, guarantorPersonRID, reason, revokedByPersonRID string) (domain.GuarantorStatus, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.vouching_guarantor_status (guarantor_person_rid, status, revoked_at, revoked_reason, revoked_by_person_rid)
		VALUES ($1, 'revoked', now(), $2, $3)
		ON CONFLICT (guarantor_person_rid) DO UPDATE SET
			status = 'revoked', revoked_at = now(), revoked_reason = $2, revoked_by_person_rid = $3
		RETURNING `+guarantorStatusColumns,
		guarantorPersonRID, reason, revokedByPersonRID,
	)
	return scanGuarantorStatus(row)
}

func scanGuarantorStatus(row rowScanner) (domain.GuarantorStatus, error) {
	var g domain.GuarantorStatus
	var status string
	if err := row.Scan(&g.GuarantorPersonRID, &status, &g.RevokedAt, &g.RevokedReason, &g.RevokedByPersonRID, &g.UpdatedAt); err != nil {
		return domain.GuarantorStatus{}, err
	}
	g.Status = domain.GuarantorStatusValue(status)
	return g, nil
}
