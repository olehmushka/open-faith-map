// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"encoding/json"
	"testing"

	"github.com/olehmushka/open-faith-map/internal/directory/domain"
)

func TestNullableText(t *testing.T) {
	if got := nullableText(""); got.Valid {
		t.Errorf("nullableText(\"\") = %+v, want Valid=false", got)
	}
	if got := nullableText("root"); !got.Valid || got.String != "root" {
		t.Errorf("nullableText(%q) = %+v, want Valid=true String=%q", "root", got, "root")
	}
}

func TestOrDefaultState(t *testing.T) {
	if got := orDefaultState(""); got != domain.StateActive {
		t.Errorf("orDefaultState(\"\") = %v, want %v", got, domain.StateActive)
	}
	if got := orDefaultState(domain.StateArchived); got != domain.StateArchived {
		t.Errorf("orDefaultState(archived) = %v, want unchanged", got)
	}
}

func TestOrDefaultMetadata(t *testing.T) {
	if got := orDefaultMetadata(nil); string(got) != "{}" {
		t.Errorf("orDefaultMetadata(nil) = %s, want {}", got)
	}
	if got := orDefaultMetadata(json.RawMessage(`{"k":"v"}`)); string(got) != `{"k":"v"}` {
		t.Errorf("orDefaultMetadata(non-empty) = %s, want unchanged", got)
	}
}
