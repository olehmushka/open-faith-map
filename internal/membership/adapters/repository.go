// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the membership module's Postgres store, ported from
// ../go-oikumenea/internal/membership/adapters (tenant_* -> directory_*, oikumenea ->
// openfaithmap schema; rank/order/facet/stats machinery dropped — see the domain package's doc
// comment). sqlc-generated (docs/architecture/decisions.md's D-Stack) — queries live in
// queries/membership.sql, generated code in membershipsql/.
package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/membership/adapters/membershipsql"
	"github.com/olehmushka/open-faith-map/internal/membership/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *membershipsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: membershipsql.New(conn)}
}

// uniqueViolation reports whether err is a Postgres unique-constraint violation on the named
// constraint/index — the signal ensurePosition/ensureFilled's resumed-retry logic depends on
// (registration/application/service.go:296-329).
func uniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

func (r *Repository) InsertPosition(ctx context.Context, unitID, code, title string) (domain.Position, error) {
	row, err := r.q.InsertPosition(ctx, membershipsql.InsertPositionParams{UnitID: unitID, Code: code, Title: title})
	if err != nil {
		if uniqueViolation(err, "membership_positions_unit_code_active_idx") {
			return domain.Position{}, domain.ErrPositionConflict
		}
		return domain.Position{}, err
	}
	return domain.Position{ID: row.ID, UnitID: row.UnitID, Code: row.Code, Title: row.Title, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) GetPosition(ctx context.Context, id string) (domain.Position, error) {
	row, err := r.q.GetPosition(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Position{}, domain.ErrPositionNotFound
	}
	if err != nil {
		return domain.Position{}, err
	}
	return domain.Position{ID: row.ID, UnitID: row.UnitID, Code: row.Code, Title: row.Title, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) ListPositionsByUnit(ctx context.Context, unitID string) ([]domain.Position, error) {
	rows, err := r.q.ListPositionsByUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Position, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Position{ID: row.ID, UnitID: row.UnitID, Code: row.Code, Title: row.Title, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}

// ListMembershipsByUnit lists unitID's active memberships — M10.7's core.conjure.yml
// ListMembershipsByUnit, replacing my-congregation's own pre-cutover
// client.membership.listMembers call.
func (r *Repository) ListMembershipsByUnit(ctx context.Context, unitID string) ([]domain.Membership, error) {
	rows, err := r.q.ListMembershipsByUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Membership, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Membership{ID: row.ID, PersonID: row.PersonID, UnitID: row.UnitID, PositionID: row.PositionID.String, Status: row.Status, EffectiveFrom: row.EffectiveFrom})
	}
	return out, nil
}

// InsertMembershipFillingPosition creates an active, position-filling membership for personID on
// positionID. positionID's owning unit_id backs unitID directly (no separate lookup) since the
// caller (application.Service.FillPosition) already resolved the position.
func (r *Repository) InsertMembershipFillingPosition(ctx context.Context, personID, unitID, positionID string) (domain.Membership, error) {
	row, err := r.q.InsertMembershipFillingPosition(ctx, membershipsql.InsertMembershipFillingPositionParams{
		PersonID: personID, UnitID: unitID, PositionID: pgtype.Text{String: positionID, Valid: true},
	})
	if err != nil {
		if uniqueViolation(err, "membership_memberships_one_holder_idx") {
			return domain.Membership{}, domain.ErrPositionAlreadyFilled
		}
		return domain.Membership{}, err
	}
	return domain.Membership{ID: row.ID, PersonID: row.PersonID, UnitID: row.UnitID, PositionID: row.PositionID.String, Status: row.Status, EffectiveFrom: row.EffectiveFrom}, nil
}

// CountRepointableMemberships previews RepointMemberships' effect (M11.8) without mutating
// anything: how many of duplicateID's active memberships would move onto survivorID untouched,
// versus how many would instead be ended as redundant because survivorID already holds the same
// unit/position membership.
func (r *Repository) CountRepointableMemberships(ctx context.Context, duplicateID, survivorID string) (toMove, toEnd int, err error) {
	row, err := r.q.CountRepointableMemberships(ctx, membershipsql.CountRepointableMembershipsParams{
		DuplicateID: duplicateID, SurvivorID: survivorID,
	})
	if err != nil {
		return 0, 0, err
	}
	return int(row.ToMove), int(row.ToEnd), nil
}

// RepointMemberships moves every one of duplicateID's active memberships onto survivorID (M11.8's
// MergePersons) — same repoint-or-end-redundant shape as authz's RepointRoleAssignments. Ended
// (not deleted) rows use this table's own status='ended'+effective_to convention, mirroring how a
// membership already ends elsewhere in this module. Must run inside the caller's own tx
// (core/application.Service.MergePersons); no Begin/Commit here.
func (r *Repository) RepointMemberships(ctx context.Context, duplicateID, survivorID string) (movedIDs, endedIDs []string, err error) {
	movedIDs, err = r.q.RepointMoveMemberships(ctx, membershipsql.RepointMoveMembershipsParams{
		DuplicateID: duplicateID, SurvivorID: survivorID,
	})
	if err != nil {
		return nil, nil, err
	}
	endedIDs, err = r.q.RepointEndRedundantMemberships(ctx, duplicateID)
	if err != nil {
		return nil, nil, err
	}
	return movedIDs, endedIDs, nil
}
