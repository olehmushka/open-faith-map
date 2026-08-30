import { getTranslations } from "next-intl/server";

import {
  buildPreviewUrl,
  createPreviewLink,
  getBlocks,
  getSite,
  listBlockTypes,
  listDocuments,
  listPatterns,
  listRevisions,
  putBlocks,
  restoreRevision,
  transitionDocument,
  updateDocument,
} from "@/lib/content";
import { DocumentTransitionAction } from "@/lib/openfaithmap/generated/content";
import { redirect } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import { BlockListEditor, type BlockSaveResult } from "./block-list-editor";
import { DocumentDetailsForm, type DetailsActionState } from "./document-details-form";

const NO_PARENT = "__none__";

// The API has no single-document GET (content.md never lists one) — find it by filtering
// listDocuments' result, the same pattern app/my-congregation/page.tsx already uses over
// listRegistrations.
export default async function DocumentEditorPage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string; documentId: string }>;
}) {
  const { locale, unitId, documentId } = await params;
  const t = await getTranslations("DocumentEditorPage");
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  const documents = await listDocuments(site.id);
  const doc = documents.find((d) => d.id === documentId);
  if (!doc) return redirect({ href: `/admin/sites/${unitId}/documents`, locale });

  // M14.14: no dedicated "list translation group siblings" endpoint exists — same "filter what
  // you already have" convention as otherPages below (documents is already one full-site fetch).
  const translations = documents.filter((d) => d.translationGroupId === doc.translationGroupId && d.id !== doc.id);

  const [blocks, blockTypes, patterns, revisions, previewToken] = await Promise.all([
    getBlocks(documentId),
    listBlockTypes(),
    listPatterns(),
    listRevisions(documentId),
    createPreviewLink(site.id),
  ]);
  const otherPages = documents.filter((d) => d.kind === "PAGE" && d.id !== documentId);
  const previewUrl = buildPreviewUrl(site, locale, previewToken);

  // M14.8: returns state instead of redirecting with ?error=<name> — see document-details-form.tsx.
  async function saveDetails(_prevState: DetailsActionState, formData: FormData): Promise<DetailsActionState> {
    "use server";
    const slug = String(formData.get("slug") ?? "");
    const parentDocumentId = String(formData.get("parentDocumentId") ?? "");
    try {
      await updateDocument(documentId, {
        slug,
        parentDocumentId: parentDocumentId && parentDocumentId !== NO_PARENT ? parentDocumentId : undefined,
        clearParent: !parentDocumentId || parentDocumentId === NO_PARENT,
      });
      return { ok: true };
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        const errorName = String((e as { errorName: string }).errorName);
        return errorName === "Content:SlugTaken"
          ? { error: "errorSlugTaken", field: "slug" }
          : { error: "errorGeneric", raw: errorName };
      }
      throw e;
    }
  }

  // M14.6: the draft-save path (both the debounced autosave and the manual "Save now" trigger in
  // BlockListEditor) never redirects — a full-page navigation every ~10s would be exactly the
  // "silently over live content" disruption the milestone's own UX research warns against. Errors
  // surface inline via the returned result instead of a query-param redirect.
  async function autosaveBlocks(
    inputs: { position: number; blockTypeCode: string; data: unknown }[],
  ): Promise<BlockSaveResult> {
    "use server";
    try {
      await putBlocks(documentId, inputs);
      return { ok: true };
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        const parameters =
          "parameters" in e ? (e as { parameters?: Record<string, unknown> }).parameters : undefined;
        return {
          ok: false,
          position: typeof parameters?.position === "number" ? parameters.position : undefined,
          field: typeof parameters?.field === "string" ? parameters.field : undefined,
        };
      }
      throw e;
    }
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

  async function restore(revisionId: string) {
    "use server";
    await restoreRevision(documentId, revisionId);
    redirect({ href: `/admin/sites/${unitId}/documents/${documentId}`, locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold">{doc.slug}</h1>
        <Badge variant="outline">{doc.state}</Badge>
      </div>

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
        {/* M14.7: opens on the tenant subdomain, never embedded here — no congregation content ever
            renders inside this admin origin (the same cross-origin guarantee D-TenantSubdomains'
            preview design relies on). */}
        <Button variant="outline" size="sm" asChild>
          <a href={previewUrl} target="_blank" rel="noopener noreferrer">
            {t("preview")}
          </a>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("detailsHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <DocumentDetailsForm action={saveDetails} doc={doc} otherPages={otherPages} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("blocksHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="mb-4 text-sm text-muted-foreground">{t("blocksHint")}</p>
          <BlockListEditor blocks={blocks} blockTypes={blockTypes} patterns={patterns} onAutosave={autosaveBlocks} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("historyHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          {revisions.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("historyEmpty")}</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {revisions.map((r) => (
                <li
                  key={r.revisionId}
                  className="flex items-center justify-between gap-3 rounded-md border p-3 text-sm"
                >
                  <span>
                    {t("revisionLabel", {
                      number: r.revisionNo,
                      date: new Date(r.createdAt).toLocaleString(locale),
                    })}
                  </span>
                  <form action={restore.bind(null, r.revisionId)}>
                    <Button type="submit" variant="outline" size="sm">
                      {t("restoreRevision")}
                    </Button>
                  </form>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("translationsHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <p className="text-sm text-muted-foreground">{t("translationsHint")}</p>
          {translations.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("translationsEmpty")}</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {translations.map((tr) => (
                <li key={tr.id} className="flex items-center justify-between gap-3 rounded-md border p-3 text-sm">
                  <a href={`/admin/sites/${unitId}/documents/${tr.id}`} className="hover:underline">
                    {t("translationItem", { locale: tr.locale, state: tr.state })}
                  </a>
                </li>
              ))}
            </ul>
          )}
          <Button variant="outline" size="sm" className="self-start" asChild>
            <a href={`/admin/sites/${unitId}/documents/new?translationGroupId=${doc.translationGroupId}&kind=${doc.kind}`}>
              {t("createTranslation")}
            </a>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
