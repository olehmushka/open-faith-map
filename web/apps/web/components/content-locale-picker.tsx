// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import type { DocumentTranslation } from "@/lib/content";

// M14.14, DS-OFM-7: a page-document-scoped locale picker, rendered inside the PAGE route itself —
// never in the shared site header/footer, which wraps every route including the root posts/events
// feed (no single translatable document behind it, so it has nothing precise to offer). `translations`
// already comes pre-filtered to PUBLISHED variants only (application.Service.resolvePublishedTranslations)
// — a picker offering a draft would 404, which DS-OFM-7 explicitly calls worse than no picker at all.
//
// Plain <a> tags, not next-intl's Link: switching content locale never touches uiLocale (the site
// chrome's own language stays put), and each translation's href is already a complete
// content-locale-prefixed path (Document's own href convention) — this just prepends uiLocale.
export async function ContentLocalePicker({
  translations,
  uiLocale,
  activeContentLocale,
}: {
  translations: DocumentTranslation[];
  uiLocale: string;
  activeContentLocale: string;
}) {
  if (translations.length < 2) return null;

  const t = await getTranslations("ContentLocalePicker");

  return (
    <nav aria-label={t("label")} className="flex flex-wrap items-center gap-2 text-sm">
      <span className="text-muted-foreground">{t("label")}:</span>
      <ul className="flex flex-wrap gap-2">
        {translations.map((translation) => {
          const isActive = translation.locale === activeContentLocale;
          return (
            <li key={translation.locale}>
              {isActive ? (
                <span aria-current="true" className="font-medium text-foreground">
                  {translation.locale}
                </span>
              ) : (
                <a href={`/${uiLocale}${translation.href}`} className="underline-offset-2 hover:underline">
                  {translation.locale}
                </a>
              )}
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
