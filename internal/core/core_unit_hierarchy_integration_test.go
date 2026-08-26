// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M12.7's RootUnit/UnitChildren against a real Postgres instance: RootUnit resolves to the
// same id every other root-guard in this package already trusts (seed.RootUnitID), and UnitChildren
// returns exactly the direct (one-hop) children of a unit — no error, just an empty list, for a unit
// with none. Both are deliberately ungated reads (no requireSubject call), matching
// GetUnit/ListUnits/UnitAncestors — this test calls them with a bare context.Background(), on
// purpose, to prove that. See core_unit_lifecycle_integration_test.go's own header comment for the
// invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestUnitHierarchyIntegration -v
package core_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	coreapplication "github.com/olehmushka/open-faith-map/internal/core/application"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
)

func TestUnitHierarchyIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	// RootUnit/UnitChildren only ever touch s.directory/s.rootUnitID — every other collaborator can
	// stay nil, the same minimal-construction precedent this package's read-only tests already use.
	dir := directoryapplication.NewService(pool)
	coreApp := coreapplication.NewService(dir, nil, nil, nil, nil, nil, nil, pool, seed.RootUnitID)

	var unitIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_edges WHERE parent_id = $1 OR child_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete edges touching %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_closure WHERE ancestor_id = $1 OR descendant_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete closure rows touching %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete unit %s: %v", id, err)
			}
		}
	})

	// --- RootUnit resolves to the same id ErrRootUnitProtected already guards (seed.RootUnitID) — a
	// bare context.Background(), no subject, proving the "ungated beyond session" read.
	root, err := coreApp.RootUnit(ctx)
	if err != nil {
		t.Fatalf("RootUnit: %v", err)
	}
	if root.ID != seed.RootUnitID {
		t.Errorf("RootUnit().ID = %q, want %q (seed.RootUnitID)", root.ID, seed.RootUnitID)
	}

	// --- UnitChildren, one hop only: a fresh unit with a single real child under it.
	parent, err := dir.CreateUnit(ctx, directorydomain.Unit{Name: "M12.7 Hierarchy Parent"})
	if err != nil {
		t.Fatalf("CreateUnit(parent): %v", err)
	}
	unitIDs = append(unitIDs, parent.ID)

	child, err := dir.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M12.7 Hierarchy Child"}, parent.ID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(child): %v", err)
	}
	unitIDs = append(unitIDs, child.ID)

	children, err := coreApp.UnitChildren(ctx, parent.ID, 0)
	if err != nil {
		t.Fatalf("UnitChildren(parent): %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("UnitChildren(parent) = %+v, want exactly [child]", children)
	}

	// --- A leaf (the child itself) has none.
	leafChildren, err := coreApp.UnitChildren(ctx, child.ID, 0)
	if err != nil {
		t.Fatalf("UnitChildren(leaf): %v", err)
	}
	if len(leafChildren) != 0 {
		t.Errorf("UnitChildren(leaf) = %+v, want none", leafChildren)
	}

	// --- An unknown (but well-formed) unit id is not a Go error — it's an empty result, the same "no
	// existence check" shape every other read in this ungated family (Ancestors/Descendants) already
	// has. unit_id binds to a uuid column, so this must be syntactically valid, just not assigned to
	// any real unit.
	unknownChildren, err := coreApp.UnitChildren(ctx, "00000000-0000-0000-0000-000000000000", 0)
	if err != nil {
		t.Fatalf("UnitChildren(unknown id): %v", err)
	}
	if len(unknownChildren) != 0 {
		t.Errorf("UnitChildren(unknown id) = %+v, want none", unknownChildren)
	}
}
