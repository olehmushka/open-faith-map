// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";

export type MobileView = "list" | "map";

export function MobileViewToggle({
  value,
  onChange,
}: {
  value: MobileView;
  onChange: (view: MobileView) => void;
}) {
  const t = useTranslations("DiscoveryMap");

  return (
    <div className="flex gap-1 px-3 pb-2 md:hidden" role="group" aria-label={t("viewToggleLabel")}>
      <Button
        type="button"
        size="sm"
        variant={value === "list" ? "default" : "outline"}
        onClick={() => onChange("list")}
        className="flex-1"
      >
        {t("listView")}
      </Button>
      <Button
        type="button"
        size="sm"
        variant={value === "map" ? "default" : "outline"}
        onClick={() => onChange("map")}
        className="flex-1"
      >
        {t("mapView")}
      </Button>
    </div>
  );
}
