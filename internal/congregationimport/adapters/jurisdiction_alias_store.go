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

const jurisdictionAliasSelectColumns = `
	id, source_code, alias_text, jurisdiction_unit_id, created_by_person_rid, created_at, updated_at`

// CreateJurisdictionAlias implements the unique-alias constraint race-safely: INSERT, catch the
// unique-violation, translate — mirrors CreateTaxonAlias exactly.
func (s *Store) CreateJurisdictionAlias(ctx context.Context, sourceCode *string, aliasText, jurisdictionUnitID, createdByPersonRID string) (domain.JurisdictionAlias, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.congregationimport_jurisdiction_aliases (source_code, alias_text, jurisdiction_unit_id, created_by_person_rid)
		VALUES ($1, $2, $3, $4)
		RETURNING `+jurisdictionAliasSelectColumns,
		sourceCode, aliasText, jurisdictionUnitID, createdByPersonRID,
	)
	a, err := scanJurisdictionAlias(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.JurisdictionAlias{}, domain.ErrAliasConflict
	}
	return a, err
}

// ListJurisdictionAliasesForMatching mirrors ListAliasesForMatching exactly (taxon_alias_store.go):
// small, operator-maintained, loaded in full so the caller can substring-match against a candidate's
// free-text jurisdiction hint.
func (s *Store) ListJurisdictionAliasesForMatching(ctx context.Context, sourceCode string) ([]domain.JurisdictionAlias, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+jurisdictionAliasSelectColumns+` FROM openfaithmap.congregationimport_jurisdiction_aliases
		WHERE source_code = $1 OR source_code IS NULL
		ORDER BY source_code NULLS LAST`,
		sourceCode,
	)
	if err != nil {
		return nil, fmt.Errorf("congregationimport: list jurisdiction aliases: %w", err)
	}
	defer rows.Close()

	var out []domain.JurisdictionAlias
	for rows.Next() {
		a, err := scanJurisdictionAlias(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListAllJurisdictionAliases returns every alias, across every source, for the alias-management UI
// (production-hardening pass) — distinct from ListJurisdictionAliasesForMatching, which always
// scopes to one connector's own matching pass.
func (s *Store) ListAllJurisdictionAliases(ctx context.Context) ([]domain.JurisdictionAlias, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+jurisdictionAliasSelectColumns+` FROM openfaithmap.congregationimport_jurisdiction_aliases
		ORDER BY source_code NULLS LAST, alias_text`,
	)
	if err != nil {
		return nil, fmt.Errorf("congregationimport: list all jurisdiction aliases: %w", err)
	}
	defer rows.Close()

	var out []domain.JurisdictionAlias
	for rows.Next() {
		a, err := scanJurisdictionAlias(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanJurisdictionAlias(row rowScanner) (domain.JurisdictionAlias, error) {
	var a domain.JurisdictionAlias
	if err := row.Scan(&a.ID, &a.SourceCode, &a.AliasText, &a.JurisdictionUnitID, &a.CreatedByPersonRID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return domain.JurisdictionAlias{}, err
	}
	return a, nil
}
