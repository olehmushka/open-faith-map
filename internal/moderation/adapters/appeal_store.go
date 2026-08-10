// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"

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

// ListAppeals has no real cursor pagination — same LIMIT-only precedent as ListReports.
func (s *Store) ListAppeals(ctx context.Context, status *domain.AppealStatus, pageSize int) ([]domain.Appeal, error) {
	var rows pgx.Rows
	var err error
	if status != nil {
		rows, err = s.pool.Query(ctx, `SELECT `+appealColumns+` FROM openfaithmap.moderation_appeals
			WHERE status = $1 ORDER BY created_at DESC LIMIT $2`, string(*status), pageSize)
	} else {
		rows, err = s.pool.Query(ctx, `SELECT `+appealColumns+` FROM openfaithmap.moderation_appeals
			ORDER BY created_at DESC LIMIT $1`, pageSize)
	}
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
