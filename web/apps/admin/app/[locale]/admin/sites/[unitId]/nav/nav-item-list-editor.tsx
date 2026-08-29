// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.10: reuses block-list-editor.tsx's dnd-kit reorder IDIOM (DndContext/SortableContext/
// useSortable, PointerSensor+KeyboardSensor, arrayMove, move-up/move-down buttons, an aria-live
// announcer, per-row remove) — not its code. A parallel component, since block-list-editor.tsx is
// tightly coupled to block-specific types (ClientBlock, BlockType, BlockDataForm).
//
// An explicit Save button, not autosave: a nav menu is edited far less often than page content and
// is typically under ten rows — M14.6's debounce/saved-indicator machinery isn't earning its keep
// here.
"use client";

import { useCallback, useState } from "react";
import { useTranslations } from "next-intl";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { ChevronDown, ChevronUp, GripVertical, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { Document, NavItem, NavItemInput } from "@/lib/content";

export type NavSaveResult =
  | { ok: true }
  | {
      ok: false;
      sortOrder?: number;
      error: "errorNavTargetInvalid" | "errorNavTargetAmbiguous" | "errorDuplicateNavItemSortOrder" | "errorGeneric";
      raw?: string;
    };

interface ClientNavItem {
  /** Client-only identity for React keys and drag-and-drop — never submitted; NavItemInput has no id. */
  key: string;
  label: string;
  mode: "page" | "external";
  targetDocumentId: string | null;
  targetUrl: string | null;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;
}

function toClientItems(items: NavItem[]): ClientNavItem[] {
  return items.map((item) => ({
    key: item.id,
    label: item.label,
    mode: item.targetUrl ? "external" : "page",
    targetDocumentId: item.targetDocumentId ?? null,
    targetUrl: item.targetUrl ?? null,
  }));
}

export function NavItemListEditor({
  items,
  pages,
  onSave,
}: {
  items: NavItem[];
  pages: Document[];
  onSave: (items: NavItemInput[]) => Promise<NavSaveResult>;
}) {
  const t = useTranslations("NavPage");
  const [rows, setRows] = useState<ClientNavItem[]>(() => toClientItems(items));
  const [liveMessage, setLiveMessage] = useState("");
  const [status, setStatus] = useState<"idle" | "saving" | "saved" | "error">("idle");
  const [error, setError] = useState<{ sortOrder?: number; message?: string }>({});

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  function rowName(row: ClientNavItem): string {
    return row.label || t("untitledItem");
  }

  function announceMoved(name: string, position: number, total: number) {
    setLiveMessage(t("itemMovedAnnouncement", { name, position, total }));
  }

  function moveRow(from: number, to: number) {
    setRows((prev) => {
      if (to < 0 || to >= prev.length) return prev;
      announceMoved(rowName(prev[from]), to + 1, prev.length);
      return arrayMove(prev, from, to);
    });
  }

  function removeRow(index: number) {
    setRows((prev) => {
      setLiveMessage(t("itemRemovedAnnouncement", { name: rowName(prev[index]) }));
      return prev.filter((_, i) => i !== index);
    });
  }

  function addRow() {
    setRows((prev) => {
      setLiveMessage(t("itemAddedAnnouncement"));
      return [...prev, { key: newKey(), label: "", mode: "page", targetDocumentId: pages[0]?.id ?? null, targetUrl: null }];
    });
  }

  function updateRow(index: number, patch: Partial<ClientNavItem>) {
    setRows((prev) => prev.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setRows((prev) => {
      const from = prev.findIndex((r) => r.key === active.id);
      const to = prev.findIndex((r) => r.key === over.id);
      if (from === -1 || to === -1) return prev;
      announceMoved(rowName(prev[from]), to + 1, prev.length);
      return arrayMove(prev, from, to);
    });
  }

  const handleSave = useCallback(async () => {
    setStatus("saving");
    const inputs: NavItemInput[] = rows.map((row, i) => ({
      label: row.label,
      targetDocumentId: row.mode === "page" ? (row.targetDocumentId ?? undefined) : undefined,
      targetUrl: row.mode === "external" ? (row.targetUrl ?? undefined) : undefined,
      sortOrder: i,
    }));
    const result = await onSave(inputs);
    if (result.ok) {
      setError({});
      setStatus("saved");
      return;
    }
    setError({ sortOrder: result.sortOrder, message: t(result.error, result.raw ? { error: result.raw } : undefined) });
    setStatus("error");
  }, [rows, onSave, t]);

  return (
    <div className="flex flex-col gap-4">
      <div aria-live="polite" className="sr-only">
        {liveMessage}
      </div>

      {rows.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("empty")}</p>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={rows.map((r) => r.key)} strategy={verticalListSortingStrategy}>
            {rows.map((row, index) => (
              <SortableNavItemRow
                key={row.key}
                itemKey={row.key}
                row={row}
                index={index}
                total={rows.length}
                pages={pages}
                hasError={index === error.sortOrder}
                onChange={(patch) => updateRow(index, patch)}
                onMoveUp={() => moveRow(index, index - 1)}
                onMoveDown={() => moveRow(index, index + 1)}
                onRemove={() => removeRow(index)}
              />
            ))}
          </SortableContext>
        </DndContext>
      )}

      <Button type="button" variant="outline" className="self-start" onClick={addRow}>
        {t("addItem")}
      </Button>

      <div className="flex items-center gap-3">
        <Button type="button" onClick={handleSave} className="self-start">
          {t("save")}
        </Button>
        <span role="status" className="text-sm text-muted-foreground">
          {status === "saving" && t("saving")}
          {status === "saved" && t("saved")}
        </span>
        {status === "error" && error.message ? <span className="text-sm text-destructive">{error.message}</span> : null}
      </div>
    </div>
  );
}

