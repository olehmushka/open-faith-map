import { getTranslations } from "next-intl/server";

import {
  getBlocks,
  getSite,
  listBlockTypes,
  listDocuments,
  putBlocks,
  transitionDocument,
  updateDocument,
} from "@/lib/content";
import { DocumentTransitionAction } from "@/lib/openfaithmap/generated/content";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const NO_PARENT = "__none__";

// The API has no single-document GET (content.md never lists one) — find it by filtering
// listDocuments' result, the same pattern app/my-congregation/page.tsx already uses over
// listRegistrations.
export default async function DocumentEditorPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; unitId: string; documentId: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { locale, unitId, documentId } = await params;
  const t = await getTranslations("DocumentEditorPage");
  const { error } = await searchParams;
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  const documents = await listDocuments(site.id);
  const doc = documents.find((d) => d.id === documentId);
  if (!doc) return redirect({ href: `/admin/sites/${unitId}/documents`, locale });

  const [blocks, blockTypes] = await Promise.all([getBlocks(documentId), listBlockTypes()]);
  const otherPages = documents.filter((d) => d.kind === "PAGE" && d.id !== documentId);
  const nextPosition = blocks.reduce((max, b) => Math.max(max, b.position), 0) + 10;

  async function saveDetails(formData: FormData) {
    "use server";
    const slug = String(formData.get("slug") ?? "");
    const parentDocumentId = String(formData.get("parentDocumentId") ?? "");
    try {
      await updateDocument(documentId, {
        slug,
        parentDocumentId: parentDocumentId && parentDocumentId !== NO_PARENT ? parentDocumentId : undefined,
        clearParent: !parentDocumentId || parentDocumentId === NO_PARENT,
      });
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect({
          href: `/admin/sites/${unitId}/documents/${documentId}?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`,
          locale,
        });
      }
      throw e;
    }
    redirect({ href: `/admin/sites/${unitId}/documents/${documentId}`, locale });
  }

  async function saveBlocks(formData: FormData) {
    "use server";
    const positions = formData.getAll("position").map(String);
    const blockTypeCodes = formData.getAll("blockTypeCode").map(String);
    const dataJson = formData.getAll("data").map(String);

    const inputs = positions.map((position, i) => ({
      position: Number(position),
      blockTypeCode: blockTypeCodes[i],
      data: JSON.parse(dataJson[i] || "{}"),
    }));

    try {
      await putBlocks(documentId, inputs);
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect({
          href: `/admin/sites/${unitId}/documents/${documentId}?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`,
          locale,
        });
      }
      throw e;
    }
    redirect({ href: `/admin/sites/${unitId}/documents/${documentId}`, locale });
  }

  async function transition(action: DocumentTransitionAction) {
    "use server";
    await transitionDocument(documentId, action);
    redirect({ href: `/admin/sites/${unitId}/documents/${documentId}`, locale });
  }
  async function publish() {
    "use server";
    await transition(DocumentTransitionAction.PUBLISH);
  }
  async function unlist() {
    "use server";
    await transition(DocumentTransitionAction.UNLIST);
  }
  async function revertToDraft() {
    "use server";
    await transition(DocumentTransitionAction.REVERT_TO_DRAFT);
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold">{doc.slug}</h1>
        <Badge variant="outline">{doc.state}</Badge>
      </div>

      {error && (
        <p className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {error === "Content:SlugTaken"
            ? t("errorSlugTaken")
            : error === "Content:BlockDataInvalid"
              ? t("errorBlockDataInvalid")
              : t("errorGeneric", { error })}
        </p>
      )}

      <div className="flex gap-3">
        <form action={publish}>
          <Button type="submit" variant="outline" size="sm" disabled={doc.state === "PUBLISHED"}>
            {t("publish")}
          </Button>
        </form>
        <form action={unlist}>
          <Button type="submit" variant="outline" size="sm" disabled={doc.state !== "PUBLISHED"}>
            {t("unlist")}
          </Button>
        </form>
        <form action={revertToDraft}>
          <Button type="submit" variant="outline" size="sm" disabled={doc.state !== "PUBLISHED"}>
            {t("backToDraft")}
          </Button>
        </form>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("detailsHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={saveDetails} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("slugLabel")}
              <Input name="slug" defaultValue={doc.slug} required pattern="[a-z0-9-]+" />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("parentPageLabel")}
              <Select name="parentDocumentId" defaultValue={doc.parentDocumentId ?? NO_PARENT}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NO_PARENT}>{t("parentPageNone")}</SelectItem>
                  {otherPages.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.slug} ({p.locale})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Label>
            <Button type="submit" className="self-start">
              {t("saveDetails")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("blocksHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="mb-4 text-sm text-muted-foreground">{t("blocksHint")}</p>
          <form action={saveBlocks} className="flex flex-col gap-4">
            {blocks.map((b) => (
              <div key={b.id} className="grid grid-cols-[6rem_10rem_1fr] gap-2 rounded-md border p-3">
                <Input name="position" type="number" defaultValue={b.position} className="h-8" />
                <Select name="blockTypeCode" defaultValue={b.blockTypeCode}>
                  <SelectTrigger size="sm" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {blockTypes.map((bt) => (
                      <SelectItem key={bt.code} value={bt.code}>
                        {bt.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Textarea name="data" defaultValue={JSON.stringify(b.data)} rows={3} className="font-mono text-xs" />
              </div>
            ))}

            <div className="grid grid-cols-[6rem_10rem_1fr] gap-2 rounded-md border border-dashed p-3">
              <Input name="position" type="number" defaultValue={nextPosition} className="h-8" />
              <Select name="blockTypeCode" defaultValue={blockTypes[0]?.code}>
                <SelectTrigger size="sm" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {blockTypes.map((bt) => (
                    <SelectItem key={bt.code} value={bt.code}>
                      {bt.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Textarea name="data" defaultValue="{}" rows={3} className="font-mono text-xs" />
            </div>

            <Button type="submit" className="self-start">
              {t("saveBlocks")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
