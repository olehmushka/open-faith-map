// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the membership module's types: the unit-owned billet (Position) and the
// reified person->unit filling (Membership). Ported from
// ../go-oikumenea/internal/membership/domain, trimmed hard per D-CorePortScope (no rank/order
// provenance — internal/rank and the order module are both dropped) and to exactly what this repo's
// one caller (internal/registration's ensurePosition/ensureFilled) needs: create a position, list a
// unit's positions (to resume after a conflict), and fill a position. No facets/stats/reach-visibility
// machinery — this repo's own internal/authz PDP gates access at the unit level, not per-record.
package domain

import (
	"errors"
	"time"
)

var (
	ErrPositionNotFound      = errors.New("membership: position not found")
	ErrPositionConflict      = errors.New("membership: position code already exists in this unit")
	ErrPositionAlreadyFilled = errors.New("membership: position already has an active filling")
)

// Position is a unit-owned billet (membership_positions).
type Position struct {
	ID        string
	UnitID    string
	Code      string
	Title     string
	Status    string // "active" | "abolished"
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Membership is a person's filling of a position (membership_memberships). This port only ever
// creates position-filling memberships (FillPosition) — plain position-less belonging isn't a
// current caller's need, so PositionID is always set here (unlike upstream, where it's optional).
type Membership struct {
	ID            string
	PersonID      string
	UnitID        string
	PositionID    string
	Status        string // "active" | "ended"
	EffectiveFrom time.Time
}
