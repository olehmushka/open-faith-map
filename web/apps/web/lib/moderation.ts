// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// openfaithmap-api's moderation module via the generated TypeScript SDK (M5). No session, ever
// (D-AdminSurface) — this app never has a token to forward, and ModerationPublicService.fileReport
// never asks for one; reporterPersonId always comes back unset. Only fileReport is exposed here —
// checkExclusion is used by web/apps/admin's registration wizard, never this app.
import { isConjureError } from "conjure-client";

import { createOpenFaithMapClient } from "./openfaithmap";
import type { IFileReportRequest, IReport } from "./openfaithmap/generated/moderation";

export type FileReportInput = IFileReportRequest;
export type Report = IReport;

export class ModerationApiError extends Error {
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
      throw new ModerationApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export async function fileReport(input: FileReportInput): Promise<Report> {
  return unwrap(client().moderationPublic.fileReport(input));
}
