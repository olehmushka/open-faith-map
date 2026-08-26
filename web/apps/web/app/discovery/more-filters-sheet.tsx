// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { ACCESSIBILITY_KEYS, ACCESSIBILITY_MESSAGE_KEYS, type AccessibilityKey } from "@/lib/accessibility";
import type { DiscoveryFilters } from "@/lib/discovery-url-state";

// Radix's Select doesn't allow an item with an empty string value — matches filter-bar.tsx's own
// sentinel convention for "no filter".
const ANY = "__any__";
const DAY_KEYS = ["day0", "day1", "day2", "day3", "day4", "day5", "day6"] as const;

type MoreFilters = Pick<DiscoveryFilters, "dayOfWeek" | "accessibility" | "onlineOnly">;

export function MoreFiltersSheet({
  initialDayOfWeek,
  initialAccessibility,
  initialOnlineOnly,
  onSubmit,
  pending,
}: {
  initialDayOfWeek: number | undefined;
  initialAccessibility: string[] | undefined;
  initialOnlineOnly: boolean | undefined;
  onSubmit: (filters: MoreFilters) => void;
  pending: boolean;
}) {
  const t = useTranslations("DiscoveryMap");
  const [open, setOpen] = useState(false);
  const [dayOfWeek, setDayOfWeek] = useState(initialDayOfWeek != null ? String(initialDayOfWeek) : ANY);
  const [accessibility, setAccessibility] = useState<Set<AccessibilityKey>>(
    new Set(initialAccessibility as AccessibilityKey[] | undefined),
  );
  const [onlineOnly, setOnlineOnly] = useState(initialOnlineOnly ?? false);

  function toggleAccessibility(key: AccessibilityKey, checked: boolean) {
    setAccessibility((prev) => {
      const next = new Set(prev);
      if (checked) next.add(key);
      else next.delete(key);
      return next;
    });
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit({
      dayOfWeek: dayOfWeek === ANY ? undefined : Number(dayOfWeek),
      accessibility: accessibility.size ? Array.from(accessibility) : undefined,
      onlineOnly: onlineOnly || undefined,
    });
    setOpen(false);
  }

  function handleClear() {
    setDayOfWeek(ANY);
    setAccessibility(new Set());
    setOnlineOnly(false);
    onSubmit({ dayOfWeek: undefined, accessibility: undefined, onlineOnly: undefined });
    setOpen(false);
  }

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button type="button" variant="outline">
          {t("moreFilters")}
        </Button>
      </SheetTrigger>
      <SheetContent side="right">
        <SheetHeader>
          <SheetTitle>{t("moreFilters")}</SheetTitle>
          <SheetDescription>{t("moreFiltersDescription")}</SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 overflow-y-auto px-4">
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">{t("dayLabel")}</span>
            <Select value={dayOfWeek} onValueChange={setDayOfWeek}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ANY}>{t("anyDay")}</SelectItem>
                {DAY_KEYS.map((key, i) => (
                  <SelectItem key={key} value={String(i)}>
                    {t(key)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>

          <Separator />

          <div className="flex flex-col gap-2">
            <span className="text-sm font-medium">{t("accessibilityLabel")}</span>
            {ACCESSIBILITY_KEYS.map((key) => (
              <div key={key} className="flex items-center gap-2">
                <Checkbox
                  id={`more-filters-${key}`}
                  checked={accessibility.has(key)}
                  onCheckedChange={(checked) => toggleAccessibility(key, checked === true)}
                />
                <label htmlFor={`more-filters-${key}`} className="text-sm">
                  {t(ACCESSIBILITY_MESSAGE_KEYS[key])}
                </label>
              </div>
            ))}
          </div>

          <Separator />

          <div className="flex items-center gap-2">
            <Switch id="more-filters-online-only" checked={onlineOnly} onCheckedChange={setOnlineOnly} />
            <label htmlFor="more-filters-online-only" className="text-sm">
              {t("onlineOnlyLabel")}
            </label>
          </div>

          <SheetFooter className="mt-0 flex-row px-0">
            <Button type="button" variant="outline" onClick={handleClear} disabled={pending}>
              {t("clearFilters")}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? t("searching") : t("applyFilters")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
