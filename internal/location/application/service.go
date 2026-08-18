// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the location module's orchestration layer: create/read for
// location_locations. Ported from ../go-oikumenea/internal/geo/application, trimmed to exactly the
// two operations this repo's consumer modules call (CreateLocation, and a GetLocation read for
// symmetry/tests) — see the domain package's own doc comment for what's deliberately not ported.
package application

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/location/adapters"
	"github.com/olehmushka/open-faith-map/internal/location/domain"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) CreateLocation(ctx context.Context, in domain.LocationInput) (domain.Location, error) {
	if err := in.Validate(); err != nil {
		return domain.Location{}, err
	}
	return adapters.NewStore(s.pool).InsertLocation(ctx, in)
}

func (s *Service) GetLocation(ctx context.Context, id string) (domain.Location, error) {
	return adapters.NewStore(s.pool).GetLocation(ctx, id)
}
