// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"
)

// TestIsReservedSlug proves D-TenantSubdomains' blocklist (M14.9) matches case-insensitively and
// leaves ordinary slugs alone. No DB needed — isReservedSlug only touches its own map literal.
func TestIsReservedSlug(t *testing.T) {
	reserved := []string{"admin", "Admin", "ADMIN", "api", "www", "localhost", "preview"}
	for _, slug := range reserved {
		if !isReservedSlug(slug) {
			t.Errorf("isReservedSlug(%q) = false, want true", slug)
		}
	}

	allowed := []string{"grace", "st-marys", "hope-church", "trinity-baptist"}
	for _, slug := range allowed {
		if isReservedSlug(slug) {
			t.Errorf("isReservedSlug(%q) = true, want false", slug)
		}
	}
}
