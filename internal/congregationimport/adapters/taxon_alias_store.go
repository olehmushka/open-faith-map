// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const taxonAliasSelectColumns = `
	id, source_code, alias_text, taxon_id, created_by_person_rid, created_at, updated_at`

// CreateTaxonAlias implements the unique-alias constraint race-safely: INSERT, catch the
// unique-violation, translate — never check-then-insert (TOCTOU), matching content's own
// InsertSite precedent (internal/content/adapters/site_store.go).
func (s *Store) CreateTaxonAlias(ctx context.Context, sourceCode *string, aliasText, taxonID, createdByPersonRID string) (domain.TaxonAlias, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.congregationimport_taxon_aliases (source_code, alias_text, taxon_id, created_by_person_rid)
		VALUES ($1, $2, $3, $4)
		RETURNING `+taxonAliasSelectColumns,
		sourceCode, aliasText, taxonID, createdByPersonRID,
	)
	a, err := scanTaxonAlias(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.TaxonAlias{}, domain.ErrAliasConflict
	}
	return a, err
}

// ListAliasesForMatching returns every alias applicable to sourceCode (source-scoped ones first,
// then global ones) — small enough (operator-maintained, not scraped) to load in full and let the
// caller do substring/keyword matching against a candidate's free-text name, rather than requiring
// an exact match no real scraped name would ever produce.
func (s *Store) ListAliasesForMatching(ctx context.Context, sourceCode string) ([]domain.TaxonAlias, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+taxonAliasSelectColumns+` FROM openfaithmap.congregationimport_taxon_aliases
		WHERE source_code = $1 OR source_code IS NULL
		ORDER BY source_code NULLS LAST`,
		sourceCode,
	)
	if err != nil {
		return nil, fmt.Errorf("congregationimport: list taxon aliases: %w", err)
	}
	defer rows.Close()

	var out []domain.TaxonAlias
	for rows.Next() {
		a, err := scanTaxonAlias(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListAllTaxonAliases returns every alias, across every source, for the alias-management UI
// (production-hardening pass) — distinct from ListAliasesForMatching, which always scopes to one
// connector's own matching pass.
func (s *Store) ListAllTaxonAliases(ctx context.Context) ([]domain.TaxonAlias, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+taxonAliasSelectColumns+` FROM openfaithmap.congregationimport_taxon_aliases
		ORDER BY source_code NULLS LAST, alias_text`,
	)
	if err != nil {
		return nil, fmt.Errorf("congregationimport: list all taxon aliases: %w", err)
	}
	defer rows.Close()

	var out []domain.TaxonAlias
	for rows.Next() {
		a, err := scanTaxonAlias(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanTaxonAlias(row rowScanner) (domain.TaxonAlias, error) {
	var a domain.TaxonAlias
	if err := row.Scan(&a.ID, &a.SourceCode, &a.AliasText, &a.TaxonID, &a.CreatedByPersonRID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return domain.TaxonAlias{}, err
	}
	return a, nil
}
