// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

// M14.12 (D-CuratedTheme): content_sites.theme's fixed token vocabulary. Every field is a name
// chosen from one of these lists — never a raw hex or an arbitrary font — enforced structurally by
// themeSchema's json-schema "enum" keywords (themevalidation.go), not merely requested in the UI.
//
// themeAccentTokens' hex values are deliberately not all safe against both backgrounds: some are
// dark/saturated enough to fail WCAG AA against the near-black dark-mode background, others are
// bright enough to fail against the white light-mode background. That spread is what makes
// checkThemeContrast's rejection real rather than decorative — a curated palette with every entry
// pre-guaranteed to pass everywhere would make the "at write time" gate pointless.

// ThemeAccentToken is one curated accent color choice.
type ThemeAccentToken struct {
	Name string
	Hex  string
}

var themeAccentTokens = []ThemeAccentToken{
	{Name: "indigo", Hex: "#4338CA"},
	{Name: "violet", Hex: "#7C3AED"},
	{Name: "rose", Hex: "#E11D48"},
	{Name: "amber", Hex: "#D97706"},
	{Name: "emerald", Hex: "#047857"},
	{Name: "teal", Hex: "#0F766E"},
	{Name: "sky", Hex: "#0369A1"},
	{Name: "slate", Hex: "#334155"},
}

// themeBackgroundHex mirrors web/apps/web/app/globals.css's --background values exactly:
// oklch(1 0 0) is pure white, and oklch(0.145 0 0) — an achromatic OKLCH color, so only L matters —
// resolves to sRGB (10,10,10)/#0A0A0A via the standard OKLab->linear-sRGB matrix (which reduces to
// linear = L^3 for a=b=0) followed by sRGB gamma encoding. Kept here as fixed hex rather than
// re-deriving OKLCH at runtime: this module has no color-space conversion needs beyond these two
// fixed reference points.
var themeBackgroundHex = map[string]string{
	"light": "#FFFFFF",
	"dark":  "#0A0A0A",
}

var themeFontPairings = []string{"modern-sans", "classic-serif", "friendly-rounded"}

var themeSpacingScales = []string{"compact", "comfortable", "spacious"}

var themeHeaderLayouts = []string{"logo-left", "centered", "stacked"}

var themeModes = []string{"light", "dark", "system"}

func accentHex(name string) (string, bool) {
	for _, t := range themeAccentTokens {
		if t.Name == name {
			return t.Hex, true
		}
	}
	return "", false
}

func accentNames() []string {
	names := make([]string, len(themeAccentTokens))
	for i, t := range themeAccentTokens {
		names[i] = t.Name
	}
	return names
}
