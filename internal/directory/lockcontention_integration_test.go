// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M10.9's lock-contention proof: a second transaction attempting to extend the same graph's closure
// while a first one already holds directory_graphs' row lock (FOR NO KEY UPDATE,
// adapters.Store.LockGraphForClosure) must genuinely block until the first commits — not
// interleave. Same DATABASE_URL-gated shape as directory_integration_test.go:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/directory/... -run TestLockContentionIntegration -v
package directory_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/directory/adapters"
	"github.com/olehmushka/open-faith-map/internal/directory/application"
	"github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
)

func TestLockContentionIntegration(t *testing.T) {
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

	dirSvc := application.NewService(pool)
	var graphID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.directory_graphs WHERE code = 'canonical'`).Scan(&graphID); err != nil {
		t.Fatalf("look up canonical graph id: %v", err)
	}

	// Two free-floating units, no edge yet — each side attaches its own as a fresh child of root, so
	// InsertEdge's own unique constraint can't be what serializes them; only the graph lock can.
	childA, err := dirSvc.CreateUnit(ctx, domain.Unit{Name: "Lock Contention Test Child A", State: domain.StateActive})
	if err != nil {
		t.Fatalf("create childA: %v", err)
	}
	childB, err := dirSvc.CreateUnit(ctx, domain.Unit{Name: "Lock Contention Test Child B", State: domain.StateActive})
	if err != nil {
		t.Fatalf("create childB: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range []string{childA.ID, childB.ID} {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_closure WHERE ancestor_id = $1 OR descendant_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete closure for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_edges WHERE child_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete edges for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete unit %s: %v", id, err)
			}
		}
	})

	// Side A: acquire the lock first, signal once held, hold it for a long, unmistakable window,
	// then release. Deliberately sequenced (not a symmetric race) so side B only ever attempts its
	// own acquisition after CONFIRMED knowledge that A already holds it — a symmetric "start both at
	// once" race is flaky by construction here, since a lock-free B could simply finish (commit)
	// before A's own acquisition attempt ever reaches the server, proving nothing either way.
	aHasLock := make(chan struct{})
	aErrCh := make(chan error, 1)
	var aReleasedAt time.Time

	go func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			aErrCh <- err
			close(aHasLock)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		store := adapters.NewStore(tx)
		if err := store.LockGraphForClosure(ctx, graphID); err != nil {
			aErrCh <- err
			close(aHasLock)
			return
		}
		close(aHasLock) // signal: A now holds the row lock

		time.Sleep(300 * time.Millisecond)

		if _, err := store.InsertEdge(ctx, graphID, seed.RootUnitID, childA.ID); err != nil {
			aErrCh <- err
			return
		}
		if err := store.ExtendClosureForEdge(ctx, graphID, seed.RootUnitID, childA.ID); err != nil {
			aErrCh <- err
			return
		}
		err = tx.Commit(ctx)
		aReleasedAt = time.Now()
		aErrCh <- err
	}()

	<-aHasLock // block here until A has genuinely acquired the lock (not just started)

	// Side B: attempt the SAME lock now, while A still holds it (300ms sleep is well underway).
	bAttemptStart := time.Now()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("side B: pool.Begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	storeB := adapters.NewStore(tx)
	if err := storeB.LockGraphForClosure(ctx, graphID); err != nil {
		t.Fatalf("side B: LockGraphForClosure: %v", err)
	}
	bAcquiredAt := time.Now()
	if _, err := storeB.InsertEdge(ctx, graphID, seed.RootUnitID, childB.ID); err != nil {
		t.Fatalf("side B: InsertEdge: %v", err)
	}
	if err := storeB.ExtendClosureForEdge(ctx, graphID, seed.RootUnitID, childB.ID); err != nil {
		t.Fatalf("side B: ExtendClosureForEdge: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("side B: commit: %v", err)
	}

	if err := <-aErrCh; err != nil {
		t.Fatalf("side A: %v", err)
	}

	blockedFor := bAcquiredAt.Sub(bAttemptStart)
	t.Logf("side B attempted the lock at %s, acquired it at %s (blocked %s) — side A released it at %s",
		bAttemptStart.Format(time.RFC3339Nano), bAcquiredAt.Format(time.RFC3339Nano), blockedFor, aReleasedAt.Format(time.RFC3339Nano))

	// The real proof: B's acquisition cannot have happened before A released — if the row lock did
	// not serialize them, B would acquire near-instantly (sub-millisecond) instead of waiting out
	// most of A's 300ms hold.
	if bAcquiredAt.Before(aReleasedAt) {
		t.Errorf("side B acquired the lock at %s, BEFORE side A released it at %s — the row lock did not serialize the two transactions",
			bAcquiredAt.Format(time.RFC3339Nano), aReleasedAt.Format(time.RFC3339Nano))
	}
	if blockedFor < 200*time.Millisecond {
		t.Errorf("side B was blocked for only %s, want close to A's 300ms hold — the row lock may not be serializing correctly", blockedFor)
	}
}
