// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useMemo, useState, useTransition } from "react";
import type { ColumnDef } from "@tanstack/react-table";

import type { AuditLogEntry, AuditLogPage } from "@/lib/core";
import { DataTable } from "@/components/data-table";
import { Button } from "@/components/ui/button";

// Cloned from moderation/report-list.tsx's shape: DataTable owns sort/global-filter/row-expansion
// only, so "Load more" state lives here (M11.2's listAuditLog has the same real keyset pagination
// as listReports/listAppeals, M7). This is the first real use of DataTable's renderExpanded prop —
// before/after are shown as pretty-printed JSON in the expanded row rather than a dedicated column,
// since either can be absent (a create has no before, a delete/revoke has no after) or arbitrarily
// shaped depending on the action.
export function AuditLogList({
  initialEntries,
  initialNextPageToken,
  loadMore,
  labels,
}: {
  initialEntries: AuditLogEntry[];
  initialNextPageToken: string | null | undefined;
  loadMore: (pageToken: string) => Promise<AuditLogPage>;
  labels: {
    noEntries: string;
    filterEntries: string;
    actorColumn: string;
    actionColumn: string;
    targetColumn: string;
    whenColumn: string;
    beforeLabel: string;
    afterLabel: string;
    loadMore: string;
    loading: string;
  };
}) {
  const [entries, setEntries] = useState(initialEntries);
  const [nextPageToken, setNextPageToken] = useState(initialNextPageToken ?? null);
  const [isPending, startTransition] = useTransition();

  function handleLoadMore() {
    if (!nextPageToken) return;
    const token = nextPageToken;
    startTransition(async () => {
      const page = await loadMore(token);
      setEntries((prev) => [...prev, ...page.entries]);
      setNextPageToken(page.nextPageToken ?? null);
    });
  }

  const columns = useMemo<ColumnDef<AuditLogEntry>[]>(
    () => [
      {
        id: "actor",
        header: labels.actorColumn,
        accessorFn: (e) => e.actorPersonName || e.actorPersonId || "",
        cell: ({ row }) => (
          <span className="text-sm">{row.original.actorPersonName || row.original.actorPersonId || "—"}</span>
        ),
      },
      {
        accessorKey: "action",
        header: labels.actionColumn,
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.action}</span>,
      },
      {
        id: "target",
        header: labels.targetColumn,
        accessorFn: (e) => `${e.targetKind} ${e.targetId}`,
        cell: ({ row }) => (
          <div className="flex flex-col">
            <span className="text-xs font-medium">{row.original.targetKind}</span>
            <span className="font-mono text-xs text-muted-foreground">{row.original.targetId}</span>
          </div>
        ),
      },
      {
        id: "when",
        header: labels.whenColumn,
        accessorKey: "createdAt",
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">{row.original.createdAt}</span>
        ),
      },
    ],
    [labels],
  );

  return (
    <div className="flex flex-col gap-3">
      <DataTable
        columns={columns}
        data={entries}
        globalFilterPlaceholder={labels.filterEntries}
        emptyMessage={labels.noEntries}
        renderExpanded={(entry) => (
          <div className="grid grid-cols-1 gap-4 text-xs sm:grid-cols-2">
            <div>
              <div className="mb-1 font-medium text-muted-foreground">{labels.beforeLabel}</div>
              <pre className="overflow-x-auto rounded bg-background p-2">
                {entry.before ? JSON.stringify(entry.before, null, 2) : "—"}
              </pre>
            </div>
            <div>
              <div className="mb-1 font-medium text-muted-foreground">{labels.afterLabel}</div>
              <pre className="overflow-x-auto rounded bg-background p-2">
                {entry.after ? JSON.stringify(entry.after, null, 2) : "—"}
              </pre>
            </div>
          </div>
        )}
      />
      {nextPageToken && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleLoadMore}
          disabled={isPending}
          className="self-start"
        >
          {isPending ? labels.loading : labels.loadMore}
        </Button>
      )}
    </div>
  );
}
