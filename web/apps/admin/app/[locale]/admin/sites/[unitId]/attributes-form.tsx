// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import type { SiteAttributes } from "@/lib/religion";

// Client component, not a plain <form action={...}> like the Theme card next to it: shadcn's
// Checkbox/Switch (Radix primitives) don't render a native <input> and so don't participate in
// FormData on their own (the installed @radix-ui/react-checkbox/-switch version's stable Root has
// no name/BubbleInput — only the still-"unstable_" Provider API does) — each field pairs a
// controlled Radix control with a hidden native input mirroring its checked state, so the
// surrounding <form>'s Server Action still receives every value the plain-<Input> forms elsewhere
// in this app get for free.
export const ACCESSIBILITY_KEYS = [
  "stepFreeEntrance",
  "accessibleRestroom",
  "hearingLoop",
  "signLanguageInterpretation",
  "accessibleParking",
  "wheelchairSeating",
  "brailleOrLargePrint",
] as const;

export function AttributesForm({
  initial,
  labels,
  onlineStreamLabel,
  submitLabel,
  action,
}: {
  initial: SiteAttributes;
  labels: Record<(typeof ACCESSIBILITY_KEYS)[number], string>;
  onlineStreamLabel: string;
  submitLabel: string;
  action: (formData: FormData) => Promise<void>;
}) {
  const [attrs, setAttrs] = useState(initial);

  return (
    <form action={action} className="flex flex-col gap-4">
      {ACCESSIBILITY_KEYS.map((key) => (
        <div key={key} className="flex items-center gap-2">
          <input type="hidden" name={key} value={attrs.accessibility[key] ? "on" : ""} />
          <Checkbox
            id={key}
            checked={attrs.accessibility[key]}
            onCheckedChange={(checked) =>
              setAttrs((prev) => ({ ...prev, accessibility: { ...prev.accessibility, [key]: checked === true } }))
            }
          />
          <Label htmlFor={key}>{labels[key]}</Label>
        </div>
      ))}
      <div className="flex items-center gap-2">
        <input type="hidden" name="onlineStream" value={attrs.onlineStream ? "on" : ""} />
        <Switch
          id="onlineStream"
          checked={attrs.onlineStream}
          onCheckedChange={(checked) => setAttrs((prev) => ({ ...prev, onlineStream: checked }))}
        />
        <Label htmlFor="onlineStream">{onlineStreamLabel}</Label>
      </div>
      <Button type="submit" className="self-start">
        {submitLabel}
      </Button>
    </form>
  );
}
