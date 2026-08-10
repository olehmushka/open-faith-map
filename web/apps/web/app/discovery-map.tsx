// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import "leaflet/dist/leaflet.css";

import Link from "next/link";
import { useState, useTransition } from "react";
import { MapContainer, Marker, Popup, TileLayer } from "react-leaflet";
import L from "leaflet";

import { searchAction } from "./actions";
import type { DiscoverySite } from "@/lib/discovery";

// A plain SVG pin div-icon, deliberately not Leaflet's default marker image — the default's icon
// URLs resolve relative to leaflet's own asset layout and break under Next's bundler without extra
// webpack config. A div-icon has no asset-resolution problem at all.
const pinIcon = L.divIcon({
  className: "",
  html: '<div style="width:16px;height:16px;border-radius:50% 50% 50% 0;background:#2563eb;border:2px solid white;transform:rotate(-45deg);box-shadow:0 1px 3px rgba(0,0,0,0.4)"></div>',
  iconSize: [16, 16],
  iconAnchor: [8, 16],
});

const DEFAULT_CENTER: [number, number] = [50.45, 30.52]; // Kyiv — D-Scope's Ukraine-first rollout
const DEFAULT_ZOOM = 6;

export function DiscoveryMap({ initialSites }: { initialSites: DiscoverySite[] }) {
  const [sites, setSites] = useState(initialSites);
  const [tradition, setTradition] = useState("");
  const [language, setLanguage] = useState("");
  const [isPending, startTransition] = useTransition();

  function runSearch(e: React.FormEvent) {
    e.preventDefault();
    startTransition(async () => {
      const result = await searchAction({
        tradition: tradition || undefined,
        language: language || undefined,
      });
      setSites(result);
    });
  }

  return (
    <div className="flex min-h-screen flex-col gap-4 p-4">
      <form onSubmit={runSearch} className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">Tradition (taxon code)</span>
          <input
            value={tradition}
            onChange={(e) => setTradition(e.target.value)}
            placeholder="orthodox"
            className="rounded border px-3 py-2"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">Service language (ISO 639-3)</span>
          <input
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            placeholder="ukr"
            className="rounded border px-3 py-2"
          />
        </label>
        <button type="submit" disabled={isPending} className="rounded border px-4 py-2">
          {isPending ? "Searching…" : "Search"}
        </button>
        <span className="text-sm text-gray-500">{sites.length} results</span>
      </form>

      <div className="h-[70vh] w-full overflow-hidden rounded border">
        <MapContainer center={DEFAULT_CENTER} zoom={DEFAULT_ZOOM} style={{ height: "100%", width: "100%" }}>
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          {sites
            .filter((s) => s.latitude != null && s.longitude != null)
            .map((s) => (
              <Marker key={s.id} position={[s.latitude as number, s.longitude as number]} icon={pinIcon}>
                <Popup>
                  {/* contentSiteId is content's internal uuid, not what getSite accepts (see
                      app/congregations/[unitId]/page.tsx's header comment) — its presence is
                      still the right "has this congregation published a site at all" signal. */}
                  {s.contentSiteId ? (
                    <Link href={`/congregations/${s.congregationUnitRid}`}>View congregation page</Link>
                  ) : (
                    <span>No published page yet.</span>
                  )}
                </Popup>
              </Marker>
            ))}
        </MapContainer>
      </div>
    </div>
  );
}
