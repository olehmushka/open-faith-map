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
	decided_by_person_id, decided_at, created_unit_id, jurisdiction_unit_id, created_at, updated_at`

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

// MarkProvisioning transitions a PENDING request to PROVISIONING, persisting the operator, the
// go-oikumenea unit createChildOrg produced, and the operator's jurisdiction choice (M4.1) — the one
// approval step that cannot be re-derived on a retry. This is the ONLY point jurisdiction_unit_id is
// ever written; a resumed PROVISIONING retry never reaches here again (ensureUnit short-circuits on
// an already-persisted created_unit_id), so the original choice is what sticks regardless of what a
// retry call's input carries. Returns domain.ErrNotPending if the row isn't currently PENDING (guards
// a double-start race — the CHECK constraint would also reject a malformed row shape).
func (s *Store) MarkProvisioning(ctx context.Context, id, decidedByPersonID, createdUnitID string, jurisdictionUnitID *string) (domain.Request, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.registration_requests
		SET status = 'PROVISIONING', decided_by_person_id = $2, created_unit_id = $3, jurisdiction_unit_id = $4
		WHERE id = $1 AND status = 'PENDING'
		RETURNING `+selectColumns,
		id, decidedByPersonID, createdUnitID, jurisdictionUnitID,
	)
	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotPending
	}
	return req, err
}

// Approve flips a PENDING or PROVISIONING request to APPROVED, recording the operator and the
// go-oikumenea unit createChildOrg produced. Returns domain.ErrNotPending if the row is in neither
// state (guards a double-approve race — the CHECK constraint would also reject a malformed row
// shape).
func (s *Store) Approve(ctx context.Context, id, decidedByPersonID, createdUnitID string) (domain.Request, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.registration_requests
		SET status = 'APPROVED', decided_by_person_id = $2, decided_at = $3, created_unit_id = $4
		WHERE id = $1 AND status IN ('PENDING', 'PROVISIONING')
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
		&r.DecidedByPersonID, &r.DecidedAt, &r.CreatedUnitID, &r.JurisdictionUnitID, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return domain.Request{}, err
	}
	r.Status = domain.Status(status)
	r.Coordinate = domain.Coordinate{Latitude: lat, Longitude: lng}
	return r, nil
}

const reparentJobSelectColumns = `
	id, registration_request_id, congregation_unit_id, old_parent_unit_id, new_parent_unit_id,
	status, performed_by_person_id, error, created_at, updated_at`

// CreateReparentJob starts a new PENDING re-parenting job. Fails with a unique-violation if a
// non-FAILED job already exists for congregationUnitID (jurisdiction_reparenting_jobs_live_unit_idx)
// — callers should check GetLiveReparentJob first to resume instead.
func (s *Store) CreateReparentJob(ctx context.Context, registrationRequestID, congregationUnitID, oldParentUnitID, newParentUnitID, performedByPersonID string) (domain.ReparentingJob, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.jurisdiction_reparenting_jobs
			(registration_request_id, congregation_unit_id, old_parent_unit_id, new_parent_unit_id, performed_by_person_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+reparentJobSelectColumns,
		registrationRequestID, congregationUnitID, oldParentUnitID, newParentUnitID, performedByPersonID,
	)
	return scanReparentJob(row)
}

// GetLiveReparentJob returns the current non-FAILED job for congregationUnitID, if one exists (at
// most one can, per the store's own unique index) — the resume path for reparentRequest.
func (s *Store) GetLiveReparentJob(ctx context.Context, congregationUnitID string) (*domain.ReparentingJob, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+reparentJobSelectColumns+`
		FROM openfaithmap.jurisdiction_reparenting_jobs
		WHERE congregation_unit_id = $1 AND status <> 'FAILED'`,
		congregationUnitID,
	)
	job, err := scanReparentJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetLatestReparentJob returns the most recently created job for registrationRequestID (FAILED or
// not), for getReparentStatus's read-only status display.
func (s *Store) GetLatestReparentJob(ctx context.Context, registrationRequestID string) (*domain.ReparentingJob, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+reparentJobSelectColumns+`
		FROM openfaithmap.jurisdiction_reparenting_jobs
		WHERE registration_request_id = $1
		ORDER BY created_at DESC LIMIT 1`,
		registrationRequestID,
	)
	job, err := scanReparentJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// AdvanceReparentJob moves job to status (NEW_EDGE_ADDED/OLD_EDGE_REMOVED/VERIFIED), clearing any
// prior error.
func (s *Store) AdvanceReparentJob(ctx context.Context, id string, status domain.ReparentStatus) (domain.ReparentingJob, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.jurisdiction_reparenting_jobs
		SET status = $2, error = NULL
		WHERE id = $1
		RETURNING `+reparentJobSelectColumns,
		id, string(status),
	)
	return scanReparentJob(row)
}

// FailReparentJob moves job to FAILED with the given error message.
func (s *Store) FailReparentJob(ctx context.Context, id, errMsg string) (domain.ReparentingJob, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.jurisdiction_reparenting_jobs
		SET status = 'FAILED', error = $2
		WHERE id = $1
		RETURNING `+reparentJobSelectColumns,
		id, errMsg,
	)
	return scanReparentJob(row)
}

func scanReparentJob(row rowScanner) (domain.ReparentingJob, error) {
	var j domain.ReparentingJob
	var status string
	if err := row.Scan(
		&j.ID, &j.RegistrationRequestID, &j.CongregationUnitID, &j.OldParentUnitID, &j.NewParentUnitID,
		&status, &j.PerformedByPersonID, &j.Error, &j.CreatedAt, &j.UpdatedAt,
	); err != nil {
		return domain.ReparentingJob{}, err
	}
	j.Status = domain.ReparentStatus(status)
	return j, nil
}
