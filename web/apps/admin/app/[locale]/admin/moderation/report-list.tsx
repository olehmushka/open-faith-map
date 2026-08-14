// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useMemo, useState, useTransition } from "react";
import { useTranslations } from "next-intl";
import type { ColumnDef } from "@tanstack/react-table";

import type { ActionKind } from "@/lib/moderation";
import type { IReport, IReportPage } from "@/lib/openfaithmap/generated/moderation";
import { DataTable } from "@/components/data-table";
import { ReportStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const ACTION_KINDS: ActionKind[] = ["HIDE", "SUSPEND", "ARCHIVE", "WARN_ADMIN", "REVOKE_VOUCH"];

// M7 (docs/modules/hardening.md): the first "Load more" pattern in this app — openfaithmap-api's
// ListReports has real keyset pagination as of this milestone, previously always returned every
// row in one shot. Pagination state lives here, purely additive to takeAction's own lifecycle
// (still a page-level Server Action that redirect()s after a mutation, resetting to page 1 —
// acceptable, since taking an action changes queue contents anyway). Rows render through
// DataTable, same reusable sort/filter shell as CandidateList; the take-action form is small
// enough to stay inline in an actions cell rather than needing a row-expansion panel.
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
    reasonPlaceholder: string;
    takeAction: string;
    loadMore: string;
    loading: string;
  };
}) {
  const t = useTranslations("ModerationQueuePage");
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

  const columns = useMemo<ColumnDef<IReport>[]>(
    () => [
      {
        id: "target",
        header: t("target"),
        accessorFn: (r) => `${r.targetKind} ${r.targetRef}`,
        cell: ({ row }) => (
          <div className="flex flex-col">
            <span className="font-medium">{row.original.targetKind}</span>
            <span className="text-xs text-muted-foreground">{row.original.targetRef}</span>
          </div>
        ),
      },
      {
        accessorKey: "reasonCode",
        header: t("reasonCode"),
      },
      {
        accessorKey: "status",
        header: t("status"),
        cell: ({ row }) => <ReportStatusBadge status={row.original.status} />,
      },
      {
        id: "filedAt",
        header: t("filedAtHeader"),
        accessorKey: "createdAt",
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">{row.original.createdAt}</span>
        ),
      },
      {
        id: "actions",
        header: "",
        enableSorting: false,
        cell: ({ row }) => {
          const r = row.original;
          return (
            <form action={takeAction} className="flex flex-wrap items-center gap-2" onClick={(e) => e.stopPropagation()}>
              <input type="hidden" name="reportId" value={r.id} />
              <Select name="actionKind" defaultValue={ACTION_KINDS[0]}>
                <SelectTrigger size="sm" className="w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ACTION_KINDS.map((kind) => (
                    <SelectItem key={kind} value={kind}>
                      {kind}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input name="reason" placeholder={labels.reasonPlaceholder} required className="h-8 w-48" />
              <Button type="submit" size="sm">
                {labels.takeAction}
              </Button>
            </form>
          );
        },
      },
    ],
    [t, takeAction, labels],
  );

  if (reports.length === 0) {
    return <p className="text-muted-foreground">{labels.noReports}</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      <DataTable
        columns={columns}
        data={reports}
        globalFilterPlaceholder={t("filterReports")}
        emptyMessage={labels.noReports}
      />
      {nextPageToken && (
        <Button type="button" variant="outline" size="sm" onClick={handleLoadMore} disabled={isPending} className="self-start">
          {isPending ? labels.loading : labels.loadMore}
        </Button>
      )}
    </div>
  );
}
