// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.17: per-tenant robots.txt, resolved the same way app/sitemap.ts resolves its tenant (Host
// header, direct — proxy.ts's matcher excludes dotted paths, so /robots.txt never gets
// host-rewritten). The apex host (discovery) gets a minimal allow-all with no sitemap reference:
// it isn't a congregation site, so it has no sitemap of its own here.
import type { MetadataRoute } from "next";
import { headers } from "next/headers";

import { parseApexHost, protocolForHost, resolveTenantSlug } from "@/lib/tenant-host";

export default async function robots(): Promise<MetadataRoute.Robots> {
  const headersList = await headers();
  const host = headersList.get("host")?.split(",")[0]?.trim();
  const apexHost = parseApexHost();
  const slug = host ? resolveTenantSlug(host, apexHost) : null;

  if (!slug || !host) {
    return { rules: { userAgent: "*", allow: "/" } };
  }

  const protocol = protocolForHost(host);
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      // Never worth indexing: a preview reflects unpublished draft content (already noindex/
      // no-store'd, M14.7 — reached at plain /{locale}/preview), and /api is a plain JSON
      // surface, not a page.
      disallow: ["/*/preview", "/api/"],
    },
    sitemap: `${protocol}://${host}/sitemap.xml`,
  };
}
