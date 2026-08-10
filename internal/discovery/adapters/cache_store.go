// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the discovery module's Postgres store — a single disposable cache table, no
// soft-delete, no audit (docs/modules/discovery.md).
package adapters

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const cacheColumns = `id, religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude, tradition_taxon_id, service_languages, service_days, refreshed_at`

// UpsertRow writes row wholesale, keyed by religion_site_rid — a cache row is never partially
// updated (docs/modules/discovery.md's invariants: "a stale row is simply overwritten") — and
// returns the persisted row (real id/refreshed_at, not row's pre-insert zero values): a freshly
// upserted row is returned straight to the caller on a cache-miss response, so a caller building a
// list key (e.g. React's) off `id` needs the real one, not an empty string.
func (s *Store) UpsertRow(ctx context.Context, row domain.CacheRow) (domain.CacheRow, error) {
	days := make([]int16, len(row.ServiceDays))
	for i, d := range row.ServiceDays {
		days[i] = int16(d)
	}
	languages := row.ServiceLanguages
	if languages == nil {
		languages = []string{}
	}
	result := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.discovery_site_cache
			(religion_site_rid, congregation_unit_rid, content_site_id, latitude, longitude,
			 tradition_taxon_id, service_languages, service_days, refreshed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (religion_site_rid) DO UPDATE SET
			congregation_unit_rid = EXCLUDED.congregation_unit_rid,
			content_site_id       = EXCLUDED.content_site_id,
			latitude              = EXCLUDED.latitude,
			longitude             = EXCLUDED.longitude,
			tradition_taxon_id    = EXCLUDED.tradition_taxon_id,
			service_languages     = EXCLUDED.service_languages,
			service_days          = EXCLUDED.service_days,
			refreshed_at          = now()
		RETURNING `+cacheColumns,
		row.ReligionSiteRID, row.CongregationUnitRID, row.ContentSiteID, row.Latitude, row.Longitude,
		row.TraditionTaxonID, languages, days,
	)
	return scanRow(result)
}

// SearchAll returns every cached row — radius/tradition/language/dayOfWeek filtering happens in
// the application layer (see domain.SearchQuery.BypassesCache's doc for why only lat/lng/radius
// can be filtered against this table today).
func (s *Store) SearchAll(ctx context.Context) ([]domain.CacheRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+cacheColumns+` FROM openfaithmap.discovery_site_cache`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRow(row rowScanner) (domain.CacheRow, error) {
	var out domain.CacheRow
	var days []int16
	var languages []string
	if err := row.Scan(
		&out.ID, &out.ReligionSiteRID, &out.CongregationUnitRID, &out.ContentSiteID,
		&out.Latitude, &out.Longitude, &out.TraditionTaxonID, &languages, &days, &out.RefreshedAt,
	); err != nil {
		return domain.CacheRow{}, err
	}
	out.ServiceLanguages = languages
	out.ServiceDays = make([]int, len(days))
	for i, d := range days {
		out.ServiceDays[i] = int(d)
	}
	return out, nil
}

func scanRows(rows pgx.Rows) ([]domain.CacheRow, error) {
	var out []domain.CacheRow
	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
