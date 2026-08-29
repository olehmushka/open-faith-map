// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { ReactNode } from "react";

import { SiteFooter } from "@/components/site-footer";
import { SiteHeader } from "@/components/site-header";
import { getSiteBySlug, getSiteChrome, listPublicNavItems } from "@/lib/content";
import { parseTheme, resolveThemeDataAttr, resolveThemeStyle } from "@/lib/theme-tokens";

// M14.11: full header/footer chrome — congregation name/logo/nav in SiteHeader, address/service
// schedule/social links in SiteFooter, both fed from one getSiteChrome call (logoUrl/socialLinks
// are content_sites' own settings; congregationName/address/schedules are composed live from
// religion at read time, per M14.11's own content.md invariant). Wraps every route under
// /_sites/[slug] (the tenant root, /preview, and the [...pageSlug] catch-all) automatically via
// Next's layout nesting, so none of those three route files needs to fetch chrome data itself.
//
// A missing site renders bare {children} — deliberately not a notFound() here, so each sibling
// route's own getSiteBySlug().catch(() => null) + notFound() stays the single source of truth for
// "this site doesn't exist."
export default async function TenantSiteLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ locale: string; slug: string }>;
}) {
  const { slug } = await params;
  const site = await getSiteBySlug(slug).catch(() => null);
  const [navItems, chrome] = site
    ? await Promise.all([listPublicNavItems(site.id).catch(() => []), getSiteChrome(site.id).catch(() => null)])
    : [[], null];

  // M14.12: theme resolution never trusts site.theme's shape — a pre-M14.12 row (or any future
  // format this code doesn't recognize) degrades to "no theme customization", not a crash.
  const theme = parseTheme(site?.theme);

  return (
    <div className="min-h-dvh bg-background text-foreground" style={resolveThemeStyle(theme)} data-theme={resolveThemeDataAttr(theme)}>
      {chrome ? <SiteHeader chrome={chrome} navItems={navItems} layout={theme.headerLayout} /> : null}
      {children}
      {chrome ? <SiteFooter chrome={chrome} /> : null}
    </div>
  );
}
