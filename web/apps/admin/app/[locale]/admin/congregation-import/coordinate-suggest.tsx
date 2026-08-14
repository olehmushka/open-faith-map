// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useRef, useState, useTransition } from "react";

// latitude/longitude come back as `number | "NaN"` — conjure-typescript's own encoding for a wire
// `double`, not something this endpoint ever actually returns as "NaN" in practice (a real geocode
// match always has real coordinates), but the type must account for it.
type SuggestResult = {
  latitude: number | "NaN";
  longitude: number | "NaN";
  precision?: string | null;
  displayName: string;
  provider: string;
};

/**
 * Renders the Latitude/Longitude fields themselves plus a "Suggest coordinates" button — ADVISORY
 * ONLY, mirrors JurisdictionField's own self-contained shape (owns the inputs it feeds, since a
 * per-item useRef can't be called from inside candidate-list.tsx's own .map() loop). Sets the
 * input VALUES via refs on a successful lookup; the operator still sees them in the normal editable
 * fields and must click the enclosing form's own Save button to persist — never auto-applied.
 */
export function CoordinateSuggest({
  defaultLatitude,
  defaultLongitude,
  onSuggest,
  labels,
}: {
  defaultLatitude?: number | string | null;
  defaultLongitude?: number | string | null;
  onSuggest: () => Promise<SuggestResult>;
  labels: {
    latitude: string;
    longitude: string;
    suggestCoordinates: string;
    suggesting: string;
    suggestedVia: string;
    geocodeNoMatch: string;
    geocodeLookupFailed: string;
  };
}) {
  const latRef = useRef<HTMLInputElement>(null);
  const lngRef = useRef<HTMLInputElement>(null);
  const [isPending, startTransition] = useTransition();
  const [message, setMessage] = useState<string | null>(null);
  const [isError, setIsError] = useState(false);

  function handleSuggest() {
    startTransition(async () => {
      try {
        const result = await onSuggest();
        if (result.latitude === "NaN" || result.longitude === "NaN") {
          setIsError(true);
          setMessage(labels.geocodeLookupFailed);
          return;
        }
        if (latRef.current) latRef.current.value = String(result.latitude);
        if (lngRef.current) lngRef.current.value = String(result.longitude);
        setIsError(false);
        setMessage(
          `${labels.suggestedVia} ${result.provider}: ${result.displayName}${result.precision ? ` (${result.precision})` : ""}`,
        );
      } catch (e) {
        setIsError(true);
        const errorName = e && typeof e === "object" && "errorName" in e ? String((e as { errorName: unknown }).errorName) : "";
        setMessage(errorName === "CongregationImport:GeocodeNoMatch" ? labels.geocodeNoMatch : labels.geocodeLookupFailed);
      }
    });
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="flex flex-wrap items-end gap-2">
        <label className="flex flex-col text-xs">
          {labels.latitude}
          <input ref={latRef} name="latitude" defaultValue={defaultLatitude ?? ""} className="w-24 rounded border px-2 py-1 text-sm" />
        </label>
        <label className="flex flex-col text-xs">
          {labels.longitude}
          <input ref={lngRef} name="longitude" defaultValue={defaultLongitude ?? ""} className="w-24 rounded border px-2 py-1 text-sm" />
        </label>
        <button type="button" onClick={handleSuggest} disabled={isPending} className="rounded border px-2 py-1 text-sm">
          {isPending ? labels.suggesting : labels.suggestCoordinates}
        </button>
      </div>
      {message && <span className={`text-xs ${isError ? "text-red-600" : "text-gray-500"}`}>{message}</span>}
    </div>
  );
}
