// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's registration module via the generated TypeScript SDK
// (./openfaithmap, M2.6) instead of a hand-rolled fetch client. Forwards the session's Google ID
// token unchanged, same as lib/core.ts. Keeps the same exported function names as the prior
// hand-written client so no call site changes — only the implementation and the source of the
// types (the Conjure contract, not a hand-copy of it) changed.
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type {
  IApproveRegistrationRequest,
  IRegistrationRequest,
  IRegistrationRequestPage,
  IReparentingJob,
  ISubmitRegistrationRequest,
  RegistrationStatus,
  ReparentStatus,
} from "./openfaithmap/generated/registration";

export type RegistrationRequest = IRegistrationRequest;
export type RegistrationRequestPage = IRegistrationRequestPage;
export type SubmitRegistrationInput = ISubmitRegistrationRequest;
export type Coordinate = IRegistrationRequest["coordinate"];
export type ReparentingJob = IReparentingJob;
export type { RegistrationStatus, ReparentStatus };

export class RegistrationApiError extends Error {
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

/** Translates a ConjureError (the SDK's transport-level error) into the errorName/parameters shape callers already handle. */
async function unwrap<T>(promise: Promise<T>): Promise<T> {
  try {
    return await promise;
  } catch (e) {
    if (isConjureError(e) && e.body && typeof e.body === "object") {
      const body = e.body as { errorName?: string; parameters?: Record<string, unknown> };
      throw new RegistrationApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export async function submitRegistration(input: SubmitRegistrationInput): Promise<RegistrationRequest> {
  return unwrap((await client()).registration.submitRequest(input));
}

export async function listRegistrations(status?: RegistrationStatus): Promise<RegistrationRequestPage> {
  return unwrap((await client()).registration.listRequests(status));
}

export async function getRegistration(id: string): Promise<RegistrationRequest> {
  return unwrap((await client()).registration.getRequest(id));
}

export async function approveRegistration(
  id: string,
  unitCode?: string,
  jurisdictionUnitId?: string,
): Promise<RegistrationRequest> {
  const request: IApproveRegistrationRequest = { unitCode, jurisdictionUnitId };
  return unwrap((await client()).registration.approveRequest(id, request));
}

export async function rejectRegistration(id: string, reason: string): Promise<RegistrationRequest> {
  return unwrap((await client()).registration.rejectRequest(id, { reason }));
}

/** Starts or resumes re-parenting an APPROVED request's congregation unit (M4.1, D-JurisdictionUnits). */
export async function reparentRegistration(id: string, newParentUnitId: string): Promise<ReparentingJob> {
  return unwrap((await client()).registration.reparentRequest(id, { newParentUnitId }));
}

export async function getReparentStatus(id: string): Promise<ReparentingJob | null> {
  return unwrap((await client()).registration.getReparentStatus(id));
}
