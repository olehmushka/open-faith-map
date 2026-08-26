// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";

import type { RegistrationRequest } from "@/lib/registration";
import { DataTable } from "@/components/data-table";
import { RegistrationStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// Unlike CandidateList/ReportList/AppealList, listRegistrations has no keyset pagination today —
// the full result set is passed straight through. Still routed through DataTable for consistent
// sort/filter chrome across the admin queues.
export function RequestList({
  requests,
  onApprove,
  onReject,
  labels,
}: {
  requests: RegistrationRequest[];
  onApprove: (formData: FormData) => void;
  onReject: (formData: FormData) => void;
  labels: {
    noRequests: string;
    jurisdictionUnitIdPlaceholder: string;
    approve: string;
    rejectionReasonPlaceholder: string;
    reject: string;
    rejectionReasonById: Record<string, string>;
    filterRequests: string;
    congregationName: string;
    status: string;
    location: string;
  };
}) {
  const columns = useMemo<ColumnDef<RegistrationRequest>[]>(
    () => [
      {
        accessorKey: "congregationName",
        header: labels.congregationName,
        cell: ({ row }) => <span className="font-medium">{row.original.congregationName}</span>,
      },
      {
        accessorKey: "status",
        header: labels.status,
        cell: ({ row }) => <RegistrationStatusBadge status={row.original.status} />,
      },
      {
        id: "location",
        header: labels.location,
        accessorFn: (r) => `${r.locality ?? ""} ${r.street ?? ""}`,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {row.original.locality ?? ""} {row.original.street ?? ""}
          </span>
        ),
      },
      {
        id: "actions",
        header: "",
        enableSorting: false,
        cell: ({ row }) => {
          const r = row.original;
          if (r.status === "PENDING") {
            return (
              <div className="flex flex-wrap items-center gap-2" onClick={(e) => e.stopPropagation()}>
                <form action={onApprove} className="flex items-center gap-2">
                  <input type="hidden" name="id" value={r.id} />
                  <Input name="jurisdictionUnitId" placeholder={labels.jurisdictionUnitIdPlaceholder} className="h-8 w-56" />
                  <Button type="submit" size="sm">
                    {labels.approve}
                  </Button>
                </form>
                <form action={onReject} className="flex items-center gap-2">
                  <input type="hidden" name="id" value={r.id} />
                  <Input name="reason" placeholder={labels.rejectionReasonPlaceholder} required className="h-8 w-40" />
                  <Button type="submit" size="sm" variant="destructive">
                    {labels.reject}
                  </Button>
                </form>
              </div>
            );
          }
          if (r.status === "REJECTED" && r.rejectionReason) {
            return <span className="text-sm text-muted-foreground">{labels.rejectionReasonById[r.id]}</span>;
          }
          return null;
        },
      },
    ],
    [labels, onApprove, onReject],
  );

  if (requests.length === 0) {
    return <p className="text-muted-foreground">{labels.noRequests}</p>;
  }

  return (
    <DataTable
      columns={columns}
      data={requests}
      globalFilterPlaceholder={labels.filterRequests}
      emptyMessage={labels.noRequests}
    />
  );
}
