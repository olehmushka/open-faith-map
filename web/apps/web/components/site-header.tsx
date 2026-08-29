// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { PublicNavItem, SiteChrome } from "@/lib/content";

// M14.11: the tenant site's header — congregation name/logo (from SiteChrome, itself composed
// server-side from content_sites' own logoUrl plus religion's live congregation name) and the nav
// menu, extracted out of layout.tsx's old bare inline <nav> so header/footer share one look.
export function SiteHeader({ chrome, navItems }: { chrome: SiteChrome; navItems: PublicNavItem[] }) {
  return (
    <header className="border-b">
      <div className="mx-auto flex max-w-3xl flex-wrap items-center gap-4 px-6 py-4">
        <div className="flex items-center gap-3 font-semibold">
          {chrome.logoUrl ? (
            // eslint-disable-next-line @next/next/no-img-element -- external, admin-supplied URL (D-ExternalMediaOnly); no first-party image optimizer to route it through.
            <img
              src={chrome.logoUrl}
              alt={chrome.congregationName}
              className="h-10 w-10 rounded object-contain"
              loading="lazy"
              referrerPolicy="no-referrer"
            />
          ) : null}
          <span>{chrome.congregationName}</span>
        </div>
        {navItems.length > 0 ? (
          <nav aria-label="Site navigation" className="ml-auto">
            <ul className="flex flex-wrap gap-4 text-sm">
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
      </div>
    </header>
  );
}
