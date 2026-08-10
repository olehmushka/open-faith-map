// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the vouching module's Postgres store. Hand-written pgx (matches every other
// module's documented single-module simplification — sqlc not required), split into one file per
// table for readability; one Store struct/package across both.
package adapters

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type rowScanner interface {
	Scan(dest ...any) error
}
