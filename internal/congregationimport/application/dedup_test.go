// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"math"
	"testing"
)

func TestHaversineMeters(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lng1, lat2, lng2 float64
		wantMeters             float64
		tolerance              float64
	}{
		{name: "identical point is zero distance", lat1: 50.4501, lng1: 30.5234, lat2: 50.4501, lng2: 30.5234, wantMeters: 0, tolerance: 1},
		// Kyiv (50.4501, 30.5234) to Lviv (49.8397, 24.0297) — a real, well-known distance (~470km).
		{name: "Kyiv to Lviv is roughly 470km", lat1: 50.4501, lng1: 30.5234, lat2: 49.8397, lng2: 24.0297, wantMeters: 470000, tolerance: 15000},
		// ~0.001 degrees of latitude is ~111 meters at any longitude.
		{name: "small latitude delta is roughly 111m", lat1: 50.0, lng1: 30.0, lat2: 50.001, lng2: 30.0, wantMeters: 111, tolerance: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := haversineMeters(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			if math.Abs(got-tt.wantMeters) > tt.tolerance {
				t.Errorf("haversineMeters(%v,%v,%v,%v) = %v, want %v ± %v", tt.lat1, tt.lng1, tt.lat2, tt.lng2, got, tt.wantMeters, tt.tolerance)
			}
		})
	}
}

func TestHaversineMetersDuplicateRadiusBoundary(t *testing.T) {
	// Two points ~100m apart (inside duplicateRadiusMeters=250) vs ~500m apart (outside) —
	// the actual precision this distinction matters for: findPossibleDuplicate's own cutoff.
	closeMeters := haversineMeters(50.0, 30.0, 50.0009, 30.0)
	if closeMeters >= duplicateRadiusMeters {
		t.Errorf("expected ~100m apart to be inside duplicateRadiusMeters=%v, got %v", duplicateRadiusMeters, closeMeters)
	}
	farMeters := haversineMeters(50.0, 30.0, 50.0045, 30.0)
	if farMeters <= duplicateRadiusMeters {
		t.Errorf("expected ~500m apart to be outside duplicateRadiusMeters=%v, got %v", duplicateRadiusMeters, farMeters)
	}
}
