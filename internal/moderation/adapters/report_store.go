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
	"fmt"
	"strconv"
	"strings"

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

// ListReports uses real keyset pagination as of M7 (docs/modules/hardening.md): the caller passes
// after, the decoded cursor of the last row of the previous page, and this queries pageSize+1 rows
// so the caller can tell whether a next page exists without a second round trip. The WHERE clause
// is assembled from a predicate list rather than branching on every (scope, status, after)
// combination — doubling the prior four-branch switch to eight for cursor presence stopped being
// readable.
func (s *Store) ListReports(ctx context.Context, scope *domain.QueueScope, status *domain.ReportStatus, pageSize int, after *domain.PageCursor) ([]domain.Report, error) {
	where := []string{"deleted_at IS NULL"}
	var args []any
	if scope != nil {
		args = append(args, string(*scope))
		where = append(where, fmt.Sprintf("queue_scope = $%d", len(args)))
	}
	if status != nil {
		args = append(args, string(*status))
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if after != nil {
		args = append(args, after.CreatedAt, after.ID)
		where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, pageSize)
	query := `SELECT ` + reportColumns + ` FROM openfaithmap.moderation_reports WHERE ` +
		strings.Join(where, " AND ") + ` ORDER BY created_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, query, args...)
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
