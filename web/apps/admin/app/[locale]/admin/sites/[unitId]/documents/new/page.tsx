import { getTranslations } from "next-intl/server";

import { createDocument, getSite, listDocuments } from "@/lib/content";
import { DocumentKind } from "@/lib/openfaithmap/generated/content";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent } from "@/components/ui/card";

const NO_PARENT = "__none__";

export default async function NewDocumentPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; unitId: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { locale, unitId } = await params;
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
    const parentDocumentIdRaw = kind === DocumentKind.PAGE ? String(formData.get("parentDocumentId") ?? "") : "";
    const parentDocumentId = parentDocumentIdRaw && parentDocumentIdRaw !== NO_PARENT ? parentDocumentIdRaw : undefined;
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
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      {error && (
        <p className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {error === "Content:SlugTaken"
            ? t("errorSlugTaken")
            : error === "Content:EventMissingStart"
              ? t("errorEventMissingStart")
              : t("errorGeneric", { error })}
        </p>
      )}

      <Card>
        <CardContent className="pt-6">
          <form action={create} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("kindLabel")}
              <Select name="kind" defaultValue={DocumentKind.PAGE}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={DocumentKind.PAGE}>{t("kindPage")}</SelectItem>
                  <SelectItem value={DocumentKind.POST}>{t("kindPost")}</SelectItem>
                  <SelectItem value={DocumentKind.EVENT}>{t("kindEvent")}</SelectItem>
                </SelectContent>
              </Select>
            </Label>

            <Label className="flex flex-col items-start gap-1">
              {t("localeLabel")}
              <Input name="locale" required placeholder="eng" />
            </Label>

            <Label className="flex flex-col items-start gap-1">
              {t("slugLabel")}
              <Input name="slug" required pattern="[a-z0-9-]+" />
            </Label>

            <Label className="flex flex-col items-start gap-1">
              {t("parentPageLabel")}
              <Select name="parentDocumentId" defaultValue={NO_PARENT}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NO_PARENT}>{t("parentPageNone")}</SelectItem>
                  {existingPages.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.slug} ({p.locale})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Label>

            <fieldset className="flex flex-col gap-4 rounded-md border p-3">
              <legend className="px-1 text-sm font-medium">{t("eventFieldsLegend")}</legend>
              <Label className="flex flex-col items-start gap-1">
                {t("eventStartsAtLabel")}
                <Input type="datetime-local" name="eventStartsAt" />
              </Label>
              <Label className="flex flex-col items-start gap-1">
                {t("eventEndsAtLabel")}
                <Input type="datetime-local" name="eventEndsAt" />
              </Label>
              <Label className="flex flex-col items-start gap-1">
                {t("eventRecurrenceLabel")}
                <Input name="eventRecurrenceRrule" placeholder="FREQ=WEEKLY;BYDAY=SU" />
              </Label>
            </fieldset>

            <Label className="flex flex-col items-start gap-1">
              {t("translationGroupLabel")}
              <Input name="translationGroupId" placeholder={t("translationGroupPlaceholder")} />
            </Label>

            <Button type="submit" className="self-start">
              {t("submit")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
