// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState, useTransition } from "react";

import type { IAppeal, IAppealPage } from "@/lib/openfaithmap/generated/moderation";

// M7 (docs/modules/hardening.md): same "Load more" pattern as ../report-list.tsx — pagination
// state lives here, purely additive to decide's own lifecycle (still a page-level Server Action
// that redirect()s after a mutation, resetting to page 1 — acceptable, deciding an appeal changes
// queue contents anyway).
export function AppealList({
  initialAppeals,
  initialNextPageToken,
  loadMore,
  decide,
  labels,
}: {
  initialAppeals: IAppeal[];
  initialNextPageToken: string | null | undefined;
  loadMore: (pageToken: string) => Promise<IAppealPage>;
  decide: (formData: FormData) => void;
  labels: {
    noAppeals: string;
    filedAt: (date: string) => string;
    notePlaceholder: string;
    uphold: string;
    overturn: string;
    loadMore: string;
    loading: string;
  };
}) {
  const [appeals, setAppeals] = useState(initialAppeals);
  const [nextPageToken, setNextPageToken] = useState(initialNextPageToken ?? null);
  const [isPending, startTransition] = useTransition();

  function handleLoadMore() {
    if (!nextPageToken) return;
    const token = nextPageToken;
    startTransition(async () => {
      const page = await loadMore(token);
      setAppeals((prev) => [...prev, ...page.appeals]);
      setNextPageToken(page.nextPageToken ?? null);
    });
  }

  if (appeals.length === 0) {
    return <p>{labels.noAppeals}</p>;
  }

  return (
    <>
      <ul className="flex flex-col gap-4">
        {appeals.map((a) => (
          <li key={a.id} className="rounded border p-4">
            <p className="text-sm">{a.statement}</p>
            <p className="text-sm text-gray-500">{labels.filedAt(a.createdAt)}</p>

            <form action={decide} className="mt-3 flex flex-wrap gap-2">
              <input type="hidden" name="appealId" value={a.id} />
              <input
                name="note"
                placeholder={labels.notePlaceholder}
                className="rounded border px-2 py-1 text-sm"
              />
              <button type="submit" name="decision" value="UPHELD" className="rounded border px-3 py-1 text-sm">
                {labels.uphold}
              </button>
              <button
                type="submit"
                name="decision"
                value="OVERTURNED"
                className="rounded border px-3 py-1 text-sm"
              >
                {labels.overturn}
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
