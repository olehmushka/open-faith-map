// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's moderation module (M5) via the generated TypeScript SDK
// (./openfaithmap, M2.6). Forwards the session's Google ID token unchanged, same as
// lib/registration.ts.
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type {
  ActionKind,
  AppealDecision,
  AppealStatus,
  IAppeal,
  IAppealPage,
  IModerationAction,
  IReport,
  IReportPage,
  QueueScope,
  ReportStatus,
} from "./openfaithmap/generated/moderation";

export type Report = IReport;
export type ReportPage = IReportPage;
export type ModerationAction = IModerationAction;
export type Appeal = IAppeal;
export type AppealPage = IAppealPage;
export type { ActionKind, AppealDecision, AppealStatus, QueueScope, ReportStatus };

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
      throw new ModerationApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export async function listReports(scope?: QueueScope, status?: ReportStatus): Promise<ReportPage> {
  return unwrap((await client()).moderation.listReports(scope, status));
}

export async function takeActionOnReport(reportId: string, actionKind: ActionKind, reason: string): Promise<ModerationAction> {
  return unwrap((await client()).moderation.takeActionOnReport(reportId, { actionKind, reason }));
}

export async function reverseAction(actionId: string, reason: string): Promise<ModerationAction> {
  return unwrap((await client()).moderation.reverseAction(actionId, { reason }));
}

export async function listAppeals(status?: AppealStatus): Promise<AppealPage> {
  return unwrap((await client()).moderation.listAppeals(status));
}

export async function decideAppeal(appealId: string, decision: AppealDecision, note?: string): Promise<Appeal> {
  return unwrap((await client()).moderation.decideAppeal(appealId, { decision, note }));
}
