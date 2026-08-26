// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useEffect, useState } from "react";

// Kyiv — same fallback D-Scope's Ukraine-first rollout used before geolocation existed.
export const DEFAULT_CENTER: [number, number] = [50.45, 30.52];
const TIMEOUT_MS = 5000;

export type GeolocationResult =
  | { status: "pending" }
  | { status: "granted"; lat: number; lng: number }
  | { status: "fallback"; lat: number; lng: number };

/**
 * One-shot browser geolocation lookup. Falls back to DEFAULT_CENTER on denial, timeout, error, or
 * when the API isn't available at all (including during SSR, where `navigator` doesn't exist).
 */
export function useGeolocation(): GeolocationResult {
  const [result, setResult] = useState<GeolocationResult>({ status: "pending" });

  useEffect(() => {
    if (typeof navigator === "undefined" || !navigator.geolocation) {
      setResult({ status: "fallback", lat: DEFAULT_CENTER[0], lng: DEFAULT_CENTER[1] });
      return;
    }

    let cancelled = false;
    navigator.geolocation.getCurrentPosition(
      (position) => {
        if (cancelled) return;
        setResult({ status: "granted", lat: position.coords.latitude, lng: position.coords.longitude });
      },
      () => {
        if (cancelled) return;
        setResult({ status: "fallback", lat: DEFAULT_CENTER[0], lng: DEFAULT_CENTER[1] });
      },
      { timeout: TIMEOUT_MS },
    );

    return () => {
      cancelled = true;
    };
  }, []);

  return result;
}