function SortableNavItemRow({
  itemKey,
  row,
  index,
  total,
  pages,
  hasError,
  onChange,
  onMoveUp,
  onMoveDown,
  onRemove,
}: {
  itemKey: string;
  row: ClientNavItem;
  index: number;
  total: number;
  pages: Document[];
  hasError: boolean;
  onChange: (patch: Partial<ClientNavItem>) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onRemove: () => void;
}) {
  const t = useTranslations("NavPage");
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: itemKey });
  const style = { transform: CSS.Transform.toString(transform), transition };
  const name = row.label || t("untitledItem");

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex flex-col items-start gap-3 rounded-md border bg-background p-3 sm:grid sm:grid-cols-[auto_auto_1fr_auto] sm:items-start"
    >
      <button
        type="button"
        className="mt-1 flex h-8 w-6 cursor-grab touch-none items-center justify-center text-muted-foreground"
        aria-label={t("dragToReorder", { name })}
        {...attributes}
        {...listeners}
      >
        <GripVertical className="size-4" />
      </button>
      <div className="mt-1 flex flex-col gap-1">
        <Button type="button" variant="ghost" size="icon" aria-label={t("moveItemUp")} disabled={index === 0} onClick={onMoveUp}>
          <ChevronUp />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={t("moveItemDown")}
          disabled={index === total - 1}
          onClick={onMoveDown}
        >
          <ChevronDown />
        </Button>
      </div>
      <div className="flex w-full flex-col gap-2">
        <Label className="flex flex-col items-start gap-1">
          {t("labelLabel")}
          <Input value={row.label} onChange={(e) => onChange({ label: e.target.value })} required aria-invalid={hasError || undefined} />
        </Label>
        <div className="flex gap-2">
          <Button
            type="button"
            size="sm"
            variant={row.mode === "page" ? "default" : "outline"}
            onClick={() => onChange({ mode: "page" })}
          >
            {t("modePage")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant={row.mode === "external" ? "default" : "outline"}
            onClick={() => onChange({ mode: "external" })}
          >
            {t("modeExternal")}
          </Button>
        </div>
        {row.mode === "page" ? (
          <Select value={row.targetDocumentId ?? undefined} onValueChange={(v) => onChange({ targetDocumentId: v })}>
            <SelectTrigger className="w-full" aria-invalid={hasError || undefined}>
              <SelectValue placeholder={t("pagePlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {pages.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.slug} ({p.locale})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <Input
            type="url"
            value={row.targetUrl ?? ""}
            onChange={(e) => onChange({ targetUrl: e.target.value })}
            placeholder="https://…"
            required
            aria-invalid={hasError || undefined}
          />
        )}
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={t("removeItem")}
        onClick={onRemove}
        className="mt-1 text-destructive"
      >
        <Trash2 />
      </Button>
    </div>
  );
}
