// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

const blockTypeColumns = `id, code, name, json_schema, status, sort_order`

func (s *Store) GetBlockTypeByCode(ctx context.Context, code string) (domain.BlockType, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+blockTypeColumns+` FROM openfaithmap.content_block_types
		WHERE code = $1 AND deleted_at IS NULL`,
		code,
	)
	bt, err := scanBlockType(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BlockType{}, domain.ErrBlockTypeNotFound
	}
	return bt, err
}

// ListActiveBlockTypes is the public read (ContentPublicService) — active types only, ordered for
// display/authoring.
func (s *Store) ListActiveBlockTypes(ctx context.Context) ([]domain.BlockType, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+blockTypeColumns+` FROM openfaithmap.content_block_types
		WHERE status = 'ACTIVE' AND deleted_at IS NULL
		ORDER BY sort_order ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("content: list block types: %w", err)
	}
	defer rows.Close()

	var out []domain.BlockType
	for rows.Next() {
		bt, err := scanBlockType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, bt)
	}
	return out, rows.Err()
}

func scanBlockType(row rowScanner) (domain.BlockType, error) {
	var bt domain.BlockType
	var status string
	if err := row.Scan(&bt.ID, &bt.Code, &bt.Name, &bt.JSONSchema, &status, &bt.SortOrder); err != nil {
		return domain.BlockType{}, err
	}
	bt.Status = domain.BlockTypeStatus(status)
	return bt, nil
}
