// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/olehmushka/open-faith-map/internal/content/domain"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// themeSchemaJSON is built from themetokens.go's vocabulary lists (never hand-duplicated) so the
// enum a submitted theme is checked against can't drift from the values checkThemeContrast and the
// admin/public frontends resolve. additionalProperties:false rejects any field outside this fixed
// shape, same as a raw hex/font would be rejected by its field's own enum.
func themeSchemaJSON() []byte {
	enumOf := func(values []string) []string { return values }
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"accent":       map[string]any{"type": "string", "enum": toAny(enumOf(accentNames()))},
			"mode":         map[string]any{"type": "string", "enum": toAny(enumOf(themeModes))},
			"fontPairing":  map[string]any{"type": "string", "enum": toAny(enumOf(themeFontPairings))},
			"spacing":      map[string]any{"type": "string", "enum": toAny(enumOf(themeSpacingScales))},
			"headerLayout": map[string]any{"type": "string", "enum": toAny(enumOf(themeHeaderLayouts))},
		},
		"additionalProperties": false,
	}
	b, _ := json.Marshal(schema)
	return b
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// themeFieldNames is the fixed, developer-authored set topLevelFieldFromValidationError-style
// filtering needs — mirrors blockvalidation.go's own reasoning: an instance-location path segment
// is only ever surfaced as a safe-arg when it names one of these, never a raw submitted key.
var themeFieldNames = map[string]bool{
	"accent": true, "mode": true, "fontPairing": true, "spacing": true, "headerLayout": true,
}

// ThemeInput is content_sites.theme's parsed, validated shape (M14.12). A zero value (all fields
// empty) is the "no theme customization yet" state every pre-M14.12 site row is in.
type ThemeInput struct {
	Accent       string `json:"accent"`
	Mode         string `json:"mode"`
	FontPairing  string `json:"fontPairing"`
	Spacing      string `json:"spacing"`
	HeaderLayout string `json:"headerLayout"`
}

// validateTheme enforces D-CuratedTheme's write-time gate: every field structurally limited to the
// curated vocabulary (themeSchemaJSON's enums), then a WCAG AA contrast check on the resolved
// accent/mode pair. A theme with an unset accent or mode (the zero-value/empty-object case every
// pre-M14.12 row already has) skips the contrast check — there is no pair to fail yet.
func validateTheme(data []byte) (ThemeInput, error) {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(themeSchemaJSON()))
	if err != nil {
		return ThemeInput{}, fmt.Errorf("theme schema: parse: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("theme", schemaDoc); err != nil {
		return ThemeInput{}, fmt.Errorf("theme schema: add resource: %w", err)
	}
	sch, err := compiler.Compile("theme")
	if err != nil {
		return ThemeInput{}, fmt.Errorf("theme schema: compile: %w", err)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return ThemeInput{}, &domain.ThemeInvalidError{}
	}
	if err := sch.Validate(instance); err != nil {
		return ThemeInput{}, &domain.ThemeInvalidError{Field: themeFieldFromValidationError(err)}
	}

	var theme ThemeInput
	if err := json.Unmarshal(data, &theme); err != nil {
		return ThemeInput{}, &domain.ThemeInvalidError{}
	}

	if theme.Accent == "" || theme.Mode == "" {
		return theme, nil
	}
	if err := checkThemeContrast(theme.Accent, theme.Mode); err != nil {
		return ThemeInput{}, err
	}
	return theme, nil
}

func themeFieldFromValidationError(err error) string {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return ""
	}
	var walk func(*jsonschema.ValidationError) string
	walk = func(v *jsonschema.ValidationError) string {
		if len(v.InstanceLocation) > 0 && themeFieldNames[v.InstanceLocation[0]] {
			return v.InstanceLocation[0]
		}
		for _, cause := range v.Causes {
			if f := walk(cause); f != "" {
				return f
			}
		}
		return ""
	}
	return walk(ve)
}

// checkThemeContrast computes the real WCAG relative-luminance contrast ratio between accent's
// curated hex and mode's background(s), rejecting anything under the 4.5:1 AA threshold for normal
// text. mode="system" is checked against both light and dark backgrounds — a visitor's OS may
// resolve either — so it's the most restrictive of the three.
func checkThemeContrast(accent, mode string) error {
	hex, ok := accentHex(accent)
	if !ok {
		return &domain.ThemeInvalidError{Field: "accent"}
	}

	backgrounds := []string{mode}
	if mode == "system" {
		backgrounds = []string{"light", "dark"}
	}
	for _, bg := range backgrounds {
		bgHex, ok := themeBackgroundHex[bg]
		if !ok {
			return &domain.ThemeInvalidError{Field: "mode"}
		}
		if contrastRatio(hex, bgHex) < 4.5 {
			return &domain.ThemeContrastFailedError{Accent: accent, Mode: mode}
		}
	}
	return nil
}

// contrastRatio implements the WCAG 2.x formula: (L1+0.05)/(L2+0.05) with L1 the lighter of the
// two relative luminances.
func contrastRatio(hexA, hexB string) float64 {
	la := relativeLuminance(hexA)
	lb := relativeLuminance(hexB)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// relativeLuminance implements the standard sRGB -> linear -> WCAG luminance pipeline
// (https://www.w3.org/TR/WCAG21/#dfn-relative-luminance).
func relativeLuminance(hex string) float64 {
	r, g, b := hexToRGB(hex)
	lin := func(c float64) float64 {
		c /= 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

func hexToRGB(hex string) (r, g, b float64) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0
	}
	ri, _ := strconv.ParseInt(hex[1:3], 16, 32)
	gi, _ := strconv.ParseInt(hex[3:5], 16, 32)
	bi, _ := strconv.ParseInt(hex[5:7], 16, 32)
	return float64(ri), float64(gi), float64(bi)
}
