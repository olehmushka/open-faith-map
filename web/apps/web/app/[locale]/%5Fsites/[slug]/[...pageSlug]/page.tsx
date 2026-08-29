// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { notFound } from "next/navigation";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { PageDocument } from "@/components/page-document";
import { ContentApiError, getPublicDocumentByPath, getSiteBySlug } from "@/lib/content";

// Same force-dynamic reasoning as the tenant root page — content changes independently of any build.
export const dynamic = "force-dynamic";

// M14.10: individual Page routes on the tenant subdomain, nested under the same /_sites/[slug] tree
// M14.9 already established (no proxy.ts/lib/tenant-host.ts change needed — injectSitesSegment
// already passes arbitrarily deep paths through untouched). pageSlug mirrors
// content_documents.parent_document_id's real 3-level nesting cap: URLs are hierarchical
// (/parent-slug/child-slug/grandchild-slug), not flat, so a wrong ancestor segment 404s instead of
// silently resolving by the last segment alone (getPublicDocumentByPath enforces this server-side;
// the length check below is just a cheap reject before any API call).
export default async function TenantPageRoute({
  params,
}: {
  params: Promise<{ locale: string; slug: string; pageSlug: string[] }>;
}) {
  const { locale, slug, pageSlug } = await params;
  if (pageSlug.length === 0 || pageSlug.length > 3) notFound();

  const site = await getSiteBySlug(slug).catch(() => null);
  if (!site) notFound();

  let resolved;
  try {
    resolved = await getPublicDocumentByPath(site.id, locale, pageSlug);
  } catch (e) {
    if (e instanceof ContentApiError && e.errorName === "Content:DocumentNotFound") notFound();
    throw e;
  }

  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-12">
      <Breadcrumbs ancestors={resolved.ancestors} current={resolved.document} locale={locale} />
      <PageDocument documentId={resolved.document.id} />
    </main>
  );
}
