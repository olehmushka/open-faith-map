// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { Metadata } from "next";
import { headers } from "next/headers";
import { notFound } from "next/navigation";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { ContentLocalePicker } from "@/components/content-locale-picker";
import { PageDocument } from "@/components/page-document";
import { ContentApiError, getPublicDocumentByPath, getSiteBySlug, type DocumentWithAncestors, type Site } from "@/lib/content";

// Same force-dynamic reasoning as the tenant root page — content changes independently of any build.
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

// First generateMetadata in this app — emits alternates.languages (hreflang) for every PUBLISHED
// sibling in the document's translation group, per DS-OFM-7's acceptance criterion. uiLocale is
// preserved across every alternate: switching content language never changes the site chrome's own
// language.
export async function generateMetadata({ params }: { params: Promise<RouteParams> }): Promise<Metadata> {
  const found = await resolve(params);
  if (!found) return {};
  const { params: routeParams, resolved } = found;

  const headersList = await headers();
  // proxy.ts's own next-intl-response-header-copy (`intlResponse.headers.forEach(...).append(...)`)
  // duplicates "host" onto the rewritten request in some cases — take the first value defensively
  // rather than emitting a broken two-host hreflang URL.
  const host = headersList.get("host")?.split(",")[0]?.trim();
  if (!host || resolved.translations.length < 2) return {};
  const protocol = host.startsWith("localhost") || host.startsWith("127.") ? "http" : "https";
  const origin = `${protocol}://${host}`;

  return {
    alternates: {
      languages: Object.fromEntries(resolved.translations.map((t) => [t.locale, `${origin}/${routeParams.locale}${t.href}`])),
    },
  };
}

export default async function TenantPageRoute({ params }: { params: Promise<RouteParams> }) {
  const found = await resolve(params);
  if (!found) notFound();
  const { params: routeParams, resolved, site } = found;

  return (
    <main
      className="mx-auto flex max-w-3xl flex-col gap-6"
      style={{ paddingInline: "calc(1.5rem * var(--of-space-scale, 1))", paddingBlock: "calc(3rem * var(--of-space-scale, 1))" }}
    >
      <Breadcrumbs ancestors={resolved.ancestors} current={resolved.document} uiLocale={routeParams.locale} contentLocale={routeParams.contentLocale} />
      <ContentLocalePicker translations={resolved.translations} uiLocale={routeParams.locale} activeContentLocale={routeParams.contentLocale} />
      <PageDocument documentId={resolved.document.id} siteId={site.id} />
    </main>
  );
}
