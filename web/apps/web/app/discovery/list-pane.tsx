// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useTranslations } from "next-intl";

import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { DistanceUnit } from "@/lib/geo";
import type { DiscoverySite } from "@/lib/discovery";
import { ResultCard } from "./result-card";

export function ListPane({
  sites,
  activeSiteId,
  onHoverSite,
  distanceOrigin,
  distanceUnit,
}: {
  sites: DiscoverySite[];
  activeSiteId: string | null;
  onHoverSite: (id: string | null) => void;
  distanceOrigin: { lat: number; lng: number } | null;
  distanceUnit: DistanceUnit;
}) {
  const t = useTranslations("DiscoveryMap");

  return (
    <div className="flex h-full flex-col gap-2 overflow-y-auto p-2">
      <p className="px-2 text-sm text-muted-foreground">{t("resultsCount", { count: sites.length })}</p>
      {sites.map((s) => (
        <Card
          key={s.id}
          onMouseEnter={() => onHoverSite(s.id)}
          onMouseLeave={() => onHoverSite(null)}
          onFocus={() => onHoverSite(s.id)}
          onBlur={() => onHoverSite(null)}
          tabIndex={0}
          className={cn(
            "cursor-pointer py-3 transition-colors",
            s.id === activeSiteId && "border-primary bg-accent",
          )}
        >
          <CardContent className="px-3">
            <ResultCard site={s} origin={distanceOrigin} unit={distanceUnit} />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
