// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// The seven SiteAttributes.Accessibility JSON keys (M13.2), same order as
// web/apps/web/lib/accessibility.ts's own copy — the two Next apps share no internal package, so
// this is a deliberate duplicate, not a drift risk: both lists are just the fixed
// AccessibilityCriteria the backend already treats as the source of truth.
//
// Deliberately its own plain module, not defined inside attributes-form.tsx (where it used to
// live): that file has "use client" at the top, and React Server Components treats *every* export
// of a "use client" module as an opaque client-only reference once imported from a Server
// Component — even a plain constant array, not just components. page.tsx (a Server Component)
// reads this array directly (ACCESSIBILITY_KEYS.map(...)) outside of JSX, which silently received a
// stub function instead of the real array when it lived in the client file, throwing
// "ACCESSIBILITY_KEYS.map is not a function" the first time this page rendered for a unit that
// already had a religion_sites row (found live, 2026-08-28 — the ternary at page.tsx's
// `attributesCard` meant the bug's code path was never reached until then).
export const ACCESSIBILITY_KEYS = [
  "stepFreeEntrance",
  "accessibleRestroom",
  "hearingLoop",
  "signLanguageInterpretation",
  "accessibleParking",
  "wheelchairSeating",
  "brailleOrLargePrint",
] as const;
