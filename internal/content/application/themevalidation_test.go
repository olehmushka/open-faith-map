// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"errors"
	"testing"

	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// TestValidateThemeCuratedVocabulary proves D-CuratedTheme's structural gate: any field value
// outside the curated enum is rejected with domain.ThemeInvalidError naming the field, never a raw
// hex/font value. No DB needed — validateTheme only touches its own byte-slice argument.
func TestValidateThemeCuratedVocabulary(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		wantErr   bool
		wantField string
	}{
		{name: "empty object is valid (no theme set yet)", data: `{}`, wantErr: false},
		{
			name:    "curated combination that passes contrast",
			data:    `{"accent":"indigo","mode":"light","fontPairing":"modern-sans","spacing":"comfortable","headerLayout":"logo-left"}`,
			wantErr: false,
		},
		{
			name:      "raw hex accent rejected",
			data:      `{"accent":"#ff00ff"}`,
			wantErr:   true,
			wantField: "accent",
		},
		{
			name:      "arbitrary font rejected",
			data:      `{"fontPairing":"Comic Sans MS"}`,
			wantErr:   true,
			wantField: "fontPairing",
		},
		{
			name:    "unexpected property rejected",
			data:    `{"customCss":"body{color:red}"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateTheme([]byte(tt.data))
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("validateTheme(%s) = %v, want nil", tt.data, err)
				}
				return
			}
			var invalid *domain.ThemeInvalidError
			if !errors.As(err, &invalid) {
				t.Fatalf("validateTheme(%s) = %v, want *domain.ThemeInvalidError", tt.data, err)
			}
			if tt.wantField != "" && invalid.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", invalid.Field, tt.wantField)
			}
		})
	}
}

// TestCheckThemeContrast proves the WCAG AA gate has real teeth in both directions: a dark
// accent against the near-black dark background fails, a bright accent against the white light
// background fails, and system mode is checked against both so it's the most restrictive.
func TestCheckThemeContrast(t *testing.T) {
	tests := []struct {
		name    string
		accent  string
		mode    string
		wantErr bool
	}{
		{name: "dark indigo on light background passes", accent: "indigo", mode: "light", wantErr: false},
		{name: "dark indigo on dark background fails", accent: "indigo", mode: "dark", wantErr: true},
		{name: "bright amber on dark background passes", accent: "amber", mode: "dark", wantErr: false},
		{name: "bright amber on light background fails", accent: "amber", mode: "light", wantErr: true},
		{name: "system mode fails if either background fails", accent: "indigo", mode: "system", wantErr: true},
		{name: "unknown accent token rejected", accent: "magenta", mode: "light", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkThemeContrast(tt.accent, tt.mode)
			if tt.wantErr && err == nil {
				t.Fatalf("checkThemeContrast(%q, %q) = nil, want error", tt.accent, tt.mode)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("checkThemeContrast(%q, %q) = %v, want nil", tt.accent, tt.mode, err)
			}
		})
	}
}

// TestContrastRatioKnownValues checks the WCAG relative-luminance/contrast-ratio math itself
// against known reference pairs (pure black/white is the maximum possible ratio, 21:1; identical
// colors are always 1:1).
func TestContrastRatioKnownValues(t *testing.T) {
	if got := contrastRatio("#000000", "#FFFFFF"); got < 20.9 || got > 21.1 {
		t.Errorf("contrastRatio(black, white) = %v, want ~21", got)
	}
	if got := contrastRatio("#4338CA", "#4338CA"); got < 0.99 || got > 1.01 {
		t.Errorf("contrastRatio(x, x) = %v, want ~1", got)
	}
}
