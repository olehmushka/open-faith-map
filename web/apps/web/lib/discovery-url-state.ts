// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// The URL query string is the single source of truth for the discovery page's filter + last-
// searched-viewport state (M13.3). Shared between the server page's initial fetch and the client
// shell's re-searches so both read/write the same shape.

export interface DiscoveryFilters {
  tradition?: string;
  language?: string;
  lat?: number;
  lng?: number;
  radiusM?: number;
  dayOfWeek?: number;
  accessibility?: string[];
  onlineOnly?: boolean;
}

interface SearchParamsLike {
  get(key: string): string | null;
}

function numberOrUndefined(value: string | null): number | undefined {
  if (value == null) return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
}

/** Works with both `URLSearchParams`/`ReadonlyURLSearchParams` (client) and `searchParamsFromRecord` (server). */
export function parseFilters(params: SearchParamsLike): DiscoveryFilters {
  const accessibility = params.get("accessibility")?.split(",").filter(Boolean);
  return {
    tradition: params.get("tradition") ?? undefined,
    language: params.get("language") ?? undefined,
    lat: numberOrUndefined(params.get("lat")),
    lng: numberOrUndefined(params.get("lng")),
    radiusM: numberOrUndefined(params.get("radiusM")),
    dayOfWeek: numberOrUndefined(params.get("dayOfWeek")),
    accessibility: accessibility?.length ? accessibility : undefined,
    onlineOnly: params.get("onlineOnly") === "true" ? true : undefined,
  };
}

export function filtersToSearchParams(filters: DiscoveryFilters): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.tradition) params.set("tradition", filters.tradition);
  if (filters.language) params.set("language", filters.language);
  if (filters.lat != null) params.set("lat", String(filters.lat));
  if (filters.lng != null) params.set("lng", String(filters.lng));
  if (filters.radiusM != null) params.set("radiusM", String(filters.radiusM));
  if (filters.dayOfWeek != null) params.set("dayOfWeek", String(filters.dayOfWeek));
  if (filters.accessibility?.length) params.set("accessibility", filters.accessibility.join(","));
  if (filters.onlineOnly) params.set("onlineOnly", "true");
  return params;
}

/** Adapts Next.js's server-side `searchParams` record (string | string[] | undefined values) to SearchParamsLike. */
export function searchParamsFromRecord(
  record: Record<string, string | string[] | undefined>,
): SearchParamsLike {
  return {
    get(key: string) {
      const value = record[key];
      if (Array.isArray(value)) return value[0] ?? null;
      return value ?? null;
    },
  };
}
