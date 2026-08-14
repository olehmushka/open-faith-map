// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the congregationimport module's Postgres store. Hand-written pgx rather than
// sqlc — matching every real module in this repo (registration/content/moderation/vouching), not
// the aspirational D-Stack convention (see conventions.md's own note that sqlc is not actually used
// anywhere yet).
package adapters

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const runSelectColumns = `
	id, source_code, status, triggered_by_person_rid, parameters, cursor_at_start, cursor_at_end,
	records_fetched, candidates_created, candidates_updated, candidates_auto_rejected, error,
	started_at, finished_at`

// CreateRun persists a new run row, including whatever parameters the caller actually supplied to
// RunConnector (nil when none were) — an empty map is stored as SQL NULL, not an empty JSON object,
// so listRuns/getRun's Parameters field is nil (not an empty map) for the common no-parameters case.
func (s *Store) CreateRun(ctx context.Context, sourceCode, triggeredByPersonRID string, parameters map[string]string, cursorAtStart *string) (domain.Run, error) {
	var paramsArg *map[string]string
	if len(parameters) > 0 {
		paramsArg = &parameters
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.congregationimport_runs (source_code, triggered_by_person_rid, parameters, cursor_at_start)
		VALUES ($1, $2, $3, $4)
		RETURNING `+runSelectColumns,
		sourceCode, triggeredByPersonRID, paramsArg, cursorAtStart,
	)
	return scanRun(row)
}

func (s *Store) GetRun(ctx context.Context, id string) (domain.Run, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+runSelectColumns+` FROM openfaithmap.congregationimport_runs WHERE id = $1`, id)
	run, err := scanRun(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, domain.ErrRunNotFound
	}
	return run, err
}

// ListRuns uses real keyset pagination, mirroring moderation's own M7 fix
// (internal/moderation/adapters/report_store.go's ListReports): the caller passes after, the
// decoded cursor of the last row of the previous page, and this queries pageSize+1 rows ordered
// (started_at DESC, id DESC) so the caller can tell whether a next page exists without a second
// round trip. The WHERE clause is assembled from a predicate list rather than branching on every
// (sourceCode, after) combination.
func (s *Store) ListRuns(ctx context.Context, sourceCode *string, pageSize int, after *domain.PageCursor) ([]domain.Run, error) {
	var where []string
	var args []any
	if sourceCode != nil {
		args = append(args, *sourceCode)
		where = append(where, fmt.Sprintf("source_code = $%d", len(args)))
	}
	if after != nil {
		args = append(args, after.CreatedAt, after.ID)
		where = append(where, fmt.Sprintf("(started_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, pageSize)
	query := `SELECT ` + runSelectColumns + ` FROM openfaithmap.congregationimport_runs`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}
	query += ` ORDER BY started_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("congregationimport: list runs: %w", err)
	}
	defer rows.Close()

	var out []domain.Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

// FinishRun records the terminal state of a run — counts, the end cursor, and either SUCCEEDED or
// FAILED (with errMsg set only in the latter case).
func (s *Store) FinishRun(ctx context.Context, id string, status domain.RunStatus, cursorAtEnd *string, recordsFetched, created, updated, autoRejected int, errMsg *string) (domain.Run, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_runs
		SET status = $2, cursor_at_end = $3, records_fetched = $4, candidates_created = $5,
		    candidates_updated = $6, candidates_auto_rejected = $7, error = $8, finished_at = $9
		WHERE id = $1
		RETURNING `+runSelectColumns,
		id, string(status), cursorAtEnd, recordsFetched, created, updated, autoRejected, errMsg, time.Now(),
	)
	run, err := scanRun(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, domain.ErrRunNotFound
	}
	return run, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(row rowScanner) (domain.Run, error) {
	var r domain.Run
	var status string
	if err := row.Scan(
		&r.ID, &r.SourceCode, &status, &r.TriggeredByPersonRID, &r.Parameters, &r.CursorAtStart, &r.CursorAtEnd,
		&r.RecordsFetched, &r.CandidatesCreated, &r.CandidatesUpdated, &r.CandidatesAutoRejected,
		&r.Error, &r.StartedAt, &r.FinishedAt,
	); err != nil {
		return domain.Run{}, err
	}
	r.Status = domain.RunStatus(status)
	return r, nil
}
