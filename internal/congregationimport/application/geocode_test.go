// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

func TestDedupNonEmptyQueries(t *testing.T) {
	street := strPtr("Lago Mascardi")
	locality := strPtr("Allen")
	adminArea1 := strPtr("Río Negro")
	country := strPtr("Argentina")

	tests := []struct {
		name  string
		in    []domain.GeocodeQuery
		wantN int
	}{
		{
			name: "full candidate has three genuinely distinct broadening steps",
			in: []domain.GeocodeQuery{
				{Street: street, Locality: locality, AdminArea1: adminArea1, Country: country},
				{Locality: locality, AdminArea1: adminArea1, Country: country},
				{AdminArea1: adminArea1, Country: country},
			},
			wantN: 3,
		},
		{
			name: "no street on the candidate collapses the first two steps into one",
			in: []domain.GeocodeQuery{
				{Locality: locality, AdminArea1: adminArea1, Country: country},
				{Locality: locality, AdminArea1: adminArea1, Country: country},
				{AdminArea1: adminArea1, Country: country},
			},
			wantN: 2,
		},
		{
			name: "a candidate with no address fields at all yields zero queries, never an empty request",
			in: []domain.GeocodeQuery{
				{}, {}, {},
			},
			wantN: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupNonEmptyQueries(tt.in)
			if len(got) != tt.wantN {
				t.Errorf("dedupNonEmptyQueries(...) returned %d queries, want %d: %+v", len(got), tt.wantN, got)
			}
		})
	}
}
