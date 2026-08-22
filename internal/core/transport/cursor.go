// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/base64"
	"encoding/json"
	"time"

	auditlogdomain "github.com/olehmushka/open-faith-map/internal/auditlog/domain"
)

// wireCursor is the on-the-wire shape of listAuditLog's pageToken — base64(JSON) of the
// (createdAt, id) of the last row in the previous page, same shape moderation's own cursor uses (M7,
// docs/modules/hardening.md).
type wireCursor struct {
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
}

func encodeAuditCursor(c auditlogdomain.PageCursor) string {
	b, _ := json.Marshal(wireCursor{CreatedAt: c.CreatedAt.Format(time.RFC3339Nano), ID: c.ID}) // struct-only Marshal cannot fail
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeAuditCursor parses an incoming pageToken. Any malformed shape — bad base64, non-JSON bytes,
// wrong fields, empty values — funnels into the same auditlogdomain.ErrInvalidPageToken, never
// silently reinterpreted as "start from page 1" (hardening.md's invariant).
func decodeAuditCursor(token string) (auditlogdomain.PageCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return auditlogdomain.PageCursor{}, auditlogdomain.ErrInvalidPageToken
	}
	var wc wireCursor
	if err := json.Unmarshal(b, &wc); err != nil {
		return auditlogdomain.PageCursor{}, auditlogdomain.ErrInvalidPageToken
	}
	if wc.ID == "" || wc.CreatedAt == "" {
		return auditlogdomain.PageCursor{}, auditlogdomain.ErrInvalidPageToken
	}
	createdAt, err := time.Parse(time.RFC3339Nano, wc.CreatedAt)
	if err != nil {
		return auditlogdomain.PageCursor{}, auditlogdomain.ErrInvalidPageToken
	}
	return auditlogdomain.PageCursor{CreatedAt: createdAt, ID: wc.ID}, nil
}
