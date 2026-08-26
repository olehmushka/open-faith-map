// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// openfaithmap-api's discovery module via the generated TypeScript SDK (M4). No session, ever
// (D-AdminSurface) — this app never has a token to forward, and DiscoveryPublicService's search
// never asks for one. See docs/modules/discovery.md's redesign.
import { isConjureError } from "conjure-client";

import { createOpenFaithMapClient } from "./openfaithmap";
import type { IDiscoverySite, IFacetsResult } from "./openfaithmap/generated/discovery";

export type DiscoverySite = IDiscoverySite;
export type DiscoveryFacets = IFacetsResult;

export class DiscoveryApiError extends Error {
  constructor(
    public status: number,
    public errorName: string,
    public parameters: Record<string, unknown>,
  ) {
    super(`${errorName} (${status})`);
  }
}

function requireBaseUrl(): string {
  const raw = process.env.OPENFAITHMAP_API_BASE_URL?.trim();
  if (!raw) {
    throw new Error("OPENFAITHMAP_API_BASE_URL is not set.");
  }
  return raw.replace(/\/+$/, "");
}

function client() {
  return createOpenFaithMapClient({ baseUrl: requireBaseUrl() });
}

async function unwrap<T>(promise: Promise<T>): Promise<T> {
  try {
    return await promise;
  } catch (e) {
    if (isConjureError(e) && e.body && typeof e.body === "object") {
      const body = e.body as { errorName?: string; parameters?: Record<string, unknown> };
      throw new DiscoveryApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export interface SearchParams {
  lat?: number;
  lng?: number;
  radiusM?: number;
  tradition?: string;
  language?: string;
  dayOfWeek?: number;
  query?: string;
}

export async function search(params: SearchParams): Promise<DiscoverySite[]> {
  const result = await unwrap(
    client().discoveryPublic.search(
      params.lat,
      params.lng,
      params.radiusM,
      params.tradition,
      params.language,
      params.dayOfWeek,
      params.query,
    ),
  );
  return result.sites;
}

// Distinct tradition/language values actually present among public sites (M13.1's facets
// endpoint) — backs the filter-bar's Select pickers so they never offer a value that would zero
// out every result. Fetched once per page load, not re-fetched per search.
export async function facets(): Promise<DiscoveryFacets> {
  return unwrap(client().discoveryPublic.facets());
}
