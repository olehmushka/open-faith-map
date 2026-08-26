// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { GeolocationResult } from "./geolocation";

export function haversineMeters(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const R = 6_371_000;
  const toRad = (d: number) => (d * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 + Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(a));
}

export type DistanceUnit = "km" | "mi";

// Rough bounding boxes for the continental US, Alaska, and Hawaii — a heuristic, not a real
// reverse-geocode. Good enough to pick a unit convention; never used for anything precision-
// sensitive. Every other resolved location (or a denied/unsupported/pending geolocation) gets km.
function isLikelyUnitedStates(lat: number, lng: number): boolean {
  const continental = lat >= 24.5 && lat <= 49.5 && lng >= -125 && lng <= -66.9;
  const alaska = lat >= 51 && lat <= 72 && lng >= -170 && lng <= -129;
  const hawaii = lat >= 18.5 && lat <= 22.5 && lng >= -160 && lng <= -154;
  return continental || alaska || hawaii;
}

export function resolveDistanceUnit(geolocation: GeolocationResult): DistanceUnit {
  if (geolocation.status === "granted" && isLikelyUnitedStates(geolocation.lat, geolocation.lng)) {
    return "mi";
  }
  return "km";
}

// "km"/"mi"/"m" are used as plain, language-neutral unit abbreviations (SI/imperial symbols read
// the same across en/es/pt/uk) rather than translated strings.
export function formatDistance(meters: number, unit: DistanceUnit): string {
  if (unit === "mi") {
    const miles = meters / 1609.344;
    return miles < 10 ? `${miles.toFixed(1)} mi` : `${Math.round(miles)} mi`;
  }
  if (meters < 1000) return `${Math.round(meters)} m`;
  const km = meters / 1000;
  return km < 10 ? `${km.toFixed(1)} km` : `${Math.round(km)} km`;
}
