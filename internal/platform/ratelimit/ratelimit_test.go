// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"testing"
)

// TestLimiterAllow exercises Limiter.allow directly, not via HTTP — proves per-(ip, endpoint)
// scoping without needing real time to pass (the provisional burst is 5, D-Hardening).
func TestLimiterAllow(t *testing.T) {
	rl := NewLimiter("test.rate_limit_rejections")

	for i := 0; i < 5; i++ {
		if !rl.allow("1.2.3.4", "/reports") {
			t.Fatalf("request %d within burst was rejected, want allowed", i+1)
		}
	}
	if rl.allow("1.2.3.4", "/reports") {
		t.Error("6th request within the same burst was allowed, want rejected")
	}

	if !rl.allow("5.6.7.8", "/reports") {
		t.Error("a different IP, same endpoint, was rejected — buckets are not scoped per key")
	}
	if !rl.allow("1.2.3.4", "/exclusion-check") {
		t.Error("a different endpoint, same IP, was rejected — buckets are not scoped per key")
	}
}
