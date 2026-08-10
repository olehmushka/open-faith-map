import { redirect } from "next/navigation";

import { auth } from "@/auth";
import { createDocument, getSite, listDocuments } from "@/lib/content";
import { DocumentKind } from "@/lib/openfaithmap/generated/content";

export default async function NewDocumentPage({
  params,
  searchParams,
}: {
  params: Promise<{ unitId: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const session = await auth();
  if (!session) redirect("/login");

  const { unitId } = await params;
  const { error } = await searchParams;
  const site = await getSite(unitId).catch(() => null);
  if (!site) redirect(`/admin/sites/${unitId}`);

  // M3 only ever creates kind=PAGE (post/event land at M4) — parent options are existing pages only.
  const existingPages = (await listDocuments(site.id)).filter((d) => d.kind === "PAGE");

  async function create(formData: FormData) {
    "use server";
    const locale = String(formData.get("locale") ?? "");
    const slug = String(formData.get("slug") ?? "");
    const parentDocumentId = String(formData.get("parentDocumentId") ?? "") || undefined;
    const translationGroupId = String(formData.get("translationGroupId") ?? "") || undefined;

    try {
      const doc = await createDocument(site!.id, {
        kind: DocumentKind.PAGE,
        locale,
        slug,
        parentDocumentId,
        translationGroupId,
      });
      redirect(`/admin/sites/${unitId}/documents/${doc.id}`);
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect(`/admin/sites/${unitId}/documents/new?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`);
      }
      throw e;
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">New page</h1>

      {error && (
        <p className="rounded border border-red-500 p-3 text-sm">
          {error === "Content:SlugTaken"
            ? "That slug is already taken for this locale."
            : `Something went wrong: ${error}`}
        </p>
      )}

      <form action={create} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Locale</span>
          <input name="locale" required placeholder="eng" className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Slug</span>
          <input name="slug" required pattern="[a-z0-9-]+" className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Parent page (optional)</span>
          <select name="parentDocumentId" className="rounded border px-3 py-2">
            <option value="">None (top level)</option>
            {existingPages.map((p) => (
              <option key={p.id} value={p.id}>
                {p.slug} ({p.locale})
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Join translation group (optional)</span>
          <input
            name="translationGroupId"
            placeholder="Leave blank to start a new page"
            className="rounded border px-3 py-2"
          />
        </label>

        <button type="submit" className="rounded border px-4 py-2">
          Create page
        </button>
      </form>
    </main>
  );
}
