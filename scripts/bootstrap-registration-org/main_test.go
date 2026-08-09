// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"reflect"
	"testing"
)

func TestPermissionsToAdd(t *testing.T) {
	tests := []struct {
		name        string
		have        []string
		wanted      []string
		wantUnion   []string
		wantChanged bool
	}{
		{
			name:        "nothing missing",
			have:        []string{"religionorg.manage", "assignment.read"},
			wanted:      []string{"religionorg.manage", "assignment.read"},
			wantUnion:   []string{"religionorg.manage", "assignment.read"},
			wantChanged: false,
		},
		{
			name:        "one missing",
			have:        []string{"religionorg.manage"},
			wanted:      []string{"religionorg.manage", "assignment.read"},
			wantUnion:   []string{"religionorg.manage", "assignment.read"},
			wantChanged: true,
		},
		{
			name:        "have is empty",
			have:        nil,
			wanted:      []string{"religionorg.manage", "assignment.read"},
			wantUnion:   []string{"religionorg.manage", "assignment.read"},
			wantChanged: true,
		},
		{
			name:        "existing permission outside wanted is kept",
			have:        []string{"religionorg.manage", "site.manage"},
			wanted:      []string{"religionorg.manage", "assignment.read"},
			wantUnion:   []string{"religionorg.manage", "site.manage", "assignment.read"},
			wantChanged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUnion, gotChanged := permissionsToAdd(tt.have, tt.wanted)
			if !reflect.DeepEqual(gotUnion, tt.wantUnion) {
				t.Errorf("union = %v, want %v", gotUnion, tt.wantUnion)
			}
			if gotChanged != tt.wantChanged {
				t.Errorf("changed = %v, want %v", gotChanged, tt.wantChanged)
			}
		})
	}
}
