import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { createDocument, getSite, listDocuments } from "@/lib/content";
import { DocumentKind } from "@/lib/openfaithmap/generated/content";
import { redirect } from "@/i18n/navigation";

export default async function NewDocumentPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; unitId: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { locale, unitId } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("NewDocumentPage");
  const { error } = await searchParams;
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  // Parent nesting is PAGE-only (content_documents_parent_pages_only) — options are existing pages.
  const existingPages = (await listDocuments(site.id)).filter((d) => d.kind === "PAGE");

  async function create(formData: FormData) {
    "use server";
    const kind = (String(formData.get("kind") ?? "PAGE") as DocumentKind) || DocumentKind.PAGE;
    const locale = String(formData.get("locale") ?? "");
    const slug = String(formData.get("slug") ?? "");
    const translationGroupId = String(formData.get("translationGroupId") ?? "") || undefined;
    // Parent nesting only applies to PAGE — never send one for POST/EVENT (DB CHECK would reject it).
    const parentDocumentId = kind === DocumentKind.PAGE ? String(formData.get("parentDocumentId") ?? "") || undefined : undefined;
    const eventStartsAt = kind === DocumentKind.EVENT ? String(formData.get("eventStartsAt") ?? "") || undefined : undefined;
    const eventEndsAt = kind === DocumentKind.EVENT ? String(formData.get("eventEndsAt") ?? "") || undefined : undefined;
    const eventRecurrenceRrule = kind === DocumentKind.EVENT ? String(formData.get("eventRecurrenceRrule") ?? "") || undefined : undefined;

    try {
      const doc = await createDocument(site!.id, {
        kind,
        locale,
        slug,
        parentDocumentId,
        translationGroupId,
        eventStartsAt: eventStartsAt ? new Date(eventStartsAt).toISOString() : undefined,
        eventEndsAt: eventEndsAt ? new Date(eventEndsAt).toISOString() : undefined,
        eventRecurrenceRrule,
      });
      redirect({ href: `/admin/sites/${unitId}/documents/${doc.id}`, locale });
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect({
          href: `/admin/sites/${unitId}/documents/new?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`,
          locale,
        });
      }
      throw e;
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      {error && (
        <p className="rounded border border-red-500 p-3 text-sm">
          {error === "Content:SlugTaken"
            ? t("errorSlugTaken")
            : error === "Content:EventMissingStart"
              ? t("errorEventMissingStart")
              : t("errorGeneric", { error })}
        </p>
      )}

      <form action={create} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("kindLabel")}</span>
          <select name="kind" defaultValue={DocumentKind.PAGE} className="rounded border px-3 py-2">
            <option value={DocumentKind.PAGE}>{t("kindPage")}</option>
            <option value={DocumentKind.POST}>{t("kindPost")}</option>
            <option value={DocumentKind.EVENT}>{t("kindEvent")}</option>
          </select>
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("localeLabel")}</span>
          <input name="locale" required placeholder="eng" className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("slugLabel")}</span>
          <input name="slug" required pattern="[a-z0-9-]+" className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("parentPageLabel")}</span>
          <select name="parentDocumentId" className="rounded border px-3 py-2">
            <option value="">{t("parentPageNone")}</option>
            {existingPages.map((p) => (
              <option key={p.id} value={p.id}>
                {p.slug} ({p.locale})
              </option>
            ))}
          </select>
        </label>

        <fieldset className="flex flex-col gap-4 rounded border p-3">
          <legend className="px-1 text-sm font-medium">{t("eventFieldsLegend")}</legend>
          <label className="flex flex-col gap-1">
            <span className="text-sm">{t("eventStartsAtLabel")}</span>
            <input type="datetime-local" name="eventStartsAt" className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm">{t("eventEndsAtLabel")}</span>
            <input type="datetime-local" name="eventEndsAt" className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm">{t("eventRecurrenceLabel")}</span>
            <input name="eventRecurrenceRrule" placeholder="FREQ=WEEKLY;BYDAY=SU" className="rounded border px-3 py-2" />
          </label>
        </fieldset>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("translationGroupLabel")}</span>
          <input
            name="translationGroupId"
            placeholder={t("translationGroupPlaceholder")}
            className="rounded border px-3 py-2"
          />
        </label>

        <button type="submit" className="rounded border px-4 py-2">
          {t("submit")}
        </button>
      </form>
    </main>
  );
}
