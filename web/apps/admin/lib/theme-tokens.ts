// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.12 (D-CuratedTheme): a frontend-only static table describing content_sites.theme's curated
// vocabulary for the admin editor's dropdowns and live preview — the same duplicated-presentation-
// data precedent M14.5's block-catalog.ts established, since there is no shared package between
// web/apps/admin and web/apps/web. This table (names, hex values, labels) must stay in sync with
// internal/content/application/themetokens.go, the actual write-time enforcement point; nothing
// here validates a theme, it only describes the choices and previews them.

export interface ThemeAccentOption {
  name: string;
  label: string;
  hex: string;
}

export const THEME_ACCENT_OPTIONS: ThemeAccentOption[] = [
  { name: "indigo", label: "Indigo", hex: "#4338CA" },
  { name: "violet", label: "Violet", hex: "#7C3AED" },
  { name: "rose", label: "Rose", hex: "#E11D48" },
  { name: "amber", label: "Amber", hex: "#D97706" },
  { name: "emerald", label: "Emerald", hex: "#047857" },
  { name: "teal", label: "Teal", hex: "#0F766E" },
  { name: "sky", label: "Sky", hex: "#0369A1" },
  { name: "slate", label: "Slate", hex: "#334155" },
];

export interface ThemeModeOption {
  name: string;
  label: string;
}

export const THEME_MODE_OPTIONS: ThemeModeOption[] = [
  { name: "light", label: "Light" },
  { name: "dark", label: "Dark" },
  { name: "system", label: "Match visitor's device" },
];

export interface ThemeFontPairingOption {
  name: string;
  label: string;
  heading: string;
  body: string;
}

export const THEME_FONT_PAIRING_OPTIONS: ThemeFontPairingOption[] = [
  { name: "modern-sans", label: "Modern Sans", heading: "ui-sans-serif, system-ui, sans-serif", body: "ui-sans-serif, system-ui, sans-serif" },
  {
    name: "classic-serif",
    label: "Classic Serif",
    heading: 'ui-serif, Georgia, Cambria, "Times New Roman", Times, serif',
    body: "ui-sans-serif, system-ui, sans-serif",
  },
  {
    name: "friendly-rounded",
    label: "Friendly Rounded",
    heading: 'ui-rounded, "Hiragino Maru Gothic ProN", system-ui, sans-serif',
    body: "ui-sans-serif, system-ui, sans-serif",
  },
];

export interface ThemeSpacingOption {
  name: string;
  label: string;
}

export const THEME_SPACING_OPTIONS: ThemeSpacingOption[] = [
  { name: "compact", label: "Compact" },
  { name: "comfortable", label: "Comfortable" },
  { name: "spacious", label: "Spacious" },
];

export interface ThemeHeaderLayoutOption {
  name: string;
  label: string;
}

export const THEME_HEADER_LAYOUT_OPTIONS: ThemeHeaderLayoutOption[] = [
  { name: "logo-left", label: "Logo left" },
  { name: "centered", label: "Centered" },
  { name: "stacked", label: "Stacked" },
];

// ThemeInput mirrors internal/content/application/themevalidation.go's ThemeInput. Every field is
// optional: "" (rendered by an empty <option>) means "not chosen yet", the state every pre-M14.12
// site's theme (`{}`) and every field an admin hasn't touched yet is in.
export interface ThemeInput {
  accent: string;
  mode: string;
  fontPairing: string;
  spacing: string;
  headerLayout: string;
}

export function parseTheme(raw: unknown): ThemeInput {
  const obj = raw && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  const pick = (key: string) => (typeof obj[key] === "string" ? (obj[key] as string) : "");
  return {
    accent: pick("accent"),
    mode: pick("mode"),
    fontPairing: pick("fontPairing"),
    spacing: pick("spacing"),
    headerLayout: pick("headerLayout"),
  };
}

export function accentHex(name: string): string | undefined {
  return THEME_ACCENT_OPTIONS.find((a) => a.name === name)?.hex;
}

// autoForeground mirrors the same computed (never curated) black/white pick used by
// web/apps/web/lib/theme-tokens.ts, so the admin preview's contrast matches what the public site
// will actually render.
export function autoForeground(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16) / 255;
  const g = parseInt(hex.slice(3, 5), 16) / 255;
  const b = parseInt(hex.slice(5, 7), 16) / 255;
  const lin = (c: number) => (c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
  const luminance = 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
  return luminance > 0.4 ? "#0A0A0A" : "#FFFFFF";
}
