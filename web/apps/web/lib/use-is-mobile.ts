// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useEffect, useState } from "react";

// Matches the `md` breakpoint used throughout app/discovery/ for CSS-only responsive layout. This
// hook exists only for the one place a plain breakpoint class can't do the job: map-pane.tsx needs
// to know in JS whether to bind Leaflet's imperative click-to-open popup at all.
const MOBILE_QUERY = "(max-width: 767.98px)";

/** SSR-safe: defaults to `false` (desktop) until mounted, then syncs to the real viewport width. */
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return;
    const mql = window.matchMedia(MOBILE_QUERY);
    setIsMobile(mql.matches);

    const listener = (e: MediaQueryListEvent) => setIsMobile(e.matches);
    mql.addEventListener("change", listener);
    return () => mql.removeEventListener("change", listener);
  }, []);

  return isMobile;
}
