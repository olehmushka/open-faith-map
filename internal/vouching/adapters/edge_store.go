// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

const vouchColumns = `id, guarantor_person_rid, claimant_person_rid, congregation_unit_rid, statement, created_at`

// InsertVouch writes a new vouching_edges row. Append-only — there is no update/delete method for
// this table anywhere in this store, matching the reject_mutation()-guarded table it backs.
func (s *Store) InsertVouch(ctx context.Context, in domain.CreateVouchInput) (domain.Vouch, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.vouching_edges (guarantor_person_rid, claimant_person_rid, congregation_unit_rid, statement)
		VALUES ($1, $2, $3, $4)
		RETURNING `+vouchColumns,
		in.GuarantorPersonRID, in.ClaimantPersonRID, in.CongregationUnitID, in.Statement,
	)
	return scanVouch(row)
}

// ListVouches has no real cursor pagination — same LIMIT-only precedent as moderation's
// ListReports/ListAppeals — nextPageToken always unset in the response.
func (s *Store) ListVouches(ctx context.Context, claimant, congregation *string, pageSize int) ([]domain.Vouch, error) {
	var rows pgx.Rows
	var err error
	switch {
	case claimant != nil && congregation != nil:
		rows, err = s.pool.Query(ctx, `SELECT `+vouchColumns+` FROM openfaithmap.vouching_edges
			WHERE claimant_person_rid = $1 AND congregation_unit_rid = $2 ORDER BY created_at DESC LIMIT $3`,
			*claimant, *congregation, pageSize)
	case claimant != nil:
		rows, err = s.pool.Query(ctx, `SELECT `+vouchColumns+` FROM openfaithmap.vouching_edges
			WHERE claimant_person_rid = $1 ORDER BY created_at DESC LIMIT $2`, *claimant, pageSize)
	case congregation != nil:
		rows, err = s.pool.Query(ctx, `SELECT `+vouchColumns+` FROM openfaithmap.vouching_edges
			WHERE congregation_unit_rid = $1 ORDER BY created_at DESC LIMIT $2`, *congregation, pageSize)
	default:
		rows, err = s.pool.Query(ctx, `SELECT `+vouchColumns+` FROM openfaithmap.vouching_edges
			ORDER BY created_at DESC LIMIT $1`, pageSize)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Vouch
	for rows.Next() {
		v, err := scanVouch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListVouchesByGuarantor returns every vouch a guarantor has ever filed, unbounded — used only by
// RevokeGuarantor's moderation-report fan-out, never a client-facing listing.
func (s *Store) ListVouchesByGuarantor(ctx context.Context, guarantorPersonRID string) ([]domain.Vouch, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+vouchColumns+` FROM openfaithmap.vouching_edges
		WHERE guarantor_person_rid = $1 ORDER BY created_at ASC`, guarantorPersonRID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Vouch
	for rows.Next() {
		v, err := scanVouch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanVouch(row rowScanner) (domain.Vouch, error) {
	var v domain.Vouch
	if err := row.Scan(&v.ID, &v.GuarantorPersonRID, &v.ClaimantPersonRID, &v.CongregationUnitID, &v.Statement, &v.CreatedAt); err != nil {
		return domain.Vouch{}, err
	}
	return v, nil
}
