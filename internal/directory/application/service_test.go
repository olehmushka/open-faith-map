// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"
	"testing"

	"github.com/olehmushka/open-faith-map/internal/directory/domain"
)

func TestDefaultGraph(t *testing.T) {
	if got := defaultGraph(""); got != domain.CanonicalGraphCode {
		t.Errorf("defaultGraph(\"\") = %q, want %q", got, domain.CanonicalGraphCode)
	}
	if got := defaultGraph("other"); got != "other" {
		t.Errorf("defaultGraph(%q) = %q, want unchanged", "other", got)
	}
}

// TestAddEdgeRejectsSelfLoopWithoutTouchingThePool proves the self-loop guard runs before any
// database access — a nil pool would panic if AddEdge tried to open a transaction first.
func TestAddEdgeRejectsSelfLoopWithoutTouchingThePool(t *testing.T) {
	svc := &Service{pool: nil}
	_, err := svc.AddEdge(context.Background(), "unit-a", "unit-a", "")
	if !errors.Is(err, domain.ErrEdgeCycle) {
		t.Errorf("AddEdge(self-loop) error = %v, want ErrEdgeCycle", err)
	}
}
