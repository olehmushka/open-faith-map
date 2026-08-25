// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the moderation module's Postgres store — sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/moderation.sql, generated
// code in moderationsql/.
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/moderation/adapters/moderationsql"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *moderationsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: moderationsql.New(conn)}
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

func (r *Repository) InsertReport(ctx context.Context, in domain.FileReportInput, scope domain.QueueScope) (domain.Report, error) {
	row, err := r.q.InsertReport(ctx, moderationsql.InsertReportParams{
		TargetKind: string(in.TargetKind),
		TargetRef:  in.TargetRef,
		ReasonCode: string(in.ReasonCode),
		Detail:     nullableText(in.Detail),
		QueueScope: string(scope),
	})
	if err != nil {
		return domain.Report{}, err
	}
	return domain.Report{
		ID: row.ID, TargetKind: domain.TargetKind(row.TargetKind), TargetRef: row.TargetRef,
		ReasonCode: domain.ReasonCode(row.ReasonCode), Detail: fromNullableText(row.Detail),
		ReporterPersonID: fromNullableText(row.ReporterPersonID), QueueScope: domain.QueueScope(row.QueueScope),
		Status: domain.ReportStatus(row.Status), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *Repository) GetReportByID(ctx context.Context, id string) (domain.Report, error) {
	row, err := r.q.GetReportByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Report{}, domain.ErrReportNotFound
	}
	if err != nil {
		return domain.Report{}, err
	}
	return domain.Report{
		ID: row.ID, TargetKind: domain.TargetKind(row.TargetKind), TargetRef: row.TargetRef,
		ReasonCode: domain.ReasonCode(row.ReasonCode), Detail: fromNullableText(row.Detail),
		ReporterPersonID: fromNullableText(row.ReporterPersonID), QueueScope: domain.QueueScope(row.QueueScope),
		Status: domain.ReportStatus(row.Status), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// ListReports uses real keyset pagination as of M7 (docs/modules/hardening.md): the caller passes
// after, the decoded cursor of the last row of the previous page, and this queries pageSize+1 rows
// so the caller can tell whether a next page exists without a second round trip.
func (r *Repository) ListReports(ctx context.Context, scope *domain.QueueScope, status *domain.ReportStatus, pageSize int, after *domain.PageCursor) ([]domain.Report, error) {
	params := moderationsql.ListReportsParams{PageSize: int32(pageSize)}
	if scope != nil {
		params.QueueScope = pgtype.Text{String: string(*scope), Valid: true}
	}
	if status != nil {
		params.Status = pgtype.Text{String: string(*status), Valid: true}
	}
	if after != nil {
		params.AfterCreatedAt = db.NullableTimeArg(&after.CreatedAt)
		params.AfterID = pgtype.Text{String: after.ID, Valid: true}
	}
	rows, err := r.q.ListReports(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Report, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Report{
			ID: row.ID, TargetKind: domain.TargetKind(row.TargetKind), TargetRef: row.TargetRef,
			ReasonCode: domain.ReasonCode(row.ReasonCode), Detail: fromNullableText(row.Detail),
			ReporterPersonID: fromNullableText(row.ReporterPersonID), QueueScope: domain.QueueScope(row.QueueScope),
			Status: domain.ReportStatus(row.Status), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *Repository) MarkReportStatus(ctx context.Context, id string, status domain.ReportStatus) (domain.Report, error) {
	row, err := r.q.MarkReportStatus(ctx, moderationsql.MarkReportStatusParams{ID: id, Status: string(status)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Report{}, domain.ErrReportNotFound
	}
	if err != nil {
		return domain.Report{}, err
	}
	return domain.Report{
		ID: row.ID, TargetKind: domain.TargetKind(row.TargetKind), TargetRef: row.TargetRef,
		ReasonCode: domain.ReasonCode(row.ReasonCode), Detail: fromNullableText(row.Detail),
		ReporterPersonID: fromNullableText(row.ReporterPersonID), QueueScope: domain.QueueScope(row.QueueScope),
		Status: domain.ReportStatus(row.Status), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func toAction(row moderationsql.OpenfaithmapModerationAction) domain.Action {
	return domain.Action{
		ID:               row.ID,
		ReportID:         fromNullableText(row.ReportID),
		ActionKind:       domain.ActionKind(row.ActionKind),
		TargetKind:       domain.TargetKind(row.TargetKind),
		TargetRef:        row.TargetRef,
		ActorPersonID:    row.ActorPersonID,
		Reason:           row.Reason,
		ReversesActionID: fromNullableText(row.ReversesActionID),
		CreatedAt:        row.CreatedAt,
	}
}

// InsertAction never sets reversed_by_action_id — moderation_actions is append-only
// (reject_mutation()-guarded); see domain.Action's doc comment for why that field is derived at
// read time instead.
func (r *Repository) InsertAction(ctx context.Context, in domain.TakeActionInput) (domain.Action, error) {
	row, err := r.q.InsertAction(ctx, moderationsql.InsertActionParams{
		ReportID:         nullableText(in.ReportID),
		ActionKind:       string(in.ActionKind),
		TargetKind:       string(in.TargetKind),
		TargetRef:        in.TargetRef,
		ActorPersonID:    in.ActorPersonID,
		Reason:           in.Reason,
		ReversesActionID: nullableText(in.ReversesActionID),
	})
	if err != nil {
		return domain.Action{}, err
	}
	return r.hydrateReversedBy(ctx, toAction(row))
}

func (r *Repository) GetActionByID(ctx context.Context, id string) (domain.Action, error) {
	row, err := r.q.GetActionByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Action{}, domain.ErrActionNotFound
	}
	if err != nil {
		return domain.Action{}, err
	}
	return r.hydrateReversedBy(ctx, toAction(row))
}

// hydrateReversedBy populates Action.ReversedByActionID by looking forward for a REVERSE row whose
// reverses_action_id points at this one — the read-time counterpart to reverses_action_id's
// backward-only, insert-time-only write (see domain.Action's doc comment).
func (r *Repository) hydrateReversedBy(ctx context.Context, action domain.Action) (domain.Action, error) {
	reverserID, err := r.q.GetReverserActionID(ctx, pgtype.Text{String: action.ID, Valid: true})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return action, nil
	case err != nil:
		return domain.Action{}, err
	}
	action.ReversedByActionID = &reverserID
	return action, nil
}

func toAppeal(row moderationsql.OpenfaithmapModerationAppeal) domain.Appeal {
	return domain.Appeal{
		ID:                        row.ID,
		ActionID:                  row.ActionID,
		CongregationAdminPersonID: row.CongregationAdminPersonID,
		Statement:                 row.Statement,
		AssignedModeratorPersonID: fromNullableText(row.AssignedModeratorPersonID),
		Status:                    domain.AppealStatus(row.Status),
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}
}

func (r *Repository) InsertAppeal(ctx context.Context, actionID, congregationAdminPersonID, statement string) (domain.Appeal, error) {
	row, err := r.q.InsertAppeal(ctx, moderationsql.InsertAppealParams{
		ActionID: actionID, CongregationAdminPersonID: congregationAdminPersonID, Statement: statement,
	})
	if err != nil {
		return domain.Appeal{}, err
	}
	return toAppeal(row), nil
}

func (r *Repository) GetAppealByID(ctx context.Context, id string) (domain.Appeal, error) {
	row, err := r.q.GetAppealByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Appeal{}, domain.ErrAppealNotFound
	}
	if err != nil {
		return domain.Appeal{}, err
	}
	return toAppeal(row), nil
}

// ListAppeals uses real keyset pagination as of M7 — same predicate-list approach as
// report_store.go's ListReports, queries pageSize+1 rows so the caller can detect a next page.
func (r *Repository) ListAppeals(ctx context.Context, status *domain.AppealStatus, pageSize int, after *domain.PageCursor) ([]domain.Appeal, error) {
	params := moderationsql.ListAppealsParams{PageSize: int32(pageSize)}
	if status != nil {
		params.Status = pgtype.Text{String: string(*status), Valid: true}
	}
	if after != nil {
		params.AfterCreatedAt = db.NullableTimeArg(&after.CreatedAt)
		params.AfterID = pgtype.Text{String: after.ID, Valid: true}
	}
	rows, err := r.q.ListAppeals(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Appeal, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAppeal(row))
	}
	return out, nil
}

func (r *Repository) DecideAppeal(ctx context.Context, id, assignedModeratorPersonID string, status domain.AppealStatus) (domain.Appeal, error) {
	row, err := r.q.DecideAppeal(ctx, moderationsql.DecideAppealParams{
		ID:                        id,
		AssignedModeratorPersonID: pgtype.Text{String: assignedModeratorPersonID, Valid: true},
		Status:                    string(status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Appeal{}, domain.ErrAppealNotFound
	}
	if err != nil {
		return domain.Appeal{}, err
	}
	return toAppeal(row), nil
}
