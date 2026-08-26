// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { DiscoveryFacets } from "@/lib/discovery";
import type { DiscoveryFilters } from "@/lib/discovery-url-state";
import { MoreFiltersSheet } from "./more-filters-sheet";

// Radix's Select doesn't allow an item with an empty string value, so "no filter" gets its own
// sentinel that's translated back to `undefined` before it ever reaches the search call.
const ANY = "__any__";

export function FilterBar({
  initialTradition,
  initialLanguage,
  initialDayOfWeek,
  initialAccessibility,
  initialOnlineOnly,
  facets,
  onSubmit,
  onMoreFiltersSubmit,
  pending,
}: {
  initialTradition: string | undefined;
  initialLanguage: string | undefined;
  initialDayOfWeek: number | undefined;
  initialAccessibility: string[] | undefined;
  initialOnlineOnly: boolean | undefined;
  facets: DiscoveryFacets;
  onSubmit: (filters: Pick<DiscoveryFilters, "tradition" | "language">) => void;
  onMoreFiltersSubmit: (filters: Pick<DiscoveryFilters, "dayOfWeek" | "accessibility" | "onlineOnly">) => void;
  pending: boolean;
}) {
  const t = useTranslations("DiscoveryMap");
  const [tradition, setTradition] = useState(initialTradition ?? ANY);
  const [language, setLanguage] = useState(initialLanguage ?? ANY);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit({
      tradition: tradition === ANY ? undefined : tradition,
      language: language === ANY ? undefined : language,
    });
  }

  return (
    <div className="flex flex-wrap items-end gap-3 p-3">
      <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t("traditionLabel")}</span>
          <Select value={tradition} onValueChange={setTradition}>
            <SelectTrigger className="min-w-40">
              <SelectValue placeholder={t("traditionPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ANY}>{t("anyTradition")}</SelectItem>
              {facets.traditions.map((tr) => (
                <SelectItem key={tr.taxonCode} value={tr.taxonCode}>
                  {tr.taxonName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t("languageLabel")}</span>
          <Select value={language} onValueChange={setLanguage}>
            <SelectTrigger className="min-w-40">
              <SelectValue placeholder={t("languagePlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ANY}>{t("anyLanguage")}</SelectItem>
              {facets.languages.map((lang) => (
                <SelectItem key={lang} value={lang}>
                  {lang}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </label>
        <Button type="submit" disabled={pending}>
          {pending ? t("searching") : t("search")}
        </Button>
      </form>
      <MoreFiltersSheet
        initialDayOfWeek={initialDayOfWeek}
        initialAccessibility={initialAccessibility}
        initialOnlineOnly={initialOnlineOnly}
        onSubmit={onMoreFiltersSubmit}
        pending={pending}
      />
    </div>
  );
}
