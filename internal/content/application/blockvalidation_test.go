// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// TestValidateBlockDataField proves M14.4's per-field error reporting: a schema-shape violation's
// domain.BlockDataInvalidError.Field names the offending top-level property when it can be safely
// determined, and is empty rather than leaking an attacker-chosen key otherwise. No DB is needed —
// validateBlockData only touches its domain.BlockType/data arguments.
func TestValidateBlockDataField(t *testing.T) {
	tests := []struct {
		name      string
		schema    string
		data      string
		wantField string
	}{
		{
			name:      "missing top-level required field",
			schema:    `{"type":"object","required":["text"],"additionalProperties":false,"properties":{"text":{"type":"string"}}}`,
			data:      `{}`,
			wantField: "text",
		},
		{
			name: "missing required field inside an array item",
			schema: `{"type":"object","required":["images"],"additionalProperties":false,"properties":{
				"images":{"type":"array","items":{"type":"object","required":["url","alt"],"additionalProperties":false,
					"properties":{"url":{"type":"string"},"alt":{"type":"string"}}}}}}`,
			data:      `{"images":[{"url":"https://example.org/a.jpg"}]}`,
			wantField: "images",
		},
		{
			name:      "unexpected property never leaks as a field",
			schema:    `{"type":"object","additionalProperties":false,"properties":{"a":{"type":"string"}}}`,
			data:      `{"unexpectedKeyAnAttackerChose":"x"}`,
			wantField: "",
		},
		{
			name: "nested column child missing a required field resolves to the outer field",
			schema: `{"type":"object","required":["columns"],"additionalProperties":false,"properties":{
				"columns":{"type":"array","items":{"type":"object","required":["blocks"],
					"properties":{"blocks":{"type":"array"}}}}}}`,
			data:      `{"columns":[{}]}`,
			wantField: "columns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockType := domain.BlockType{Code: "test_type", JSONSchema: json.RawMessage(tt.schema)}
			err := validateBlockData(blockType, 0, json.RawMessage(tt.data))

			var invalid *domain.BlockDataInvalidError
			if !errors.As(err, &invalid) {
				t.Fatalf("validateBlockData() = %v (%T), want *domain.BlockDataInvalidError", err, err)
			}
			if invalid.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", invalid.Field, tt.wantField)
			}
		})
	}
}
