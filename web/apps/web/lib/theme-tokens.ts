// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { CSSProperties } from "react";

// M14.12 (D-CuratedTheme): a frontend-only static table resolving content_sites.theme's curated
// token names into concrete CSS values — the same "duplicated presentation data next to the Go
// source of truth" precedent M14.5's block-catalog.ts established, since there is no shared package
// between web/apps/admin and web/apps/web. The curated vocabulary and accent hex values must stay
// in sync with internal/content/application/themetokens.go, which is the actual write-time
// enforcement point; this file only resolves an already-validated theme into styling, it never
// validates one.

export interface ThemeAccent {
  name: string;
  hex: string;
}

export const THEME_ACCENTS: ThemeAccent[] = [
  { name: "indigo", hex: "#4338CA" },
  { name: "violet", hex: "#7C3AED" },
  { name: "rose", hex: "#E11D48" },
  { name: "amber", hex: "#D97706" },
  { name: "emerald", hex: "#047857" },
  { name: "teal", hex: "#0F766E" },
  { name: "sky", hex: "#0369A1" },
  { name: "slate", hex: "#334155" },
];

export const THEME_MODES = ["light", "dark", "system"] as const;
export type ThemeMode = (typeof THEME_MODES)[number];

export interface ThemeFontPairing {
  name: string;
  heading: string;
  body: string;
}

export const THEME_FONT_PAIRINGS: ThemeFontPairing[] = [
  { name: "modern-sans", heading: "ui-sans-serif, system-ui, sans-serif", body: "ui-sans-serif, system-ui, sans-serif" },
  {
    name: "classic-serif",
    heading: 'ui-serif, Georgia, Cambria, "Times New Roman", Times, serif',
    body: "ui-sans-serif, system-ui, sans-serif",
  },
  {
    name: "friendly-rounded",
    heading: 'ui-rounded, "Hiragino Maru Gothic ProN", system-ui, sans-serif',
    body: "ui-sans-serif, system-ui, sans-serif",
  },
];

export interface ThemeSpacingScale {
  name: string;
  scale: number;
}

export const THEME_SPACING_SCALES: ThemeSpacingScale[] = [
  { name: "compact", scale: 0.75 },
  { name: "comfortable", scale: 1 },
  { name: "spacious", scale: 1.35 },
];

export const THEME_HEADER_LAYOUTS = ["logo-left", "centered", "stacked"] as const;
export type ThemeHeaderLayout = (typeof THEME_HEADER_LAYOUTS)[number];

// ThemeInput mirrors internal/content/application/themevalidation.go's ThemeInput — a site's
// theme column, parsed defensively (every field optional: a pre-M14.12 site's theme is `{}`, and
// even a validated theme leaves accent/mode unset until an admin has chosen one).
export interface ThemeInput {
  accent?: string;
  mode?: string;
  fontPairing?: string;
  spacing?: string;
  headerLayout?: string;
}

// parseTheme never throws: an unrecognized shape (a pre-M14.12 free-text row, or a future format)
// degrades to "no theme set" rather than crashing the tenant layout, mirroring M14.2's renderer
// precedent ("degrades ... rather than crashing" for a legacy shape it doesn't recognize).
export function parseTheme(raw: unknown): ThemeInput {
  if (!raw || typeof raw !== "object") return {};
  const obj = raw as Record<string, unknown>;
  const pick = (key: string) => (typeof obj[key] === "string" ? (obj[key] as string) : undefined);
  return {
    accent: pick("accent"),
    mode: pick("mode"),
    fontPairing: pick("fontPairing"),
    spacing: pick("spacing"),
    headerLayout: pick("headerLayout"),
  };
}

function accentHex(name: string | undefined): string | undefined {
  return THEME_ACCENTS.find((a) => a.name === name)?.hex;
}

// autoForeground picks black or white for text drawn on top of an accent-colored background,
// using the same WCAG relative-luminance formula the write-time contrast gate uses — this is
// computed, never a curated choice, so it always passes contrast by construction.
function autoForeground(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16) / 255;
  const g = parseInt(hex.slice(3, 5), 16) / 255;
  const b = parseInt(hex.slice(5, 7), 16) / 255;
  const lin = (c: number) => (c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
  const luminance = 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
  return luminance > 0.4 ? "#0A0A0A" : "#FFFFFF";
}

// resolveThemeStyle turns a validated theme into CSS custom properties the tenant layout applies
// on a wrapping element. --primary/--primary-foreground are already read by every shadcn-derived
// Tailwind utility used throughout blocks (web/apps/web/app/globals.css's own @theme mapping), so
// overriding them here restyles the whole site with no new CSS classes required anywhere else.
// forcedPalette mirrors globals.css's own :root (light) and @media(prefers-color-scheme: dark)
// (dark) blocks for the handful of tokens actually painted as backgrounds/borders across the
// tenant renderer. Applied only when a site forces light/dark — "system" leaves the OS-driven
// media query in globals.css untouched, which is why this table exists here rather than as a
// third globals.css block: it's a deliberate override, not a new default.
const forcedPalette: Record<"light" | "dark", Record<string, string>> = {
  light: {
    "--background": "oklch(1 0 0)",
    "--foreground": "oklch(0.145 0 0)",
    "--card": "oklch(1 0 0)",
    "--card-foreground": "oklch(0.145 0 0)",
    "--border": "oklch(0.922 0 0)",
  },
  dark: {
    "--background": "oklch(0.145 0 0)",
    "--foreground": "oklch(0.985 0 0)",
    "--card": "oklch(0.205 0 0)",
    "--card-foreground": "oklch(0.985 0 0)",
    "--border": "oklch(1 0 0 / 10%)",
  },
};

export function resolveThemeStyle(theme: ThemeInput): CSSProperties {
  const style: Record<string, string> = {};

  if (theme.mode === "light" || theme.mode === "dark") {
    Object.assign(style, forcedPalette[theme.mode]);
    style["colorScheme"] = theme.mode;
  }

  const accent = accentHex(theme.accent);
  if (accent) {
    style["--primary"] = accent;
    style["--primary-foreground"] = autoForeground(accent);
  }

  // --font-heading feeds the `font-heading` Tailwind utility (globals.css's own @theme inline
  // mapping) that heading-rendering blocks already apply. Body text has no such utility hook — the
  // base body { font-family: system-ui, sans-serif } rule is a literal, not var-driven — so its
  // pairing value is set directly as this wrapper's own `font-family`, which every descendant
  // inherits unless it sets its own (exactly what the heading override above does).
  const pairing = THEME_FONT_PAIRINGS.find((p) => p.name === theme.fontPairing);
  if (pairing) {
    style["--font-heading"] = pairing.heading;
    style["fontFamily"] = pairing.body;
  }

  const spacing = THEME_SPACING_SCALES.find((s) => s.name === theme.spacing);
  if (spacing) {
    style["--of-space-scale"] = String(spacing.scale);
  }

  return style as CSSProperties;
}

// resolveThemeDataAttr returns the `data-theme` value forcing light/dark (globals.css mirrors its
// media-query dark palette under `:root[data-theme="dark"]` for this), or undefined for "system"/
// unset, which leaves the existing prefers-color-scheme behavior untouched.
export function resolveThemeDataAttr(theme: ThemeInput): "light" | "dark" | undefined {
  return theme.mode === "light" || theme.mode === "dark" ? theme.mode : undefined;
}
