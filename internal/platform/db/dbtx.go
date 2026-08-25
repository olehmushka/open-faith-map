// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the minimal pgx surface sqlc-generated Queries need, satisfied identically by
// *pgxpool.Pool and pgx.Tx. It mirrors each <module>sql package's own generated DBTX interface
// (never hand-edited) so service-layer inTx closures can depend on this one shared type instead of
// importing a per-module generated package just for its interface.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
