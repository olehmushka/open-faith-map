// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

// next/dynamic's `ssr: false` is only usable from a Client Component (page.tsx is a Server
// Component, async, and fetches data server-side — it can't use it directly). Leaflet touches
// `window` at module-evaluation time, which crashes SSR outright ("ReferenceError: window is not
// defined") — found by actually loading the page, not by review; `npm run build`'s static-page
// check doesn't execute the map component with real network conditions the way a live request
// does. This thin wrapper is the only reason this file exists.
import { useTranslations } from "next-intl";
import dynamic from "next/dynamic";

function MapLoadingFallback() {
  const t = useTranslations("DiscoveryMap");
  return <div className="flex min-h-[70vh] items-center justify-center p-4">{t("loadingMap")}</div>;
}

export const DiscoveryMap = dynamic(() => import("./discovery-map").then((m) => m.DiscoveryMap), {
  ssr: false,
  loading: MapLoadingFallback,
});
