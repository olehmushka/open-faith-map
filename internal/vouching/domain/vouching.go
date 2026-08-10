// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the vouching module's plain Go types — no Conjure, no SQL, no HTTP
// (docs/architecture/overview.md's transport → application → domain → adapters layering).
package domain

import (
	"errors"
	"time"
)

// GuarantorStatusValue values match the DB CHECK constraint (migrations/0008_vouching.sql)
// verbatim — lowercase, per vouching.md's own literal SQL text. This is a deliberate, scoped
// deviation from the "Conjure enum matches DB CHECK verbatim, no case conversion" convention every
// other module follows: the Conjure GuarantorStatus enum stays uppercase (TRUSTED/REVOKED, matching
// every other Conjure enum in this repo), and the one-line case shim is confined to
// transport/convert.go.
type GuarantorStatusValue string

const (
	StatusTrusted GuarantorStatusValue = "trusted"
	StatusRevoked GuarantorStatusValue = "revoked"
)

// Vouch is an immutable event-log entry: guarantor vouched for claimant, at a point in time, for a
// specific congregation claim. Append-only (reject_mutation()-guarded) — never edited or deleted.
type Vouch struct {
	ID                 string
	GuarantorPersonRID string
	ClaimantPersonRID  string
	CongregationUnitID string
	Statement          *string
	CreatedAt          time.Time
}

type CreateVouchInput struct {
	GuarantorPersonRID          string
	ClaimantPersonRID           string
	CongregationUnitID          string
	GuarantorCongregationUnitID string
	Statement                   *string
}

// GuarantorStatus is the mutable overlay recording whether a guarantor is currently trusted. Its
// absence (no row in vouching_guarantor_status) means StatusTrusted — the table's own DEFAULT,
// synthesized by the store rather than requiring a row to exist up front.
type GuarantorStatus struct {
	GuarantorPersonRID string
	Status             GuarantorStatusValue
	RevokedAt          *time.Time
	RevokedReason      *string
	RevokedByPersonRID *string
	// UpdatedAt is unset when this record was synthesized (no underlying row exists).
	UpdatedAt *time.Time
}

var (
	ErrForbidden        = errors.New("caller does not hold the required target-scoped grant")
	ErrGuarantorRevoked = errors.New("guarantor currently holds revoked status and may not file a new vouch")
	// ErrGuarantorRevokeFanoutIncomplete wraps a successful revocation (the load-bearing status
	// write already committed) where one or more of the best-effort moderation-report fan-out
	// calls failed — vouching.md's invariant that revocation itself must take effect immediately,
	// even if the review queue isn't fully populated yet.
	ErrGuarantorRevokeFanoutIncomplete = errors.New("guarantor revoked, but not every affected vouch could be queued for moderator review")
)
