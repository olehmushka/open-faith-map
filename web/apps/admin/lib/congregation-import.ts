// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's congregationimport module (M8) via the generated TypeScript SDK
// (./openfaithmap, M2.6). Forwards the session's Google ID token unchanged, same as
// lib/moderation.ts/lib/registration.ts.
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type {
  ICandidate,
  ICandidatePage,
  ICreateJurisdictionAliasRequest,
  ICreateTaxonAliasRequest,
  IEditCandidateRequest,
  IImportRun,
  IJurisdictionAlias,
  IJurisdictionAliasList,
  IRunPage,
  ITaxonAlias,
  ITaxonAliasList,
} from "./openfaithmap/generated/congregationimport";

export type Candidate = ICandidate;
export type CandidatePage = ICandidatePage;
export type ImportRun = IImportRun;
export type RunPage = IRunPage;
export type TaxonAlias = ITaxonAlias;
export type TaxonAliasList = ITaxonAliasList;
export type JurisdictionAlias = IJurisdictionAlias;
export type JurisdictionAliasList = IJurisdictionAliasList;

export class CongregationImportApiError extends Error {
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
      throw new CongregationImportApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export async function runConnector(sourceCode: string): Promise<ImportRun> {
  return unwrap((await client()).congregationImport.runConnector({ sourceCode }));
}

export async function listRuns(sourceCode?: string, pageSize?: number, pageToken?: string): Promise<RunPage> {
  return unwrap((await client()).congregationImport.listRuns(sourceCode, pageSize, pageToken));
}

export async function listCandidates(status?: string, pageSize?: number, pageToken?: string): Promise<CandidatePage> {
  return unwrap((await client()).congregationImport.listCandidates(status, pageSize, pageToken));
}

export async function editCandidate(candidateId: string, request: IEditCandidateRequest): Promise<Candidate> {
  return unwrap((await client()).congregationImport.editCandidate(candidateId, request));
}

export async function approveCandidate(candidateId: string, jurisdictionUnitId?: string): Promise<Candidate> {
  return unwrap((await client()).congregationImport.approveCandidate(candidateId, { jurisdictionUnitId }));
}

export async function rejectCandidate(candidateId: string, reason: string): Promise<Candidate> {
  return unwrap((await client()).congregationImport.rejectCandidate(candidateId, { reason }));
}

export async function listTaxonAliases(sourceCode?: string): Promise<TaxonAliasList> {
  return unwrap((await client()).congregationImport.listTaxonAliases(sourceCode));
}

export async function createTaxonAlias(request: ICreateTaxonAliasRequest): Promise<TaxonAlias> {
  return unwrap((await client()).congregationImport.createTaxonAlias(request));
}

export async function listJurisdictionAliases(sourceCode?: string): Promise<JurisdictionAliasList> {
  return unwrap((await client()).congregationImport.listJurisdictionAliases(sourceCode));
}

export async function createJurisdictionAlias(request: ICreateJurisdictionAliasRequest): Promise<JurisdictionAlias> {
  return unwrap((await client()).congregationImport.createJurisdictionAlias(request));
}
