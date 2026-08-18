// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"errors"
)

// ErrPermissionDenied is the PDP's negative decision surfaced to callers.
var ErrPermissionDenied = errors.New("permission denied")

// Scope is a role assignment's reach (D-Inherit). `unit` applies at target only; `subtree` cascades
// to target + all descendants in the assignment's graph, when that graph is authority-bearing.
type Scope string

const (
	ScopeUnit    Scope = "unit"
	ScopeSubtree Scope = "subtree"
)

// ClosurePort is the directory-graph closure surface the PDP depends on (cross-module query,
// D-InProcessAuthz amendment #4). internal/directory implements it; internal/authz/domain owns the
// interface and imports no other module — that inversion is what keeps M10.3 independent of M10.4.
type ClosurePort interface {
	// IsAncestorOrSelf reports whether ancestorUnitID reaches descendantUnitID in the graph's
	// closure. The PDP additionally treats target == unit as self-authorized WITHOUT this call.
	IsAncestorOrSelf(ctx context.Context, graphID, ancestorUnitID, descendantUnitID string) (bool, error)
	// IsAuthorityBearing reports whether the graph cascades authority (D-DirectoryGraphs). A subtree
	// grant on a directory-only graph confers nothing in the PDP.
	IsAuthorityBearing(ctx context.Context, graphID string) (bool, error)
}
