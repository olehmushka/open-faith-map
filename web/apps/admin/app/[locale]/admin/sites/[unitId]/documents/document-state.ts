// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import type { StatusTone } from "@/components/status-badge";
import type { Document } from "@/lib/content";

// M14.15/D-PublishOnRead: both DocumentsPage and the document editor must render effectiveState,
// never the raw state column — a SCHEDULED document past its publishAt reads as Published here,
// exactly like a real one. Shared so the two pages can't drift on what each state is called or
// colored.
export function documentStateLabel(
  t: (key: string) => string,
  state: Document["effectiveState"],
): string {
  switch (state) {
    case "DRAFT":
      return t("stateDraft");
    case "PUBLISHED":
      return t("statePublished");
    case "UNLISTED":
      return t("stateUnlisted");
    case "SCHEDULED":
      return t("stateScheduled");
  }
}

export const DOCUMENT_STATE_TONE: Record<Document["effectiveState"], StatusTone> = {
  DRAFT: "neutral",
  SCHEDULED: "warning",
  PUBLISHED: "success",
  UNLISTED: "info",
};
