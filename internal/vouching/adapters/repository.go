// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the vouching module's Postgres store — sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/vouching.sql, generated code
// in vouchingsql/.
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
	"github.com/olehmushka/open-faith-map/internal/vouching/adapters/vouchingsql"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

type Repository struct {
	q *vouchingsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: vouchingsql.New(conn)}
}

func nullableText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromNullableText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func toVouch(row vouchingsql.OpenfaithmapVouchingEdge) domain.Vouch {
	return domain.Vouch{
		ID:                 row.ID,
		GuarantorPersonRID: row.GuarantorPersonRid,
		ClaimantPersonRID:  row.ClaimantPersonRid,
		CongregationUnitID: row.CongregationUnitRid,
		Statement:          fromNullableText(row.Statement),
		CreatedAt:          row.CreatedAt,
	}
}

// InsertVouch writes a new vouching_edges row. Append-only — there is no update/delete method for
// this table anywhere in this store, matching the reject_mutation()-guarded table it backs.
func (r *Repository) InsertVouch(ctx context.Context, in domain.CreateVouchInput) (domain.Vouch, error) {
	row, err := r.q.InsertVouch(ctx, vouchingsql.InsertVouchParams{
		GuarantorPersonRid:  in.GuarantorPersonRID,
		ClaimantPersonRid:   in.ClaimantPersonRID,
		CongregationUnitRid: in.CongregationUnitID,
		Statement:           nullableText(in.Statement),
	})
	if err != nil {
		return domain.Vouch{}, err
	}
	return toVouch(row), nil
}

// ListVouches has no real cursor pagination — same LIMIT-only precedent as moderation's
// ListReports/ListAppeals — nextPageToken always unset in the response.
func (r *Repository) ListVouches(ctx context.Context, claimant, congregation *string, pageSize int) ([]domain.Vouch, error) {
	rows, err := r.q.ListVouches(ctx, vouchingsql.ListVouchesParams{
		ClaimantPersonRid:   nullableText(claimant),
		CongregationUnitRid: nullableText(congregation),
		PageSize:            int32(pageSize),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Vouch, 0, len(rows))
	for _, row := range rows {
		out = append(out, toVouch(row))
	}
	return out, nil
}

// ListVouchesByGuarantor returns every vouch a guarantor has ever filed, unbounded — used only by
// RevokeGuarantor's moderation-report fan-out, never a client-facing listing.
func (r *Repository) ListVouchesByGuarantor(ctx context.Context, guarantorPersonRID string) ([]domain.Vouch, error) {
	rows, err := r.q.ListVouchesByGuarantor(ctx, guarantorPersonRID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Vouch, 0, len(rows))
	for _, row := range rows {
		out = append(out, toVouch(row))
	}
	return out, nil
}

func toGuarantorStatus(row vouchingsql.OpenfaithmapVouchingGuarantorStatus) domain.GuarantorStatus {
	updatedAt := row.UpdatedAt
	return domain.GuarantorStatus{
		GuarantorPersonRID: row.GuarantorPersonRid,
		Status:             domain.GuarantorStatusValue(row.Status),
		RevokedAt:          db.NullableTime(row.RevokedAt),
		RevokedReason:      fromNullableText(row.RevokedReason),
		RevokedByPersonRID: fromNullableText(row.RevokedByPersonRid),
		UpdatedAt:          &updatedAt,
	}
}

// GetGuarantorStatus returns the guarantor's current status. The absence of any row means
// StatusTrusted (the column's own DEFAULT) — synthesized here rather than surfaced as a not-found
// error, so callers never need a separate "ensure row exists" write path.
func (r *Repository) GetGuarantorStatus(ctx context.Context, guarantorPersonRID string) (domain.GuarantorStatus, error) {
	row, err := r.q.GetGuarantorStatus(ctx, guarantorPersonRID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.GuarantorStatus{GuarantorPersonRID: guarantorPersonRID, Status: domain.StatusTrusted}, nil
	}
	if err != nil {
		return domain.GuarantorStatus{}, err
	}
	return toGuarantorStatus(row), nil
}

// UpsertRevoked marks a guarantor revoked, inserting a fresh row if none exists yet.
func (r *Repository) UpsertRevoked(ctx context.Context, guarantorPersonRID, reason, revokedByPersonRID string) (domain.GuarantorStatus, error) {
	row, err := r.q.UpsertRevokedGuarantor(ctx, vouchingsql.UpsertRevokedGuarantorParams{
		GuarantorPersonRid: guarantorPersonRID,
		RevokedReason:      nullableText(&reason),
		RevokedByPersonRid: nullableText(&revokedByPersonRID),
	})
	if err != nil {
		return domain.GuarantorStatus{}, err
	}
	return toGuarantorStatus(row), nil
}
