// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the discovery module's Postgres store — a single disposable cache table, no
// soft-delete, no audit (docs/modules/discovery.md). sqlc-generated (docs/architecture/decisions.md's
// D-Stack) — queries live in queries/discovery.sql, generated code in discoverysql/.
package adapters

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/discovery/adapters/discoverysql"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
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

// attributesFromJSON unmarshals a religion_sites/discovery_site_cache attributes column into its Go
// shape; an empty/invalid document (including sqlc's zero-value json.RawMessage(nil), which occurs
// on a scan where the column somehow came back empty) degrades to the zero-value SiteAttributes
// (every criterion unset) rather than erroring — matches the column's own `NOT NULL DEFAULT '{}'`.
func attributesFromJSON(raw json.RawMessage) religiondomain.SiteAttributes {
	var a religiondomain.SiteAttributes
	if len(raw) == 0 {
		return a
	}
	_ = json.Unmarshal(raw, &a)
	return a
}

func attributesToJSON(a religiondomain.SiteAttributes) json.RawMessage {
	b, err := json.Marshal(a)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

// cacheRowFromFields builds a domain.CacheRow from the column set every SELECT/RETURNING on
// discovery_site_cache shares — sqlc generates a distinct row struct per query even when the
// column list is identical, so this is the one place that shape gets translated.
func cacheRowFromFields(
	id, religionSiteRid, congregationUnitRid string,
	contentSiteID pgtype.Text,
	latitude, longitude pgtype.Numeric,
	name string,
	addressLine, traditionTaxonID, traditionTaxonCode, traditionTaxonName pgtype.Text,
	serviceLanguages []string,
	serviceDays []int16,
	attributes json.RawMessage,
	refreshedAt time.Time,
) domain.CacheRow {
	days := make([]int, len(serviceDays))
	for i, d := range serviceDays {
		days[i] = int(d)
	}
	return domain.CacheRow{
		ID:                  id,
		ReligionSiteRID:     religionSiteRid,
		CongregationUnitRID: congregationUnitRid,
		ContentSiteID:       fromNullableText(contentSiteID),
		Latitude:            floatFromNumeric(latitude),
		Longitude:           floatFromNumeric(longitude),
		Name:                name,
		Address:             fromNullableText(addressLine),
		TraditionTaxonID:    fromNullableText(traditionTaxonID),
		TraditionTaxonCode:  fromNullableText(traditionTaxonCode),
		TraditionTaxonName:  fromNullableText(traditionTaxonName),
		ServiceLanguages:    serviceLanguages,
		ServiceDays:         days,
		Attributes:          attributesFromJSON(attributes),
		RefreshedAt:         refreshedAt,
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
		Name:                row.Name,
		AddressLine:         nullableText(row.Address),
		TraditionTaxonID:    nullableText(row.TraditionTaxonID),
		TraditionTaxonCode:  nullableText(row.TraditionTaxonCode),
		TraditionTaxonName:  nullableText(row.TraditionTaxonName),
		ServiceLanguages:    languages,
		ServiceDays:         days,
		Attributes:          attributesToJSON(row.Attributes),
	})
	if err != nil {
		return domain.CacheRow{}, err
	}
	return cacheRowFromFields(result.ID, result.ReligionSiteRid, result.CongregationUnitRid, result.ContentSiteID,
		result.Latitude, result.Longitude, result.Name, result.AddressLine, result.TraditionTaxonID,
		result.TraditionTaxonCode, result.TraditionTaxonName, result.ServiceLanguages, result.ServiceDays,
		result.Attributes, result.RefreshedAt), nil
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
		out = append(out, cacheRowFromFields(row.ID, row.ReligionSiteRid, row.CongregationUnitRid, row.ContentSiteID,
			row.Latitude, row.Longitude, row.Name, row.AddressLine, row.TraditionTaxonID,
			row.TraditionTaxonCode, row.TraditionTaxonName, row.ServiceLanguages, row.ServiceDays,
			row.Attributes, row.RefreshedAt))
	}
	return out, nil
}
