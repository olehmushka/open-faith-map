// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";

import type { ReparentingJob, RegistrationRequest } from "@/lib/registration";
import { DataTable } from "@/components/data-table";
import { ReparentStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type Row = { request: RegistrationRequest; job: ReparentingJob | null };

export function ReparentList({
  requests,
  jobByRequestId,
  onReparent,
  labels,
}: {
  requests: RegistrationRequest[];
  jobByRequestId: Map<string, ReparentingJob>;
  onReparent: (formData: FormData) => void;
  labels: {
    noApprovedCongregations: string;
    congregationName: string;
    currentJurisdictionById: Record<string, string>;
    lastMoveById: Record<string, string>;
    unitLabelById: Record<string, string>;
    newParentUnitIdPlaceholder: string;
    resumeMove: string;
    reparentButton: string;
    filterCongregations: string;
  };
}) {
  const rows: Row[] = useMemo(
    () => requests.map((request) => ({ request, job: jobByRequestId.get(request.id) ?? null })),
    [requests, jobByRequestId],
  );

  const columns = useMemo<ColumnDef<Row>[]>(
    () => [
      {
        id: "congregationName",
        header: labels.congregationName,
        accessorFn: (r) => r.request.congregationName,
        cell: ({ row }) => (
          <div className="flex flex-col">
            <span className="font-medium">{row.original.request.congregationName}</span>
            <span className="text-xs text-muted-foreground">
              {labels.unitLabelById[row.original.request.id]}
            </span>
          </div>
        ),
      },
      {
        id: "jurisdiction",
        header: "",
        accessorFn: (r) => r.request.jurisdictionUnitId ?? "",
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {labels.currentJurisdictionById[row.original.request.id]}
          </span>
        ),
      },
      {
        id: "job",
        header: "",
        enableSorting: false,
        cell: ({ row }) => {
          const job = row.original.job;
          if (!job) return null;
          return (
            <div className="flex flex-col gap-1 text-sm">
              <ReparentStatusBadge status={job.status} />
              <span className="text-xs text-muted-foreground">
                {labels.lastMoveById[row.original.request.id]}
                {job.error && <span className="text-destructive"> ({job.error})</span>}
              </span>
            </div>
          );
        },
      },
      {
        id: "actions",
        header: "",
        enableSorting: false,
        cell: ({ row }) => {
          const { request, job } = row.original;
          const resuming = job && job.status !== "VERIFIED" && job.status !== "FAILED";
          return (
            <form action={onReparent} className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
              <input type="hidden" name="id" value={request.id} />
              <Input name="newParentUnitId" placeholder={labels.newParentUnitIdPlaceholder} required className="h-8 w-48" />
              <Button type="submit" size="sm">
                {resuming ? labels.resumeMove : labels.reparentButton}
              </Button>
            </form>
          );
        },
      },
    ],
    [labels, onReparent],
  );

  if (requests.length === 0) {
    return <p className="text-muted-foreground">{labels.noApprovedCongregations}</p>;
  }

  return (
    <DataTable
      columns={columns}
      data={rows}
      globalFilterPlaceholder={labels.filterCongregations}
      emptyMessage={labels.noApprovedCongregations}
    />
  );
}
