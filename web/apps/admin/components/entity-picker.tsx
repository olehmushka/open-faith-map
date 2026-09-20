// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { ChevronsUpDown } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";

export type EntityPickerOption = { id: string; label: string; sublabel?: string };

/** Imperative escape hatch for the one caller (congregation-import's unit picker) that needs to
 * set the selection from outside — e.g. after its own "create missing unit" dialog succeeds.
 * Every other caller just reads the hidden input via the enclosing form/searchParams, the same
 * as a plain <Input> always has. */
export type EntityPickerHandle = { select: (option: EntityPickerOption) => void };

// Matches command-palette.tsx's own debounce window for the same reason: fast enough to feel
// live, slow enough that normal typing speed doesn't fire a request per keystroke.
const SEARCH_DEBOUNCE_MS = 200;

/**
 * Search-and-select a person/unit/taxon by name, replacing a bare id <input>. Renders a hidden
 * input carrying the real id, so every consuming form/server-action/searchParams read is
 * unchanged (`formData.get(name)` or the querystring both still resolve as before) — this is a
 * drop-in <Input> replacement, not a form-shape change. Generalizes
 * congregation-import/jurisdiction-field.tsx's JurisdictionField (M17): same hidden-input
 * mechanic, but built on the cmdk primitives in components/ui/command.tsx for live debounced
 * search with keyboard nav, instead of a manual Search button.
 *
 * Uncontrolled by design (own internal selection state seeded from `defaultValue`, a plain
 * serializable option) rather than a controlled value/onChange pair — a Server Component parent
 * can only pass plain data or Server Actions as props to a Client Component, never a synchronous
 * state setter, so a controlled API would be unusable from most of this component's callers.
 */
export const EntityPicker = React.forwardRef<
  EntityPickerHandle,
  {
    name: string;
    defaultValue?: EntityPickerOption | null;
    onSearch: (query: string) => Promise<EntityPickerOption[]>;
    placeholder: string;
    disabled?: boolean;
    className?: string;
  }
>(function EntityPicker(
  { name, defaultValue = null, onSearch, placeholder, disabled, className },
  ref,
) {
  const t = useTranslations("EntityPicker");
  const [value, setValue] = React.useState<EntityPickerOption | null>(defaultValue);
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [results, setResults] = React.useState<EntityPickerOption[]>([]);
  const [pending, setPending] = React.useState(false);

  React.useImperativeHandle(ref, () => ({
    select(option: EntityPickerOption) {
      setValue(option);
      setOpen(false);
      setQuery("");
    },
  }));

  React.useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed) {
      setResults([]);
      setPending(false);
      return;
    }
    setPending(true);
    const timeout = setTimeout(() => {
      onSearch(trimmed)
        .then(setResults)
        .catch(() => setResults([]))
        .finally(() => setPending(false));
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timeout);
  }, [query, onSearch]);

  function select(option: EntityPickerOption) {
    setValue(option);
    setOpen(false);
    setQuery("");
  }

  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <input type="hidden" name={name} value={value?.id ?? ""} />
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="outline"
            role="combobox"
            aria-expanded={open}
            disabled={disabled}
            className="h-8 w-full justify-between font-normal"
          >
            <span className={cn("truncate text-left", !value && "text-muted-foreground")}>
              {value ? value.label : placeholder}
            </span>
            <ChevronsUpDown className="size-3.5 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-72 p-0" align="start">
          {/* shouldFilter=false: results are already server-filtered by onSearch, so cmdk's own
              client-side fuzzy match must not additionally hide any of them. */}
          <Command shouldFilter={false}>
            <CommandInput
              placeholder={t("searchPlaceholder")}
              value={query}
              onValueChange={setQuery}
            />
            <CommandList>
              {!pending && query.trim() !== "" && results.length === 0 && (
                <CommandEmpty>{t("noMatches")}</CommandEmpty>
              )}
              {results.map((option) => (
                <CommandItem
                  key={option.id}
                  value={option.id}
                  data-checked={value?.id === option.id}
                  onSelect={() => select(option)}
                >
                  <span className="flex flex-col">
                    <span>{option.label}</span>
                    {option.sublabel && (
                      <span className="text-xs text-muted-foreground">{option.sublabel}</span>
                    )}
                  </span>
                </CommandItem>
              ))}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
});
