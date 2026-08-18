// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the location module's Postgres store, ported from
// ../go-oikumenea/internal/geo/adapters (oikumenea -> openfaithmap schema; no MGRS derivation — see
// the domain package's own doc comment for why).
package adapters

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/location/domain"
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	q Querier
}

func NewStore(q Querier) *Store {
	return &Store{q: q}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// sourceCoordinate captures the input as originally supplied, for source_coordinate — this module
// only ever receives plain lat/lon (see the domain package's doc comment), so the shape is fixed.
type sourceCoordinate struct {
	Format    string  `json:"format"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (s *Store) InsertLocation(ctx context.Context, in domain.LocationInput) (domain.Location, error) {
	raw, err := json.Marshal(sourceCoordinate{Format: "latlon", Latitude: in.Latitude, Longitude: in.Longitude})
	if err != nil {
		return domain.Location{}, err
	}
	row := s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.location_locations
			(geom, mgrs, source_coordinate, country_id, admin_area_1, admin_area_2, locality, street,
			 house_number, postal_code, raw_address, type_id)
		VALUES (
			ST_SetSRID(ST_MakePoint($1::double precision, $2::double precision), 4326)::geography,
			NULL, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, ST_Y(geom::geometry)::double precision, ST_X(geom::geometry)::double precision,
			country_id, coalesce(admin_area_1,''), coalesce(admin_area_2,''), coalesce(locality,''),
			coalesce(street,''), coalesce(house_number,''), coalesce(postal_code,''),
			coalesce(raw_address,''), coalesce(type_id::text,''), created_at, updated_at`,
		in.Longitude, in.Latitude, raw, in.CountryID,
		nullIfEmpty(in.AdminArea1), nullIfEmpty(in.AdminArea2), nullIfEmpty(in.Locality),
		nullIfEmpty(in.Street), nullIfEmpty(in.HouseNumber), nullIfEmpty(in.PostalCode),
		nullIfEmpty(in.RawAddress), nullIfEmpty(in.TypeID))
	return scanLocation(row)
}

func (s *Store) GetLocation(ctx context.Context, id string) (domain.Location, error) {
	row := s.q.QueryRow(ctx, `
		SELECT id, ST_Y(geom::geometry)::double precision, ST_X(geom::geometry)::double precision,
			country_id, coalesce(admin_area_1,''), coalesce(admin_area_2,''), coalesce(locality,''),
			coalesce(street,''), coalesce(house_number,''), coalesce(postal_code,''),
			coalesce(raw_address,''), coalesce(type_id::text,''), created_at, updated_at
		FROM openfaithmap.location_locations WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanLocation(row)
}

func scanLocation(row pgx.Row) (domain.Location, error) {
	var l domain.Location
	err := row.Scan(&l.ID, &l.Latitude, &l.Longitude, &l.CountryID, &l.AdminArea1, &l.AdminArea2,
		&l.Locality, &l.Street, &l.HouseNumber, &l.PostalCode, &l.RawAddress, &l.TypeID,
		&l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Location{}, domain.ErrLocationNotFound
	}
	return l, err
}
