// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"testing"
)

func intPtr(v int) *int { return &v }

func TestPageSizeOrDefault(t *testing.T) {
	tests := []struct {
		name string
		in   *int
		want int
	}{
		{name: "nil falls back to default", in: nil, want: defaultPageSize},
		{name: "zero falls back to default", in: intPtr(0), want: defaultPageSize},
		{name: "negative falls back to default", in: intPtr(-5), want: defaultPageSize},
		{name: "within range passes through", in: intPtr(10), want: 10},
		{name: "at the max ceiling passes through", in: intPtr(maxPageSize), want: maxPageSize},
		{name: "over the max ceiling clamps down", in: intPtr(maxPageSize + 1), want: maxPageSize},
		{name: "far over the max ceiling clamps down", in: intPtr(500), want: maxPageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pageSizeOrDefault(tt.in); got != tt.want {
				t.Errorf("pageSizeOrDefault(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
