import { getTranslations } from "next-intl/server";
import { FileText } from "lucide-react";

import { ACCESSIBILITY_KEYS } from "@/lib/accessibility";
import { createSite, getSite, updateSiteTheme } from "@/lib/content";
import { getSite as getReligionSite, isSiteNotFound, updateSiteAttributes } from "@/lib/religion";
import { Link, redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

import { AttributesForm } from "./attributes-form";

// Site creation + theme editor for one congregation's unit (M3, docs/modules/content.md). getSite
// is a public-read call (ContentPublicService) — no site yet is a normal, expected state (a
// congregation admin's first visit), not an error, hence .catch(() => null) rather than surfacing
// Content:SiteNotFound.
//
// M13.2 adds a second, unrelated "site" concept to this same page: religion_sites (physical/online
// presence — accessibility criteria, online-stream flag), fetched via the religion module's own
// site.manage-gated getSite(unitId). A unit with no religion_sites row yet (never approved through
// registration, or created via the super-admin unit tree) is a normal state too — the Accessibility
// card is simply omitted, not an error.
export default async function SitePage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { locale, unitId } = await params;
  const t = await getTranslations("SitePage");
  const site = await getSite(unitId).catch(() => null);
  const religionSite = await getReligionSite(unitId).catch((e) => {
    if (isSiteNotFound(e)) return null;
    throw e;
  });

  async function saveAttributes(formData: FormData) {
    "use server";
    await updateSiteAttributes(unitId, {
      accessibility: {
        stepFreeEntrance: formData.get("stepFreeEntrance") === "on",
        accessibleRestroom: formData.get("accessibleRestroom") === "on",
        hearingLoop: formData.get("hearingLoop") === "on",
        signLanguageInterpretation: formData.get("signLanguageInterpretation") === "on",
        accessibleParking: formData.get("accessibleParking") === "on",
        wheelchairSeating: formData.get("wheelchairSeating") === "on",
        brailleOrLargePrint: formData.get("brailleOrLargePrint") === "on",
      },
      onlineStream: formData.get("onlineStream") === "on",
    });
    redirect({ href: `/admin/sites/${unitId}`, locale });
  }

  const attributesCard = religionSite ? (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t("attributesHeading")}</CardTitle>
      </CardHeader>
      <CardContent>
        <AttributesForm
          initial={religionSite.attributes}
          labels={Object.fromEntries(ACCESSIBILITY_KEYS.map((key) => [key, t(`${key}Label`)])) as Record<(typeof ACCESSIBILITY_KEYS)[number], string>}
          onlineStreamLabel={t("onlineStreamLabel")}
          submitLabel={t("saveAttributes")}
          action={saveAttributes}
        />
      </CardContent>
    </Card>
  ) : null;

  if (!site) {
    async function create(formData: FormData) {
      "use server";
      const slug = String(formData.get("slug") ?? "");
      try {
        await createSite({ congregationUnitId: unitId, slug });
      } catch (e) {
        if (e && typeof e === "object" && "errorName" in e) {
          redirect({
            href: `/admin/sites/${unitId}?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`,
            locale,
          });
        }
        throw e;
      }
      redirect({ href: `/admin/sites/${unitId}`, locale });
    }

    return (
      <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
        <h1 className="text-2xl font-semibold">{t("createHeading")}</h1>
        <Card>
          <CardHeader>
            <CardDescription>{t("createIntro")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form action={create} className="flex flex-col gap-4">
              <Label className="flex flex-col items-start gap-1">
                {t("slugLabel")}
                <Input name="slug" required pattern="[a-z0-9-]+" />
              </Label>
              <Button type="submit" className="self-start">
                {t("createSubmit")}
              </Button>
            </form>
          </CardContent>
        </Card>
        {attributesCard}
      </div>
    );
  }

  const theme = (site.theme ?? {}) as { accentColor?: string; fontPairing?: string; headerLayout?: string };

  async function saveTheme(formData: FormData) {
    "use server";
    await updateSiteTheme(site!.id, {
      accentColor: String(formData.get("accentColor") ?? ""),
      fontPairing: String(formData.get("fontPairing") ?? ""),
      headerLayout: String(formData.get("headerLayout") ?? ""),
    });
    redirect({ href: `/admin/sites/${unitId}`, locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("siteHeading", { slug: site.slug })}</h1>
        <Button variant="outline" size="sm" asChild>
          <Link href={`/admin/sites/${unitId}/documents`}>
            <FileText className="size-3.5" />
            {t("managePages")}
          </Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("themeHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={saveTheme} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("accentColorLabel")}
              <Input name="accentColor" defaultValue={theme.accentColor ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("fontPairingLabel")}
              <Input name="fontPairing" defaultValue={theme.fontPairing ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("headerLayoutLabel")}
              <Input name="headerLayout" defaultValue={theme.headerLayout ?? ""} />
            </Label>
            <Button type="submit" className="self-start">
              {t("saveTheme")}
            </Button>
          </form>
        </CardContent>
      </Card>
      {attributesCard}
    </div>
  );
}
