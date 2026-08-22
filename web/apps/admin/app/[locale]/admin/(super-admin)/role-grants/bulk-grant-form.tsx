// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { type FormEvent, useState, useTransition } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useRouter } from "@/i18n/navigation";
import type { Person, Role } from "@/lib/core";

// M11.7 — the search-and-select half of the bulk role-assign flow. A client component (unlike the
// rest of this page) since checking/unchecking people needs real interactive state; ends in a plain
// Server Action + redirect() like every other mutation on this page, not useActionState — that
// pattern (people/invite/invite-form.tsx) exists specifically because a one-time secret token can't
// hit a URL, which doesn't apply here.
export function BulkGrantForm({
  unitId,
  personQuery,
  persons,
  roles,
  action,
}: {
  unitId: string;
  personQuery: string;
  persons: Person[];
  roles: Role[];
  action: (formData: FormData) => Promise<void>;
}) {
  const t = useTranslations("SuperAdminRoleGrantsPage");
  const router = useRouter();
  const [query, setQuery] = useState(personQuery);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [, startTransition] = useTransition();

  function runSearch(e: FormEvent) {
    e.preventDefault();
    startTransition(() => {
      router.replace(
        `/admin/role-grants?unitId=${encodeURIComponent(unitId)}&personQuery=${encodeURIComponent(query)}`,
      );
    });
  }

  function toggle(personId: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(personId)) {
        next.delete(personId);
      } else {
        next.add(personId);
      }
      return next;
    });
  }

  return (
    <div className="flex flex-col gap-4 border-t pt-4">
      <h3 className="text-sm font-medium">{t("bulkHeading")}</h3>

      <form onSubmit={runSearch} className="flex gap-2">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t("personSearchPlaceholder")}
        />
        <Button type="submit" variant="outline">
          {t("load")}
        </Button>
      </form>

      {personQuery && persons.length === 0 && (
        <p className="text-sm text-muted-foreground">{t("noPersonResults")}</p>
      )}

      {persons.length > 0 && (
        <ul className="flex max-h-64 flex-col divide-y overflow-y-auto rounded-md border">
          {persons.map((p) => (
            <li key={p.id} className="flex items-center gap-2 px-3 py-2 text-sm">
              <input
                type="checkbox"
                checked={selected.has(p.id)}
                onChange={() => toggle(p.id)}
                className="size-4 rounded border-input"
              />
              <span>{p.displayName}</span>
            </li>
          ))}
        </ul>
      )}

      <form action={action} className="flex flex-col gap-4">
        <input type="hidden" name="unitId" value={unitId} />
        {[...selected].map((id) => (
          <input key={id} type="hidden" name="personIds" value={id} />
        ))}
        <Label className="flex flex-col items-start gap-1">
          {t("roleLabel")}
          <select
            name="roleId"
            required
            className="h-8 w-full rounded-lg border border-input bg-transparent px-2.5 py-1 text-sm"
          >
            <option value="">{t("rolePlaceholder")}</option>
            {roles.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>
        </Label>
        <Button type="submit" disabled={selected.size === 0} className="self-start">
          {t("assignToSelected", { count: selected.size })}
        </Button>
      </form>
    </div>
  );
}
