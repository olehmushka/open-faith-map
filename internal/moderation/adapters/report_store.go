// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the moderation module's Postgres store. Hand-written pgx (matches
// internal/registration's/internal/content's documented single-module simplification — sqlc not
// required), split into one file per table for readability; one Store struct/package across all
// three.
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const reportColumns = `id, target_kind, target_ref, reason_code, detail, reporter_person_id, queue_scope, status, created_at, updated_at`

func (s *Store) InsertReport(ctx context.Context, in domain.FileReportInput, scope domain.QueueScope) (domain.Report, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.moderation_reports (target_kind, target_ref, reason_code, detail, queue_scope)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+reportColumns,
		string(in.TargetKind), in.TargetRef, string(in.ReasonCode), in.Detail, string(scope),
	)
	return scanReport(row)
}

func (s *Store) GetReportByID(ctx context.Context, id string) (domain.Report, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+reportColumns+` FROM openfaithmap.moderation_reports WHERE id = $1 AND deleted_at IS NULL`, id)
	report, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Report{}, domain.ErrReportNotFound
	}
	return report, err
}

// ListReports has no real cursor pagination (registration's ListRequests precedent — LIMIT-only,
// no OFFSET/keyset — nextPageToken always unset in the response).
func (s *Store) ListReports(ctx context.Context, scope *domain.QueueScope, status *domain.ReportStatus, pageSize int) ([]domain.Report, error) {
	var rows pgx.Rows
	var err error
	switch {
	case scope != nil && status != nil:
		rows, err = s.pool.Query(ctx, `SELECT `+reportColumns+` FROM openfaithmap.moderation_reports
			WHERE deleted_at IS NULL AND queue_scope = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3`,
			string(*scope), string(*status), pageSize)
	case scope != nil:
		rows, err = s.pool.Query(ctx, `SELECT `+reportColumns+` FROM openfaithmap.moderation_reports
			WHERE deleted_at IS NULL AND queue_scope = $1 ORDER BY created_at DESC LIMIT $2`,
			string(*scope), pageSize)
	case status != nil:
		rows, err = s.pool.Query(ctx, `SELECT `+reportColumns+` FROM openfaithmap.moderation_reports
			WHERE deleted_at IS NULL AND status = $1 ORDER BY created_at DESC LIMIT $2`,
			string(*status), pageSize)
	default:
		rows, err = s.pool.Query(ctx, `SELECT `+reportColumns+` FROM openfaithmap.moderation_reports
			WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1`, pageSize)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Report
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, report)
	}
	return out, rows.Err()
}

func (s *Store) MarkReportStatus(ctx context.Context, id string, status domain.ReportStatus) (domain.Report, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.moderation_reports SET status = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+reportColumns,
		id, string(status),
	)
	report, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Report{}, domain.ErrReportNotFound
	}
	return report, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReport(row rowScanner) (domain.Report, error) {
	var r domain.Report
	var targetKind, reasonCode, queueScope, status string
	if err := row.Scan(&r.ID, &targetKind, &r.TargetRef, &reasonCode, &r.Detail, &r.ReporterPersonID, &queueScope, &status, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return domain.Report{}, err
	}
	r.TargetKind = domain.TargetKind(targetKind)
	r.ReasonCode = domain.ReasonCode(reasonCode)
	r.QueueScope = domain.QueueScope(queueScope)
	r.Status = domain.ReportStatus(status)
	return r, nil
}
