// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.5: a categorized, searchable block inserter, replacing the single hardcoded "new block" row
// (whose type <Select> was native/uncontrolled, so it couldn't preview the chosen type until
// save+reload). Built on shadcn's Command (cmdk + Radix Dialog), already installed for other admin
// screens — auto-focus, arrow-key navigation, Enter-to-select, and Escape-to-close all come from
// that primitive for free, so this component adds none of its own keyboard handling.
"use client";

import { useState, type ComponentProps } from "react";
import { useTranslations } from "next-intl";
import { Plus } from "lucide-react";

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
import type { BlockType } from "@/lib/content";

import { BLOCK_CATEGORY_LABELS, BLOCK_CATEGORY_ORDER, blockCatalogEntry } from "./block-catalog";

export function BlockInserter({
  blockTypes,
  onInsert,
  triggerLabel,
  triggerVariant = "outline",
  triggerSize = "sm",
}: {
  blockTypes: BlockType[];
  onInsert: (blockType: BlockType) => void;
  /** M14.8: lets the empty-state CTA (block-list-editor.tsx) reuse this same dialog with its own
   * wording/prominence instead of duplicating the Command/category logic for a second trigger. */
  triggerLabel?: string;
  triggerVariant?: ComponentProps<typeof Button>["variant"];
  triggerSize?: ComponentProps<typeof Button>["size"];
}) {
  const [open, setOpen] = useState(false);
  const t = useTranslations("DocumentEditorPage");

  const groups = BLOCK_CATEGORY_ORDER.map((category) => ({
    category,
    types: blockTypes
      .filter((bt) => blockCatalogEntry(bt.code).category === category)
      .sort((a, b) => a.sortOrder - b.sortOrder),
  })).filter((group) => group.types.length > 0);

  return (
    <>
      <Button type="button" variant={triggerVariant} size={triggerSize} className="self-start" onClick={() => setOpen(true)}>
        <Plus /> {triggerLabel ?? t("addBlock")}
      </Button>
      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        title={t("addBlockDialogTitle")}
        description={t("addBlockDialogDescription")}
      >
        <Command>
          <CommandInput placeholder={t("searchBlockTypesPlaceholder")} />
          <CommandList>
            <CommandEmpty>{t("noMatchingBlockType")}</CommandEmpty>
            {groups.map(({ category, types }) => (
              <CommandGroup key={category} heading={BLOCK_CATEGORY_LABELS[category]}>
                {types.map((bt) => {
                  const { description } = blockCatalogEntry(bt.code);
                  return (
                    <CommandItem
                      key={bt.code}
                      value={`${bt.name} ${description}`}
                      onSelect={() => {
                        onInsert(bt);
                        setOpen(false);
                      }}
                    >
                      <div className="flex flex-col">
                        <span>{bt.name}</span>
                        {description ? <span className="text-xs text-muted-foreground">{description}</span> : null}
                      </div>
                    </CommandItem>
                  );
                })}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </CommandDialog>
    </>
  );
}
