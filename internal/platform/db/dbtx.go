// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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

// NullableTime converts a sqlc-generated pgtype.Timestamptz (sqlc.yaml deliberately leaves nullable
// timestamptz columns at this default rather than overriding to *time.Time — see sqlc.yaml's own
// comment) to the *time.Time every domain struct's nullable timestamp field expects.
func NullableTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// NullableTimeArg is NullableTime's inverse, for passing a domain *time.Time into a sqlc Params
// field typed pgtype.Timestamptz.
func NullableTimeArg(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
