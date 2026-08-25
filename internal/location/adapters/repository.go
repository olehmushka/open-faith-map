// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the location module's Postgres store, ported from
// ../go-oikumenea/internal/geo/adapters (oikumenea -> openfaithmap schema; no MGRS derivation — see
// the domain package's own doc comment for why). sqlc-generated (docs/architecture/decisions.md's
// D-Stack) — queries live in queries/location.sql, generated code in locationsql/.
package adapters

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/location/adapters/locationsql"
	"github.com/olehmushka/open-faith-map/internal/location/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *locationsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: locationsql.New(conn)}
}

func nullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// sourceCoordinate captures the input as originally supplied, for source_coordinate — this module
// only ever receives plain lat/lon (see the domain package's doc comment), so the shape is fixed.
type sourceCoordinate struct {
	Format    string  `json:"format"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (r *Repository) InsertLocation(ctx context.Context, in domain.LocationInput) (domain.Location, error) {
	raw, err := json.Marshal(sourceCoordinate{Format: "latlon", Latitude: in.Latitude, Longitude: in.Longitude})
	if err != nil {
		return domain.Location{}, err
	}
	row, err := r.q.InsertLocation(ctx, locationsql.InsertLocationParams{
		Longitude:        in.Longitude,
		Latitude:         in.Latitude,
		SourceCoordinate: json.RawMessage(raw),
		CountryID:        in.CountryID,
		AdminArea1:       nullableText(in.AdminArea1),
		AdminArea2:       nullableText(in.AdminArea2),
		Locality:         nullableText(in.Locality),
		Street:           nullableText(in.Street),
		HouseNumber:      nullableText(in.HouseNumber),
		PostalCode:       nullableText(in.PostalCode),
		RawAddress:       nullableText(in.RawAddress),
		TypeID:           nullableText(in.TypeID),
	})
	if err != nil {
		return domain.Location{}, err
	}
	return domain.Location{
		ID:          row.ID,
		Latitude:    row.Latitude,
		Longitude:   row.Longitude,
		CountryID:   row.CountryID,
		AdminArea1:  row.AdminArea1,
		AdminArea2:  row.AdminArea2,
		Locality:    row.Locality,
		Street:      row.Street,
		HouseNumber: row.HouseNumber,
		PostalCode:  row.PostalCode,
		RawAddress:  row.RawAddress,
		TypeID:      row.TypeID,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *Repository) GetLocation(ctx context.Context, id string) (domain.Location, error) {
	row, err := r.q.GetLocation(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Location{}, domain.ErrLocationNotFound
		}
		return domain.Location{}, err
	}
	return domain.Location{
		ID:          row.ID,
		Latitude:    row.Latitude,
		Longitude:   row.Longitude,
		CountryID:   row.CountryID,
		AdminArea1:  row.AdminArea1,
		AdminArea2:  row.AdminArea2,
		Locality:    row.Locality,
		Street:      row.Street,
		HouseNumber: row.HouseNumber,
		PostalCode:  row.PostalCode,
		RawAddress:  row.RawAddress,
		TypeID:      row.TypeID,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}
