// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the audit-log module's Postgres store — sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/auditlog.sql, generated code
// in auditlogsql/.
package adapters

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/auditlog/adapters/auditlogsql"
	"github.com/olehmushka/open-faith-map/internal/auditlog/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *auditlogsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: auditlogsql.New(conn)}
}

func nullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// InsertEntry writes one append-only row. actorPersonID empty means NULL (the actor's person was
// deleted between the mutation and this write — vanishingly unlikely in a single request, but
// identity_audit_log.actor_person_id is nullable for exactly this case). before/after nil means
// SQL NULL, not the JSON literal "null" — nullableJSON keeps that distinction rather than letting an
// empty json.RawMessage insert an empty byte string.
func (r *Repository) InsertEntry(ctx context.Context, actorPersonID, action, targetKind, targetID string, before, after []byte) error {
	return r.q.InsertAuditEntry(ctx, auditlogsql.InsertAuditEntryParams{
		ActorPersonID: nullableText(actorPersonID),
		Action:        action,
		TargetKind:    targetKind,
		TargetID:      targetID,
		Before:        nullableJSON(before),
		After:         nullableJSON(after),
	})
}

func nullableJSON(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}

// ListEntries uses the same real keyset pagination as moderation's ListReports (M7,
// docs/modules/hardening.md): the caller passes after, the decoded cursor of the last row of the
// previous page, and this queries pageSize+1 rows so the caller can tell whether a next page exists
// without a second round trip.
func (r *Repository) ListEntries(ctx context.Context, filter domain.Filter, pageSize int, after *domain.PageCursor) ([]domain.Entry, error) {
	params := auditlogsql.ListAuditEntriesParams{
		ActorPersonID: nullableText(filter.ActorPersonID),
		TargetKind:    nullableText(filter.TargetKind),
		TargetID:      nullableText(filter.TargetID),
		CreatedFrom:   db.NullableTimeArg(filter.From),
		CreatedTo:     db.NullableTimeArg(filter.To),
		PageSize:      int32(pageSize),
	}
	if after != nil {
		params.AfterCreatedAt = db.NullableTimeArg(&after.CreatedAt)
		params.AfterID = pgtype.Text{String: after.ID, Valid: true}
	}
	rows, err := r.q.ListAuditEntries(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Entry, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Entry{
			ID:              row.ID,
			ActorPersonID:   row.ActorPersonID,
			ActorPersonName: row.ActorPersonName,
			Action:          row.Action,
			TargetKind:      row.TargetKind,
			TargetID:        row.TargetID,
			Before:          row.Before,
			After:           row.After,
			CreatedAt:       row.CreatedAt,
		})
	}
	return out, nil
}
