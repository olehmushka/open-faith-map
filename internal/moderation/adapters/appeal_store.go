// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

const appealColumns = `id, action_id, congregation_admin_person_id, statement, assigned_moderator_person_id, status, created_at, updated_at`

func (s *Store) InsertAppeal(ctx context.Context, actionID, congregationAdminPersonID, statement string) (domain.Appeal, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.moderation_appeals (action_id, congregation_admin_person_id, statement)
		VALUES ($1, $2, $3)
		RETURNING `+appealColumns,
		actionID, congregationAdminPersonID, statement,
	)
	return scanAppeal(row)
}

func (s *Store) GetAppealByID(ctx context.Context, id string) (domain.Appeal, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+appealColumns+` FROM openfaithmap.moderation_appeals WHERE id = $1`, id)
	appeal, err := scanAppeal(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Appeal{}, domain.ErrAppealNotFound
	}
	return appeal, err
}

// ListAppeals uses real keyset pagination as of M7 — same predicate-list approach as
// report_store.go's ListReports, queries pageSize+1 rows so the caller can detect a next page.
func (s *Store) ListAppeals(ctx context.Context, status *domain.AppealStatus, pageSize int, after *domain.PageCursor) ([]domain.Appeal, error) {
	var where []string
	var args []any
	if status != nil {
		args = append(args, string(*status))
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if after != nil {
		args = append(args, after.CreatedAt, after.ID)
		where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, pageSize)
	query := `SELECT ` + appealColumns + ` FROM openfaithmap.moderation_appeals`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Appeal
	for rows.Next() {
		appeal, err := scanAppeal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, appeal)
	}
	return out, rows.Err()
}

func (s *Store) DecideAppeal(ctx context.Context, id, assignedModeratorPersonID string, status domain.AppealStatus) (domain.Appeal, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.moderation_appeals SET assigned_moderator_person_id = $2, status = $3
		WHERE id = $1
		RETURNING `+appealColumns,
		id, assignedModeratorPersonID, string(status),
	)
	appeal, err := scanAppeal(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Appeal{}, domain.ErrAppealNotFound
	}
	return appeal, err
}

func scanAppeal(row rowScanner) (domain.Appeal, error) {
	var a domain.Appeal
	var status string
	if err := row.Scan(&a.ID, &a.ActionID, &a.CongregationAdminPersonID, &a.Statement, &a.AssignedModeratorPersonID, &status, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return domain.Appeal{}, err
	}
	a.Status = domain.AppealStatus(status)
	return a, nil
}
