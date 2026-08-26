// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import "leaflet/dist/leaflet.css";
import "react-leaflet-cluster/dist/assets/MarkerCluster.css";
import "react-leaflet-cluster/dist/assets/MarkerCluster.Default.css";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { MapContainer, Marker, Popup, TileLayer, useMap, useMapEvents } from "react-leaflet";
import MarkerClusterGroup from "react-leaflet-cluster";
import L from "leaflet";

import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import type { DiscoverySite } from "@/lib/discovery";
import type { GeolocationResult } from "@/lib/geolocation";
import type { DistanceUnit } from "@/lib/geo";
import { useIsMobile } from "@/lib/use-is-mobile";
import { Link } from "@/i18n/navigation";
import { ResultCard } from "./result-card";

// A plain SVG pin div-icon, deliberately not Leaflet's default marker image — the default's icon
// URLs resolve relative to leaflet's own asset layout and break under Next's bundler without extra
// webpack config. A div-icon has no asset-resolution problem at all.
function makePinIcon(active: boolean) {
  const size = active ? 22 : 16;
  const color = active ? "#ea580c" : "#2563eb";
  return L.divIcon({
    className: "",
    html: `<div style="width:${size}px;height:${size}px;border-radius:50% 50% 50% 0;background:${color};border:2px solid white;transform:rotate(-45deg);box-shadow:0 1px 3px rgba(0,0,0,0.4)"></div>`,
    iconSize: [size, size],
    iconAnchor: [size / 2, size],
  });
}
const pinIcon = makePinIcon(false);
const pinIconActive = makePinIcon(true);

export const GEOLOCATION_ZOOM = 11;

export interface PendingViewport {
  lat: number;
  lng: number;
  radiusM: number;
}

/** Fires once, only after geolocation actually resolves to a real position — never for the Kyiv fallback, which is already the map's initial center. */
function GeolocationRecenter({
  geolocation,
  enabled,
}: {
  geolocation: GeolocationResult;
  enabled: boolean;
}) {
  const map = useMap();
  const recenteredRef = useRef(false);

  useEffect(() => {
    if (!enabled || recenteredRef.current || geolocation.status !== "granted") return;
    recenteredRef.current = true;
    map.setView([geolocation.lat, geolocation.lng], GEOLOCATION_ZOOM);
  }, [enabled, geolocation, map]);

  return null;
}

function ViewportWatcher({ onViewportChange }: { onViewportChange: (viewport: PendingViewport) => void }) {
  useMapEvents({
    moveend(e) {
      const map = e.target;
      const center = map.getCenter();
      const radiusM = center.distanceTo(map.getBounds().getNorthEast());
      onViewportChange({ lat: center.lat, lng: center.lng, radiusM });
    },
  });
  return null;
}

/**
 * On mobile, MapPane's container is toggled between `display:none` and visible (discovery-shell.tsx's
 * List↔Map switch keeps the map mounted rather than remounting it). Leaflet doesn't recompute a
 * map's tile grid on its own when its container comes back from `display:none` — it needs an
 * explicit `invalidateSize()` or it renders blank/mis-tiled until the next manual pan/zoom.
 */
function InvalidateSizeOnShow({ active }: { active: boolean }) {
  const map = useMap();

  useEffect(() => {
    if (!active) return;
    const id = requestAnimationFrame(() => map.invalidateSize());
    return () => cancelAnimationFrame(id);
  }, [active, map]);

  return null;
}

export function MapPane({
  sites,
  center,
  zoom,
  activeSiteId,
  onHoverSite,
  onViewportChange,
  geolocation,
  geolocationEnabled,
  distanceOrigin,
  distanceUnit,
  active = true,
}: {
  sites: DiscoverySite[];
  center: [number, number];
  zoom: number;
  activeSiteId: string | null;
  onHoverSite: (id: string | null) => void;
  onViewportChange: (viewport: PendingViewport) => void;
  geolocation: GeolocationResult;
  geolocationEnabled: boolean;
  distanceOrigin: { lat: number; lng: number } | null;
  distanceUnit: DistanceUnit;
  /** Whether this pane is the one currently visible on a narrow (< md) viewport. Always treated as
   * visible at `md` and up, where discovery-shell.tsx renders both panes side by side. */
  active?: boolean;
}) {
  const t = useTranslations("DiscoveryMap");
  const isMobile = useIsMobile();
  const [tappedSiteId, setTappedSiteId] = useState<string | null>(null);
  const tappedSite = sites.find((s) => s.id === tappedSiteId) ?? null;

  function handleMarkerClick(site: DiscoverySite) {
    if (!isMobile) return;
    onHoverSite(site.id);
    setTappedSiteId(site.id);
  }

  function closeTappedSheet() {
    setTappedSiteId(null);
    onHoverSite(null);
  }

  return (
    <>
      <MapContainer center={center} zoom={zoom} style={{ height: "100%", width: "100%" }}>
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        <ViewportWatcher onViewportChange={onViewportChange} />
        <GeolocationRecenter geolocation={geolocation} enabled={geolocationEnabled} />
        <InvalidateSizeOnShow active={active} />
        <MarkerClusterGroup chunkedLoading>
          {sites
            .filter((s) => s.latitude != null && s.longitude != null)
            .map((s) => (
              <Marker
                key={s.id}
                position={[s.latitude as number, s.longitude as number]}
                icon={s.id === activeSiteId ? pinIconActive : pinIcon}
                eventHandlers={{
                  mouseover: () => onHoverSite(s.id),
                  mouseout: () => onHoverSite(null),
                  click: () => handleMarkerClick(s),
                }}
              >
                {/* On mobile, tapping a pin opens the bottom sheet below instead — no Leaflet popup
                    bound at all, so there's nothing for Leaflet's own click-to-open to trigger. */}
                {!isMobile && (
                  <Popup minWidth={224}>
                    <ResultCard site={s} origin={distanceOrigin} unit={distanceUnit} />
                    {/* contentSiteId is content's internal uuid, not what getSite accepts (see
                        app/congregations/[unitId]/page.tsx's header comment) — its presence is
                        still the right "has this congregation published a site at all" signal. */}
                    <div className="pt-1">
                      {s.contentSiteId ? (
                        <Link href={`/congregations/${s.congregationUnitRid}`}>{t("viewCongregationPage")}</Link>
                      ) : (
                        <span className="text-xs text-muted-foreground">{t("noPublishedPage")}</span>
                      )}
                    </div>
                  </Popup>
                )}
              </Marker>
            ))}
        </MarkerClusterGroup>
      </MapContainer>
      <Sheet open={tappedSiteId != null} onOpenChange={(open) => !open && closeTappedSheet()}>
        <SheetContent side="bottom">
          <SheetHeader className="sr-only">
            <SheetTitle>{tappedSite?.name || t("unnamedSite")}</SheetTitle>
          </SheetHeader>
          {tappedSite ? (
            <div className="px-4 pb-4">
              <ResultCard site={tappedSite} origin={distanceOrigin} unit={distanceUnit} />
              <div className="pt-1">
                {tappedSite.contentSiteId ? (
                  <Link href={`/congregations/${tappedSite.congregationUnitRid}`}>
                    {t("viewCongregationPage")}
                  </Link>
                ) : (
                  <span className="text-xs text-muted-foreground">{t("noPublishedPage")}</span>
                )}
              </div>
            </div>
          ) : null}
        </SheetContent>
      </Sheet>
    </>
  );
}
