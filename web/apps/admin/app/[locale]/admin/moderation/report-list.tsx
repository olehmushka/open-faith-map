// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState, useTransition } from "react";

import type { ActionKind } from "@/lib/moderation";
import type { IReport, IReportPage } from "@/lib/openfaithmap/generated/moderation";

const ACTION_KINDS: ActionKind[] = ["HIDE", "SUSPEND", "ARCHIVE", "WARN_ADMIN", "REVOKE_VOUCH"];

// M7 (docs/modules/hardening.md): the first "Load more" pattern in this app — openfaithmap-api's
// ListReports has real keyset pagination as of this milestone, previously always returned every
// row in one shot. Pagination state lives here, purely additive to takeAction's own lifecycle
// (still a page-level Server Action that redirect()s after a mutation, resetting to page 1 —
// acceptable, since taking an action changes queue contents anyway).
export function ReportList({
  initialReports,
  initialNextPageToken,
  loadMore,
  takeAction,
  labels,
}: {
  initialReports: IReport[];
  initialNextPageToken: string | null | undefined;
  loadMore: (pageToken: string) => Promise<IReportPage>;
  takeAction: (formData: FormData) => void;
  labels: {
    noReports: string;
    filedAt: (date: string) => string;
    reasonPlaceholder: string;
    takeAction: string;
    loadMore: string;
    loading: string;
  };
}) {
  const [reports, setReports] = useState(initialReports);
  const [nextPageToken, setNextPageToken] = useState(initialNextPageToken ?? null);
  const [isPending, startTransition] = useTransition();

  function handleLoadMore() {
    if (!nextPageToken) return;
    const token = nextPageToken;
    startTransition(async () => {
      const page = await loadMore(token);
      setReports((prev) => [...prev, ...page.reports]);
      setNextPageToken(page.nextPageToken ?? null);
    });
  }

  if (reports.length === 0) {
    return <p>{labels.noReports}</p>;
  }

  return (
    <>
      <ul className="flex flex-col gap-4">
        {reports.map((r) => (
          <li key={r.id} className="rounded border p-4">
            <div className="flex items-baseline justify-between">
              <span className="font-medium">
                {r.targetKind}: {r.targetRef}
              </span>
              <span className="text-sm">{r.reasonCode}</span>
            </div>
            {r.detail && <p className="text-sm">{r.detail}</p>}
            <p className="text-sm text-gray-500">{labels.filedAt(r.createdAt)}</p>

            <form action={takeAction} className="mt-3 flex flex-wrap gap-2">
              <input type="hidden" name="reportId" value={r.id} />
              <select name="actionKind" className="rounded border px-2 py-1 text-sm" defaultValue={ACTION_KINDS[0]}>
                {ACTION_KINDS.map((kind) => (
                  <option key={kind} value={kind}>
                    {kind}
                  </option>
                ))}
              </select>
              <input
                name="reason"
                placeholder={labels.reasonPlaceholder}
                required
                className="rounded border px-2 py-1 text-sm"
              />
              <button type="submit" className="rounded border px-3 py-1 text-sm">
                {labels.takeAction}
              </button>
            </form>
          </li>
        ))}
      </ul>
      {nextPageToken && (
        <button
          type="button"
          onClick={handleLoadMore}
          disabled={isPending}
          className="self-start rounded border px-3 py-1 text-sm"
        >
          {isPending ? labels.loading : labels.loadMore}
        </button>
      )}
    </>
  );
}
