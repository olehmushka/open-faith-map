// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { rootUnit, unitChildren } from "@/lib/core";

import { UnitTree } from "./unit-tree";

// M12.7 — the admin hierarchy tree, global and root-down (unlike units/[unitId]/page.tsx's own
// Ancestors card, which only ever walks up from wherever the admin already is). The root's first
// level is fetched here, server-side, so opening this page needs no client round trip; every level
// below that loads lazily as UnitTree expands it, via the fetchChildren server action below.
export default async function SuperAdminUnitTreePage() {
  const t = await getTranslations("SuperAdminUnitTreePage");
  const root = await rootUnit();
  const rootChildren = await unitChildren(root.id);

  async function fetchChildren(unitId: string) {
    "use server";
    return unitChildren(unitId);
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <UnitTree root={root} rootChildren={rootChildren} fetchChildren={fetchChildren} />
    </div>
  );
}
