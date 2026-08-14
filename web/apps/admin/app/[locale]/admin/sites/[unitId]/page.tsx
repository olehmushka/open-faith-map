import { getTranslations } from "next-intl/server";
import { FileText } from "lucide-react";

import { createSite, getSite, updateSiteTheme } from "@/lib/content";
import { Link, redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

// Site creation + theme editor for one congregation's unit (M3, docs/modules/content.md). getSite
// is a public-read call (ContentPublicService) — no site yet is a normal, expected state (a
// congregation admin's first visit), not an error, hence .catch(() => null) rather than surfacing
// Content:SiteNotFound.
export default async function SitePage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { locale, unitId } = await params;
  const t = await getTranslations("SitePage");
  const site = await getSite(unitId).catch(() => null);

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
    </div>
  );
}
