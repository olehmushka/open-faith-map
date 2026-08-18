// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"testing"
)

func TestLocationInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      LocationInput
		wantErr error
	}{
		{"valid", LocationInput{Latitude: 50.45, Longitude: 30.52, CountryID: "country-1"}, nil},
		{"latitude out of range", LocationInput{Latitude: 91, Longitude: 30.52, CountryID: "country-1"}, ErrInvalidLocation},
		{"longitude out of range", LocationInput{Latitude: 50.45, Longitude: 181, CountryID: "country-1"}, ErrInvalidLocation},
		{"missing country", LocationInput{Latitude: 50.45, Longitude: 30.52}, ErrInvalidLocation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.in.Validate(); err != tt.wantErr {
				t.Errorf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
