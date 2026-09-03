import { getTranslations } from "next-intl/server";
import { FileText, Mail, Menu } from "lucide-react";

import { ACCESSIBILITY_KEYS } from "@/lib/accessibility";
import { createSite, getSite, updateSiteChrome, updateSiteTheme } from "@/lib/content";
import { getSite as getReligionSite, isSiteNotFound, updateSiteAttributes } from "@/lib/religion";
import { Link, redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

import { AttributesForm } from "./attributes-form";
import { parseTheme } from "@/lib/theme-tokens";
import { ThemeForm } from "./theme-form";

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
  searchParams,
}: {
  params: Promise<{ locale: string; unitId: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { locale, unitId } = await params;
  const { error } = await searchParams;
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

  const theme = parseTheme(site.theme);

  // M14.12: theme fields are D-CuratedTheme's fixed vocabulary — omit rather than send "" for an
  // unset <select> (an empty string isn't a valid enum value; the write-time gate would reject it
  // as unset behaves differently from "chosen but empty"). A rejection (ThemeInvalid, e.g. a
  // stale/tampered request; ThemeContrastFailed, e.g. an accent/mode pair that fails AA) redirects
  // with the typed error name in ?error=, the same shape `create` above already uses.
  async function saveTheme(formData: FormData) {
    "use server";
    const submitted: Record<string, string> = {};
    for (const key of ["accent", "mode", "fontPairing", "spacing", "headerLayout"]) {
      const v = String(formData.get(key) ?? "").trim();
      if (v !== "") submitted[key] = v;
    }
    try {
      await updateSiteTheme(site!.id, submitted);
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

  // M14.11: logoUrl/socialLinks are content_sites' own site-level settings — same full-replace
  // plain-form shape as theme just above (a handful of fixed optional fields, not a dynamic list
  // like the nav menu, which is why this doesn't need nav-item-list-editor.tsx's client-state
  // machinery).
  async function saveChrome(formData: FormData) {
    "use server";
    const optional = (name: string) => {
      const v = String(formData.get(name) ?? "").trim();
      return v === "" ? null : v;
    };
    await updateSiteChrome(site!.id, optional("logoUrl"), {
      facebook: optional("facebook") ?? undefined,
      instagram: optional("instagram") ?? undefined,
      youtube: optional("youtube") ?? undefined,
      twitter: optional("twitter") ?? undefined,
      website: optional("website") ?? undefined,
    });
    redirect({ href: `/admin/sites/${unitId}`, locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("siteHeading", { slug: site.slug })}</h1>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href={`/admin/sites/${unitId}/documents`}>
              <FileText className="size-3.5" />
              {t("managePages")}
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link href={`/admin/sites/${unitId}/nav`}>
              <Menu className="size-3.5" />
              {t("manageNav")}
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link href={`/admin/sites/${unitId}/messages`}>
              <Mail className="size-3.5" />
              {t("manageMessages")}
            </Link>
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("themeHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <ThemeForm
            initial={theme}
            labels={{
              accent: t("accentColorLabel"),
              mode: t("modeLabel"),
              fontPairing: t("fontPairingLabel"),
              spacing: t("spacingLabel"),
              headerLayout: t("headerLayoutLabel"),
              notSet: t("notSetOption"),
              preview: t("previewHeading"),
              submit: t("saveTheme"),
            }}
            action={saveTheme}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("chromeHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={saveChrome} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("logoUrlLabel")}
              <Input name="logoUrl" type="url" defaultValue={site.logoUrl ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("facebookLabel")}
              <Input name="facebook" type="url" defaultValue={site.socialLinks.facebook ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("instagramLabel")}
              <Input name="instagram" type="url" defaultValue={site.socialLinks.instagram ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("youtubeLabel")}
              <Input name="youtube" type="url" defaultValue={site.socialLinks.youtube ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("twitterLabel")}
              <Input name="twitter" type="url" defaultValue={site.socialLinks.twitter ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("websiteLabel")}
              <Input name="website" type="url" defaultValue={site.socialLinks.website ?? ""} />
            </Label>
            <Button type="submit" className="self-start">
              {t("saveChrome")}
            </Button>
          </form>
        </CardContent>
      </Card>
      {attributesCard}
    </div>
  );
}
