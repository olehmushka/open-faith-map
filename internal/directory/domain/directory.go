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
