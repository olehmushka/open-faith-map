// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package db holds small cross-cutting Postgres helpers that don't belong to any one module.
package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LockBootSeed serializes the boot-time seeding section (the first-admin bootstrap, internal/identity/
// bootstrap) across replicas. The value is arbitrary but must be stable and unique within this
// database.
const LockBootSeed int64 = 0x01_6F_69_6B_B0_07_5E_ED

// WithAdvisoryLock runs fn while holding the session-level advisory lock key, taken on a dedicated
// pooled connection (session locks belong to a connection; the pool must not recycle it mid-fn). It
// BLOCKS until the lock is granted — for boot seeding that is the point: the loser waits for the
// winner's seed to finish, then finds everything already seeded. The lock is released (and the
// connection returned) when fn returns.
func WithAdvisoryLock(ctx context.Context, pool *pgxpool.Pool, key int64, fn func(context.Context) error) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		return err
	}
	defer func() {
		// Best-effort unlock; if it fails the session is broken and the lock dies with the connection.
		_, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", key)
	}()
	return fn(ctx)
}
