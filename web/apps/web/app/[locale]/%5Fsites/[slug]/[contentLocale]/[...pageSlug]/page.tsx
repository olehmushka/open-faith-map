// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { ContentLocalePicker } from "@/components/content-locale-picker";
import { JsonLd } from "@/components/json-ld";
import { PageDocument } from "@/components/page-document";
import { ContentApiError, getPublicBlocks, getPublicDocumentByPath, getSiteBySlug, getSiteChrome, type DocumentWithAncestors, type Site } from "@/lib/content";
import { deriveDescription, deriveTitle, resolveOrigin } from "@/lib/seo";
import { breadcrumbListJsonLd } from "@/lib/structured-data";

// M14.17: force-dynamic stays — this catch-all's real cost was never route-level static-shell reuse
// (a per-tenant, per-path page can't share a build-time-generated shell across arbitrary Host
// headers/slugs it has no generateStaticParams for; verified empirically: removing this here made
// Next 16/Turbopack attempt static optimization anyway and throw DYNAMIC_SERVER_USAGE on
// resolveOrigin()'s headers() call for any path outside its build-time static set). The real win
// this milestone asked for — not re-querying openfaithmap-api on every anonymous view — comes from
// lib/content.ts's tagged fetch Data Cache (getPublicDocumentByPath/getPublicBlocks, 60s TTL +
// tag-based revalidation), which persists independent of this route-level setting.
export const dynamic = "force-dynamic";

// M14.10: individual Page routes on the tenant subdomain, nested under the same /_sites/[slug] tree
// M14.9 already established (no proxy.ts/lib/tenant-host.ts change needed — injectSitesSegment
// already passes arbitrarily deep paths through untouched). pageSlug mirrors
// content_documents.parent_document_id's real 3-level nesting cap: URLs are hierarchical
// (/parent-slug/child-slug/grandchild-slug), not flat, so a wrong ancestor segment 404s instead of
// silently resolving by the last segment alone (getPublicDocumentByPath enforces this server-side;
// the length check below is just a cheap reject before any API call).
//
// M14.14: `locale` (next-intl's route segment) is now the site chrome's UI language only; the
// document's own content locale is the new `contentLocale` segment, decoupled per DS-OFM-7's
// resolution — a congregation can author a page in any language, not only the 4 chrome locales.
type RouteParams = { locale: string; slug: string; contentLocale: string; pageSlug: string[] };

async function resolve(params: Promise<RouteParams>): Promise<{ params: RouteParams; resolved: DocumentWithAncestors; site: Site } | null> {
  const routeParams = await params;
  const { slug, contentLocale, pageSlug } = routeParams;
  if (pageSlug.length === 0 || pageSlug.length > 3) return null;

  const site = await getSiteBySlug(slug).catch(() => null);
  if (!site) return null;

  try {
    const resolved = await getPublicDocumentByPath(site.id, contentLocale, pageSlug);
    return { params: routeParams, resolved, site };
  } catch (e) {
    if (e instanceof ContentApiError && e.errorName === "Content:DocumentNotFound") return null;
    throw e;
  }
}

// First generateMetadata in this app (M14.14 added it for alternates.languages only; M14.17 fills
// in the rest) — title/description (explicit metaTitle/metaDescription override, else derived from
// the document's own blocks), canonical, and OpenGraph/Twitter, alongside the existing hreflang.
export async function generateMetadata({ params }: { params: Promise<RouteParams> }): Promise<Metadata> {
  const found = await resolve(params);
  if (!found) return {};
  const { params: routeParams, resolved, site } = found;
  const { document } = resolved;

  const [blocks, chrome, origin] = await Promise.all([
    getPublicBlocks(document.id),
    getSiteChrome(site.id).catch(() => null),
    resolveOrigin(),
  ]);

  const title = document.metaTitle || deriveTitle(blocks, document.slug);
  const description = document.metaDescription || deriveDescription(blocks);
  const path = `/${routeParams.locale}/${routeParams.contentLocale}/${routeParams.pageSlug.join("/")}`;
  const canonical = origin ? `${origin}${path}` : undefined;
  const ogImage = firstImageBlockUrl(blocks) ?? chrome?.logoUrl ?? undefined;

  return {
    title,
    description,
    alternates: {
      canonical,
      ...(resolved.translations.length >= 2 && origin
        ? { languages: Object.fromEntries(resolved.translations.map((t) => [t.locale, `${origin}/${routeParams.locale}${t.href}`])) }
        : {}),
    },
    openGraph: { title, description, url: canonical, images: ogImage ? [ogImage] : undefined },
    twitter: { card: "summary_large_image", title, description, images: ogImage ? [ogImage] : undefined },
  };
}

function firstImageBlockUrl(blocks: { blockTypeCode: string; position: number; data: unknown }[]): string | undefined {
  const sorted = [...blocks].sort((a, b) => a.position - b.position);
  for (const b of sorted) {
    if (b.blockTypeCode !== "image") continue;
    const url = (b.data as Record<string, unknown> | null)?.url;
    if (typeof url === "string" && url) return url;
  }
  return undefined;
}

export default async function TenantPageRoute({ params }: { params: Promise<RouteParams> }) {
  const found = await resolve(params);
  if (!found) notFound();
  const { params: routeParams, resolved, site } = found;
  const origin = await resolveOrigin();

  return (
    <main
      className="mx-auto flex max-w-3xl flex-col gap-6"
      style={{ paddingInline: "calc(1.5rem * var(--of-space-scale, 1))", paddingBlock: "calc(3rem * var(--of-space-scale, 1))" }}
    >
      {resolved.ancestors.length > 0 && origin ? (
        <JsonLd
          data={breadcrumbListJsonLd(
            [...resolved.ancestors, resolved.document],
            (slugChain) => `${origin}/${routeParams.locale}/${routeParams.contentLocale}/${slugChain.join("/")}`,
          )}
        />
      ) : null}
      <Breadcrumbs ancestors={resolved.ancestors} current={resolved.document} uiLocale={routeParams.locale} contentLocale={routeParams.contentLocale} />
      <ContentLocalePicker translations={resolved.translations} uiLocale={routeParams.locale} activeContentLocale={routeParams.contentLocale} />
      <PageDocument documentId={resolved.document.id} siteId={site.id} />
    </main>
  );
}
