// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { Blocks } from "@/app/blocks";
import { Badge } from "@/components/ui/badge";
import { ACCESSIBILITY_KEYS, ACCESSIBILITY_MESSAGE_KEYS } from "@/lib/accessibility";
import { getPreviewBlocks, getPublicBlocks, listPreviewDocuments, listPublicDocuments, type Site } from "@/lib/content";
import { getSite as getDiscoverySite } from "@/lib/discovery";
import { fileReport, type FileReportInput } from "@/lib/moderation";
import { redirect } from "@/i18n/navigation";

const REPORT_REASON_CODES: FileReportInput["reasonCode"][] = [
  "SPAM",
  "INCORRECT_INFORMATION",
  "INAPPROPRIATE_CONTENT",
  "DUPLICATE",
  "OTHER",
];

// Sunday..Saturday, matching DiscoverySite.serviceDays' own 0-6 convention — same const-array
// precedent as app/discovery/more-filters-sheet.tsx's own DAY_KEYS. Exported for site-footer.tsx's
// own day-of-week labels (ServiceSchedule.dayOfWeek uses the identical 0-6 convention).
export const DAY_KEYS = ["day0", "day1", "day2", "day3", "day4", "day5", "day6"] as const;

// The real public-site renderer (M14.9, D-TenantSubdomains) — the "extractable module" the
// milestone names: the site's discovery-derived header, its Posts/Events feed, and the report
// form, keyed by an already-resolved Site rather than fetching one itself, so this component has
// no opinion about which route or host reached it. Rendered today from
// app/[locale]/%5Fsites/[slug]/page.tsx (the tenant-subdomain route the proxy rewrites into);
// Phase 2 (openfaithmap-sites, out of scope here) can move this file into its own deployment
// without a rewrite.
//
// M14.10: Pages no longer render inline here — each gets its own route
// (app/[locale]/%5Fsites/[slug]/[...pageSlug]/page.tsx via components/page-document.tsx), reached
// through the nav menu (M14.10's layout.tsx) or a direct URL. This component's own root route
// (the site's "/") is now posts/events plus the discovery header/report form only.
export async function SitePage({
  site,
  locale,
  reported,
  previewToken,
}: {
  site: Site;
  locale: string;
  reported?: string;
  // M14.7: when set, every document/blocks read below swaps to the token-gated preview source
  // (draft revisions, every document state) instead of the public one — the render tree below is
  // completely unchanged either way, per D-ContentRevisions ("a draft is content, not a special
  // code path").
  previewToken?: string;
}) {
  const t = await getTranslations("CongregationPage");
  const tm = await getTranslations("DiscoveryMap");

  const listDocuments = previewToken
    ? (kind: string) => listPreviewDocuments(site.id, previewToken, kind)
    : (kind: string) => listPublicDocuments(site.id, kind);

  // M5, D-AdminSurface: this app never holds a session, so the report is filed anonymously — the
  // caller identity is never asked (ModerationPublicService.fileReport, docs/modules/moderation.md).
  async function report(formData: FormData) {
    "use server";
    const reasonCode = String(formData.get("reasonCode")) as FileReportInput["reasonCode"];
    const detail = String(formData.get("detail") ?? "").trim() || undefined;
    await fileReport({
      targetKind: "CONGREGATION",
      targetRef: site.congregationUnitId,
      reasonCode,
      detail,
    });
    redirect({ href: `/_sites/${site.slug}?reported=1`, locale });
  }

  const [posts, events, discoverySite] = await Promise.all([
    listDocuments("POST"),
    listDocuments("EVENT"),
    getDiscoverySite(site.congregationUnitId).catch(() => null),
  ]);

  const hasCoords =
    typeof discoverySite?.latitude === "number" && typeof discoverySite?.longitude === "number";

  return (
    <main
      className="mx-auto flex max-w-3xl flex-col gap-10"
      style={{ paddingInline: "calc(1.5rem * var(--of-space-scale, 1))", paddingBlock: "calc(3rem * var(--of-space-scale, 1))" }}
    >
      {discoverySite ? (
        <section className="flex flex-col gap-2 border-b pb-8">
          {discoverySite.address ? (
            <p className="text-sm text-muted-foreground">{discoverySite.address}</p>
          ) : null}
          <div className="flex flex-wrap gap-1">
            {discoverySite.traditionTaxonName ? (
              <Badge variant="secondary">{discoverySite.traditionTaxonName}</Badge>
            ) : null}
            {discoverySite.attributes.onlineStream ? (
              <Badge variant="outline">{tm("onlineStream")}</Badge>
            ) : null}
            {ACCESSIBILITY_KEYS.filter((key) => discoverySite.attributes.accessibility[key]).map((key) => (
              <Badge key={key} variant="outline">
                {tm(ACCESSIBILITY_MESSAGE_KEYS[key])}
              </Badge>
            ))}
          </div>
          {discoverySite.serviceLanguages.length > 0 ? (
            <p className="text-sm">
              {t("serviceLanguages")}: {discoverySite.serviceLanguages.join(", ")}
            </p>
          ) : null}
          {discoverySite.serviceDays.length > 0 ? (
            <p className="text-sm">
              {tm("dayLabel")}: {discoverySite.serviceDays.map((d) => tm(DAY_KEYS[d])).join(", ")}
            </p>
          ) : null}
          {hasCoords ? (
            <a
              href={`https://www.google.com/maps/dir/?api=1&destination=${discoverySite.latitude},${discoverySite.longitude}`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm font-medium text-primary underline-offset-2 hover:underline"
            >
              {tm("getDirections")}
            </a>
          ) : null}
        </section>
      ) : null}

      {events.length > 0 && (
        <section className="flex flex-col gap-4 border-t pt-8">
          <h2 className="text-xl font-semibold">{t("upcomingEvents")}</h2>
          {events.map((e) => (
            <div key={e.id} className="rounded border p-4">
              <p className="text-sm text-gray-500">
                {e.eventStartsAt ? new Date(e.eventStartsAt).toLocaleString() : t("dateTbd")}
              </p>
              <EventBlocks documentId={e.id} siteId={site.id} previewToken={previewToken} />
            </div>
          ))}
        </section>
      )}

      {posts.length > 0 && (
        <section className="flex flex-col gap-4 border-t pt-8">
          <h2 className="text-xl font-semibold">{t("news")}</h2>
          {posts.map((p) => (
            <article key={p.id} className="rounded border p-4">
              <PostBlocks documentId={p.id} siteId={site.id} previewToken={previewToken} />
            </article>
          ))}
        </section>
      )}

      <section className="flex flex-col gap-3 border-t pt-8">
        <h2 className="text-xl font-semibold">{t("reportHeading")}</h2>
        {reported === "1" ? (
          <p className="text-sm">{t("reportThanks")}</p>
        ) : (
          <form action={report} className="flex flex-col gap-2">
            <select name="reasonCode" required className="rounded border px-2 py-1 text-sm" defaultValue="">
              <option value="" disabled>
                {t("reportReasonPlaceholder")}
              </option>
              {REPORT_REASON_CODES.map((code) => (
                <option key={code} value={code}>
                  {code}
                </option>
              ))}
            </select>
            <textarea
              name="detail"
              placeholder={t("reportDetailPlaceholder")}
              className="rounded border px-2 py-1 text-sm"
            />
            <button type="submit" className="self-start rounded border px-3 py-1 text-sm">
              {t("reportSubmit")}
            </button>
          </form>
        )}
      </section>
    </main>
  );
}

async function EventBlocks({ documentId, siteId, previewToken }: { documentId: string; siteId: string; previewToken?: string }) {
  const blocks = previewToken ? await getPreviewBlocks(documentId, previewToken) : await getPublicBlocks(documentId);
  return <Blocks blocks={blocks} siteId={siteId} />;
}

async function PostBlocks({ documentId, siteId, previewToken }: { documentId: string; siteId: string; previewToken?: string }) {
  const blocks = previewToken ? await getPreviewBlocks(documentId, previewToken) : await getPublicBlocks(documentId);
  return <Blocks blocks={blocks} siteId={siteId} />;
}
