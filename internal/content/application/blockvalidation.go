// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"bytes"
	"fmt"

	"github.com/olehmushka/open-faith-map/internal/content/domain"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// validateBlockData resolves blockTypeCode to an active block-type row and validates data against
// its json_schema — content.md's "blocks always schema-valid at write time" invariant. A retired or
// unknown code is domain.ErrBlockTypeNotFound; a schema violation is
// domain.BlockDataInvalidError{BlockTypeCode, Position} (never the raw validator message, which
// could echo arbitrary submitted content into a safe-arg — see transport/errors.go).
func validateBlockData(blockType domain.BlockType, position int, data []byte) error {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(blockType.JSONSchema))
	if err != nil {
		return fmt.Errorf("block type %q: parse json_schema: %w", blockType.Code, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(blockType.Code, schemaDoc); err != nil {
		return fmt.Errorf("block type %q: add schema resource: %w", blockType.Code, err)
	}
	sch, err := compiler.Compile(blockType.Code)
	if err != nil {
		return fmt.Errorf("block type %q: compile schema: %w", blockType.Code, err)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return &domain.BlockDataInvalidError{BlockTypeCode: blockType.Code, Position: position}
	}
	if err := sch.Validate(instance); err != nil {
		return &domain.BlockDataInvalidError{BlockTypeCode: blockType.Code, Position: position}
	}
	return nil
}
