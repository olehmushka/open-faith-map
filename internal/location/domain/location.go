// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the location module's types: the shared place object (location_locations).
// Ported from ../go-oikumenea/internal/geo/domain, trimmed per D-CorePortScope and this milestone's
// own scope note: no geo_places (WOF gazetteer, dropped table), and — deliberately, not an
// oversight — no MGRS/UTM/СК-42 coordinate-format support. Upstream's coordinate conversion (and its
// MGRS derivation) both depend on github.com/wroge/wgs84 for the UTM projection maths; grepping this
// repo's own consumer modules (internal/registration, internal/congregationimport) shows every
// CoordinateInput they ever build is plain lat/lon — no caller has ever supplied UTM/MGRS/СК-42. A
// port with no callers for the feature is speculative, not required, and the location_locations.mgrs
// column stays nullable, always NULL in this port. Add real conversion the day a caller needs one.
package domain

import (
	"errors"
	"time"
)

var (
	ErrLocationNotFound = errors.New("location: not found")
	ErrInvalidLocation  = errors.New("location: invalid request")
)

// Location is the shared, standalone place entity (location_locations): a precise WGS84 coordinate
// plus a structured postal address over refdata_countries. A location carries no owner/visibility —
// a referencing module (religion_sites) owns the meaning on its own link.
type Location struct {
	ID          string
	Latitude    float64
	Longitude   float64
	CountryID   string
	AdminArea1  string
	AdminArea2  string
	Locality    string
	Street      string
	HouseNumber string
	PostalCode  string
	RawAddress  string
	TypeID      string // "" = unset
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LocationInput is the create request: a plain lat/lon coordinate plus the structured address.
type LocationInput struct {
	Latitude    float64
	Longitude   float64
	CountryID   string
	AdminArea1  string
	AdminArea2  string
	Locality    string
	Street      string
	HouseNumber string
	PostalCode  string
	RawAddress  string
	TypeID      string
}

// Validate enforces the create-time invariants: a coordinate on Earth and a country.
func (in LocationInput) Validate() error {
	if in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
		return ErrInvalidLocation
	}
	if in.CountryID == "" {
		return ErrInvalidLocation
	}
	return nil
}
