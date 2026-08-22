// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	auditlogdomain "github.com/olehmushka/open-faith-map/internal/auditlog/domain"
)

// encodeRaw base64-encodes arbitrary JSON text without going through the real wireCursor struct —
// lets the tamper tests exercise "well-formed base64/JSON but the wrong shape or values" without
// hand-rolling raw bytes for every case (mirrors moderation/transport/cursor_test.go's own helper).
func encodeRaw(json string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(json))
}

func TestEncodeDecodeAuditCursorRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		cursor auditlogdomain.PageCursor
	}{
		{name: "typical row", cursor: auditlogdomain.PageCursor{CreatedAt: time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC), ID: "entry-1"}},
		{name: "sub-second precision preserved", cursor: auditlogdomain.PageCursor{CreatedAt: time.Date(2026, 8, 10, 12, 0, 0, 123456789, time.UTC), ID: "entry-2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := decodeAuditCursor(encodeAuditCursor(tt.cursor))
			if err != nil {
				t.Fatalf("decodeAuditCursor(encodeAuditCursor(%v)) returned error: %v", tt.cursor, err)
			}
			if !decoded.CreatedAt.Equal(tt.cursor.CreatedAt) || decoded.ID != tt.cursor.ID {
				t.Errorf("round-trip = %+v, want %+v", decoded, tt.cursor)
			}
		})
	}
}

func TestDecodeAuditCursorTamperCases(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "empty string", token: ""},
		{name: "not valid base64", token: "!!!not-base64!!!"},
		{name: "valid base64, not JSON", token: "bm90IGpzb24"}, // "not json"
		{name: "valid JSON, wrong shape", token: encodeRaw(`{"foo":"bar"}`)},
		{name: "valid JSON, empty id", token: encodeRaw(`{"createdAt":"2026-08-10T12:00:00Z","id":""}`)},
		{name: "valid JSON, empty createdAt", token: encodeRaw(`{"createdAt":"","id":"entry-1"}`)},
		{name: "valid JSON, unparseable createdAt", token: encodeRaw(`{"createdAt":"not-a-date","id":"entry-1"}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeAuditCursor(tt.token)
			if !errors.Is(err, auditlogdomain.ErrInvalidPageToken) {
				t.Errorf("decodeAuditCursor(%q) error = %v, want %v", tt.token, err, auditlogdomain.ErrInvalidPageToken)
			}
		})
	}
}
