"use client";

import { useTranslations } from "next-intl";
import { Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { toggleCommandPalette } from "@/components/command-palette";

export function CommandPaletteButton() {
  const t = useTranslations("AdminShell");

  return (
    <Button
      variant="outline"
      size="sm"
      className="gap-2 text-muted-foreground"
      onClick={toggleCommandPalette}
    >
      <Search className="size-3.5" />
      <span className="hidden sm:inline">{t("commandPalette")}</span>
      <kbd className="hidden rounded border bg-muted px-1.5 py-0.5 text-[10px] font-mono sm:inline">
        ⌘K
      </kbd>
    </Button>
  );
}
