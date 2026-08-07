// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the registration module's Postgres store. Hand-written pgx queries rather
// than sqlc (deviates from D-Stack's "pgx + sqlc" convention) — a deliberate, documented
// simplification for this module's small, single-table surface; revisit if the query count grows.
package adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/olehmushka/open-faith-map/internal/registration/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectColumns = `
	id, submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
	street, house_number, postal_code, latitude, longitude, status, rejection_reason,
	decided_by_person_id, decided_at, created_unit_id, created_at, updated_at`

func (s *Store) Insert(ctx context.Context, in domain.SubmitInput) (domain.Request, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.registration_requests
			(submitted_by_person_id, taxon_id, congregation_name, country_id, admin_area1, locality,
			 street, house_number, postal_code, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+selectColumns,
		in.SubmittedByPersonID, in.TaxonID, in.CongregationName, in.CountryID, in.AdminArea1,
		in.Locality, in.Street, in.HouseNumber, in.PostalCode, in.Coordinate.Latitude, in.Coordinate.Longitude,
	)
	return scanRequest(row)
}

func (s *Store) Get(ctx context.Context, id string) (domain.Request, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+selectColumns+` FROM openfaithmap.registration_requests WHERE id = $1`, id)
	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	}
	return req, err
}

// List returns requests, most recent first, optionally filtered by status.
func (s *Store) List(ctx context.Context, status *domain.Status, pageSize int) ([]domain.Request, error) {
	var rows pgx.Rows
	var err error
	if status != nil {
		rows, err = s.pool.Query(ctx, `
			SELECT `+selectColumns+` FROM openfaithmap.registration_requests
			WHERE status = $1 ORDER BY created_at DESC LIMIT $2`, string(*status), pageSize)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT `+selectColumns+` FROM openfaithmap.registration_requests
			ORDER BY created_at DESC LIMIT $1`, pageSize)
	}
	if err != nil {
		return nil, fmt.Errorf("registration: list: %w", err)
	}
	defer rows.Close()

	var out []domain.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

// ListBySubmitter returns a single submitter's own requests, most recent first.
func (s *Store) ListBySubmitter(ctx context.Context, submittedByPersonID string, pageSize int) ([]domain.Request, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+selectColumns+` FROM openfaithmap.registration_requests
		WHERE submitted_by_person_id = $1 ORDER BY created_at DESC LIMIT $2`, submittedByPersonID, pageSize)
	if err != nil {
		return nil, fmt.Errorf("registration: list by submitter: %w", err)
	}
	defer rows.Close()

	var out []domain.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

// Approve flips a PENDING request to APPROVED, recording the operator and the go-oikumenea unit
// createChildOrg produced. Returns domain.ErrNotPending if the row isn't currently PENDING (guards
// a double-approve race — the CHECK constraint would also reject a malformed row shape).
func (s *Store) Approve(ctx context.Context, id, decidedByPersonID, createdUnitID string) (domain.Request, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.registration_requests
		SET status = 'APPROVED', decided_by_person_id = $2, decided_at = $3, created_unit_id = $4
		WHERE id = $1 AND status = 'PENDING'
		RETURNING `+selectColumns,
		id, decidedByPersonID, time.Now(), createdUnitID,
	)
	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotPending
	}
	return req, err
}

// Reject flips a PENDING request to REJECTED with a reason. Returns domain.ErrNotPending if the
// row isn't currently PENDING.
func (s *Store) Reject(ctx context.Context, id, decidedByPersonID, reason string) (domain.Request, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.registration_requests
		SET status = 'REJECTED', decided_by_person_id = $2, decided_at = $3, rejection_reason = $4
		WHERE id = $1 AND status = 'PENDING'
		RETURNING `+selectColumns,
		id, decidedByPersonID, time.Now(), reason,
	)
	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotPending
	}
	return req, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRequest(row rowScanner) (domain.Request, error) {
	var r domain.Request
	var status string
	var lat, lng float64
	if err := row.Scan(
		&r.ID, &r.SubmittedByPersonID, &r.TaxonID, &r.CongregationName, &r.CountryID,
		&r.AdminArea1, &r.Locality, &r.Street, &r.HouseNumber, &r.PostalCode,
		&lat, &lng, &status, &r.RejectionReason,
		&r.DecidedByPersonID, &r.DecidedAt, &r.CreatedUnitID, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return domain.Request{}, err
	}
	r.Status = domain.Status(status)
	r.Coordinate = domain.Coordinate{Latitude: lat, Longitude: lng}
	return r, nil
}
