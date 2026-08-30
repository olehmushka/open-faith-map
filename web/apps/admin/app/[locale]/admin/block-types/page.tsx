// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { createBlockType, listBlockTypesForCatalog, updateBlockType } from "@/lib/content";
import { redirect } from "@/i18n/navigation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

// M14.13, D-SitePatterns: content.catalog.manage-gated server-side (platform-moderator standing,
// checked against the shared root unit — internal/content/application/authorize.go's
// requireCatalogManage) — this page adds no local "isModerator" gate of its own, same discipline
// app/[locale]/admin/moderation/page.tsx already follows for moderation.standing. A non-moderator's
// call to any action below simply comes back Content:Forbidden, surfaced via ?error=.
//
// jsonSchema/uiSchema are entered as raw JSON once, at creation — the owner's decision (this
// session) to lock a block type's schema after creation: updateBlockType's request has no such
// field at all (see api/content.conjure.yml's UpdateBlockTypeRequest doc comment), so editing here
// is limited to name/status/sortOrder. A moderator wanting a different shape retires the old type
// and creates a new one.
export default async function BlockTypesPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { locale } = await params;
  const { error } = await searchParams;
  const t = await getTranslations("BlockTypesPage");
  const blockTypes = await listBlockTypesForCatalog();

  async function toggleStatus(formData: FormData) {
    "use server";
    const blockTypeId = String(formData.get("blockTypeId"));
    const nextStatus = String(formData.get("nextStatus")) as "ACTIVE" | "RETIRED";
    try {
      await updateBlockType(blockTypeId, { status: nextStatus });
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect({ href: `/admin/block-types?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`, locale });
      }
      throw e;
    }
    redirect({ href: "/admin/block-types", locale });
  }

  async function create(formData: FormData) {
    "use server";
    const code = String(formData.get("code") ?? "").trim();
    const name = String(formData.get("name") ?? "").trim();
    const sortOrder = Number(formData.get("sortOrder") ?? 0);
    let jsonSchema: unknown;
    let uiSchema: unknown = {};
    try {
      jsonSchema = JSON.parse(String(formData.get("jsonSchema") ?? "{}"));
      const uiSchemaRaw = String(formData.get("uiSchema") ?? "").trim();
      if (uiSchemaRaw) uiSchema = JSON.parse(uiSchemaRaw);
    } catch {
      redirect({ href: "/admin/block-types?error=InvalidJson", locale });
      return;
    }
    try {
      await createBlockType({ code, name, jsonSchema, uiSchema, sortOrder });
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect({ href: `/admin/block-types?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`, locale });
      }
      throw e;
    }
    redirect({ href: "/admin/block-types", locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-muted-foreground">{t("intro")}</p>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("catalogHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {blockTypes.length === 0 && <p className="text-sm text-muted-foreground">{t("empty")}</p>}
          {blockTypes
            .slice()
            .sort((a, b) => a.sortOrder - b.sortOrder)
            .map((bt) => (
              <div key={bt.id} className="flex items-center justify-between gap-3 rounded-md border p-3">
                <div className="flex flex-col">
                  <span className="font-medium">{bt.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {bt.code} · {t("sortOrderLabel")} {bt.sortOrder}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant={bt.status === "ACTIVE" ? "default" : "secondary"}>{bt.status}</Badge>
                  <form action={toggleStatus}>
                    <input type="hidden" name="blockTypeId" value={bt.id} />
                    <input type="hidden" name="nextStatus" value={bt.status === "ACTIVE" ? "RETIRED" : "ACTIVE"} />
                    <Button type="submit" variant="outline" size="sm">
                      {bt.status === "ACTIVE" ? t("retire") : t("reactivate")}
                    </Button>
                  </form>
                </div>
              </div>
            ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("createHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={create} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="code">{t("codeLabel")}</Label>
              <Input id="code" name="code" required pattern="[a-z][a-z0-9_]*" placeholder="feast_banner" />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="name">{t("nameLabel")}</Label>
              <Input id="name" name="name" required placeholder="Feast banner" />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="sortOrder">{t("sortOrderLabel")}</Label>
              <Input id="sortOrder" name="sortOrder" type="number" defaultValue={100} required />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="jsonSchema">{t("jsonSchemaLabel")}</Label>
              <Textarea id="jsonSchema" name="jsonSchema" rows={6} className="font-mono text-xs" required defaultValue='{"type":"object","properties":{}}' />
              <p className="text-xs text-muted-foreground">{t("jsonSchemaHint")}</p>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="uiSchema">{t("uiSchemaLabel")}</Label>
              <Textarea id="uiSchema" name="uiSchema" rows={4} className="font-mono text-xs" defaultValue="{}" />
            </div>
            <Button type="submit" className="self-start">
              {t("createSubmit")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
