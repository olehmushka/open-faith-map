// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's discovery module via the generated TypeScript SDK
// (./openfaithmap, M4). `search` needs no session (DiscoveryPublicService never looks at the
// bearer token — same shape as lib/content.ts's public reads); `refreshRegion` is an operator tool
// (DiscoveryService, content.manage-equivalent target-scoped check) and forwards the session's
// Google ID token like every other admin-only call in this app.
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type { IDiscoverySite, IRefreshRegionRequest, IRefreshResult } from "./openfaithmap/generated/discovery";

export type DiscoverySite = IDiscoverySite;
export type RefreshRegionInput = IRefreshRegionRequest;
export type RefreshResult = IRefreshResult;

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

async function client() {
  const session = await auth();
  return createOpenFaithMapClient({
    baseUrl: requireBaseUrl(),
    token: session?.idToken,
  });
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

// ---- operator tool (target-scoped, root-unit check) ----

export async function refreshRegion(input: RefreshRegionInput): Promise<RefreshResult> {
  return unwrap((await client()).discovery.refresh(input));
}

// ~5km padding around a single point — enough to cover discovery_site_cache's region-bucketing
// without refreshing a huge area for a one-congregation approval.
const REFRESH_PADDING_DEGREES = 0.05;

/**
 * Best-effort nudge so a just-approved congregation shows up on the public map's default
 * (no-filter) view immediately, instead of waiting for the next filtered search or an operator's
 * manual refreshRegion call — discovery_site_cache only self-refreshes on a cache-miss or a
 * tradition/language/dayOfWeek/query search (docs/modules/discovery.md's "lazy-only, no scheduled
 * job"), never automatically on approval. Swallows failures deliberately: the cache is
 * self-healing by design, so a refresh hiccup here must never fail the approval itself.
 */
export async function refreshRegionAroundPoint(
  latitude: number | "NaN" | null | undefined,
  longitude: number | "NaN" | null | undefined,
): Promise<void> {
  if (latitude == null || longitude == null || latitude === "NaN" || longitude === "NaN") return;
  try {
    await refreshRegion({
      minLat: latitude - REFRESH_PADDING_DEGREES,
      maxLat: latitude + REFRESH_PADDING_DEGREES,
      minLng: longitude - REFRESH_PADDING_DEGREES,
      maxLng: longitude + REFRESH_PADDING_DEGREES,
    });
  } catch {
    // Best-effort — see doc comment above.
  }
}
