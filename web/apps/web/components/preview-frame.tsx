// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const DEVICE_WIDTHS = {
  mobile: "375px",
  tablet: "768px",
  full: "100%",
} as const;

type Device = keyof typeof DEVICE_WIDTHS;

// M14.7's device-width toggle. No iframe: the site renders once, in this same document, through the
// real public renderer (components/site-page.tsx) — this component only constrains the width of
// whatever it's handed, so there is exactly one render pipeline to keep pixel-identical to
// publishing, not a second one to drift.
export function PreviewFrame({ children }: { children: React.ReactNode }) {
  const t = useTranslations("Preview");
  const [device, setDevice] = useState<Device>("full");

  return (
    <div className="flex min-h-screen flex-col bg-muted">
      <div className="sticky top-0 z-50 flex flex-wrap items-center justify-between gap-2 border-b bg-foreground px-4 py-2 text-background">
        <span className="text-sm font-medium">{t("banner")}</span>
        <div className="flex gap-1">
          {(Object.keys(DEVICE_WIDTHS) as Device[]).map((key) => (
            <Button
              key={key}
              type="button"
              size="sm"
              variant={device === key ? "secondary" : "ghost"}
              className={cn(device !== key && "text-background hover:text-background")}
              onClick={() => setDevice(key)}
            >
              {t(key === "mobile" ? "deviceMobile" : key === "tablet" ? "deviceTablet" : "deviceFull")}
            </Button>
          ))}
        </div>
      </div>
      <div
        className="mx-auto w-full flex-1 bg-background transition-[max-width] duration-200"
        style={{ maxWidth: DEVICE_WIDTHS[device] }}
      >
        {children}
      </div>
    </div>
  );
}
