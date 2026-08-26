// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the directory module's types: the unit hierarchy (Unit/Graph/Edge) and its
// materialized transitive closure. Ported from ../go-oikumenea/internal/tenant, trimmed per
// D-CorePortScope: no organizations/domains/unit-kinds/visibility (single-tenant product, no
// shadow-unit concept — see docs/architecture/decisions.md's D-CorePortScope amendment).
package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrUnitNotFound  = errors.New("unit not found")
	ErrGraphNotFound = errors.New("graph not found")
	// ErrEdgeCycle: a self-loop, or a new parent->child edge whose child already reaches the parent
	// in the same graph.
	ErrEdgeCycle = errors.New("edge would create a cycle in its graph")
	// ErrEdgeExists: this exact (graph, parent, child) edge already exists — distinct from
	// ErrEdgeCycle so a caller doing resumable multi-step work (internal/registration's re-parenting
	// state machine) can treat a repeat AddEdge as success rather than a real cycle rejection.
	ErrEdgeExists = errors.New("edge already exists")
	// ErrUnitHasChildren: DeleteUnit's orphan-protection — a unit with a live parent->child edge in
	// any graph cannot be soft-deleted (M12.1).
	ErrUnitHasChildren = errors.New("unit has child units")
	// ErrUnitHasNoCurrentParent: Move/CurrentParent found no live parent edge for the unit in the
	// graph (e.g. it is that graph's root) — there is nothing to move it from (M12.2).
	ErrUnitHasNoCurrentParent = errors.New("unit has no current parent in this graph")
	// ErrUnitMoveConflict: Move's unitID already has a live (non-FAILED) job targeting a different
	// parent than the one just requested — the caller must resolve it (retry with the same
	// newParentUnitID to resume, or wait for it to fail out) before starting a move elsewhere (M12.2).
	ErrUnitMoveConflict = errors.New("unit already has a live move job targeting a different parent")
	// ErrUnitAlreadyAtParent: Move's newParentUnitID is already unitID's current parent — rejected
	// upfront, never started as a job. The add-before-remove state machine cannot represent a
	// same-parent move: with only one edge to begin with, "add the new edge" no-ops (it already
	// exists) and "remove the old edge" deletes that same edge, orphaning the unit with zero parents
	// (found in browser verification, 2026-08-26; M12.2).
	ErrUnitAlreadyAtParent = errors.New("unit is already at the requested parent")
)

// State is a unit's lifecycle state.
type State string

const (
	StateActive    State = "active"
	StateSuspended State = "suspended"
	StateArchived  State = "archived"
)

// Unit is directory_units — a node in the hierarchy graph.
type Unit struct {
	ID        string
	Code      string // optional, mutable, unique among active coded units
	Name      string
	Level     *int16 // optional ordinal for sort/filter; never a PDP input
	State     State
	Metadata  json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeleteEligibility is UnitDeleteEligibility's read-only preview of DeleteUnit's own orphan-protection
// checks (M12.5) — CanDelete is the AND of the three negations plus !IsRoot, computed once
// server-side so the client never has to re-derive the rule.
type DeleteEligibility struct {
	IsRoot                   bool
	HasChildren              bool
	HasOrgProfile            bool
	HasActiveRoleAssignments bool
	CanDelete                bool
}

// Graph is directory_graphs — a named hierarchy. "canonical" is the default + authority-bearing
// graph the PDP cascades subtree grants over (D-InProcessAuthz).
type Graph struct {
	ID                 string
	Code               string
	Name               string
	IsDefault          bool
	IsAuthorityBearing bool
}

// Edge is directory_unit_edges — a reified parent->child edge within one graph.
type Edge struct {
	ID        string
	GraphID   string
	GraphCode string
	ParentID  string
	ChildID   string
	CreatedAt time.Time
}

// UnitRef is the shape Ancestors/Descendants return: enough to render a breadcrumb or subtree list
// without a second round trip per unit.
type UnitRef struct {
	ID    string
	Code  string
	Name  string
	Depth int
}

// MoveStatus is a MoveJob's resumable state machine step (M12.2, generalized from
// internal/registration's former private ReparentStatus).
type MoveStatus string

const (
	MovePending        MoveStatus = "PENDING"
	MoveNewEdgeAdded   MoveStatus = "NEW_EDGE_ADDED"
	MoveOldEdgeRemoved MoveStatus = "OLD_EDGE_REMOVED"
	MoveVerified       MoveStatus = "VERIFIED"
	MoveFailed         MoveStatus = "FAILED"
)

// MoveJob is directory_unit_move_jobs — one move attempt's durable, resumable state (M12.2). At most
// one non-FAILED job exists per (GraphID, UnitID) at a time (the store's own unique index).
type MoveJob struct {
	ID                  string
	GraphID             string
	UnitID              string
	OldParentUnitID     string
	NewParentUnitID     string
	Status              MoveStatus
	PerformedByPersonID string
	Error               *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ClosureReport is RebuildClosure/VerifyClosure's per-graph result.
type ClosureReport struct {
	GraphCode    string
	MissingCount int
	ExtraCount   int
	InDrift      bool
	Sample       json.RawMessage
}

// CanonicalGraphCode is the default, always-authority-bearing graph
// (migrations/0022_core_seed.sql's seed row) — the analog of upstream's "command" graph.
const CanonicalGraphCode = "canonical"
