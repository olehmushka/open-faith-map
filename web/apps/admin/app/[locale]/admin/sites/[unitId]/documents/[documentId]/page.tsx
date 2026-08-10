import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
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
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

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
        parentDocumentId: parentDocumentId || undefined,
        clearParent: parentDocumentId === "",
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
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-8 px-6 py-12">
      <h1 className="text-2xl font-semibold">
        {doc.slug} — <span className="text-base font-normal">{doc.state}</span>
      </h1>

      {error && (
        <p className="rounded border border-red-500 p-3 text-sm">
          {error === "Content:SlugTaken"
            ? t("errorSlugTaken")
            : error === "Content:BlockDataInvalid"
              ? t("errorBlockDataInvalid")
              : t("errorGeneric", { error })}
        </p>
      )}

      <section className="flex gap-3">
        <form action={publish}>
          <button type="submit" className="rounded border px-3 py-1 text-sm" disabled={doc.state === "PUBLISHED"}>
            {t("publish")}
          </button>
        </form>
        <form action={unlist}>
          <button type="submit" className="rounded border px-3 py-1 text-sm" disabled={doc.state !== "PUBLISHED"}>
            {t("unlist")}
          </button>
        </form>
        <form action={revertToDraft}>
          <button type="submit" className="rounded border px-3 py-1 text-sm" disabled={doc.state !== "PUBLISHED"}>
            {t("backToDraft")}
          </button>
        </form>
      </section>

      <section>
        <h2 className="text-lg font-medium">{t("detailsHeading")}</h2>
        <form action={saveDetails} className="mt-2 flex flex-col gap-4">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">{t("slugLabel")}</span>
            <input name="slug" defaultValue={doc.slug} required pattern="[a-z0-9-]+" className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">{t("parentPageLabel")}</span>
            <select name="parentDocumentId" defaultValue={doc.parentDocumentId ?? ""} className="rounded border px-3 py-2">
              <option value="">{t("parentPageNone")}</option>
              {otherPages.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.slug} ({p.locale})
                </option>
              ))}
            </select>
          </label>
          <button type="submit" className="rounded border px-4 py-2">
            {t("saveDetails")}
          </button>
        </form>
      </section>

      <section>
        <h2 className="text-lg font-medium">{t("blocksHeading")}</h2>
        <p className="text-sm">{t("blocksHint")}</p>
        <form action={saveBlocks} className="mt-2 flex flex-col gap-4">
          {blocks.map((b) => (
            <div key={b.id} className="grid grid-cols-[6rem_10rem_1fr] gap-2 rounded border p-3">
              <input name="position" type="number" defaultValue={b.position} className="rounded border px-2 py-1 text-sm" />
              <select name="blockTypeCode" defaultValue={b.blockTypeCode} className="rounded border px-2 py-1 text-sm">
                {blockTypes.map((bt) => (
                  <option key={bt.code} value={bt.code}>
                    {bt.name}
                  </option>
                ))}
              </select>
              <textarea name="data" defaultValue={JSON.stringify(b.data)} rows={3} className="rounded border px-2 py-1 font-mono text-xs" />
            </div>
          ))}

          <div className="grid grid-cols-[6rem_10rem_1fr] gap-2 rounded border border-dashed p-3">
            <input name="position" type="number" defaultValue={nextPosition} className="rounded border px-2 py-1 text-sm" />
            <select name="blockTypeCode" defaultValue={blockTypes[0]?.code} className="rounded border px-2 py-1 text-sm">
              {blockTypes.map((bt) => (
                <option key={bt.code} value={bt.code}>
                  {bt.name}
                </option>
              ))}
            </select>
            <textarea name="data" defaultValue="{}" rows={3} className="rounded border px-2 py-1 font-mono text-xs" />
          </div>

          <button type="submit" className="rounded border px-4 py-2">
            {t("saveBlocks")}
          </button>
        </form>
      </section>
    </main>
  );
}
