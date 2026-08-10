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
