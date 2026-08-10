// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"fmt"

	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

const blockColumns = `b.id, b.document_id, b.block_type_id, t.code, b.position, b.data, b.created_at, b.updated_at`

func (s *Store) ListBlocks(ctx context.Context, documentID string) ([]domain.Block, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+blockColumns+`
		FROM openfaithmap.content_blocks b
		JOIN openfaithmap.content_block_types t ON t.id = b.block_type_id
		WHERE b.document_id = $1 AND b.deleted_at IS NULL
		ORDER BY b.position ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("content: list blocks: %w", err)
	}
	defer rows.Close()

	var out []domain.Block
	for rows.Next() {
		block, err := scanBlock(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, block)
	}
	return out, rows.Err()
}

// ReplaceBlocks is a transactional delete-then-insert — application.Service has already validated
// every block's data against its type's json_schema and rejected duplicate positions before this
// is ever called, so this method trusts its input completely.
func (s *Store) ReplaceBlocks(ctx context.Context, documentID string, blocks []domain.BlockInput) ([]domain.Block, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("content: replace blocks: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM openfaithmap.content_blocks WHERE document_id = $1`, documentID); err != nil {
		return nil, fmt.Errorf("content: replace blocks: delete: %w", err)
	}

	for _, b := range blocks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO openfaithmap.content_blocks (document_id, block_type_id, position, data)
			SELECT $1, t.id, $2, $3 FROM openfaithmap.content_block_types t WHERE t.code = $4 AND t.deleted_at IS NULL`,
			documentID, b.Position, []byte(b.Data), b.BlockTypeCode,
		); err != nil {
			return nil, fmt.Errorf("content: replace blocks: insert position %d: %w", b.Position, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("content: replace blocks: commit: %w", err)
	}
	return s.ListBlocks(ctx, documentID)
}

func scanBlock(row rowScanner) (domain.Block, error) {
	var b domain.Block
	if err := row.Scan(&b.ID, &b.DocumentID, &b.BlockTypeID, &b.BlockTypeCode, &b.Position, &b.Data, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return domain.Block{}, err
	}
	return b, nil
}
