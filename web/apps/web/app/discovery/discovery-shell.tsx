// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useMemo, useState, useTransition } from "react";
import { useSearchParams } from "next/navigation";

import { searchAction } from "../actions";
import type { DiscoverySite, DiscoveryFacets } from "@/lib/discovery";
import { DEFAULT_CENTER, useGeolocation } from "@/lib/geolocation";
import { filtersToSearchParams, parseFilters, type DiscoveryFilters } from "@/lib/discovery-url-state";
import { haversineMeters, resolveDistanceUnit } from "@/lib/geo";
import { useRouter } from "@/i18n/navigation";
import { FilterBar } from "./filter-bar";
import { ListPane } from "./list-pane";
import { MapPane, GEOLOCATION_ZOOM, type PendingViewport } from "./map-pane";
import { SearchThisAreaButton } from "./search-this-area-button";

const DEFAULT_ZOOM = 6;
const DEFAULT_RADIUS_M = 20_000;

// A pending viewport counts as "searched" already if it's within this margin of the last search —
// small float drift from Leaflet's own setup shouldn't pop the button right after mount/recenter.
function viewportsDiffer(a: PendingViewport, b: PendingViewport): boolean {
  const centerDistanceM = haversineMeters(a.lat, a.lng, b.lat, b.lng);
  const threshold = Math.max(500, b.radiusM * 0.15);
  return centerDistanceM > threshold || Math.abs(a.radiusM - b.radiusM) > b.radiusM * 0.15;
}

export function DiscoveryShell({
  initialSites,
  facets,
}: {
  initialSites: DiscoverySite[];
  facets: DiscoveryFacets;
}) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const filters = useMemo(() => parseFilters(searchParams), [searchParams]);

  const [sites, setSites] = useState(initialSites);
  const [activeSiteId, setActiveSiteId] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  const hasExplicitLocation = filters.lat != null && filters.lng != null;
  const geolocation = useGeolocation();

  const [lastSearchedViewport, setLastSearchedViewport] = useState<PendingViewport | null>(
    hasExplicitLocation
      ? { lat: filters.lat as number, lng: filters.lng as number, radiusM: filters.radiusM ?? DEFAULT_RADIUS_M }
      : null,
  );
  const [pendingViewport, setPendingViewport] = useState<PendingViewport | null>(null);

  const initialCenter: [number, number] = hasExplicitLocation
    ? [filters.lat as number, filters.lng as number]
    : DEFAULT_CENTER;
  const initialZoom = hasExplicitLocation ? GEOLOCATION_ZOOM : DEFAULT_ZOOM;

  function runSearch(next: DiscoveryFilters) {
    startTransition(async () => {
      const result = await searchAction({
        tradition: next.tradition,
        language: next.language,
        lat: next.lat,
        lng: next.lng,
        radiusM: next.radiusM,
        dayOfWeek: next.dayOfWeek,
        accessibility: next.accessibility?.join(","),
        onlineOnly: next.onlineOnly,
      });
      setSites(result);
      router.replace({ pathname: "/", query: Object.fromEntries(filtersToSearchParams(next)) }, { scroll: false });
    });
  }

  function handleFilterSubmit(next: Pick<DiscoveryFilters, "tradition" | "language">) {
    const location = lastSearchedViewport;
    runSearch({
      ...filters,
      ...next,
      lat: location?.lat,
      lng: location?.lng,
      radiusM: location?.radiusM,
    });
  }

  function handleMoreFiltersSubmit(next: Pick<DiscoveryFilters, "dayOfWeek" | "accessibility" | "onlineOnly">) {
    const location = lastSearchedViewport;
    runSearch({
      ...filters,
      ...next,
      lat: location?.lat,
      lng: location?.lng,
      radiusM: location?.radiusM,
    });
  }

  function handleSearchThisArea() {
    if (!pendingViewport) return;
    runSearch({ ...filters, ...pendingViewport });
    setLastSearchedViewport(pendingViewport);
  }

  const showSearchThisArea =
    pendingViewport != null &&
    (lastSearchedViewport == null || viewportsDiffer(pendingViewport, lastSearchedViewport));

  const distanceOrigin = lastSearchedViewport
    ? { lat: lastSearchedViewport.lat, lng: lastSearchedViewport.lng }
    : null;
  const distanceUnit = resolveDistanceUnit(geolocation);

  return (
    <div className="grid h-[calc(100vh-6rem)] grid-rows-[auto_1fr] gap-2">
      <FilterBar
        initialTradition={filters.tradition}
        initialLanguage={filters.language}
        initialDayOfWeek={filters.dayOfWeek}
        initialAccessibility={filters.accessibility}
        initialOnlineOnly={filters.onlineOnly}
        facets={facets}
        onSubmit={handleFilterSubmit}
        onMoreFiltersSubmit={handleMoreFiltersSubmit}
        pending={isPending}
      />
      <div className="grid min-h-0 grid-cols-[minmax(280px,360px)_1fr] gap-2 px-3 pb-3">
        <div className="min-h-0 overflow-hidden rounded border">
          <ListPane
            sites={sites}
            activeSiteId={activeSiteId}
            onHoverSite={setActiveSiteId}
            distanceOrigin={distanceOrigin}
            distanceUnit={distanceUnit}
          />
        </div>
        <div className="relative min-h-0 overflow-hidden rounded border">
          <MapPane
            sites={sites}
            center={initialCenter}
            zoom={initialZoom}
            activeSiteId={activeSiteId}
            onHoverSite={setActiveSiteId}
            onViewportChange={setPendingViewport}
            geolocation={geolocation}
            geolocationEnabled={!hasExplicitLocation}
            distanceOrigin={distanceOrigin}
            distanceUnit={distanceUnit}
          />
          {showSearchThisArea ? (
            <SearchThisAreaButton onClick={handleSearchThisArea} pending={isPending} />
          ) : null}
        </div>
      </div>
    </div>
  );
}
