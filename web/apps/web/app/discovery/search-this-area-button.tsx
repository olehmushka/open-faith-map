// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";

export function SearchThisAreaButton({ onClick, pending }: { onClick: () => void; pending: boolean }) {
  const t = useTranslations("DiscoveryMap");

  return (
    <div className="pointer-events-none absolute inset-x-0 top-3 z-[1000] flex justify-center">
      <Button onClick={onClick} disabled={pending} className="pointer-events-auto shadow-md">
        {pending ? t("searching") : t("searchThisArea")}
      </Button>
    </div>
  );
}
