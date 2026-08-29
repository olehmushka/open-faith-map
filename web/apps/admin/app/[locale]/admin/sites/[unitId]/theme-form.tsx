// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  accentHex,
  autoForeground,
  THEME_ACCENT_OPTIONS,
  THEME_FONT_PAIRING_OPTIONS,
  THEME_HEADER_LAYOUT_OPTIONS,
  THEME_MODE_OPTIONS,
  THEME_SPACING_OPTIONS,
  type ThemeInput,
} from "@/lib/theme-tokens";

// Client component (like attributes-form.tsx next to it), not a plain <form action={...}> like the
// M14.11 chrome card next to it: M14.12's live-preview acceptance criterion needs the selection to
// re-render before the form is ever submitted. Native <select> elements (unlike Radix Checkbox/
// Switch) already participate in FormData on their own, so no hidden-input mirroring is needed —
// the form action below reads accent/mode/fontPairing/spacing/headerLayout straight off the
// selects' own `name`s.
export function ThemeForm({
  initial,
  labels,
  action,
}: {
  initial: ThemeInput;
  labels: {
    accent: string;
    mode: string;
    fontPairing: string;
    spacing: string;
    headerLayout: string;
    notSet: string;
    preview: string;
    submit: string;
  };
  action: (formData: FormData) => Promise<void>;
}) {
  const [theme, setTheme] = useState(initial);

  const select = (field: keyof ThemeInput, label: string, options: { name: string; label: string }[]) => (
    <Label className="flex flex-col items-start gap-1">
      {label}
      <select
        name={field}
        value={theme[field]}
        onChange={(e) => setTheme((prev) => ({ ...prev, [field]: e.target.value }))}
        className="border-input h-9 w-full rounded-md border bg-transparent px-3 text-sm shadow-xs"
      >
        <option value="">{labels.notSet}</option>
        {options.map((opt) => (
          <option key={opt.name} value={opt.name}>
            {opt.label}
          </option>
        ))}
      </select>
    </Label>
  );

  const accent = accentHex(theme.accent);
  const pairing = THEME_FONT_PAIRING_OPTIONS.find((p) => p.name === theme.fontPairing);
  const previewBg = theme.mode === "dark" ? "#0A0A0A" : "#FFFFFF";
  const previewFg = theme.mode === "dark" ? "#FFFFFF" : "#0A0A0A";

  return (
    <form action={action} className="flex flex-col gap-4">
      {select("accent", labels.accent, THEME_ACCENT_OPTIONS)}
      {select("mode", labels.mode, THEME_MODE_OPTIONS)}
      {select("fontPairing", labels.fontPairing, THEME_FONT_PAIRING_OPTIONS)}
      {select("spacing", labels.spacing, THEME_SPACING_OPTIONS)}
      {select("headerLayout", labels.headerLayout, THEME_HEADER_LAYOUT_OPTIONS)}

      <div className="flex flex-col gap-2">
        <span className="text-sm font-medium">{labels.preview}</span>
        <div
          className="flex flex-col gap-2 rounded-md border p-4"
          style={{ background: previewBg, color: previewFg, fontFamily: pairing?.body }}
        >
          <span className="text-lg font-semibold" style={{ fontFamily: pairing?.heading }}>
            {labels.preview}
          </span>
          {accent ? (
            <span
              className="inline-flex w-fit items-center rounded px-3 py-1.5 text-sm font-medium"
              style={{ background: accent, color: autoForeground(accent) }}
            >
              {theme.accent}
            </span>
          ) : null}
        </div>
      </div>

      <Button type="submit" className="self-start">
        {labels.submit}
      </Button>
    </form>
  );
}
