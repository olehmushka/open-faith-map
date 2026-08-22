// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the audit-log module's plain Go types — no Conjure, no SQL, no HTTP
// (docs/architecture/overview.md's transport → application → domain → adapters layering).
package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// ErrInvalidPageToken mirrors moderation's own error of the same name (M7,
// docs/modules/hardening.md's invariant) — a malformed pageToken must never be silently
// reinterpreted as "start from page 1".
var ErrInvalidPageToken = errors.New("pageToken is malformed or tampered")

// Entry is one append-only row of identity_audit_log — an admin mutation, recorded once, never
// edited. ActorPersonID/ActorPersonName are empty when the acting person was later deleted
// (actor_person_id is ON DELETE SET NULL — the log entry survives, the actor reference does not).
type Entry struct {
	ID              string
	ActorPersonID   string
	ActorPersonName string
	Action          string
	TargetKind      string
	TargetID        string
	Before          json.RawMessage
	After           json.RawMessage
	CreatedAt       time.Time
}

// Filter narrows ListEntries by actor/target/date — every field optional, ANDed together when set.
type Filter struct {
	ActorPersonID string
	TargetKind    string
	TargetID      string
	From          *time.Time
	To            *time.Time
}

// PageCursor is the decoded shape of an opaque pageToken — encodes the (createdAt, id) of the last
// row in the previous page for keyset pagination, same shape moderation's own PageCursor uses. The
// wire encoding (base64/JSON) is a transport-only concern; domain only knows these two fields.
type PageCursor struct {
	CreatedAt time.Time
	ID        string
}
