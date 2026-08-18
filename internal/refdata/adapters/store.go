// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the refdata module's Postgres store: a single read over refdata_countries +
// refdata_country_names (both seeded at M10.1, migrations/0021_core_refdata.sql).
package adapters

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/refdata/domain"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Store struct {
	q Querier
}

func NewStore(q Querier) *Store {
	return &Store{q: q}
}

// ListCountries returns every country with all its locale names, ordered by sort_order (matching
// refdata_countries' own seed order, which mirrors upstream's ListCountries ordering).
func (s *Store) ListCountries(ctx context.Context) ([]domain.Country, error) {
	rows, err := s.q.Query(ctx, `
		SELECT c.id, c.code, c.name, n.locale, n.name
		FROM openfaithmap.refdata_countries c
		LEFT JOIN openfaithmap.refdata_country_names n ON n.code = c.code
		ORDER BY c.sort_order NULLS LAST, c.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[string]*domain.Country{}
	var order []string
	for rows.Next() {
		var id, code, name string
		var locale, localeName *string
		if err := rows.Scan(&id, &code, &name, &locale, &localeName); err != nil {
			return nil, err
		}
		c, ok := byID[id]
		if !ok {
			c = &domain.Country{ID: id, Code: code, Name: name, Names: map[string]string{}}
			byID[id] = c
			order = append(order, id)
		}
		if locale != nil && localeName != nil {
			c.Names[*locale] = *localeName
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]domain.Country, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}
