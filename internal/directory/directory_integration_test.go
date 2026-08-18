// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// The closure-maintenance SQL (incremental extend/shrink, WITH RECURSIVE rebuild/verify) is where
// this module's real correctness risk lives — a unit test with a mocked store would prove nothing
// about it. This file proves it against a real Postgres instead, following this repo's own
// established integration-test convention (internal/coreintegration/client_integration_test.go): a
// real Go test, skipped unless pointed at a live database, runnable with:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/directory/... -run TestDirectoryClosureIntegration -v
//
// (adjust the DSN if not going through the docker-compose port mapping — DATABASE_URL inside the
// openfaithmap-api container itself points at the postgres service by its compose hostname, not
// localhost).
package directory_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/directory/application"
	"github.com/olehmushka/open-faith-map/internal/directory/domain"
)

func TestDirectoryClosureIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	// t.Cleanup runs AFTER the test function returns (and thus after any plain `defer` in this
	// function body already ran) — a deferred pool.Close() here would close the pool before a
	// t.Cleanup-registered DB-cleanup callback ever got to use it. Register the close via Cleanup
	// too, first, so LIFO ordering runs the DB cleanup (registered second, below) before it.
	t.Cleanup(pool.Close)

	var unitIDs []string
	graphID, graphCode := createTestGraph(t, ctx, pool)
	// Single, explicitly-ordered cleanup — directory_unit_edges references directory_units with
	// ON DELETE RESTRICT, so edges/closure must go before units, and units before the graph.
	// Errors are asserted, not swallowed — a silent cleanup failure here previously left 5 real rows
	// behind in the live dev database without the test ever reporting it.
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_closure WHERE graph_id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete closure rows: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_edges WHERE graph_id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete edges: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_closure_status WHERE graph_id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete closure status: %v", err)
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete unit %s: %v", id, err)
			}
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_graphs WHERE id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete graph: %v", err)
		}
	})

	svc := application.NewService(pool)

	// --- CreateUnit (root) -> CreateUnitWithEdge two levels deep -> Ancestors nearest-first.
	root, err := svc.CreateUnit(ctx, domain.Unit{Name: "Root"})
	if err != nil {
		t.Fatalf("CreateUnit(root): %v", err)
	}
	unitIDs = append(unitIDs, root.ID)

	child, err := svc.CreateUnitWithEdge(ctx, domain.Unit{Name: "Child"}, root.ID, graphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(child): %v", err)
	}
	unitIDs = append(unitIDs, child.ID)

	grandchild, err := svc.CreateUnitWithEdge(ctx, domain.Unit{Name: "Grandchild"}, child.ID, graphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(grandchild): %v", err)
	}
	unitIDs = append(unitIDs, grandchild.ID)

	ancestors, err := svc.Ancestors(ctx, grandchild.ID, graphCode)
	if err != nil {
		t.Fatalf("Ancestors: %v", err)
	}
	if len(ancestors) != 2 {
		t.Fatalf("Ancestors(grandchild) = %d entries, want 2 (child, root)", len(ancestors))
	}
	if ancestors[0].ID != child.ID || ancestors[1].ID != root.ID {
		t.Errorf("Ancestors(grandchild) = %+v, want [child, root] nearest-first", ancestors)
	}

	// --- AddEdge that would close a cycle is rejected.
	_, err = svc.AddEdge(ctx, root.ID, grandchild.ID, graphCode)
	if err != domain.ErrEdgeCycle {
		t.Errorf("AddEdge(grandchild -> root, a cycle) error = %v, want ErrEdgeCycle", err)
	}

	// --- A second, unrelated root plus AddEdge (not CreateUnitWithEdge) to exercise the incremental
	// extend path independently, then RemoveEdge to exercise the shrink path.
	other, err := svc.CreateUnit(ctx, domain.Unit{Name: "Other root"})
	if err != nil {
		t.Fatalf("CreateUnit(other): %v", err)
	}
	unitIDs = append(unitIDs, other.ID)

	leaf, err := svc.CreateUnit(ctx, domain.Unit{Name: "Leaf"})
	if err != nil {
		t.Fatalf("CreateUnit(leaf): %v", err)
	}
	unitIDs = append(unitIDs, leaf.ID)

	if _, err := svc.AddEdge(ctx, leaf.ID, other.ID, graphCode); err != nil {
		t.Fatalf("AddEdge(leaf -> other): %v", err)
	}
	leafAncestors, err := svc.Ancestors(ctx, leaf.ID, graphCode)
	if err != nil {
		t.Fatalf("Ancestors(leaf) after AddEdge: %v", err)
	}
	if len(leafAncestors) != 1 || leafAncestors[0].ID != other.ID {
		t.Fatalf("Ancestors(leaf) after AddEdge = %+v, want [other]", leafAncestors)
	}

	if err := svc.RemoveEdge(ctx, leaf.ID, other.ID, graphCode); err != nil {
		t.Fatalf("RemoveEdge(leaf, other): %v", err)
	}
	leafAncestorsAfterRemove, err := svc.Ancestors(ctx, leaf.ID, graphCode)
	if err != nil {
		t.Fatalf("Ancestors(leaf) after RemoveEdge: %v", err)
	}
	if len(leafAncestorsAfterRemove) != 0 {
		t.Errorf("Ancestors(leaf) after RemoveEdge = %+v, want none (shrink should have removed it)", leafAncestorsAfterRemove)
	}

	// --- RemoveEdge on an already-absent edge is a true no-op: succeeds, no error, no shrink to
	// break anything that's still there.
	if err := svc.RemoveEdge(ctx, leaf.ID, other.ID, graphCode); err != nil {
		t.Errorf("RemoveEdge on an already-absent edge should be a no-op, got: %v", err)
	}

	// --- VerifyClosure reports zero drift for a correctly-maintained graph.
	reports, err := svc.VerifyClosure(ctx, &graphCode)
	if err != nil {
		t.Fatalf("VerifyClosure: %v", err)
	}
	if len(reports) != 1 || reports[0].InDrift {
		t.Fatalf("VerifyClosure (before corruption) = %+v, want a single report with InDrift=false", reports)
	}

	// --- RebuildClosure from scratch agrees with the incrementally-maintained closure: the two
	// algorithms cross-checking each other is the strongest correctness signal available here.
	if _, err := svc.RebuildClosure(ctx, &graphCode); err != nil {
		t.Fatalf("RebuildClosure: %v", err)
	}
	postRebuild, err := svc.VerifyClosure(ctx, &graphCode)
	if err != nil {
		t.Fatalf("VerifyClosure (post-rebuild): %v", err)
	}
	if len(postRebuild) != 1 || postRebuild[0].InDrift {
		t.Fatalf("VerifyClosure (post-rebuild) = %+v, want zero drift", postRebuild)
	}
	postRebuildAncestors, err := svc.Ancestors(ctx, grandchild.ID, graphCode)
	if err != nil {
		t.Fatalf("Ancestors (post-rebuild): %v", err)
	}
	if len(postRebuildAncestors) != 2 {
		t.Errorf("Ancestors(grandchild) post-rebuild = %+v, want the same 2 ancestors as before", postRebuildAncestors)
	}

	// --- Manually corrupt one closure row and confirm VerifyClosure's drift detection catches it.
	if _, err := pool.Exec(ctx, `
		DELETE FROM openfaithmap.directory_unit_closure
		WHERE graph_id = $1 AND ancestor_id = $2 AND descendant_id = $3`, graphID, root.ID, grandchild.ID); err != nil {
		t.Fatalf("corrupt closure row: %v", err)
	}
	drifted, err := svc.VerifyClosure(ctx, &graphCode)
	if err != nil {
		t.Fatalf("VerifyClosure (after corruption): %v", err)
	}
	if len(drifted) != 1 || !drifted[0].InDrift || drifted[0].MissingCount != 1 {
		t.Errorf("VerifyClosure (after corruption) = %+v, want InDrift=true, MissingCount=1", drifted)
	}

	// --- ListUnits (M10.7): a code/name ILIKE search finds the grandchild by its distinctive name.
	found, err := svc.ListUnits(ctx, "Grandchild", 10)
	if err != nil {
		t.Fatalf("ListUnits: %v", err)
	}
	var sawGrandchild bool
	for _, u := range found {
		if u.ID == grandchild.ID {
			sawGrandchild = true
		}
	}
	if !sawGrandchild {
		t.Errorf("ListUnits(%q) = %+v, want it to include the grandchild unit", "Grandchild", found)
	}
}

// createTestGraph inserts a throwaway authority-bearing graph directly via SQL — Service exposes no
// graph-management methods this milestone (nothing needs custom graph creation yet).
func createTestGraph(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (id, code string) {
	t.Helper()
	code = "test-directory-" + t.Name()
	err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.directory_graphs (id, code, name, is_authority_bearing)
		VALUES (openfaithmap.new_id(3,1,2), $1, $1, true)
		RETURNING id`, code).Scan(&id)
	if err != nil {
		t.Fatalf("createTestGraph: %v", err)
	}
	return id, code
}
