"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { MapPinned } from "lucide-react";

import { useRouter } from "@/i18n/navigation";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { NAV } from "@/components/admin-sidebar";
import type { JurisdictionUnit } from "@/lib/jurisdiction";

export const COMMAND_PALETTE_TOGGLE_EVENT = "admin:command-palette:toggle";

export function toggleCommandPalette() {
  window.dispatchEvent(new Event(COMMAND_PALETTE_TOGGLE_EVENT));
}

export function CommandPalette({
  onSearchSite,
}: {
  onSearchSite: (query: string) => Promise<JurisdictionUnit[]>;
}) {
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [siteResults, setSiteResults] = React.useState<JurisdictionUnit[]>([]);
  const router = useRouter();
  const t = useTranslations("AdminShell");

  React.useEffect(() => {
    const keyHandler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen((v) => !v);
      }
    };
    // The topbar's own ⌘K button lives outside this component, so it toggles the palette via a
    // plain DOM event rather than lifting open-state into a shared context for one boolean.
    const toggleHandler = () => setOpen((v) => !v);
    document.addEventListener("keydown", keyHandler);
    window.addEventListener(COMMAND_PALETTE_TOGGLE_EVENT, toggleHandler);
    return () => {
      document.removeEventListener("keydown", keyHandler);
      window.removeEventListener(COMMAND_PALETTE_TOGGLE_EVENT, toggleHandler);
    };
  }, []);

  React.useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed) {
      setSiteResults([]);
      return;
    }
    const timeout = setTimeout(() => {
      onSearchSite(trimmed).then(setSiteResults).catch(() => setSiteResults([]));
    }, 200);
    return () => clearTimeout(timeout);
  }, [query, onSearchSite]);

  function go(href: string) {
    setOpen(false);
    setQuery("");
    router.push(href);
  }

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title={t("commandPalette")}
      description={t("commandPalettePlaceholder")}
    >
      <CommandInput
        placeholder={t("commandPalettePlaceholder")}
        value={query}
        onValueChange={setQuery}
      />
      <CommandList>
        <CommandEmpty>{t("commandPaletteNoResults")}</CommandEmpty>
        <CommandGroup heading={t("sections")}>
          {NAV.flatMap((item) => [
            { href: item.href, labelKey: item.labelKey },
            ...(item.children ?? []),
          ]).map((entry) => (
            <CommandItem key={entry.href} onSelect={() => go(entry.href)}>
              {t(entry.labelKey)}
            </CommandItem>
          ))}
        </CommandGroup>
        {siteResults.length > 0 && (
          <CommandGroup heading={t("commandPaletteSites")}>
            {siteResults.map((unit) => (
              <CommandItem key={unit.id} onSelect={() => go(`/admin/sites/${unit.id}`)}>
                <MapPinned />
                {unit.name}
              </CommandItem>
            ))}
          </CommandGroup>
        )}
      </CommandList>
    </CommandDialog>
  );
}
