// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.17: per-tenant sitemap.xml. proxy.ts's own matcher ("/((?!api|_next|_vercel|.*\\..*).*)")
// excludes any dotted path, so a request for /sitemap.xml never gets host-rewritten into
// /_sites/[slug] the way every other tenant route does — this file resolves the tenant itself, from
// the raw Host header, exactly like proxy.ts does.
//
// Scope call, named rather than silently dropped: each PAGE gets one representative URL, at the
// site chrome's default UI locale (routing.defaultLocale). A document's own translations are
// discoverable from that URL's own <link rel="alternate" hreflang> tags (the page route's
// generateMetadata, M14.14/M14.17) — duplicating every UI-locale x content-locale combination into
// the sitemap itself would multiply entries for little indexing benefit.
import type { MetadataRoute } from "next";
import { headers } from "next/headers";

import { routing } from "@/i18n/routing";
import { getSiteBySlug, listSitemapEntries } from "@/lib/content";
import { parseApexHost, protocolForHost, resolveTenantSlug } from "@/lib/tenant-host";

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const headersList = await headers();
  const host = headersList.get("host")?.split(",")[0]?.trim();
  if (!host) return [];

  const apexHost = parseApexHost();
  const slug = resolveTenantSlug(host, apexHost);
  if (!slug) return [];

  const site = await getSiteBySlug(slug).catch(() => null);
  if (!site) return [];

  const origin = `${protocolForHost(host)}://${host}`;
  const locale = routing.defaultLocale;

  const entries = await listSitemapEntries(site.id);

  return [
    { url: `${origin}/${locale}` },
    ...entries.map((e) => ({
      url: `${origin}/${locale}${e.href}`,
      lastModified: new Date(e.updatedAt),
    })),
  ];
}
