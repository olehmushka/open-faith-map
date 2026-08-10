// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";

import { Blocks } from "@/app/blocks";
import { getPublicBlocks, getSite, listPublicDocuments } from "@/lib/content";
import { fileReport, type FileReportInput } from "@/lib/moderation";
import { redirect } from "@/i18n/navigation";

const REPORT_REASON_CODES: FileReportInput["reasonCode"][] = [
  "SPAM",
  "INCORRECT_INFORMATION",
  "INAPPROPRIATE_CONTENT",
  "DUPLICATE",
  "OTHER",
];

// Same force-dynamic reasoning as app/page.tsx — content changes independently of any build.
export const dynamic = "force-dynamic";

// Public per-congregation page (M4, docs/modules/web-facade.md's "Discovery map/search" surface).
// Keyed by the go-oikumenea congregation unit RID (not content_sites.id) — the only key
// ContentPublicService.getSite accepts. Renders every published PAGE inline (an MVP one-pager;
// content.md's parent/child page nesting isn't surfaced as separate routes yet — an open seam,
// not a regression, since M3 shipped no public rendering at all), plus a Posts feed and an
// Upcoming events list.
export default async function CongregationPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; unitId: string }>;
  searchParams: Promise<{ reported?: string }>;
}) {
  const { locale, unitId } = await params;
  const { reported } = await searchParams;
  const t = await getTranslations("CongregationPage");
  const site = await getSite(unitId).catch(() => null);
  if (!site) notFound();

  // M5, D-AdminSurface: this app never holds a session, so the report is filed anonymously — the
  // caller identity is never asked (ModerationPublicService.fileReport, docs/modules/moderation.md).
  async function report(formData: FormData) {
    "use server";
    const reasonCode = String(formData.get("reasonCode")) as FileReportInput["reasonCode"];
    const detail = String(formData.get("detail") ?? "").trim() || undefined;
    await fileReport({ targetKind: "CONGREGATION", targetRef: unitId, reasonCode, detail });
    redirect({ href: `/congregations/${unitId}?reported=1`, locale });
  }

  const [pages, posts, events] = await Promise.all([
    listPublicDocuments(site.id, "PAGE"),
    listPublicDocuments(site.id, "POST"),
    listPublicDocuments(site.id, "EVENT"),
  ]);

  const pageBlocks = await Promise.all(pages.map((p) => getPublicBlocks(p.id)));

  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-10 px-6 py-12">
      {pages.map((page, i) => (
        <section key={page.id} className="flex flex-col gap-4">
          <Blocks blocks={pageBlocks[i]} />
        </section>
      ))}

      {events.length > 0 && (
        <section className="flex flex-col gap-4 border-t pt-8">
          <h2 className="text-xl font-semibold">{t("upcomingEvents")}</h2>
          {events.map((e) => (
            <div key={e.id} className="rounded border p-4">
              <p className="text-sm text-gray-500">
                {e.eventStartsAt ? new Date(e.eventStartsAt).toLocaleString() : t("dateTbd")}
              </p>
              <EventBlocks documentId={e.id} />
            </div>
          ))}
        </section>
      )}

      {posts.length > 0 && (
        <section className="flex flex-col gap-4 border-t pt-8">
          <h2 className="text-xl font-semibold">{t("news")}</h2>
          {posts.map((p) => (
            <article key={p.id} className="rounded border p-4">
              <PostBlocks documentId={p.id} />
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

async function EventBlocks({ documentId }: { documentId: string }) {
  const blocks = await getPublicBlocks(documentId);
  return <Blocks blocks={blocks} />;
}

async function PostBlocks({ documentId }: { documentId: string }) {
  const blocks = await getPublicBlocks(documentId);
  return <Blocks blocks={blocks} />;
}
