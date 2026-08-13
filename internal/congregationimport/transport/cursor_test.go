// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// encodeRaw base64-encodes arbitrary JSON text without going through the real wireCursor struct —
// lets the tamper tests exercise "well-formed base64/JSON but the wrong shape or values" without
// hand-rolling raw bytes for every case.
func encodeRaw(json string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(json))
}

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		cursor domain.PageCursor
	}{
		{name: "typical row", cursor: domain.PageCursor{CreatedAt: time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC), ID: "candidate-1"}},
		{name: "sub-second precision preserved", cursor: domain.PageCursor{CreatedAt: time.Date(2026, 8, 10, 12, 0, 0, 123456789, time.UTC), ID: "candidate-2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := decodeCursor(encodeCursor(tt.cursor))
			if err != nil {
				t.Fatalf("decodeCursor(encodeCursor(%v)) returned error: %v", tt.cursor, err)
			}
			if !decoded.CreatedAt.Equal(tt.cursor.CreatedAt) || decoded.ID != tt.cursor.ID {
				t.Errorf("round-trip = %+v, want %+v", decoded, tt.cursor)
			}
		})
	}
}

func TestDecodeCursorTamperCases(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "empty string", token: ""},
		{name: "not valid base64", token: "!!!not-base64!!!"},
		{name: "valid base64, not JSON", token: "bm90IGpzb24"}, // "not json"
		{name: "valid JSON, wrong shape", token: encodeRaw(`{"foo":"bar"}`)},
		{name: "valid JSON, empty id", token: encodeRaw(`{"createdAt":"2026-08-10T12:00:00Z","id":""}`)},
		{name: "valid JSON, empty createdAt", token: encodeRaw(`{"createdAt":"","id":"candidate-1"}`)},
		{name: "valid JSON, unparseable createdAt", token: encodeRaw(`{"createdAt":"not-a-date","id":"candidate-1"}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeCursor(tt.token)
			if !errors.Is(err, domain.ErrInvalidPageToken) {
				t.Errorf("decodeCursor(%q) error = %v, want %v", tt.token, err, domain.ErrInvalidPageToken)
			}
		})
	}
}
