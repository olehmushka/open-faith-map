// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { PublicNavItem, SiteChrome } from "@/lib/content";
import type { ThemeHeaderLayout } from "@/lib/theme-tokens";

// Base padding this header used before M14.12's spacing-scale token existed — kept as the literal
// multiplier base so "comfortable" (scale 1) renders pixel-identical to the pre-M14.12 layout.
const PADDING_INLINE_REM = 1.5;
const PADDING_BLOCK_REM = 1;

// M14.11: the tenant site's header — congregation name/logo (from SiteChrome, itself composed
// server-side from content_sites' own logoUrl plus religion's live congregation name) and the nav
// menu, extracted out of layout.tsx's old bare inline <nav> so header/footer share one look.
//
// M14.12: `layout` (D-CuratedTheme's headerLayout token) branches the three curated arrangements —
// "logo-left" is this component's original, only-ever layout, now the default for an unset/legacy
// theme. The padding itself scales with --of-space-scale (M14.12's spacing-scale token), falling
// back to 1 (today's fixed padding) when unset.
export function SiteHeader({
  chrome,
  navItems,
  uiLocale,
  layout = "logo-left",
}: {
  chrome: SiteChrome;
  navItems: PublicNavItem[];
  // M14.14: internal nav item hrefs are already content-locale-prefixed (buildPublicHref, backend)
  // but never carried the site chrome's own UI language — this app's route now needs both segments.
  uiLocale: string;
  layout?: ThemeHeaderLayout | string;
}) {
  const padding = {
    paddingInline: `calc(${PADDING_INLINE_REM}rem * var(--of-space-scale, 1))`,
    paddingBlock: `calc(${PADDING_BLOCK_REM}rem * var(--of-space-scale, 1))`,
  };

  const logo = (
    <div className="flex items-center gap-3 font-heading font-semibold">
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
  );

  const nav =
    navItems.length > 0 ? (
      <nav aria-label="Site navigation">
        <ul className="flex flex-wrap gap-4 text-sm">
          {navItems.map((item) => (
            <li key={`${item.label}-${item.href}`}>
              {item.external ? (
                <a href={item.href} target="_blank" rel="noopener noreferrer" className="hover:underline">
                  {item.label}
                </a>
              ) : (
                <a href={`/${uiLocale}${item.href}`} className="hover:underline">
                  {item.label}
                </a>
              )}
            </li>
          ))}
        </ul>
      </nav>
    ) : null;

  if (layout === "centered") {
    return (
      <header className="border-b">
        <div className="mx-auto flex max-w-3xl flex-col items-center gap-3 text-center" style={padding}>
          {logo}
          {nav}
        </div>
      </header>
    );
  }

  if (layout === "stacked") {
    return (
      <header className="border-b">
        <div className="mx-auto flex max-w-3xl flex-col gap-3" style={padding}>
          {logo}
          {nav}
        </div>
      </header>
    );
  }

  // "logo-left" (default): logo/name left, nav right, wrapping on narrow viewports — today's only
  // layout, unchanged in markup shape.
  return (
    <header className="border-b">
      <div className="mx-auto flex max-w-3xl flex-wrap items-center gap-4" style={padding}>
        {logo}
        {nav ? <div className="ml-auto">{nav}</div> : null}
      </div>
    </header>
  );
}
