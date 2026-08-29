// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { ReactNode } from "react";

import { getSiteBySlug, listPublicNavItems } from "@/lib/content";

// M14.10: minimal nav chrome only — full header/footer (congregation name, logo, social links,
// footer service times read live from religion_service_schedules) is M14.11's job, not built ahead
// of it here. Wraps every route under /_sites/[slug] (the tenant root, /preview, and the new
// [...pageSlug] catch-all) automatically via Next's layout nesting, so none of those three route
// files needs to fetch the nav menu itself.
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
  const navItems = site ? await listPublicNavItems(site.id).catch(() => []) : [];

  return (
    <>
      {navItems.length > 0 ? (
        <nav aria-label="Site navigation" className="border-b">
          <ul className="mx-auto flex max-w-3xl flex-wrap gap-4 px-6 py-3 text-sm">
            {navItems.map((item) => (
              <li key={`${item.label}-${item.href}`}>
                {item.external ? (
                  <a href={item.href} target="_blank" rel="noopener noreferrer" className="hover:underline">
                    {item.label}
                  </a>
                ) : (
                  <a href={item.href} className="hover:underline">
                    {item.label}
                  </a>
                )}
              </li>
            ))}
          </ul>
        </nav>
      ) : null}
      {children}
    </>
  );
}
