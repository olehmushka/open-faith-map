// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the refdata module's orchestration layer — one read, ListCountries.
package application

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/refdata/adapters"
	"github.com/olehmushka/open-faith-map/internal/refdata/domain"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) ListCountries(ctx context.Context) ([]domain.Country, error) {
	return adapters.NewStore(s.pool).ListCountries(ctx)
}
