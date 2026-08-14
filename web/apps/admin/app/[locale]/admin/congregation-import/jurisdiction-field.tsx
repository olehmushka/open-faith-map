// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useEffect, useRef, useState, useTransition } from "react";
import { createPortal } from "react-dom";
import { useTranslations } from "next-intl";

type UnitOption = { id: string; code: string | null; name: string };

// slugify: no such helper exists anywhere in web/apps/admin yet (checked) — a small, local one for
// the create-unit modal's Code field default, editable before submit either way.
function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "") // combining diacritical marks left behind by NFKD
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64);
}

/**
 * Search-and-select a jurisdiction unit, plus "create a missing one on the spot" — replaces a bare
 * jurisdictionUnitId <input>. Renders a hidden input carrying the real value, so it must be a
 * descendant of the enclosing Approve <form> — but the <dialog> itself is portaled to
 * document.body, since a <dialog><form> nested inside that outer <form> would be invalid HTML
 * (forms cannot nest) and behave unpredictably across browsers.
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
  const [mounted, setMounted] = useState(false);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const nameInputId = `unit-name-${candidateId}`;

  useEffect(() => setMounted(true), []);

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
      dialogRef.current?.close();
    });
  }

  return (
    <div className="flex flex-col gap-1">
      <input type="hidden" name="jurisdictionUnitId" value={selectedId} />
      <span className="text-xs">
        {labels.jurisdictionUnitId}: {selectedLabel || labels.jurisdictionNone}
      </span>
      <div className="flex flex-wrap gap-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={labels.jurisdictionSearchPlaceholder}
          className="rounded border px-2 py-1 text-sm"
        />
        <button type="button" onClick={handleSearch} disabled={isPending} className="rounded border px-2 py-1 text-sm">
          {labels.jurisdictionSearch}
        </button>
        <button
          type="button"
          onClick={() => dialogRef.current?.showModal()}
          className="rounded border px-2 py-1 text-sm"
        >
          {labels.createUnit}
        </button>
      </div>
      {results && (
        <ul className="flex flex-col gap-1 text-sm">
          {results.length === 0 && <li>{labels.jurisdictionNoMatches}</li>}
          {results.map((u) => (
            <li key={u.id}>
              <button
                type="button"
                onClick={() => select(u)}
                className="rounded border px-2 py-0.5 text-left text-sm hover:bg-gray-50"
              >
                {u.name} {u.code && <span className="text-gray-500">({u.code})</span>}
              </button>
            </li>
          ))}
        </ul>
      )}

      {mounted &&
        createPortal(
          <dialog ref={dialogRef} className="rounded border p-4 backdrop:bg-black/30">
            <form action={handleCreate} className="flex flex-col gap-3" style={{ minWidth: "20rem" }}>
              <h3 className="font-medium">{labels.createUnitHeading}</h3>
              <label className="flex flex-col text-xs" htmlFor={nameInputId}>
                {labels.createUnitName}
                <input
                  id={nameInputId}
                  name="name"
                  required
                  defaultValue={jurisdictionHint ?? candidateName}
                  className="rounded border px-2 py-1 text-sm"
                />
              </label>
              <label className="flex flex-col text-xs">
                {labels.createUnitCode}
                <input
                  name="code"
                  required
                  defaultValue={slugify(jurisdictionHint ?? candidateName)}
                  className="rounded border px-2 py-1 text-sm"
                />
              </label>
              <label className="flex flex-col text-xs">
                {labels.createUnitParentUnitId}
                <input name="parentUnitId" defaultValue={rootUnitId} className="rounded border px-2 py-1 text-sm" />
              </label>
              <div className="flex justify-end gap-2">
                <button type="button" onClick={() => dialogRef.current?.close()} className="rounded border px-3 py-1 text-sm">
                  {labels.createUnitCancel}
                </button>
                <button type="submit" disabled={isPending} className="rounded border px-3 py-1 text-sm">
                  {labels.createUnitSubmit}
                </button>
              </div>
            </form>
          </dialog>,
          document.body,
        )}
    </div>
  );
}
