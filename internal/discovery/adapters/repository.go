// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the discovery module's Postgres store — a single disposable cache table, no
// soft-delete, no audit (docs/modules/discovery.md). sqlc-generated (docs/architecture/decisions.md's
// D-Stack) — queries live in queries/discovery.sql, generated code in discoverysql/.
package adapters

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/discovery/adapters/discoverysql"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *discoverysql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: discoverysql.New(conn)}
}

func nullableText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromNullableText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func numericFromFloat(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(*f, 'f', -1, 64))
	return n
}

func floatFromNumeric(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	return &f.Float64
}

func toCacheRow(row discoverysql.OpenfaithmapDiscoverySiteCache) domain.CacheRow {
	days := make([]int, len(row.ServiceDays))
	for i, d := range row.ServiceDays {
		days[i] = int(d)
	}
	return domain.CacheRow{
		ID:                  row.ID,
		ReligionSiteRID:     row.ReligionSiteRid,
		CongregationUnitRID: row.CongregationUnitRid,
		ContentSiteID:       fromNullableText(row.ContentSiteID),
		Latitude:            floatFromNumeric(row.Latitude),
		Longitude:           floatFromNumeric(row.Longitude),
		TraditionTaxonID:    fromNullableText(row.TraditionTaxonID),
		ServiceLanguages:    row.ServiceLanguages,
		ServiceDays:         days,
		RefreshedAt:         row.RefreshedAt,
	}
}

// UpsertRow writes row wholesale, keyed by religion_site_rid — a cache row is never partially
// updated (docs/modules/discovery.md's invariants: "a stale row is simply overwritten") — and
// returns the persisted row (real id/refreshed_at, not row's pre-insert zero values): a freshly
// upserted row is returned straight to the caller on a cache-miss response, so a caller building a
// list key (e.g. React's) off `id` needs the real one, not an empty string.
func (r *Repository) UpsertRow(ctx context.Context, row domain.CacheRow) (domain.CacheRow, error) {
	days := make([]int16, len(row.ServiceDays))
	for i, d := range row.ServiceDays {
		days[i] = int16(d)
	}
	languages := row.ServiceLanguages
	if languages == nil {
		languages = []string{}
	}
	result, err := r.q.UpsertCacheRow(ctx, discoverysql.UpsertCacheRowParams{
		ReligionSiteRid:     row.ReligionSiteRID,
		CongregationUnitRid: row.CongregationUnitRID,
		ContentSiteID:       nullableText(row.ContentSiteID),
		Latitude:            numericFromFloat(row.Latitude),
		Longitude:           numericFromFloat(row.Longitude),
		TraditionTaxonID:    nullableText(row.TraditionTaxonID),
		ServiceLanguages:    languages,
		ServiceDays:         days,
	})
	if err != nil {
		return domain.CacheRow{}, err
	}
	return toCacheRow(result), nil
}

// SearchAll returns every cached row — radius/tradition/language/dayOfWeek filtering happens in
// the application layer (see domain.SearchQuery.BypassesCache's doc for why only lat/lng/radius
// can be filtered against this table today).
func (r *Repository) SearchAll(ctx context.Context) ([]domain.CacheRow, error) {
	rows, err := r.q.SearchAllCacheRows(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CacheRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCacheRow(row))
	}
	return out, nil
}
