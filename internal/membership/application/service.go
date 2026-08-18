// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the membership module's orchestration layer: CreatePosition, ListPositions
// (by unit), and FillPosition — see the domain package's doc comment for why the surface stops there.
package application

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/membership/adapters"
	"github.com/olehmushka/open-faith-map/internal/membership/domain"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) CreatePosition(ctx context.Context, unitID, code, title string) (domain.Position, error) {
	return adapters.NewStore(s.pool).InsertPosition(ctx, unitID, code, title)
}

func (s *Service) ListPositionsByUnit(ctx context.Context, unitID string) ([]domain.Position, error) {
	return adapters.NewStore(s.pool).ListPositionsByUnit(ctx, unitID)
}

// ListMembershipsByUnit lists unitID's active memberships — M10.7's core.conjure.yml surface.
func (s *Service) ListMembershipsByUnit(ctx context.Context, unitID string) ([]domain.Membership, error) {
	return adapters.NewStore(s.pool).ListMembershipsByUnit(ctx, unitID)
}

// FillPosition creates an active membership filling positionID for personID. unitID is the
// position's own owning unit — callers already have it from CreatePosition/ListPositionsByUnit, so
// this does not re-derive it from positionID to save a lookup.
func (s *Service) FillPosition(ctx context.Context, personID, unitID, positionID string) (domain.Membership, error) {
	return adapters.NewStore(s.pool).InsertMembershipFillingPosition(ctx, personID, unitID, positionID)
}
