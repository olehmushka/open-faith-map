// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the refdata module's Postgres store: a single read over refdata_countries +
// refdata_country_names (both seeded at M10.1, migrations/0012_core_refdata.sql). sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/refdata.sql, generated code in
// refdatasql/.
package adapters

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/platform/db"
	"github.com/olehmushka/open-faith-map/internal/refdata/adapters/refdatasql"
	"github.com/olehmushka/open-faith-map/internal/refdata/domain"
)

type Repository struct {
	q *refdatasql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: refdatasql.New(conn)}
}

// ListCountries returns every country with all its locale names, ordered by sort_order (matching
// refdata_countries' own seed order, which mirrors upstream's ListCountries ordering).
func (r *Repository) ListCountries(ctx context.Context) ([]domain.Country, error) {
	rows, err := r.q.ListCountries(ctx)
	if err != nil {
		return nil, err
	}

	byID := map[string]*domain.Country{}
	var order []string
	for _, row := range rows {
		c, ok := byID[row.ID]
		if !ok {
			c = &domain.Country{ID: row.ID, Code: row.Code, Name: row.Name, Names: map[string]string{}}
			byID[row.ID] = c
			order = append(order, row.ID)
		}
		if row.Locale.Valid && row.LocaleName.Valid {
			c.Names[row.Locale.String] = row.LocaleName.String
		}
	}
	out := make([]domain.Country, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}
