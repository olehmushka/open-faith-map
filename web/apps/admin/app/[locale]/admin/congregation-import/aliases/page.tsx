// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";
import { ArrowLeft } from "lucide-react";

import {
  createJurisdictionAlias,
  createTaxonAlias,
  listJurisdictionAliases,
  listTaxonAliases,
} from "@/lib/congregation-import";
import { Link, redirect } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

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
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("heading")}</h1>
        <Button variant="ghost" size="sm" asChild>
          <Link href="/admin/congregation-import">
            <ArrowLeft className="size-3.5" />
            {t("backToQueue")}
          </Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("taxonAliasesHeading")}</CardTitle>
          <CardDescription>{t("taxonAliasesHint")}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form action={addTaxonAlias} className="flex flex-wrap items-end gap-3">
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("sourceCode")}
              <Input name="sourceCode" placeholder={t("sourceCodeAllPlaceholder")} className="h-8" />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("aliasText")}
              <Input name="aliasText" required className="h-8" />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("taxonId")}
              <Input name="taxonId" required className="h-8" />
            </Label>
            <Button type="submit" size="sm">
              {t("add")}
            </Button>
          </form>
          <ul className="flex flex-col gap-1 text-sm">
            {taxonAliases.length === 0 && <li className="text-muted-foreground">{t("noAliases")}</li>}
            {taxonAliases.map((a) => (
              <li key={a.id} className="flex flex-wrap items-center gap-2">
                <code className="rounded bg-muted px-1">{a.sourceCode ?? t("global")}</code>
                <span>{a.aliasText}</span>
                <span className="text-muted-foreground">→</span>
                <code className="rounded bg-muted px-1">{a.taxonId}</code>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("jurisdictionAliasesHeading")}</CardTitle>
          <CardDescription>{t("jurisdictionAliasesHint")}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form action={addJurisdictionAlias} className="flex flex-wrap items-end gap-3">
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("sourceCode")}
              <Input name="sourceCode" placeholder={t("sourceCodeAllPlaceholder")} className="h-8" />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("aliasText")}
              <Input name="aliasText" required className="h-8" />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("jurisdictionUnitId")}
              <Input name="jurisdictionUnitId" required className="h-8" />
            </Label>
            <Button type="submit" size="sm">
              {t("add")}
            </Button>
          </form>
          <ul className="flex flex-col gap-1 text-sm">
            {jurisdictionAliases.length === 0 && <li className="text-muted-foreground">{t("noAliases")}</li>}
            {jurisdictionAliases.map((a) => (
              <li key={a.id} className="flex flex-wrap items-center gap-2">
                <code className="rounded bg-muted px-1">{a.sourceCode ?? t("global")}</code>
                <span>{a.aliasText}</span>
                <span className="text-muted-foreground">→</span>
                <code className="rounded bg-muted px-1">{a.jurisdictionUnitId}</code>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
