// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the content module's Postgres store. Hand-written pgx (matches
// internal/registration's documented single-module simplification — sqlc not required), split into
// one file per table for readability; one Store struct/package across all four.
package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const siteColumns = `id, congregation_unit_rid, slug, theme, created_at, updated_at`

// InsertSite implements U5's resolution race-safely: INSERT, catch the unique-violation, translate
// — never check-then-insert (TOCTOU).
func (s *Store) InsertSite(ctx context.Context, in domain.CreateSiteInput) (domain.Site, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.content_sites (congregation_unit_rid, slug)
		VALUES ($1, $2)
		RETURNING `+siteColumns,
		in.CongregationUnitRID, in.Slug,
	)
	site, err := scanSite(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "content_sites_slug_idx" {
			return domain.Site{}, &domain.SlugTakenError{Slug: in.Slug, Scope: "site"}
		}
		return domain.Site{}, fmt.Errorf("content: site already exists for this congregation unit: %w", err)
	}
	return site, err
}

func (s *Store) GetSiteByID(ctx context.Context, id string) (domain.Site, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+siteColumns+` FROM openfaithmap.content_sites WHERE id = $1 AND deleted_at IS NULL`, id)
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	return site, err
}

func (s *Store) GetSiteByUnit(ctx context.Context, congregationUnitRID string) (domain.Site, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+siteColumns+` FROM openfaithmap.content_sites WHERE congregation_unit_rid = $1 AND deleted_at IS NULL`, congregationUnitRID)
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	return site, err
}

func (s *Store) UpdateSiteTheme(ctx context.Context, id string, theme []byte) (domain.Site, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.content_sites SET theme = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+siteColumns,
		id, theme,
	)
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	return site, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSite(row rowScanner) (domain.Site, error) {
	var site domain.Site
	if err := row.Scan(&site.ID, &site.CongregationUnitRID, &site.Slug, &site.Theme, &site.CreatedAt, &site.UpdatedAt); err != nil {
		return domain.Site{}, err
	}
	return site, nil
}
