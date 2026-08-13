// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import {
  createJurisdictionAlias,
  createTaxonAlias,
  listJurisdictionAliases,
  listTaxonAliases,
} from "@/lib/congregation-import";
import { redirect } from "@/i18n/navigation";

// A secondary page under congregation-import (mirrors /admin/registrations/reparent's own
// "secondary page under the same feature" shape) — previously SQL-only
// (docs/modules/congregationimport.md's Known limitations), operators must not need direct
// Postgres access to add an alias in production. Deliberately minimal: create + list only, no
// delete/deactivate (named follow-up, not silently dropped — an alias is normalized/validated
// before insert, so a wrong one is rare, not a daily operator task).
export default async function CongregationImportAliasesPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("CongregationImportAliasesPage");
  const [{ aliases: taxonAliases }, { aliases: jurisdictionAliases }] = await Promise.all([
    listTaxonAliases(),
    listJurisdictionAliases(),
  ]);

  async function addTaxonAlias(formData: FormData) {
    "use server";
    const sourceCode = String(formData.get("sourceCode") ?? "").trim() || undefined;
    const aliasText = String(formData.get("aliasText") ?? "").trim();
    const taxonId = String(formData.get("taxonId") ?? "").trim();
    if (!aliasText || !taxonId) return;
    await createTaxonAlias({ sourceCode, aliasText, taxonId });
    redirect({ href: "/admin/congregation-import/aliases", locale });
  }

  async function addJurisdictionAlias(formData: FormData) {
    "use server";
    const sourceCode = String(formData.get("sourceCode") ?? "").trim() || undefined;
    const aliasText = String(formData.get("aliasText") ?? "").trim();
    const jurisdictionUnitId = String(formData.get("jurisdictionUnitId") ?? "").trim();
    if (!aliasText || !jurisdictionUnitId) return;
    await createJurisdictionAlias({ sourceCode, aliasText, jurisdictionUnitId });
    redirect({ href: "/admin/congregation-import/aliases", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <a href={`/${locale}/admin/congregation-import`} className="text-sm underline">
        {t("backToQueue")}
      </a>

      <section className="rounded border p-4">
        <h2 className="font-medium">{t("taxonAliasesHeading")}</h2>
        <p className="text-sm text-gray-500">{t("taxonAliasesHint")}</p>
        <form action={addTaxonAlias} className="mt-3 flex flex-wrap items-end gap-2">
          <label className="flex flex-col text-xs">
            {t("sourceCode")}
            <input name="sourceCode" placeholder={t("sourceCodeAllPlaceholder")} className="rounded border px-2 py-1 text-sm" />
          </label>
          <label className="flex flex-col text-xs">
            {t("aliasText")}
            <input name="aliasText" required className="rounded border px-2 py-1 text-sm" />
          </label>
          <label className="flex flex-col text-xs">
            {t("taxonId")}
            <input name="taxonId" required className="rounded border px-2 py-1 text-sm" />
          </label>
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("add")}
          </button>
        </form>
        <ul className="mt-3 flex flex-col gap-1 text-sm">
          {taxonAliases.length === 0 && <li>{t("noAliases")}</li>}
          {taxonAliases.map((a) => (
            <li key={a.id} className="flex flex-wrap gap-2">
              <code className="rounded bg-gray-100 px-1">{a.sourceCode ?? t("global")}</code>
              <span>{a.aliasText}</span>
              <span>→</span>
              <code className="rounded bg-gray-100 px-1">{a.taxonId}</code>
            </li>
          ))}
        </ul>
      </section>

      <section className="rounded border p-4">
        <h2 className="font-medium">{t("jurisdictionAliasesHeading")}</h2>
        <p className="text-sm text-gray-500">{t("jurisdictionAliasesHint")}</p>
        <form action={addJurisdictionAlias} className="mt-3 flex flex-wrap items-end gap-2">
          <label className="flex flex-col text-xs">
            {t("sourceCode")}
            <input name="sourceCode" placeholder={t("sourceCodeAllPlaceholder")} className="rounded border px-2 py-1 text-sm" />
          </label>
          <label className="flex flex-col text-xs">
            {t("aliasText")}
            <input name="aliasText" required className="rounded border px-2 py-1 text-sm" />
          </label>
          <label className="flex flex-col text-xs">
            {t("jurisdictionUnitId")}
            <input name="jurisdictionUnitId" required className="rounded border px-2 py-1 text-sm" />
          </label>
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("add")}
          </button>
        </form>
        <ul className="mt-3 flex flex-col gap-1 text-sm">
          {jurisdictionAliases.length === 0 && <li>{t("noAliases")}</li>}
          {jurisdictionAliases.map((a) => (
            <li key={a.id} className="flex flex-wrap gap-2">
              <code className="rounded bg-gray-100 px-1">{a.sourceCode ?? t("global")}</code>
              <span>{a.aliasText}</span>
              <span>→</span>
              <code className="rounded bg-gray-100 px-1">{a.jurisdictionUnitId}</code>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
