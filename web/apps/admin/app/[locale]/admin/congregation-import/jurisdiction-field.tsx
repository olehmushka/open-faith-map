// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState, useTransition } from "react";
import { useTranslations } from "next-intl";
import { Plus, Search } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type UnitOption = { id: string; code: string | null; name: string };

// slugify: no such helper exists anywhere in web/apps/admin yet (checked) — a small, local one for
// the create-unit modal's Code field default, editable before submit either way.
function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "") // combining diacritical marks left behind by NFKD
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64);
}

/**
 * Search-and-select a jurisdiction unit, plus "create a missing one on the spot" — replaces a bare
 * jurisdictionUnitId <input>. Renders a hidden input carrying the real value, so it must be a
 * descendant of the enclosing Approve <form> — the shadcn Dialog itself still portals to
 * document.body (Radix's own DialogPortal), so it's safe to nest visually inside that outer
 * <form> without producing invalid nested-<form> HTML.
 */
export function JurisdictionField({
  candidateId,
  candidateName,
  jurisdictionHint,
  suggestedJurisdictionUnitId,
  rootUnitId,
  onSearch,
  onCreateUnit,
  labels,
}: {
  candidateId: string;
  candidateName: string;
  jurisdictionHint?: string | null;
  suggestedJurisdictionUnitId?: string | null;
  rootUnitId: string;
  onSearch: (query: string) => Promise<UnitOption[]>;
  onCreateUnit: (parentUnitId: string, code: string, name: string) => Promise<UnitOption>;
  labels: {
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
  };
}) {
  const t = useTranslations("CongregationImportPage");
  const [selectedId, setSelectedId] = useState(suggestedJurisdictionUnitId ?? "");
  const [selectedLabel, setSelectedLabel] = useState(
    suggestedJurisdictionUnitId ? `${suggestedJurisdictionUnitId} (${t("suggestedJurisdiction")})` : "",
  );
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<UnitOption[] | null>(null);
  const [isPending, startTransition] = useTransition();
  const [dialogOpen, setDialogOpen] = useState(false);
  const nameInputId = `unit-name-${candidateId}`;

  function select(unit: UnitOption) {
    setSelectedId(unit.id);
    setSelectedLabel(unit.code ? `${unit.name} (${unit.code})` : unit.name);
  }

  function handleSearch() {
    if (!query.trim()) return;
    startTransition(async () => {
      setResults(await onSearch(query.trim()));
    });
  }

  function handleCreate(formData: FormData) {
    const parentUnitId = String(formData.get("parentUnitId") ?? "").trim() || rootUnitId;
    const code = String(formData.get("code") ?? "").trim();
    const name = String(formData.get("name") ?? "").trim();
    if (!code || !name) return;
    startTransition(async () => {
      const unit = await onCreateUnit(parentUnitId, code, name);
      select(unit);
      setDialogOpen(false);
    });
  }

  return (
    <div className="flex flex-col gap-1">
      <input type="hidden" name="jurisdictionUnitId" value={selectedId} />
      <span className="text-xs text-muted-foreground">
        {labels.jurisdictionUnitId}: {selectedLabel || labels.jurisdictionNone}
      </span>
      <div className="flex flex-wrap gap-2">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={labels.jurisdictionSearchPlaceholder}
          className="h-8 w-56"
        />
        <Button type="button" variant="outline" size="sm" onClick={handleSearch} disabled={isPending}>
          <Search className="size-3.5" />
          {labels.jurisdictionSearch}
        </Button>
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <Button type="button" variant="outline" size="sm" onClick={() => setDialogOpen(true)}>
            <Plus className="size-3.5" />
            {labels.createUnit}
          </Button>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{labels.createUnitHeading}</DialogTitle>
            </DialogHeader>
            <form
              action={handleCreate}
              id={`create-unit-form-${candidateId}`}
              className="flex flex-col gap-3"
            >
              <Label htmlFor={nameInputId} className="flex flex-col items-start gap-1 text-xs">
                {labels.createUnitName}
                <Input
                  id={nameInputId}
                  name="name"
                  required
                  defaultValue={jurisdictionHint ?? candidateName}
                />
              </Label>
              <Label className="flex flex-col items-start gap-1 text-xs">
                {labels.createUnitCode}
                <Input name="code" required defaultValue={slugify(jurisdictionHint ?? candidateName)} />
              </Label>
              <Label className="flex flex-col items-start gap-1 text-xs">
                {labels.createUnitParentUnitId}
                <Input name="parentUnitId" defaultValue={rootUnitId} />
              </Label>
            </form>
            <DialogFooter>
              <DialogClose asChild>
                <Button type="button" variant="outline">
                  {labels.createUnitCancel}
                </Button>
              </DialogClose>
              <Button type="submit" form={`create-unit-form-${candidateId}`} disabled={isPending}>
                {labels.createUnitSubmit}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
      {results && (
        <ul className="flex flex-col gap-1 text-sm">
          {results.length === 0 && <li className="text-muted-foreground">{labels.jurisdictionNoMatches}</li>}
          {results.map((u) => (
            <li key={u.id}>
              <button
                type="button"
                onClick={() => select(u)}
                className="rounded border px-2 py-0.5 text-left text-sm hover:bg-muted"
              >
                {u.name} {u.code && <span className="text-muted-foreground">({u.code})</span>}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
