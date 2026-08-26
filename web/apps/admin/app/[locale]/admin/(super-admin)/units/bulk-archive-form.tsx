// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";
import { Building2 } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Link } from "@/i18n/navigation";
import type { Unit } from "@/lib/core";

// M12.5 — the units list's own multi-select + bulk-archive action, mirroring role-grants/
// bulk-grant-form.tsx's client-owns-selection-state shape but simpler: results already come from the
// page's own search form, so there's no secondary search-and-load step here.
export function BulkArchiveForm({
  units,
  action,
}: {
  units: Unit[];
  action: (formData: FormData) => Promise<void>;
}) {
  const t = useTranslations("SuperAdminUnitsPage");
  const [selected, setSelected] = useState<Set<string>>(new Set());

  function toggle(unitId: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(unitId)) {
        next.delete(unitId);
      } else {
        next.add(unitId);
      }
      return next;
    });
  }

  return (
    <div className="flex flex-col gap-4">
      <ul className="flex flex-col divide-y rounded-md border">
        {units.map((u) => (
          <li key={u.id} className="flex items-center gap-2 px-3 py-2 text-sm">
            <input
              type="checkbox"
              checked={selected.has(u.id)}
              onChange={() => toggle(u.id)}
              className="size-4 rounded border-input"
            />
            <Link href={`/admin/units/${u.id}`} className="flex flex-1 items-center gap-2 hover:underline">
              <Building2 className="size-4 text-muted-foreground" />
              <span className="flex-1">{u.name}</span>
              {u.code && <span className="text-xs text-muted-foreground">{u.code}</span>}
            </Link>
          </li>
        ))}
      </ul>

      <form action={action} className="flex flex-col gap-2 border-t pt-4">
        <h3 className="text-sm font-medium">{t("bulkArchiveHeading")}</h3>
        {[...selected].map((id) => (
          <input key={id} type="hidden" name="unitIds" value={id} />
        ))}
        <Button type="submit" variant="outline" size="sm" disabled={selected.size === 0} className="self-start">
          {t("archiveSelected", { count: selected.size })}
        </Button>
      </form>
    </div>
  );
}
