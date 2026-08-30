// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.13, D-SitePatterns: a searchable palette of starter patterns, the same shadcn Command/
// CommandDialog primitive block-inserter.tsx already uses — deliberately a separate, flat list
// (name + description only, no categories) rather than folded into the block-type inserter, since
// a pattern is a different kind of thing to insert (many blocks at once, unsynced) from a single
// block. Selecting one hands the whole Pattern back to the caller, which copies its blocks into
// local editor state — this component itself never talks to the API.
"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { LayoutTemplate } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import type { Pattern } from "@/lib/content";

export function PatternInserter({
  patterns,
  onInsert,
  triggerLabel,
  triggerVariant = "outline",
  triggerSize = "sm",
}: {
  patterns: Pattern[];
  onInsert: (pattern: Pattern) => void;
  triggerLabel?: string;
  triggerVariant?: "outline" | "default";
  triggerSize?: "sm" | "default";
}) {
  const [open, setOpen] = useState(false);
  const t = useTranslations("DocumentEditorPage");

  if (patterns.length === 0) return null;

  return (
    <>
      <Button type="button" variant={triggerVariant} size={triggerSize} className="self-start" onClick={() => setOpen(true)}>
        <LayoutTemplate /> {triggerLabel ?? t("insertPattern")}
      </Button>
      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        title={t("insertPatternDialogTitle")}
        description={t("insertPatternDialogDescription")}
      >
        <Command>
          <CommandInput placeholder={t("searchPatternsPlaceholder")} />
          <CommandList>
            <CommandEmpty>{t("noMatchingPattern")}</CommandEmpty>
            <CommandGroup>
              {patterns.map((p) => (
                <CommandItem
                  key={p.id}
                  value={`${p.name} ${p.description}`}
                  onSelect={() => {
                    onInsert(p);
                    setOpen(false);
                  }}
                >
                  <div className="flex flex-col">
                    <span>{p.name}</span>
                    {p.description ? <span className="text-xs text-muted-foreground">{p.description}</span> : null}
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </CommandDialog>
    </>
  );
}
