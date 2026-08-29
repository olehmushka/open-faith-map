// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { getSite, listDocuments, listNavItems, putNavItems, type NavItemInput } from "@/lib/content";
import { redirect } from "@/i18n/navigation";

import { NavItemListEditor, type NavSaveResult } from "./nav-item-list-editor";

// M14.10: manages a site's hand-built nav menu (content_site_nav_items) — independent of
// parent_document_id (M14.0's own nav-model ruling). putNavItems is a full replace, same shape as
// putBlocks before M14.6's revision refactor, so this page's one server action always saves the
// whole ordered list.
export default async function NavPage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { locale, unitId } = await params;
  const t = await getTranslations("NavPage");
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  const [navItems, documents] = await Promise.all([listNavItems(site.id), listDocuments(site.id)]);
  const pages = documents.filter((d) => d.kind === "PAGE");

  async function save(items: NavItemInput[]): Promise<NavSaveResult> {
    "use server";
    try {
      await putNavItems(site!.id, items);
      return { ok: true };
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        const errorName = String((e as { errorName: string }).errorName);
        const parameters =
          "parameters" in e ? (e as { parameters?: Record<string, unknown> }).parameters : undefined;
        switch (errorName) {
          case "Content:NavTargetInvalid": {
            // Unlike NavTargetAmbiguous/DuplicateNavItemSortOrder (which carry sortOrder directly),
            // NavTargetInvalid's own safe-arg is the offending targetDocumentId — the row it
            // belongs to is found by matching it back against the items actually submitted.
            const targetDocumentId = typeof parameters?.targetDocumentId === "string" ? parameters.targetDocumentId : undefined;
            const sortOrder = items.find((item) => item.targetDocumentId === targetDocumentId)?.sortOrder;
            return { ok: false, sortOrder, error: "errorNavTargetInvalid" };
          }
          case "Content:NavTargetAmbiguous":
            return { ok: false, sortOrder: typeof parameters?.sortOrder === "number" ? parameters.sortOrder : undefined, error: "errorNavTargetAmbiguous" };
          case "Content:DuplicateNavItemSortOrder":
            return { ok: false, sortOrder: typeof parameters?.sortOrder === "number" ? parameters.sortOrder : undefined, error: "errorDuplicateNavItemSortOrder" };
          default:
            return { ok: false, error: "errorGeneric", raw: errorName };
        }
      }
      throw e;
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-muted-foreground">{t("hint")}</p>
      <NavItemListEditor items={navItems} pages={pages} onSave={save} />
    </div>
  );
}
