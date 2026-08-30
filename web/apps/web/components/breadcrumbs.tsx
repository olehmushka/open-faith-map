// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { Document } from "@/lib/content";

// M14.10: rendered only at depth >= 2 (i.e. ancestors.length >= 1) — a top-level page shows no
// breadcrumb at all, per the milestone's own acceptance criterion. Plain <a> tags, not next-intl's
// Link: every href here is already a complete, locale-prefixed absolute path (matching exactly how
// the tenant-subdomain catch-all route itself is addressed), so re-running it through Link's own
// locale-prefixing would double it up.
//
// Documents have no title/display-name field in this schema, only slug — labels are the ancestor's
// own slug, humanized (kebab-case -> Title Case). The current (leaf) page renders as plain text,
// not a link, per standard breadcrumb convention.
//
// M14.14: uiLocale (next-intl chrome language) and contentLocale (the document's own locale) are
// now separate URL segments — see the page route this renders under.
export function Breadcrumbs({
  ancestors,
  current,
  uiLocale,
  contentLocale,
}: {
  ancestors: Document[];
  current: Document;
  uiLocale: string;
  contentLocale: string;
}) {
  if (ancestors.length === 0) return null;

  const chain = [...ancestors, current];

  return (
    <nav aria-label="Breadcrumb" className="text-sm text-muted-foreground">
      <ol className="flex flex-wrap items-center gap-1">
        {chain.map((doc, i) => {
          const href = `/${uiLocale}/${contentLocale}/${chain
            .slice(0, i + 1)
            .map((d) => d.slug)
            .join("/")}`;
          const isLast = i === chain.length - 1;
          return (
            <li key={doc.id} className="flex items-center gap-1">
              {i > 0 ? <span aria-hidden="true">/</span> : null}
              {isLast ? (
                <span aria-current="page" className="font-medium text-foreground">
                  {humanizeSlug(doc.slug)}
                </span>
              ) : (
                <a href={href} className="underline-offset-2 hover:underline">
                  {humanizeSlug(doc.slug)}
                </a>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

function humanizeSlug(slug: string): string {
  return slug
    .split("-")
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}
