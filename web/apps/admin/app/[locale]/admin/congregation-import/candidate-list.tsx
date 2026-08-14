// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useMemo, useState, useTransition } from "react";
import { useTranslations } from "next-intl";
import type { ColumnDef } from "@tanstack/react-table";
import { ChevronRight } from "lucide-react";

import type { Candidate, CandidatePage } from "@/lib/congregation-import";
import { DataTable } from "@/components/data-table";
import { CandidateStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { CoordinateSuggest } from "./coordinate-suggest";
import { JurisdictionField } from "./jurisdiction-field";

type PickerOption = { id: string; name: string };
type UnitOption = { id: string; code: string | null; name: string };

// Radix's Select.Item forbids an empty-string value (that's reserved to mean "nothing selected"),
// so the taxon/country "unset" option needs a real sentinel value instead — page.tsx's edit()
// action strips it back to `undefined` before calling editCandidate.
export const UNSET_OPTION = "__unset__";

// Mirrors moderation's own ReportList exactly (M7's "Load more" pattern,
// docs/modules/hardening.md): pagination state lives here, purely additive to the
// edit/approve/reject Server Actions' own lifecycle (each still redirect()s after a mutation,
// resetting to page 1 — acceptable, since a mutation changes queue contents anyway). Every
// formatted string is produced by this client component's own useTranslations call, never passed
// in as a function prop — a real bug M7 hit once already (a Server Component closure crossing into
// a Client Component is rejected by Next.js, only on a real request, not by any static check).
//
// Rows render through DataTable (@tanstack/react-table) for sort/filter over whatever's been
// loaded so far; "Load more" keeps driving the real server-side keyset pagination underneath it
// unchanged. Per-row edit/approve/reject stays the same rich form set as before (taxon/country
// pickers, JurisdictionField's own search+create dialog, CoordinateSuggest's geocode round-trip) —
// too much for a table cell, so it now lives in an expandable row panel instead of a card.
export function CandidateList({
  initialCandidates,
  initialNextPageToken,
  loadMore,
  onEdit,
  onApprove,
  onReject,
  taxa,
  countries,
  rootUnitId,
  onSearchJurisdiction,
  onCreateUnit,
  onSuggestCoordinates,
  labels,
}: {
  initialCandidates: Candidate[];
  initialNextPageToken: string | null | undefined;
  loadMore: (pageToken: string) => Promise<CandidatePage>;
  onEdit: (formData: FormData) => void;
  onApprove: (formData: FormData) => void;
  onReject: (formData: FormData) => void;
  taxa: PickerOption[];
  countries: PickerOption[];
  rootUnitId: string;
  onSearchJurisdiction: (query: string) => Promise<UnitOption[]>;
  onCreateUnit: (parentUnitId: string, code: string, name: string) => Promise<UnitOption>;
  onSuggestCoordinates: (
    candidateId: string,
  ) => Promise<{ latitude: number | "NaN"; longitude: number | "NaN"; precision?: string | null; displayName: string; provider: string }>;
  labels: {
    noCandidates: string;
    taxonId: string;
    taxonUnset: string;
    countryId: string;
    countryUnset: string;
    latitude: string;
    longitude: string;
    suggestCoordinates: string;
    suggesting: string;
    suggestedVia: string;
    geocodeNoMatch: string;
    geocodeLookupFailed: string;
    save: string;
    jurisdictionUnitId: string;
    jurisdictionNone: string;
    jurisdictionSearchPlaceholder: string;
    jurisdictionSearch: string;
    jurisdictionNoMatches: string;
    createUnit: string;
    createUnitHeading: string;
    createUnitName: string;
    createUnitCode: string;
    createUnitParentUnitId: string;
    createUnitSubmit: string;
    createUnitCancel: string;
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

  const columns = useMemo<ColumnDef<Candidate>[]>(
    () => [
      {
        id: "expand",
        header: "",
        cell: ({ row }) => (
          <ChevronRight
            className={`size-4 text-muted-foreground transition-transform ${row.getIsExpanded() ? "rotate-90" : ""}`}
          />
        ),
        enableSorting: false,
      },
      {
        accessorKey: "name",
        header: t("name"),
        cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      },
      {
        accessorKey: "status",
        header: t("status"),
        cell: ({ row }) => <CandidateStatusBadge status={row.original.status} />,
      },
      {
        id: "source",
        header: t("source"),
        accessorFn: (c) => `${c.sourceCode} ${c.sourceRecordId}`,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {row.original.sourceCode} · {row.original.sourceRecordId}
          </span>
        ),
      },
      {
        accessorKey: "taxonHint",
        header: t("taxonHint"),
        cell: ({ row }) => row.original.taxonHint ?? "—",
      },
      {
        accessorKey: "jurisdictionHint",
        header: t("jurisdictionHint"),
        cell: ({ row }) => row.original.jurisdictionHint ?? "—",
      },
    ],
    [t],
  );

  if (candidates.length === 0) {
    return <p className="text-muted-foreground">{labels.noCandidates}</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      <DataTable
        columns={columns}
        data={candidates}
        globalFilterPlaceholder={t("filterCandidates")}
        emptyMessage={labels.noCandidates}
        renderExpanded={(c) => (
          <div className="flex flex-col gap-4 py-2">
            {c.suggestedJurisdictionUnitId && (
              <p className="text-sm">
                {t("suggestedJurisdiction")}: <code className="rounded bg-muted px-1">{c.suggestedJurisdictionUnitId}</code>
              </p>
            )}

            <form action={onEdit} className="flex flex-wrap items-end gap-3">
              <input type="hidden" name="id" value={c.id} />
              <Label className="flex flex-col items-start gap-1 text-xs">
                {labels.taxonId}
                {/* Radix's Select.Root renders a hidden native <select> for `name` when given one,
                    so this still participates in onEdit's plain FormData the same as before. */}
                <Select name="taxonId" defaultValue={c.taxonId ?? UNSET_OPTION}>
                  <SelectTrigger size="sm" className="w-48">
                    <SelectValue placeholder={labels.taxonUnset} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={UNSET_OPTION}>{labels.taxonUnset}</SelectItem>
                    {taxa.map((taxon) => (
                      <SelectItem key={taxon.id} value={taxon.id}>
                        {taxon.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Label>
              <Label className="flex flex-col items-start gap-1 text-xs">
                {labels.countryId}
                <Select name="countryId" defaultValue={c.countryId ?? UNSET_OPTION}>
                  <SelectTrigger size="sm" className="w-48">
                    <SelectValue placeholder={labels.countryUnset} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={UNSET_OPTION}>{labels.countryUnset}</SelectItem>
                    {countries.map((country) => (
                      <SelectItem key={country.id} value={country.id}>
                        {country.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Label>
              <CoordinateSuggest
                defaultLatitude={c.latitude}
                defaultLongitude={c.longitude}
                onSuggest={() => onSuggestCoordinates(c.id)}
                labels={labels}
              />
              <Button type="submit" size="sm">
                {labels.save}
              </Button>
            </form>

            <div className="flex flex-wrap gap-4 border-t pt-3">
              <form action={onApprove} className="flex flex-col gap-2">
                <input type="hidden" name="id" value={c.id} />
                <JurisdictionField
                  candidateId={c.id}
                  candidateName={c.name}
                  jurisdictionHint={c.jurisdictionHint}
                  suggestedJurisdictionUnitId={c.suggestedJurisdictionUnitId}
                  rootUnitId={rootUnitId}
                  onSearch={onSearchJurisdiction}
                  onCreateUnit={onCreateUnit}
                  labels={labels}
                />
                <Button type="submit" size="sm" className="self-start">
                  {labels.approve}
                </Button>
              </form>
              <form action={onReject} className="flex items-end gap-2">
                <input type="hidden" name="id" value={c.id} />
                <Input name="reason" placeholder={labels.reasonPlaceholder} required className="h-8 w-56" />
                <Button type="submit" size="sm" variant="destructive">
                  {labels.reject}
                </Button>
              </form>
            </div>
          </div>
        )}
      />
      {nextPageToken && (
        <Button type="button" variant="outline" size="sm" onClick={handleLoadMore} disabled={isPending} className="self-start">
          {isPending ? labels.loading : labels.loadMore}
        </Button>
      )}
    </div>
  );
}

