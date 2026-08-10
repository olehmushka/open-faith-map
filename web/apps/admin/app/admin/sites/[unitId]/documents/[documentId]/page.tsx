import { redirect } from "next/navigation";

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

// The API has no single-document GET (content.md never lists one) — find it by filtering
// listDocuments' result, the same pattern app/my-congregation/page.tsx already uses over
// listRegistrations.
export default async function DocumentEditorPage({
  params,
  searchParams,
}: {
  params: Promise<{ unitId: string; documentId: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const session = await auth();
  if (!session) redirect("/login");

  const { unitId, documentId } = await params;
  const { error } = await searchParams;
  const site = await getSite(unitId).catch(() => null);
  if (!site) redirect(`/admin/sites/${unitId}`);

  const documents = await listDocuments(site.id);
  const doc = documents.find((d) => d.id === documentId);
  if (!doc) redirect(`/admin/sites/${unitId}/documents`);

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
        redirect(
          `/admin/sites/${unitId}/documents/${documentId}?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`,
        );
      }
      throw e;
    }
    redirect(`/admin/sites/${unitId}/documents/${documentId}`);
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
        redirect(
          `/admin/sites/${unitId}/documents/${documentId}?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`,
        );
      }
      throw e;
    }
    redirect(`/admin/sites/${unitId}/documents/${documentId}`);
  }

  async function transition(action: DocumentTransitionAction) {
    "use server";
    await transitionDocument(documentId, action);
    redirect(`/admin/sites/${unitId}/documents/${documentId}`);
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
            ? "That slug is already taken for this locale."
            : error === "Content:BlockDataInvalid"
              ? "One of the blocks failed validation against its block type's schema."
              : `Something went wrong: ${error}`}
        </p>
      )}

      <section className="flex gap-3">
        <form action={publish}>
          <button type="submit" className="rounded border px-3 py-1 text-sm" disabled={doc.state === "PUBLISHED"}>
            Publish
          </button>
        </form>
        <form action={unlist}>
          <button type="submit" className="rounded border px-3 py-1 text-sm" disabled={doc.state !== "PUBLISHED"}>
            Unlist
          </button>
        </form>
        <form action={revertToDraft}>
          <button type="submit" className="rounded border px-3 py-1 text-sm" disabled={doc.state !== "PUBLISHED"}>
            Back to draft
          </button>
        </form>
      </section>

      <section>
        <h2 className="text-lg font-medium">Details</h2>
        <form action={saveDetails} className="mt-2 flex flex-col gap-4">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Slug</span>
            <input name="slug" defaultValue={doc.slug} required pattern="[a-z0-9-]+" className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Parent page</span>
            <select name="parentDocumentId" defaultValue={doc.parentDocumentId ?? ""} className="rounded border px-3 py-2">
              <option value="">None (top level)</option>
              {otherPages.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.slug} ({p.locale})
                </option>
              ))}
            </select>
          </label>
          <button type="submit" className="rounded border px-4 py-2">
            Save details
          </button>
        </form>
      </section>

      <section>
        <h2 className="text-lg font-medium">Blocks</h2>
        <p className="text-sm">
          Full replace on save — reorder by editing the position numbers. Data is raw JSON, validated
          against the block type&apos;s schema.
        </p>
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
            Save blocks
          </button>
        </form>
      </section>
    </main>
  );
}
