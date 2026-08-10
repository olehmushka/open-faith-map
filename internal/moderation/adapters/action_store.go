// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

const actionColumns = `id, report_id, action_kind, target_kind, target_ref, actor_person_id, reason, reverses_action_id, created_at`

// InsertAction never sets reversed_by_action_id — moderation_actions is append-only
// (reject_mutation()-guarded); see domain.Action's doc comment for why that field is derived at
// read time instead.
func (s *Store) InsertAction(ctx context.Context, in domain.TakeActionInput) (domain.Action, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.moderation_actions (report_id, action_kind, target_kind, target_ref, actor_person_id, reason, reverses_action_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+actionColumns,
		in.ReportID, string(in.ActionKind), string(in.TargetKind), in.TargetRef, in.ActorPersonID, in.Reason, in.ReversesActionID,
	)
	action, err := scanAction(row)
	if err != nil {
		return domain.Action{}, err
	}
	return s.hydrateReversedBy(ctx, action)
}

func (s *Store) GetActionByID(ctx context.Context, id string) (domain.Action, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+actionColumns+` FROM openfaithmap.moderation_actions WHERE id = $1`, id)
	action, err := scanAction(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Action{}, domain.ErrActionNotFound
	}
	if err != nil {
		return domain.Action{}, err
	}
	return s.hydrateReversedBy(ctx, action)
}

// hydrateReversedBy populates Action.ReversedByActionID by looking forward for a REVERSE row whose
// reverses_action_id points at this one — the read-time counterpart to reverses_action_id's
// backward-only, insert-time-only write (see domain.Action's doc comment).
func (s *Store) hydrateReversedBy(ctx context.Context, action domain.Action) (domain.Action, error) {
	var reverserID string
	err := s.pool.QueryRow(ctx, `SELECT id FROM openfaithmap.moderation_actions WHERE reverses_action_id = $1`, action.ID).Scan(&reverserID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return action, nil
	case err != nil:
		return domain.Action{}, err
	}
	action.ReversedByActionID = &reverserID
	return action, nil
}

func scanAction(row rowScanner) (domain.Action, error) {
	var a domain.Action
	var actionKind, targetKind string
	if err := row.Scan(&a.ID, &a.ReportID, &actionKind, &targetKind, &a.TargetRef, &a.ActorPersonID, &a.Reason, &a.ReversesActionID, &a.CreatedAt); err != nil {
		return domain.Action{}, err
	}
	a.ActionKind = domain.ActionKind(actionKind)
	a.TargetKind = domain.TargetKind(targetKind)
	return a, nil
}
