// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/json"
)

// marshalAny converts a Conjure `any` field (decoded JSON, so a plain Go interface{} —
// map[string]interface{}, []interface{}, string, float64, bool, or nil) into the raw JSON bytes
// domain types store and the adapters layer writes straight into a jsonb column.
func marshalAny(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

// unmarshalAny is marshalAny's inverse: raw JSON bytes (as read back from a jsonb column) into the
// interface{} shape a Conjure `any` response field expects.
func unmarshalAny(raw []byte) interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return map[string]interface{}{}
	}
	return v
}
