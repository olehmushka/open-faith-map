// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the registration module's Postgres store — sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/registration.sql, generated
// code in registrationsql/.
package adapters

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
	"github.com/olehmushka/open-faith-map/internal/registration/adapters/registrationsql"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
)

type Repository struct {
	q *registrationsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: registrationsql.New(conn)}
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

func toRequest(row registrationsql.OpenfaithmapRegistrationRequest) domain.Request {
	return domain.Request{
		ID:                  row.ID,
		SubmittedByPersonID: row.SubmittedByPersonID,
		TaxonID:             row.TaxonID,
		CongregationName:    row.CongregationName,
		CountryID:           row.CountryID,
		AdminArea1:          fromNullableText(row.AdminArea1),
		Locality:            fromNullableText(row.Locality),
		Street:              fromNullableText(row.Street),
		HouseNumber:         fromNullableText(row.HouseNumber),
		PostalCode:          fromNullableText(row.PostalCode),
		Coordinate:          domain.Coordinate{Latitude: row.Latitude, Longitude: row.Longitude},
		Status:              domain.Status(row.Status),
		RejectionReason:     fromNullableText(row.RejectionReason),
		DecidedByPersonID:   fromNullableText(row.DecidedByPersonID),
		DecidedAt:           db.NullableTime(row.DecidedAt),
		CreatedUnitID:       fromNullableText(row.CreatedUnitID),
		JurisdictionUnitID:  fromNullableText(row.JurisdictionUnitID),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func (r *Repository) Insert(ctx context.Context, in domain.SubmitInput) (domain.Request, error) {
	row, err := r.q.InsertRegistrationRequest(ctx, registrationsql.InsertRegistrationRequestParams{
		SubmittedByPersonID: in.SubmittedByPersonID,
		TaxonID:             in.TaxonID,
		CongregationName:    in.CongregationName,
		CountryID:           in.CountryID,
		AdminArea1:          nullableText(in.AdminArea1),
		Locality:            nullableText(in.Locality),
		Street:              nullableText(in.Street),
		HouseNumber:         nullableText(in.HouseNumber),
		PostalCode:          nullableText(in.PostalCode),
		Latitude:            in.Coordinate.Latitude,
		Longitude:           in.Coordinate.Longitude,
	})
	if err != nil {
		return domain.Request{}, err
	}
	return toRequest(row), nil
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Request, error) {
	row, err := r.q.GetRegistrationRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Request{}, err
	}
	return toRequest(row), nil
}

// List returns requests, most recent first, optionally filtered by status.
func (r *Repository) List(ctx context.Context, status *domain.Status, pageSize int) ([]domain.Request, error) {
	var statusArg pgtype.Text
	if status != nil {
		statusArg = pgtype.Text{String: string(*status), Valid: true}
	}
	rows, err := r.q.ListRegistrationRequests(ctx, registrationsql.ListRegistrationRequestsParams{
		Status:   statusArg,
		PageSize: int32(pageSize),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Request, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRequest(row))
	}
	return out, nil
}

// ListBySubmitter returns a single submitter's own requests, most recent first.
func (r *Repository) ListBySubmitter(ctx context.Context, submittedByPersonID string, pageSize int) ([]domain.Request, error) {
	rows, err := r.q.ListRegistrationRequestsBySubmitter(ctx, registrationsql.ListRegistrationRequestsBySubmitterParams{
		SubmittedByPersonID: submittedByPersonID,
		PageSize:            int32(pageSize),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Request, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRequest(row))
	}
	return out, nil
}

// MarkProvisioning transitions a PENDING request to PROVISIONING, persisting the operator, the
// go-oikumenea unit createChildOrg produced, and the operator's jurisdiction choice (M4.1) — the one
// approval step that cannot be re-derived on a retry. This is the ONLY point jurisdiction_unit_id is
// ever written; a resumed PROVISIONING retry never reaches here again (ensureUnit short-circuits on
// an already-persisted created_unit_id), so the original choice is what sticks regardless of what a
// retry call's input carries. Returns domain.ErrNotPending if the row isn't currently PENDING (guards
// a double-start race — the CHECK constraint would also reject a malformed row shape).
func (r *Repository) MarkProvisioning(ctx context.Context, id, decidedByPersonID, createdUnitID string, jurisdictionUnitID *string) (domain.Request, error) {
	row, err := r.q.MarkRegistrationRequestProvisioning(ctx, registrationsql.MarkRegistrationRequestProvisioningParams{
		ID:                 id,
		DecidedByPersonID:  nullableText(&decidedByPersonID),
		CreatedUnitID:      nullableText(&createdUnitID),
		JurisdictionUnitID: nullableText(jurisdictionUnitID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotPending
	}
	if err != nil {
		return domain.Request{}, err
	}
	return toRequest(row), nil
}

// Approve flips a PENDING or PROVISIONING request to APPROVED, recording the operator and the
// go-oikumenea unit createChildOrg produced. Returns domain.ErrNotPending if the row is in neither
// state (guards a double-approve race — the CHECK constraint would also reject a malformed row
// shape).
func (r *Repository) Approve(ctx context.Context, id, decidedByPersonID, createdUnitID string) (domain.Request, error) {
	row, err := r.q.ApproveRegistrationRequest(ctx, registrationsql.ApproveRegistrationRequestParams{
		ID:                id,
		DecidedByPersonID: nullableText(&decidedByPersonID),
		DecidedAt:         db.NullableTimeArg(timePtr(time.Now())),
		CreatedUnitID:     nullableText(&createdUnitID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotPending
	}
	if err != nil {
		return domain.Request{}, err
	}
	return toRequest(row), nil
}

// Reject flips a PENDING request to REJECTED with a reason. Returns domain.ErrNotPending if the
// row isn't currently PENDING.
func (r *Repository) Reject(ctx context.Context, id, decidedByPersonID, reason string) (domain.Request, error) {
	row, err := r.q.RejectRegistrationRequest(ctx, registrationsql.RejectRegistrationRequestParams{
		ID:                id,
		DecidedByPersonID: nullableText(&decidedByPersonID),
		DecidedAt:         db.NullableTimeArg(timePtr(time.Now())),
		RejectionReason:   nullableText(&reason),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotPending
	}
	if err != nil {
		return domain.Request{}, err
	}
	return toRequest(row), nil
}

func timePtr(t time.Time) *time.Time { return &t }
