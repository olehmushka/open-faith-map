"use client";

import * as React from "react";
import {
  type ColumnDef,
  type ExpandedState,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

/**
 * Generic sort/filter shell over @tanstack/react-table for the admin queues (candidates, reports,
 * appeals). Deliberately does NOT own pagination — every queue's "Load more" is real server-side
 * keyset pagination (lib/congregation-import.ts, lib/moderation.ts), which this component can't
 * replace; callers keep accumulating fetched rows themselves and just hand the growing array here
 * for client-side sort/filter over whatever has been loaded so far.
 */
export function DataTable<TData>({
  columns,
  data,
  globalFilterPlaceholder,
  renderExpanded,
  emptyMessage,
}: {
  columns: ColumnDef<TData>[];
  data: TData[];
  globalFilterPlaceholder?: string;
  renderExpanded?: (row: TData) => React.ReactNode;
  emptyMessage?: string;
}) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [globalFilter, setGlobalFilter] = React.useState("");
  const [expanded, setExpanded] = React.useState<ExpandedState>({});

  const table = useReactTable({
    data,
    columns,
    state: { sorting, globalFilter, expanded },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    onExpandedChange: setExpanded,
    getRowCanExpand: () => Boolean(renderExpanded),
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
  });

  return (
    <div className="flex flex-col gap-3">
      {globalFilterPlaceholder && (
        <Input
          value={globalFilter}
          onChange={(e) => setGlobalFilter(e.target.value)}
          placeholder={globalFilterPlaceholder}
          className="max-w-xs"
        />
      )}
      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  const sortable = header.column.getCanSort();
                  const sortDirection = header.column.getIsSorted();
                  return (
                    <TableHead key={header.id}>
                      {header.isPlaceholder ? null : sortable ? (
                        <button
                          type="button"
                          onClick={header.column.getToggleSortingHandler()}
                          className="flex items-center gap-1 hover:text-foreground"
                        >
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {sortDirection === "asc" ? (
                            <ArrowUp className="size-3.5" />
                          ) : sortDirection === "desc" ? (
                            <ArrowDown className="size-3.5" />
                          ) : (
                            <ArrowUpDown className="size-3.5 text-muted-foreground/50" />
                          )}
                        </button>
                      ) : (
                        flexRender(header.column.columnDef.header, header.getContext())
                      )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <React.Fragment key={row.id}>
                  <TableRow
                    aria-expanded={row.getIsExpanded() || undefined}
                    className={cn(renderExpanded && "cursor-pointer")}
                    onClick={renderExpanded ? () => row.toggleExpanded() : undefined}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                  {row.getIsExpanded() && renderExpanded && (
                    <TableRow>
                      <TableCell colSpan={row.getVisibleCells().length} className="bg-muted/30">
                        <div onClick={(e) => e.stopPropagation()}>
                          {renderExpanded(row.original)}
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </React.Fragment>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                  {emptyMessage}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
