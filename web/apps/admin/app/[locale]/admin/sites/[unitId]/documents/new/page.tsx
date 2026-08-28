import { getTranslations } from "next-intl/server";

import { createDocument, getSite, listDocuments } from "@/lib/content";
import { DocumentKind } from "@/lib/openfaithmap/generated/content";
import { redirect } from "@/i18n/navigation";
import { Card, CardContent } from "@/components/ui/card";

import { NewDocumentForm, type CreateActionState } from "./new-document-form";

const NO_PARENT = "__none__";

export default async function NewDocumentPage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { locale, unitId } = await params;
  const t = await getTranslations("NewDocumentPage");
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  // Parent nesting is PAGE-only (content_documents_parent_pages_only) — options are existing pages.
  const existingPages = (await listDocuments(site.id)).filter((d) => d.kind === "PAGE");

  // M14.8: returns state instead of redirecting with ?error=<name> on failure — see
  // new-document-form.tsx. The success path still redirects, same as before.
  async function create(_prevState: CreateActionState, formData: FormData): Promise<CreateActionState> {
    "use server";
    const kind = (String(formData.get("kind") ?? "PAGE") as DocumentKind) || DocumentKind.PAGE;
    // Renamed from the original's `locale` to avoid shadowing the outer routing `locale` (en/uk/es/
    // pt) with the document's own content locale (e.g. "eng") — the original shadow meant the
    // success-path redirect below was passing the wrong value as next-intl's routing locale.
    const docLocale = String(formData.get("locale") ?? "");
    const slug = String(formData.get("slug") ?? "");
    const translationGroupId = String(formData.get("translationGroupId") ?? "") || undefined;
    // Parent nesting only applies to PAGE — never send one for POST/EVENT (DB CHECK would reject it).
    const parentDocumentIdRaw = kind === DocumentKind.PAGE ? String(formData.get("parentDocumentId") ?? "") : "";
    const parentDocumentId = parentDocumentIdRaw && parentDocumentIdRaw !== NO_PARENT ? parentDocumentIdRaw : undefined;
    const eventStartsAt = kind === DocumentKind.EVENT ? String(formData.get("eventStartsAt") ?? "") || undefined : undefined;
    const eventEndsAt = kind === DocumentKind.EVENT ? String(formData.get("eventEndsAt") ?? "") || undefined : undefined;
    const eventRecurrenceRrule = kind === DocumentKind.EVENT ? String(formData.get("eventRecurrenceRrule") ?? "") || undefined : undefined;

    try {
      const doc = await createDocument(site!.id, {
        kind,
        locale: docLocale,
        slug,
        parentDocumentId,
        translationGroupId,
        eventStartsAt: eventStartsAt ? new Date(eventStartsAt).toISOString() : undefined,
        eventEndsAt: eventEndsAt ? new Date(eventEndsAt).toISOString() : undefined,
        eventRecurrenceRrule,
      });
      redirect({ href: `/admin/sites/${unitId}/documents/${doc.id}`, locale });
      return null; // unreachable — redirect() always throws; satisfies the function's return type.
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        const errorName = String((e as { errorName: string }).errorName);
        if (errorName === "Content:SlugTaken") return { error: "errorSlugTaken", field: "slug" };
        if (errorName === "Content:EventMissingStart") {
          return { error: "errorEventMissingStart", field: "eventStartsAt" };
        }
        return { error: "errorGeneric", raw: errorName };
      }
      throw e;
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <Card>
        <CardContent className="pt-6">
          <NewDocumentForm action={create} existingPages={existingPages} />
        </CardContent>
      </Card>
    </div>
  );
}
