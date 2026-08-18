// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"errors"
	"testing"
)

func TestValidateSeedForEnvironment(t *testing.T) {
	real := AdminSeed{Issuer: "https://accounts.google.com", Subject: "109876543210987654321", Email: "admin@example.org"}
	placeholder := AdminSeed{Issuer: "https://accounts.google.com", Subject: PlaceholderSubject}
	empty := AdminSeed{}

	tests := []struct {
		name    string
		seed    AdminSeed
		env     string
		wantErr bool
	}{
		{"local allows the placeholder", placeholder, "local", false},
		{"dev allows the placeholder", placeholder, "dev", false},
		{"local allows an empty seed (no bootstrap-admin configured yet)", empty, "local", false},
		{"staging refuses the placeholder", placeholder, "staging", true},
		{"prod refuses the placeholder", placeholder, "prod", true},
		{"prod refuses an empty subject", empty, "prod", true},
		{"unrecognized environment refuses the placeholder (fail closed)", placeholder, "production", true},
		{"prod accepts a real-looking seed", real, "prod", false},
		{"staging accepts a real-looking seed", real, "staging", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSeedForEnvironment(tt.seed, tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSeedForEnvironment(env=%q) error = %v, wantErr %v", tt.env, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrPlaceholderSeed) {
				t.Errorf("expected ErrPlaceholderSeed, got %v", err)
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "b", "c"); got != "b" {
		t.Errorf("firstNonEmpty = %q, want b", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("firstNonEmpty with all-empty = %q, want empty", got)
	}
}
