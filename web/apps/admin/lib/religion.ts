// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's religion module via the generated TypeScript SDK
// (./openfaithmap, M13.2) — religion's first transport layer, so this is its first client wrapper
// too. Same shape as lib/content.ts: forwards the session's Google ID token unchanged, site.manage-
// gated (congregation-admin on their own unit, or registration-operator).
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type { ISite, ISiteAttributes, IUpdateSiteAttributesRequest } from "./openfaithmap/generated/religion";

export type Site = ISite;
export type SiteAttributes = ISiteAttributes;

export class ReligionApiError extends Error {
  constructor(
    public status: number,
    public errorName: string,
    public parameters: Record<string, unknown>,
  ) {
    super(`${errorName} (${status})`);
  }
}

/** True for a ReligionApiError raised by Religion:SiteNotFound — the unit has no site yet, a normal state, not a failure. */
export function isSiteNotFound(e: unknown): e is ReligionApiError {
  return e instanceof ReligionApiError && e.errorName === "Religion:SiteNotFound";
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
    // M11.3, D-SessionTracking: every authenticated request needs a valid X-Session-Id alongside
    // the bearer (internal/identity/middleware.Authenticator.Handle) — see lib/content.ts's own
    // client() for the pattern this mirrors.
    fetch: session?.sessionId
      ? (url, init) =>
          fetch(url, { ...init, headers: { ...init?.headers, "X-Session-Id": session.sessionId! } })
      : undefined,
  });
}

/** Translates a ConjureError (the SDK's transport-level error) into the errorName/parameters shape callers already handle. */
async function unwrap<T>(promise: Promise<T>): Promise<T> {
  try {
    return await promise;
  } catch (e) {
    if (isConjureError(e) && e.body && typeof e.body === "object") {
      const body = e.body as { errorName?: string; parameters?: Record<string, unknown> };
      throw new ReligionApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export async function getSite(unitId: string): Promise<Site> {
  return unwrap((await client()).religion.getSite(unitId));
}

export async function updateSiteAttributes(unitId: string, attributes: SiteAttributes): Promise<Site> {
  const request: IUpdateSiteAttributesRequest = { attributes };
  return unwrap((await client()).religion.updateSiteAttributes(unitId, request));
}
