// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the audit-log module's Postgres store — hand-written pgx, one Store, matching
// this repo's own single-module-simplification convention (internal/moderation/adapters).
package adapters

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/auditlog/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// InsertEntry writes one append-only row. actorPersonID empty means NULL (the actor's person was
// deleted between the mutation and this write — vanishingly unlikely in a single request, but
// identity_audit_log.actor_person_id is nullable for exactly this case). before/after nil means
// SQL NULL, not the JSON literal "null" — nullableJSON keeps that distinction rather than letting an
// empty json.RawMessage insert an empty byte string.
func (s *Store) InsertEntry(ctx context.Context, actorPersonID, action, targetKind, targetID string, before, after []byte) error {
	var actorArg any
	if actorPersonID != "" {
		actorArg = actorPersonID
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO openfaithmap.identity_audit_log (actor_person_id, action, target_kind, target_id, before, after)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		actorArg, action, targetKind, targetID, nullableJSON(before), nullableJSON(after))
	return err
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

const entryColumns = `l.id, COALESCE(l.actor_person_id::text, ''), COALESCE(p.display_name, ''), l.action, l.target_kind, l.target_id, l.before, l.after, l.created_at`

// ListEntries uses the same real keyset pagination as moderation's ListReports (M7,
// docs/modules/hardening.md): the caller passes after, the decoded cursor of the last row of the
// previous page, and this queries pageSize+1 rows so the caller can tell whether a next page exists
// without a second round trip. actor_person_name is denormalized in via a LEFT JOIN (not INNER —
// actor_person_id can be NULL after a person deletion, and the row must still list).
func (s *Store) ListEntries(ctx context.Context, filter domain.Filter, pageSize int, after *domain.PageCursor) ([]domain.Entry, error) {
	where := []string{"1=1"}
	var args []any
	if filter.ActorPersonID != "" {
		args = append(args, filter.ActorPersonID)
		where = append(where, fmt.Sprintf("l.actor_person_id = $%d", len(args)))
	}
	if filter.TargetKind != "" {
		args = append(args, filter.TargetKind)
		where = append(where, fmt.Sprintf("l.target_kind = $%d", len(args)))
	}
	if filter.TargetID != "" {
		args = append(args, filter.TargetID)
		where = append(where, fmt.Sprintf("l.target_id = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		where = append(where, fmt.Sprintf("l.created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		where = append(where, fmt.Sprintf("l.created_at <= $%d", len(args)))
	}
	if after != nil {
		args = append(args, after.CreatedAt, after.ID)
		where = append(where, fmt.Sprintf("(l.created_at, l.id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, pageSize)
	query := `SELECT ` + entryColumns + `
		FROM openfaithmap.identity_audit_log l
		LEFT JOIN openfaithmap.identity_persons p ON p.id = l.actor_person_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY l.created_at DESC, l.id DESC LIMIT $` + strconv.Itoa(len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.ActorPersonID, &e.ActorPersonName, &e.Action, &e.TargetKind, &e.TargetID, &e.Before, &e.After, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
