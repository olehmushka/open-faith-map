// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"errors"
)

// ErrGeocodeNoMatch means the configured provider ran successfully but found nothing for the
// given query — distinct from a network/timeout/provider failure (which propagates unchanged, the
// same "pass through as-is" convention transport/errors.go's mapErr already uses for every other
// non-domain error).
var ErrGeocodeNoMatch = errors.New("congregationimport: no geocode match")

// GeocodeQuery is structured, not a single free-text string — most providers (Nominatim included)
// resolve a structured street/city/state/country query more reliably than one concatenated string,
// and it maps directly onto the address fields a Candidate already carries.
type GeocodeQuery struct {
	Street     *string
	Locality   *string
	AdminArea1 *string
	Country    *string
}

// GeocodeResult is always a SUGGESTION, never applied automatically — mirrors
// suggestedJurisdictionUnitId's own established invariant (D-JurisdictionUnits): the caller must
// still go through EditCandidate to actually persist Latitude/Longitude.
type GeocodeResult struct {
	Latitude    float64
	Longitude   float64
	Precision   *string // the provider's own reported precision/place type, shown to the operator, never trusted blindly
	DisplayName string  // the resolved address, so the operator can sanity-check the match before trusting it
	Provider    string  // which Geocoder produced this (its own Code())
}

// Geocoder is the Strategy-pattern interface every provider (Nominatim, and later LocationIQ/
// Google/...) implements identically — mirrors Connector's own shape and reasoning exactly: the
// application layer never branches on provider, and adding a new one is a new adapter package plus
// one registration line in main.go, not an interface or endpoint change.
type Geocoder interface {
	// Code is the stable provider identifier, stored on every GeocodeResult.Provider.
	Code() string
	// Geocode returns a match for query, or (nil, nil) when the provider found nothing — a real,
	// expected outcome (see ErrGeocodeNoMatch), not an error condition.
	Geocode(ctx context.Context, query GeocodeQuery) (*GeocodeResult, error)
}
