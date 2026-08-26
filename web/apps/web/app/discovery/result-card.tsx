// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";
import { ACCESSIBILITY_KEYS, ACCESSIBILITY_MESSAGE_KEYS } from "@/lib/accessibility";
import { formatDistance, haversineMeters, type DistanceUnit } from "@/lib/geo";
import type { DiscoverySite } from "@/lib/discovery";

// Shared between the list pane's card and the map's marker popup (M13.4) — one field layout, one
// place to keep it in sync.
export function ResultCard({
  site,
  origin,
  unit,
}: {
  site: DiscoverySite;
  origin: { lat: number; lng: number } | null;
  unit: DistanceUnit;
}) {
  const t = useTranslations("DiscoveryMap");

  const hasCoords = typeof site.latitude === "number" && typeof site.longitude === "number";
  const distance =
    origin && hasCoords
      ? formatDistance(
          haversineMeters(origin.lat, origin.lng, site.latitude as number, site.longitude as number),
          unit,
        )
      : null;

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-sm font-medium">{site.name || t("unnamedSite")}</span>
        {distance ? <span className="shrink-0 text-xs text-muted-foreground">{distance}</span> : null}
      </div>
      {site.address ? <span className="text-xs text-muted-foreground">{site.address}</span> : null}
      <div className="flex flex-wrap gap-1 pt-0.5">
        {site.traditionTaxonName ? <Badge variant="secondary">{site.traditionTaxonName}</Badge> : null}
        {site.attributes.onlineStream ? <Badge variant="outline">{t("onlineStream")}</Badge> : null}
        {ACCESSIBILITY_KEYS.filter((key) => site.attributes.accessibility[key]).map((key) => (
          <Badge key={key} variant="outline">
            {t(ACCESSIBILITY_MESSAGE_KEYS[key])}
          </Badge>
        ))}
      </div>
      {hasCoords ? (
        <a
          href={`https://www.google.com/maps/dir/?api=1&destination=${site.latitude},${site.longitude}`}
          target="_blank"
          rel="noopener noreferrer"
          className="pt-0.5 text-xs font-medium text-primary underline-offset-2 hover:underline"
          onClick={(e) => e.stopPropagation()}
        >
          {t("getDirections")}
        </a>
      ) : null}
    </div>
  );
}
