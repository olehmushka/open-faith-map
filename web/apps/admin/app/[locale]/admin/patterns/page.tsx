// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { createPattern, deletePattern, listPatterns, updatePattern } from "@/lib/content";
import { redirect } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

// M14.13, D-SitePatterns: content.catalog.manage-gated server-side, same discipline
// block-types/page.tsx already documents (platform-moderator standing, no local frontend gate).
//
// blocks is entered as one JSON array of {blockTypeCode,position,data} objects — the same shape
// putBlocks already accepts and this page's own read (Pattern.blocks) already returns, so an
// existing pattern's blocks can be copied out, hand-edited, and pasted back in without any
// translation step. A per-block form (like block-data-form.tsx) is deliberately not built here:
// that's real UI investment for a moderator-only, low-frequency screen managing at most a handful
// of seeded rows.
export default async function PatternsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { locale } = await params;
  const { error } = await searchParams;
  const t = await getTranslations("PatternsPage");
  const patterns = await listPatterns();

  function redirectWithError(e: unknown): never {
    if (e && typeof e === "object" && "errorName" in e) {
      redirect({ href: `/admin/patterns?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`, locale });
    }
    throw e;
  }

  async function create(formData: FormData) {
    "use server";
    const name = String(formData.get("name") ?? "").trim();
    const description = String(formData.get("description") ?? "").trim();
    const sortOrder = Number(formData.get("sortOrder") ?? 0);
    let blocks: unknown;
    try {
      blocks = JSON.parse(String(formData.get("blocks") ?? "[]"));
    } catch {
      redirect({ href: "/admin/patterns?error=InvalidJson", locale });
      return;
    }
    try {
      await createPattern({ name, description, blocks: blocks as never, sortOrder });
    } catch (e) {
      redirectWithError(e);
    }
    redirect({ href: "/admin/patterns", locale });
  }

  async function update(formData: FormData) {
    "use server";
    const patternId = String(formData.get("patternId"));
    const name = String(formData.get("name") ?? "").trim();
    const description = String(formData.get("description") ?? "").trim();
    const sortOrder = Number(formData.get("sortOrder") ?? 0);
    let blocks: unknown;
    try {
      blocks = JSON.parse(String(formData.get("blocks") ?? "[]"));
    } catch {
      redirect({ href: "/admin/patterns?error=InvalidJson", locale });
      return;
    }
    try {
      await updatePattern(patternId, { name, description, blocks: blocks as never, sortOrder });
    } catch (e) {
      redirectWithError(e);
    }
    redirect({ href: "/admin/patterns", locale });
  }

  async function remove(formData: FormData) {
    "use server";
    const patternId = String(formData.get("patternId"));
    try {
      await deletePattern(patternId);
    } catch (e) {
      redirectWithError(e);
    }
    redirect({ href: "/admin/patterns", locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-muted-foreground">{t("intro")}</p>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {patterns
        .slice()
        .sort((a, b) => a.sortOrder - b.sortOrder)
        .map((p) => (
          <Card key={p.id}>
            <CardHeader>
              <CardTitle className="text-base">{p.name}</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <form action={update} className="flex flex-col gap-3">
                <input type="hidden" name="patternId" value={p.id} />
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor={`name-${p.id}`}>{t("nameLabel")}</Label>
                  <Input id={`name-${p.id}`} name="name" defaultValue={p.name} required />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor={`description-${p.id}`}>{t("descriptionLabel")}</Label>
                  <Input id={`description-${p.id}`} name="description" defaultValue={p.description} />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor={`sortOrder-${p.id}`}>{t("sortOrderLabel")}</Label>
                  <Input id={`sortOrder-${p.id}`} name="sortOrder" type="number" defaultValue={p.sortOrder} required />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor={`blocks-${p.id}`}>{t("blocksLabel")}</Label>
                  <Textarea
                    id={`blocks-${p.id}`}
                    name="blocks"
                    rows={8}
                    className="font-mono text-xs"
                    defaultValue={JSON.stringify(p.blocks, null, 2)}
                    required
                  />
                </div>
                <div className="flex gap-2">
                  <Button type="submit" size="sm">
                    {t("saveSubmit")}
                  </Button>
                </div>
              </form>
              <form action={remove}>
                <input type="hidden" name="patternId" value={p.id} />
                <Button type="submit" variant="destructive" size="sm">
                  {t("delete")}
                </Button>
              </form>
            </CardContent>
          </Card>
        ))}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("createHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={create} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="name">{t("nameLabel")}</Label>
              <Input id="name" name="name" required placeholder="Getting here" />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="description">{t("descriptionLabel")}</Label>
              <Input id="description" name="description" placeholder={t("descriptionPlaceholder")} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="sortOrder">{t("sortOrderLabel")}</Label>
              <Input id="sortOrder" name="sortOrder" type="number" defaultValue={100} required />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="blocks">{t("blocksLabel")}</Label>
              <Textarea
                id="blocks"
                name="blocks"
                rows={8}
                className="font-mono text-xs"
                required
                defaultValue={JSON.stringify(
                  [{ blockTypeCode: "heading", position: 0, data: { level: 2, text: [{ type: "text", text: "New pattern" }] } }],
                  null,
                  2,
                )}
              />
              <p className="text-xs text-muted-foreground">{t("blocksHint")}</p>
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
