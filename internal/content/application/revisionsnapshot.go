// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// revisionSnapshotBlock is the on-disk shape of one element inside a
// content_document_revisions.data array (M14.6, D-ContentRevisions) — the same
// {blockTypeCode,position,data} triple PutBlocks already validates, never a second copy of
// content_blocks' own per-row shape. A block has never had an id stable across saves even before
// this milestone: Repository.ReplaceBlocks (this snapshot's predecessor) was a delete-then-insert
// that assigned every block a fresh row id on every PutBlocks call, so dropping per-block ids here
// changes nothing a caller could have relied on.
type revisionSnapshotBlock struct {
	BlockTypeCode string          `json:"blockTypeCode"`
	Position      int             `json:"position"`
	Data          json.RawMessage `json:"data"`
}

// marshalBlocksSnapshot serializes an already-validated block list into a revision's data column.
func marshalBlocksSnapshot(blocks []domain.BlockInput) (json.RawMessage, error) {
	out := make([]revisionSnapshotBlock, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, revisionSnapshotBlock{BlockTypeCode: b.BlockTypeCode, Position: b.Position, Data: b.Data})
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("content: marshal blocks snapshot: %w", err)
	}
	return data, nil
}

// unmarshalBlocksSnapshot reconstructs a response block list from a revision's stored snapshot.
// Each block's ID is synthesized as "<documentID>:<position>" rather than read back from a stored
// row id — see revisionSnapshotBlock's own comment on why that was never a stable identity to
// begin with. CreatedAt/UpdatedAt both take the owning revision's own timestamp: a snapshot has no
// finer-grained per-block history than "when was this whole revision saved."
func unmarshalBlocksSnapshot(documentID string, revisionCreatedAt time.Time, data json.RawMessage) ([]domain.Block, error) {
	var raw []revisionSnapshotBlock
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("content: unmarshal blocks snapshot: %w", err)
	}
	out := make([]domain.Block, 0, len(raw))
	for _, b := range raw {
		out = append(out, domain.Block{
			ID:            fmt.Sprintf("%s:%d", documentID, b.Position),
			DocumentID:    documentID,
			BlockTypeCode: b.BlockTypeCode,
			Position:      b.Position,
			Data:          b.Data,
			CreatedAt:     revisionCreatedAt,
			UpdatedAt:     revisionCreatedAt,
		})
	}
	return out, nil
}
