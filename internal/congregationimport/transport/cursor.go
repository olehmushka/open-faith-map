// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// wireCursor is the on-the-wire shape of a pageToken — base64(JSON) of the (createdAt, id) of the
// last row in the previous page, byte-for-byte the same shape moderation's own M7 pagination fix
// uses (internal/moderation/transport/cursor.go). Domain knows only the decoded fields
// (domain.PageCursor); base64/JSON encoding is a transport-only concern.
type wireCursor struct {
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
}

// encodeCursor renders a domain.PageCursor as the opaque token returned in nextPageToken.
// URL-safe, unpadded base64 (RawURLEncoding) since the token rides in a ?pageToken= query string.
func encodeCursor(c domain.PageCursor) string {
	b, _ := json.Marshal(wireCursor{CreatedAt: c.CreatedAt.Format(time.RFC3339Nano), ID: c.ID}) // struct-only Marshal cannot fail
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor parses an incoming pageToken. Any malformed shape — bad base64, non-JSON bytes,
// wrong fields, empty values — funnels into the same domain.ErrInvalidPageToken, never silently
// reinterpreted as "start from page 1".
func decodeCursor(token string) (domain.PageCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return domain.PageCursor{}, domain.ErrInvalidPageToken
	}
	var wc wireCursor
	if err := json.Unmarshal(b, &wc); err != nil {
		return domain.PageCursor{}, domain.ErrInvalidPageToken
	}
	if wc.ID == "" || wc.CreatedAt == "" {
		return domain.PageCursor{}, domain.ErrInvalidPageToken
	}
	createdAt, err := time.Parse(time.RFC3339Nano, wc.CreatedAt)
	if err != nil {
		return domain.PageCursor{}, domain.ErrInvalidPageToken
	}
	return domain.PageCursor{CreatedAt: createdAt, ID: wc.ID}, nil
}
