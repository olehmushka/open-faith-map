// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";

import { PreviewFrame } from "@/components/preview-frame";
import { SitePage } from "@/components/site-page";
import { ContentApiError, getSiteBySlug } from "@/lib/content";

// Same reasoning as the tenant root page — content changes independently of any build, and a
// preview link's whole point is to reflect the draft as it stands right now.
export const dynamic = "force-dynamic";

// M14.7: reached on the tenant subdomain (nests under the existing /_sites/[slug] tree, so
// proxy.ts/lib/tenant-host.ts need no changes — the apex-host 404 guard already covers any path
// under _sites). SitePage is called here as a plain awaited function, not JSX, specifically so a
// Content:PreviewTokenInvalid thrown by its own data fetching can be caught here and turned into a
// clear message — never the app's generic error boundary, and never a silent fall-back to published
// content.
export default async function PreviewPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; slug: string }>;
  searchParams: Promise<{ token?: string }>;
}) {
  const { locale, slug } = await params;
  const { token } = await searchParams;
  const site = await getSiteBySlug(slug).catch(() => null);
  if (!site) notFound();

  if (!token) {
    return (
      <PreviewFrame>
        <InvalidPreviewLink />
      </PreviewFrame>
    );
  }

  try {
    const rendered = await SitePage({ site, locale, previewToken: token });
    return <PreviewFrame>{rendered}</PreviewFrame>;
  } catch (e) {
    if (e instanceof ContentApiError && e.errorName === "Content:PreviewTokenInvalid") {
      return (
        <PreviewFrame>
          <InvalidPreviewLink />
        </PreviewFrame>
      );
    }
    throw e;
  }
}

async function InvalidPreviewLink() {
  const t = await getTranslations("Preview");
  return (
    <main className="mx-auto flex max-w-md flex-col gap-2 px-6 py-24 text-center">
      <h1 className="text-lg font-semibold">{t("invalidHeading")}</h1>
      <p className="text-sm text-muted-foreground">{t("invalidBody")}</p>
    </main>
  );
}
