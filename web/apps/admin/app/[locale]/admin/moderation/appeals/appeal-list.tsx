// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useMemo, useState, useTransition } from "react";
import { useTranslations } from "next-intl";
import type { ColumnDef } from "@tanstack/react-table";

import type { IAppeal, IAppealPage } from "@/lib/openfaithmap/generated/moderation";
import { DataTable } from "@/components/data-table";
import { AppealStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// M7 (docs/modules/hardening.md): same "Load more" pattern as ../report-list.tsx — pagination
// state lives here, purely additive to decide's own lifecycle (still a page-level Server Action
// that redirect()s after a mutation, resetting to page 1 — acceptable, deciding an appeal changes
// queue contents anyway). Rows render through DataTable, same shell report-list.tsx uses.
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
    notePlaceholder: string;
    uphold: string;
    overturn: string;
    loadMore: string;
    loading: string;
  };
}) {
  const t = useTranslations("AppealsPage");
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

  const columns = useMemo<ColumnDef<IAppeal>[]>(
    () => [
      {
        accessorKey: "statement",
        header: t("statement"),
        cell: ({ row }) => <span className="line-clamp-2 max-w-md">{row.original.statement}</span>,
      },
      {
        accessorKey: "status",
        header: t("status"),
        cell: ({ row }) => <AppealStatusBadge status={row.original.status} />,
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
          const a = row.original;
          return (
            <form action={decide} className="flex flex-wrap items-center gap-2" onClick={(e) => e.stopPropagation()}>
              <input type="hidden" name="appealId" value={a.id} />
              <Input name="note" placeholder={labels.notePlaceholder} className="h-8 w-48" />
              <Button type="submit" name="decision" value="UPHELD" size="sm" variant="outline">
                {labels.uphold}
              </Button>
              <Button type="submit" name="decision" value="OVERTURNED" size="sm">
                {labels.overturn}
              </Button>
            </form>
          );
        },
      },
    ],
    [t, decide, labels],
  );

  if (appeals.length === 0) {
    return <p className="text-muted-foreground">{labels.noAppeals}</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      <DataTable
        columns={columns}
        data={appeals}
        globalFilterPlaceholder={t("filterAppeals")}
        emptyMessage={labels.noAppeals}
      />
      {nextPageToken && (
        <Button type="button" variant="outline" size="sm" onClick={handleLoadMore} disabled={isPending} className="self-start">
          {isPending ? labels.loading : labels.loadMore}
        </Button>
      )}
    </div>
  );
}
