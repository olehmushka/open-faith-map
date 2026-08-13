// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState, useTransition } from "react";
import { useTranslations } from "next-intl";

import type { Candidate, CandidatePage } from "@/lib/congregation-import";

// Mirrors moderation's own ReportList exactly (M7's "Load more" pattern,
// docs/modules/hardening.md): pagination state lives here, purely additive to the
// edit/approve/reject Server Actions' own lifecycle (each still redirect()s after a mutation,
// resetting to page 1 — acceptable, since a mutation changes queue contents anyway). Every
// formatted string is produced by this client component's own useTranslations call, never passed
// in as a function prop — a real bug M7 hit once already (a Server Component closure crossing into
// a Client Component is rejected by Next.js, only on a real request, not by any static check).
export function CandidateList({
  initialCandidates,
  initialNextPageToken,
  loadMore,
  onEdit,
  onApprove,
  onReject,
  labels,
}: {
  initialCandidates: Candidate[];
  initialNextPageToken: string | null | undefined;
  loadMore: (pageToken: string) => Promise<CandidatePage>;
  onEdit: (formData: FormData) => void;
  onApprove: (formData: FormData) => void;
  onReject: (formData: FormData) => void;
  labels: {
    noCandidates: string;
    taxonId: string;
    countryId: string;
    latitude: string;
    longitude: string;
    save: string;
    jurisdictionUnitId: string;
    approve: string;
    reasonPlaceholder: string;
    reject: string;
    loadMore: string;
    loading: string;
  };
}) {
  const t = useTranslations("CongregationImportPage");
  const [candidates, setCandidates] = useState(initialCandidates);
  const [nextPageToken, setNextPageToken] = useState(initialNextPageToken ?? null);
  const [isPending, startTransition] = useTransition();

  function handleLoadMore() {
    if (!nextPageToken) return;
    const token = nextPageToken;
    startTransition(async () => {
      const page = await loadMore(token);
      setCandidates((prev) => [...prev, ...page.candidates]);
      setNextPageToken(page.nextPageToken ?? null);
    });
  }

  if (candidates.length === 0) {
    return <p>{labels.noCandidates}</p>;
  }

  return (
    <>
      <ul className="flex flex-col gap-4">
        {candidates.map((c) => (
          <li key={c.id} className="rounded border p-4">
            <div className="flex items-baseline justify-between gap-2">
              <span className="font-medium">{c.name}</span>
              <span className="text-sm">{c.status}</span>
            </div>
            <p className="text-sm text-gray-500">
              {c.sourceCode} · {c.sourceRecordId}
            </p>
            {c.taxonHint && (
              <p className="text-sm">
                {t("taxonHint")}: {c.taxonHint}
              </p>
            )}
            {c.jurisdictionHint && (
              <p className="text-sm">
                {t("jurisdictionHint")}: {c.jurisdictionHint}
              </p>
            )}
            {c.suggestedJurisdictionUnitId && (
              <p className="text-sm">
                {t("suggestedJurisdiction")}: <code className="rounded bg-gray-100 px-1">{c.suggestedJurisdictionUnitId}</code>
              </p>
            )}

            <form action={onEdit} className="mt-3 flex flex-wrap items-end gap-2">
              <input type="hidden" name="id" value={c.id} />
              <label className="flex flex-col text-xs">
                {labels.taxonId}
                <input name="taxonId" defaultValue={c.taxonId ?? ""} className="rounded border px-2 py-1 text-sm" />
              </label>
              <label className="flex flex-col text-xs">
                {labels.countryId}
                <input name="countryId" defaultValue={c.countryId ?? ""} className="rounded border px-2 py-1 text-sm" />
              </label>
              <label className="flex flex-col text-xs">
                {labels.latitude}
                <input name="latitude" defaultValue={c.latitude ?? ""} className="w-24 rounded border px-2 py-1 text-sm" />
              </label>
              <label className="flex flex-col text-xs">
                {labels.longitude}
                <input name="longitude" defaultValue={c.longitude ?? ""} className="w-24 rounded border px-2 py-1 text-sm" />
              </label>
              <button type="submit" className="rounded border px-3 py-1 text-sm">
                {labels.save}
              </button>
            </form>

            <div className="mt-2 flex flex-wrap gap-2">
              <form action={onApprove} className="flex gap-2">
                <input type="hidden" name="id" value={c.id} />
                <input
                  name="jurisdictionUnitId"
                  defaultValue={c.suggestedJurisdictionUnitId ?? ""}
                  placeholder={labels.jurisdictionUnitId}
                  className="rounded border px-2 py-1 text-sm"
                />
                <button type="submit" className="rounded border px-3 py-1 text-sm">
                  {labels.approve}
                </button>
              </form>
              <form action={onReject} className="flex gap-2">
                <input type="hidden" name="id" value={c.id} />
                <input
                  name="reason"
                  placeholder={labels.reasonPlaceholder}
                  required
                  className="rounded border px-2 py-1 text-sm"
                />
                <button type="submit" className="rounded border px-3 py-1 text-sm">
                  {labels.reject}
                </button>
              </form>
            </div>
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
