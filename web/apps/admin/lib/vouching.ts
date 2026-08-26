// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's vouching module (M6) via the generated TypeScript SDK
// (./openfaithmap, M2.6). Forwards the session's Google ID token unchanged, same as
// lib/moderation.ts/lib/registration.ts. Vouching has no anonymous/public service at all (unlike
// moderation/content/discovery) — every call here requires a real session.
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type {
  GuarantorStatus,
  IGuarantorStatusRecord,
  IVouch,
  IVouchPage,
} from "./openfaithmap/generated/vouching";

export type Vouch = IVouch;
export type VouchPage = IVouchPage;
export type GuarantorStatusRecord = IGuarantorStatusRecord;
export type { GuarantorStatus };

export class VouchingApiError extends Error {
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
    // M11.3, D-SessionTracking: every authenticated request needs a valid X-Session-Id alongside
    // the bearer (internal/identity/middleware.Authenticator.Handle) — omitted here until now,
    // silently 401-ing every call through this file since M11.3 shipped (see lib/core.ts's own
    // client() for the pattern this mirrors).
    fetch: session?.sessionId
      ? (url, init) =>
          fetch(url, { ...init, headers: { ...init?.headers, "X-Session-Id": session.sessionId! } })
      : undefined,
  });
}

async function unwrap<T>(promise: Promise<T>): Promise<T> {
  try {
    return await promise;
  } catch (e) {
    if (isConjureError(e) && e.body && typeof e.body === "object") {
      const body = e.body as { errorName?: string; parameters?: Record<string, unknown> };
      throw new VouchingApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export async function createVouch(
  claimantPersonId: string,
  congregationUnitId: string,
  guarantorCongregationUnitId: string,
  statement?: string,
): Promise<Vouch> {
  return unwrap(
    (await client()).vouching.createVouch({
      claimantPersonId,
      congregationUnitId,
      guarantorCongregationUnitId,
      statement,
    }),
  );
}

export async function listVouches(claimant?: string, congregation?: string): Promise<VouchPage> {
  return unwrap((await client()).vouching.listVouches(claimant, congregation));
}

export async function revokeGuarantor(personRid: string, reason: string): Promise<GuarantorStatusRecord> {
  return unwrap((await client()).vouching.revokeGuarantor(personRid, { reason }));
}

export async function getGuarantorStatus(personRid: string): Promise<GuarantorStatusRecord> {
  return unwrap((await client()).vouching.getGuarantorStatus(personRid));
}
