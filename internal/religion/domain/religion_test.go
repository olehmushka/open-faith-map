// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"testing"
)

func TestCoarsen(t *testing.T) {
	const lat, lng = 50.450123, 30.523456

	tests := []struct {
		name      string
		precision string
		wantOK    bool
		wantLat   float64
		wantLng   float64
	}{
		{"empty precision passes through exact", "", true, lat, lng},
		{"exact passes through unchanged", PrecisionExact, true, lat, lng},
		{"hidden omits the coordinate entirely", PrecisionHidden, false, 0, 0},
		{"street rounds to 4 decimals", PrecisionStreet, true, 50.4501, 30.5235},
		{"neighborhood rounds to 3 decimals", PrecisionNeighborhood, true, 50.45, 30.523},
		{"city rounds to 2 decimals", PrecisionCity, true, 50.45, 30.52},
		{"unknown precision passes through exact", "bogus", true, lat, lng},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLat, gotLng, ok := Coarsen(lat, lng, tt.precision)
			if ok != tt.wantOK {
				t.Fatalf("Coarsen(%q) ok = %v, want %v", tt.precision, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if gotLat != tt.wantLat || gotLng != tt.wantLng {
				t.Errorf("Coarsen(%q) = (%v, %v), want (%v, %v)", tt.precision, gotLat, gotLng, tt.wantLat, tt.wantLng)
			}
		})
	}
}

func TestCoarsenAddress(t *testing.T) {
	const locality, adminArea1, adminArea2, street, houseNumber, postalCode = "Kyiv", "Kyiv City", "Kyiv Raion", "Khreshchatyk St", "22", "01001"

	tests := []struct {
		name      string
		precision string
		wantOK    bool
		want      string
	}{
		{"empty precision shows the full address", "", true, "Khreshchatyk St 22, Kyiv, Kyiv Raion, Kyiv City, 01001"},
		{"exact shows the full address", PrecisionExact, true, "Khreshchatyk St 22, Kyiv, Kyiv Raion, Kyiv City, 01001"},
		{"street shows the full address (coordinate is already ~11m via Coarsen)", PrecisionStreet, true, "Khreshchatyk St 22, Kyiv, Kyiv Raion, Kyiv City, 01001"},
		{"neighborhood shows locality + admin area 1 only", PrecisionNeighborhood, true, "Kyiv, Kyiv City"},
		{"city shows locality + admin area 1 only", PrecisionCity, true, "Kyiv, Kyiv City"},
		{"hidden omits the address entirely", PrecisionHidden, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CoarsenAddress(locality, adminArea1, adminArea2, street, houseNumber, postalCode, tt.precision)
			if ok != tt.wantOK {
				t.Fatalf("CoarsenAddress(%q) ok = %v, want %v", tt.precision, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("CoarsenAddress(%q) = %q, want %q", tt.precision, got, tt.want)
			}
		})
	}

	t.Run("neighborhood with no locality or admin area 1 has nothing to show", func(t *testing.T) {
		if _, ok := CoarsenAddress("", "", adminArea2, street, houseNumber, postalCode, PrecisionNeighborhood); ok {
			t.Errorf("CoarsenAddress with empty locality/adminArea1 at neighborhood precision, ok = true, want false")
		}
	})
}
