// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the membership module's Postgres store, ported from
// ../go-oikumenea/internal/membership/adapters (tenant_* -> directory_*, oikumenea ->
// openfaithmap schema; rank/order/facet/stats machinery dropped — see the domain package's doc
// comment).
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olehmushka/open-faith-map/internal/membership/domain"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Store struct {
	q Querier
}

func NewStore(q Querier) *Store {
	return &Store{q: q}
}

// uniqueViolation reports whether err is a Postgres unique-constraint violation on the named
// constraint/index — the signal ensurePosition/ensureFilled's resumed-retry logic depends on
// (registration/application/service.go:296-329).
func uniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

func scanPosition(row pgx.Row) (domain.Position, error) {
	var p domain.Position
	err := row.Scan(&p.ID, &p.UnitID, &p.Code, &p.Title, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Position{}, domain.ErrPositionNotFound
	}
	return p, err
}

const positionCols = `id, unit_id, code, title, status, created_at, updated_at`

func (s *Store) InsertPosition(ctx context.Context, unitID, code, title string) (domain.Position, error) {
	row := s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.membership_positions (unit_id, code, title)
		VALUES ($1, $2, $3) RETURNING `+positionCols, unitID, code, title)
	p, err := scanPosition(row)
	if err != nil {
		if uniqueViolation(err, "membership_positions_unit_code_active_idx") {
			return domain.Position{}, domain.ErrPositionConflict
		}
		return domain.Position{}, err
	}
	return p, nil
}

func (s *Store) GetPosition(ctx context.Context, id string) (domain.Position, error) {
	return scanPosition(s.q.QueryRow(ctx, `
		SELECT `+positionCols+` FROM openfaithmap.membership_positions
		WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (s *Store) ListPositionsByUnit(ctx context.Context, unitID string) ([]domain.Position, error) {
	rows, err := s.q.Query(ctx, `
		SELECT `+positionCols+` FROM openfaithmap.membership_positions
		WHERE unit_id = $1 AND deleted_at IS NULL ORDER BY sort_order NULLS LAST, code`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Position
	for rows.Next() {
		p, err := scanPosition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

const membershipCols = `id, person_id, unit_id, position_id, status, effective_from`

func scanMembership(row pgx.Row) (domain.Membership, error) {
	var m domain.Membership
	err := row.Scan(&m.ID, &m.PersonID, &m.UnitID, &m.PositionID, &m.Status, &m.EffectiveFrom)
	return m, err
}

// ListMembershipsByUnit lists unitID's active memberships — M10.7's core.conjure.yml
// ListMembershipsByUnit, replacing my-congregation's own pre-cutover
// client.membership.listMembers call.
func (s *Store) ListMembershipsByUnit(ctx context.Context, unitID string) ([]domain.Membership, error) {
	rows, err := s.q.Query(ctx, `
		SELECT `+membershipCols+` FROM openfaithmap.membership_memberships
		WHERE unit_id = $1 AND status = 'active' AND deleted_at IS NULL
		ORDER BY effective_from`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Membership
	for rows.Next() {
		m, err := scanMembership(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// InsertMembershipFillingPosition creates an active, position-filling membership for personID on
// positionID. positionID's owning unit_id backs unitID directly (no separate lookup) since the
// caller (application.Service.FillPosition) already resolved the position.
func (s *Store) InsertMembershipFillingPosition(ctx context.Context, personID, unitID, positionID string) (domain.Membership, error) {
	var m domain.Membership
	err := s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.membership_memberships (person_id, unit_id, position_id)
		VALUES ($1, $2, $3)
		RETURNING id, person_id, unit_id, position_id, status, effective_from`,
		personID, unitID, positionID,
	).Scan(&m.ID, &m.PersonID, &m.UnitID, &m.PositionID, &m.Status, &m.EffectiveFrom)
	if err != nil {
		if uniqueViolation(err, "membership_memberships_one_holder_idx") {
			return domain.Membership{}, domain.ErrPositionAlreadyFilled
		}
		return domain.Membership{}, err
	}
	return m, nil
}
