// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Mirrors web/apps/admin/lib/accessibility.ts's own ACCESSIBILITY_KEYS — same seven
// SiteAttributes.Accessibility JSON keys, same order. The two Next apps share no internal package,
// so this is a deliberate duplicate, not a drift risk: both lists are just the fixed
// AccessibilityCriteria the backend already treats as the source of truth.
export const ACCESSIBILITY_KEYS = [
  "stepFreeEntrance",
  "accessibleRestroom",
  "hearingLoop",
  "signLanguageInterpretation",
  "accessibleParking",
  "wheelchairSeating",
  "brailleOrLargePrint",
] as const;

export type AccessibilityKey = (typeof ACCESSIBILITY_KEYS)[number];

// Maps each key to its `DiscoveryMap` translation-message key (flat, matching this namespace's
// existing non-nested style).
export const ACCESSIBILITY_MESSAGE_KEYS: Record<AccessibilityKey, string> = {
  stepFreeEntrance: "a11yStepFreeEntrance",
  accessibleRestroom: "a11yAccessibleRestroom",
  hearingLoop: "a11yHearingLoop",
  signLanguageInterpretation: "a11ySignLanguageInterpretation",
  accessibleParking: "a11yAccessibleParking",
  wheelchairSeating: "a11yWheelchairSeating",
  brailleOrLargePrint: "a11yBrailleOrLargePrint",
};
