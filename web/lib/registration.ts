// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: a typed fetch client for openfaithmap-api's registration module (M2). Plain
// fetch, not a generated SDK — openfaithmap-api has no TypeScript SDK generation pipeline set up
// yet (unlike go-oikumenea's clients/typescript), a deliberate, documented scope cut for this pass.
// Forwards the session's Google ID token unchanged, same as web/lib/oikumenea.ts.
import "server-only";

import { auth } from "@/auth";

export type RegistrationStatus = "PENDING" | "APPROVED" | "REJECTED";

export interface Coordinate {
  latitude: number;
  longitude: number;
}

export interface RegistrationRequest {
  id: string;
  submittedByPersonId: string;
  taxonId: string;
  congregationName: string;
  countryId: string;
  adminArea1?: string;
  locality?: string;
  street?: string;
  houseNumber?: string;
  postalCode?: string;
  coordinate: Coordinate;
  status: RegistrationStatus;
  rejectionReason?: string;
  decidedByPersonId?: string;
  decidedAt?: string;
  createdUnitId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SubmitRegistrationInput {
  taxonId: string;
  congregationName: string;
  countryId: string;
  adminArea1?: string;
  locality?: string;
  street?: string;
  houseNumber?: string;
  postalCode?: string;
  coordinate: Coordinate;
}

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

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  const session = await auth();
  const res = await fetch(`${requireBaseUrl()}${path}`, {
    ...init,
    headers: {
      ...(session?.idToken ? { Authorization: `Bearer ${session.idToken}` } : {}),
      "Content-Type": "application/json",
      ...init?.headers,
    },
    cache: "no-store",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new RegistrationApiError(res.status, body.errorName ?? "Unknown", body.parameters ?? {});
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export function submitRegistration(input: SubmitRegistrationInput): Promise<RegistrationRequest> {
  return call<RegistrationRequest>("/registration/v1/requests", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function listRegistrations(status?: RegistrationStatus): Promise<{ requests: RegistrationRequest[] }> {
  const qs = status ? `?status=${status}` : "";
  return call(`/registration/v1/requests${qs}`);
}

export function getRegistration(id: string): Promise<RegistrationRequest> {
  return call(`/registration/v1/requests/${id}`);
}

export function approveRegistration(id: string, unitCode?: string): Promise<RegistrationRequest> {
  return call(`/registration/v1/requests/${id}/approve`, {
    method: "POST",
    body: JSON.stringify(unitCode ? { unitCode } : {}),
  });
}

export function rejectRegistration(id: string, reason: string): Promise<RegistrationRequest> {
  return call(`/registration/v1/requests/${id}/reject`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
}
