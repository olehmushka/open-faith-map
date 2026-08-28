// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { notFound } from "next/navigation";

import { SitePage } from "@/components/site-page";
import { getSiteBySlug } from "@/lib/content";

// Same force-dynamic reasoning as app/page.tsx — content changes independently of any build.
export const dynamic = "force-dynamic";

// The tenant-subdomain route (M14.9, D-TenantSubdomains): proxy.ts rewrites a request whose Host
// header resolves to a congregation's slug into /{locale}/_sites/{slug} (the directory is actually
// named %5Fsites — Next.js's private-folder convention would otherwise exclude a literal `_sites`
// folder from routing entirely; %5F is the documented escape for a URL segment that starts with an
// underscore). This page is deliberately thin — the real rendering lives in components/site-page.tsx,
// "an extractable module" per the milestone, so Phase 2 (openfaithmap-sites) is a move, not a rewrite.
export default async function TenantSitePage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; slug: string }>;
  searchParams: Promise<{ reported?: string }>;
}) {
  const { locale, slug } = await params;
  const { reported } = await searchParams;
  const site = await getSiteBySlug(slug).catch(() => null);
  if (!site) notFound();

  return <SitePage site={site} locale={locale} reported={reported} />;
}
