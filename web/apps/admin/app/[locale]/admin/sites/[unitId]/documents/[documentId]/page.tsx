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

import { BlockListEditor } from "./block-list-editor";

const NO_PARENT = "__none__";

// The API has no single-document GET (content.md never lists one) — find it by filtering
// listDocuments' result, the same pattern app/my-congregation/page.tsx already uses over
// listRegistrations.
export default async function DocumentEditorPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; unitId: string; documentId: string }>;
  searchParams: Promise<{ error?: string; position?: string; field?: string }>;
}) {
  const { locale, unitId, documentId } = await params;
  const t = await getTranslations("DocumentEditorPage");
  const { error, position: erroredPositionParam, field: erroredField } = await searchParams;
  const erroredPosition = erroredPositionParam !== undefined ? Number(erroredPositionParam) : undefined;
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  const documents = await listDocuments(site.id);
  const doc = documents.find((d) => d.id === documentId);
  if (!doc) return redirect({ href: `/admin/sites/${unitId}/documents`, locale });

  const [blocks, blockTypes] = await Promise.all([getBlocks(documentId), listBlockTypes()]);
  const otherPages = documents.filter((d) => d.kind === "PAGE" && d.id !== documentId);

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
        const errorName = String((e as { errorName: string }).errorName);
        const parameters =
          "parameters" in e ? (e as { parameters?: Record<string, unknown> }).parameters : undefined;
        const params = new URLSearchParams({ error: errorName });
        if (typeof parameters?.position === "number") params.set("position", String(parameters.position));
        if (typeof parameters?.field === "string" && parameters.field) params.set("field", parameters.field);
        redirect({
          href: `/admin/sites/${unitId}/documents/${documentId}?${params.toString()}`,
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

      {error && !((error === "Content:BlockDataInvalid" || error === "Content:BlockUrlNotAllowed") && erroredField) && (
        <p className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {error === "Content:SlugTaken"
            ? t("errorSlugTaken")
            : error === "Content:BlockDataInvalid" || error === "Content:BlockUrlNotAllowed"
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
          <BlockListEditor
            documentId={documentId}
            blocks={blocks}
            blockTypes={blockTypes}
            erroredPosition={erroredPosition}
            erroredField={erroredField}
            action={saveBlocks}
          />
        </CardContent>
      </Card>
    </div>
  );
}
