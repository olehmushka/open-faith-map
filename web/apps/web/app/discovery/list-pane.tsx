// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useTranslations } from "next-intl";

import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { DiscoverySite } from "@/lib/discovery";

// Minimal card: name + tradition. The full result card (distance/address/tags/badges/directions
// CTA) is M13.4's scope — this only needs enough surface for hover-sync with the map to be real.
export function ListPane({
  sites,
  activeSiteId,
  onHoverSite,
}: {
  sites: DiscoverySite[];
  activeSiteId: string | null;
  onHoverSite: (id: string | null) => void;
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
          <CardContent className="flex flex-col gap-0.5 px-3">
            <span className="text-sm font-medium">{s.name || t("unnamedSite")}</span>
            {s.traditionTaxonName ? (
              <span className="text-xs text-muted-foreground">{s.traditionTaxonName}</span>
            ) : null}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
