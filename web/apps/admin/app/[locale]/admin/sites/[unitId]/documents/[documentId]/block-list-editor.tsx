// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.5: replaces the server-rendered, uncontrolled-form block list (manually-typed `position`
// numbers, one hardcoded "new block" row, no remove control) with a client-managed array supporting
// drag-and-drop reorder, first-class keyboard move-up/move-down, a categorized inserter
// (block-inserter.tsx), and per-block removal.
//
// M14.6: the old single "Save" button that submitted a <form action={action}> and redirected is
// replaced by useDebouncedAutosave (hooks/use-debounced-autosave.ts) — saves on a ~10s debounce
// after the last edit, with a visible saved/unsaved indicator, plus a manual "Save now" trigger for
// anyone who doesn't want to wait. This never navigates, so the pre-submit sessionStorage snapshot
// M14.5 added (to survive a redirect-on-error round trip) is gone too — that failure mode no longer
// exists once saving doesn't redirect, and the server now persists the draft on its own debounce,
// which is what actually satisfies "survive a refresh mid-edit" per M14.6's acceptance criteria.
//
// M14.8: adds session-local undo/redo (hooks/use-block-history.ts, reusing this file's own
// getSnapshot) and a real empty state for a document with zero blocks. The milestone text says an
// empty site should offer "start from a template" — M14.13 (content_patterns) doesn't exist
// anywhere in this repo yet, so the empty state offers a clear CTA into the existing inserter
// instead; M14.13, when built, is what would make it offer real starter layouts. A block-level
// undo/redo mutation is just another change to useDebouncedAutosave's polled form state — no
// plumbing connects the two hooks, it's autosaved like any manual edit.
"use client";

import { useCallback, useEffect, useRef, useState } from "react";
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
import { ChevronDown, ChevronUp, GripVertical, Redo2, Trash2, Undo2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { Block, BlockType } from "@/lib/content";
import { useDebouncedAutosave, type AutosaveStatus } from "@/hooks/use-debounced-autosave";
import { useBlockHistory } from "@/hooks/use-block-history";

import { BlockDataForm } from "./block-data-form";
import { BlockInserter } from "./block-inserter";

export interface BlockSaveInput {
  position: number;
  blockTypeCode: string;
  data: unknown;
}

export type BlockSaveResult = { ok: true } | { ok: false; position?: number; field?: string };

interface ClientBlock {
  /** Client-only identity for React keys and drag-and-drop — never submitted; BlockInput has no id. */
  key: string;
  blockTypeCode: string;
  data: unknown;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;
}

function statusLabel(t: ReturnType<typeof useTranslations>, status: AutosaveStatus): string {
  switch (status) {
    case "saving":
      return t("autosaveSaving");
    case "saved":
      return t("autosaveSaved");
    case "unsaved":
      return t("autosaveUnsaved");
    case "error":
      return t("autosaveError");
    default:
      return "";
  }
}

export function BlockListEditor({
  blocks,
  blockTypes,
  onAutosave,
}: {
  blocks: Block[];
  blockTypes: BlockType[];
  onAutosave: (inputs: BlockSaveInput[]) => Promise<BlockSaveResult>;
}) {
  const t = useTranslations("DocumentEditorPage");
  const formRef = useRef<HTMLFormElement>(null);
  const [items, setItems] = useState<ClientBlock[]>(() =>
    blocks.map((b) => ({ key: b.id, blockTypeCode: b.blockTypeCode, data: b.data })),
  );
  const [liveMessage, setLiveMessage] = useState("");
  const [lastError, setLastError] = useState<{ position?: number; field?: string }>({});

  const getSnapshot = useCallback((): BlockSaveInput[] | null => {
    const form = formRef.current;
    if (!form) return null;
    const formData = new FormData(form);
    const positions = formData.getAll("position").map(String);
    const blockTypeCodes = formData.getAll("blockTypeCode").map(String);
    const dataJson = formData.getAll("data").map(String);
    return positions.map((position, i) => ({
      position: Number(position),
      blockTypeCode: blockTypeCodes[i],
      data: JSON.parse(dataJson[i] || "{}"),
    }));
  }, []);

  const save = useCallback(
    async (inputs: BlockSaveInput[]): Promise<{ ok: boolean }> => {
      const result = await onAutosave(inputs);
      if (result.ok) {
        setLastError({});
        return { ok: true };
      }
      setLastError({ position: result.position, field: result.field });
      return { ok: false };
    },
    [onAutosave],
  );

  const { status, flush } = useDebouncedAutosave(getSnapshot, save);
  const { canUndo, canRedo, undo, redo } = useBlockHistory(getSnapshot);

  const restoreSnapshot = useCallback((snapshot: BlockSaveInput[]) => {
    // Fresh keys force every SortableBlockRow (and the BlockDataForm nested inside it) to remount,
    // which is what re-seeds BlockDataForm's own local `data` state from the restored value — it
    // never lifts per-keystroke edits up into `items`, so a prop change alone wouldn't be picked up.
    setItems(
      snapshot
        .slice()
        .sort((a, b) => a.position - b.position)
        .map((s) => ({ key: newKey(), blockTypeCode: s.blockTypeCode, data: s.data })),
    );
  }, []);

  const handleUndo = useCallback(() => {
    const snapshot = undo();
    if (snapshot) {
      restoreSnapshot(snapshot);
      setLiveMessage(t("undoAnnouncement"));
    }
  }, [undo, restoreSnapshot, t]);

  const handleRedo = useCallback(() => {
    const snapshot = redo();
    if (snapshot) {
      restoreSnapshot(snapshot);
      setLiveMessage(t("redoAnnouncement"));
    }
  }, [redo, restoreSnapshot, t]);

  // Global Ctrl/Cmd+Z (+ Shift for redo), mirroring components/command-palette.tsx's own
  // keydown-listener pattern. Skipped while focus is inside a text field so a user correcting a
  // typo keeps the browser's native per-field undo instead of having it hijacked into this coarser,
  // settle-window-granularity block restore.
  useEffect(() => {
    function keyHandler(e: KeyboardEvent) {
      if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== "z") return;
      const target = e.target as HTMLElement | null;
      if (target?.tagName === "INPUT" || target?.tagName === "TEXTAREA" || target?.isContentEditable) return;
      e.preventDefault();
      if (e.shiftKey) {
        handleRedo();
      } else {
        handleUndo();
      }
    }
    document.addEventListener("keydown", keyHandler);
    return () => document.removeEventListener("keydown", keyHandler);
  }, [handleUndo, handleRedo]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  function nameFor(code: string): string {
    return blockTypes.find((bt) => bt.code === code)?.name ?? code;
  }

  function announceMoved(name: string, position: number, total: number) {
    setLiveMessage(t("blockMovedAnnouncement", { name, position, total }));
  }

  function moveBlock(from: number, to: number) {
    setItems((prev) => {
      if (to < 0 || to >= prev.length) return prev;
      announceMoved(nameFor(prev[from].blockTypeCode), to + 1, prev.length);
      return arrayMove(prev, from, to);
    });
  }

  function removeBlock(index: number) {
    setItems((prev) => {
      setLiveMessage(t("blockRemovedAnnouncement", { name: nameFor(prev[index].blockTypeCode) }));
      return prev.filter((_, i) => i !== index);
    });
  }

  function insertBlock(blockType: BlockType) {
    setItems((prev) => {
      setLiveMessage(t("blockAddedAnnouncement", { name: blockType.name }));
      return [...prev, { key: newKey(), blockTypeCode: blockType.code, data: {} }];
    });
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setItems((prev) => {
      const from = prev.findIndex((b) => b.key === active.id);
      const to = prev.findIndex((b) => b.key === over.id);
      if (from === -1 || to === -1) return prev;
      announceMoved(nameFor(prev[from].blockTypeCode), to + 1, prev.length);
      return arrayMove(prev, from, to);
    });
  }

  return (
    <form ref={formRef} onSubmit={(e) => e.preventDefault()} className="flex flex-col gap-4">
      <div aria-live="polite" className="sr-only">
        {liveMessage}
      </div>

      {/* Rendered unconditionally (not inside the empty-state branch below): deleting the last
          block is exactly when a visible undo control matters most. */}
      <div className="flex items-center gap-1">
        <Button type="button" variant="ghost" size="icon" aria-label={t("undo")} disabled={!canUndo} onClick={handleUndo}>
          <Undo2 />
        </Button>
        <Button type="button" variant="ghost" size="icon" aria-label={t("redo")} disabled={!canRedo} onClick={handleRedo}>
          <Redo2 />
        </Button>
      </div>

      {items.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-md border border-dashed p-8 text-center">
          <p className="text-sm font-medium">{t("blocksEmptyHeading")}</p>
          <p className="text-sm text-muted-foreground">{t("blocksEmptyBody")}</p>
          <BlockInserter
            blockTypes={blockTypes}
            onInsert={insertBlock}
            triggerLabel={t("blocksEmptyCta")}
            triggerVariant="default"
            triggerSize="default"
          />
        </div>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={items.map((b) => b.key)} strategy={verticalListSortingStrategy}>
            {items.map((block, index) => {
              const blockType = blockTypes.find((bt) => bt.code === block.blockTypeCode);
              return (
                <SortableBlockRow
                  key={block.key}
                  itemKey={block.key}
                  block={block}
                  index={index}
                  total={items.length}
                  blockType={blockType}
                  blockTypes={blockTypes}
                  erroredField={index === lastError.position ? lastError.field : undefined}
                  labels={{
                    moveUp: t("moveBlockUp"),
                    moveDown: t("moveBlockDown"),
                    remove: t("removeBlock"),
                    dragToReorder: (name: string) => t("dragToReorder", { name }),
                  }}
                  onTypeChange={(code) =>
                    setItems((prev) => prev.map((b, i) => (i === index ? { ...b, blockTypeCode: code } : b)))
                  }
                  onMoveUp={() => moveBlock(index, index - 1)}
                  onMoveDown={() => moveBlock(index, index + 1)}
                  onRemove={() => removeBlock(index)}
                />
              );
            })}
          </SortableContext>
        </DndContext>
      )}

      {items.length > 0 && <BlockInserter blockTypes={blockTypes} onInsert={insertBlock} />}

      <div className="flex items-center gap-3">
        <Button type="button" variant="outline" className="self-start" onClick={flush}>
          {t("saveNow")}
        </Button>
        <span role="status" className="text-sm text-muted-foreground">
          {statusLabel(t, status)}
        </span>
      </div>
    </form>
  );
}

interface RowLabels {
  moveUp: string;
  moveDown: string;
  remove: string;
  dragToReorder: (name: string) => string;
}

function SortableBlockRow({
  itemKey,
  block,
  index,
  total,
  blockType,
  blockTypes,
  erroredField,
  labels,
  onTypeChange,
  onMoveUp,
  onMoveDown,
  onRemove,
}: {
  itemKey: string;
  block: ClientBlock;
  index: number;
  total: number;
  blockType: BlockType | undefined;
  blockTypes: BlockType[];
  erroredField?: string;
  labels: RowLabels;
  onTypeChange: (code: string) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onRemove: () => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: itemKey });
  const style = { transform: CSS.Transform.toString(transform), transition };
  const label = blockType?.name ?? block.blockTypeCode;

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex flex-col items-start gap-2 rounded-md border bg-background p-3 sm:grid sm:grid-cols-[auto_auto_10rem_1fr_auto] sm:items-start"
    >
      <input type="hidden" name="position" value={index} readOnly />
      <button
        type="button"
        className="mt-1 flex h-8 w-6 cursor-grab touch-none items-center justify-center text-muted-foreground"
        aria-label={labels.dragToReorder(label)}
        {...attributes}
        {...listeners}
      >
        <GripVertical className="size-4" />
      </button>
      <div className="mt-1 flex flex-col gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={labels.moveUp}
          disabled={index === 0}
          onClick={onMoveUp}
        >
          <ChevronUp />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={labels.moveDown}
          disabled={index === total - 1}
          onClick={onMoveDown}
        >
          <ChevronDown />
        </Button>
      </div>
      <Select name="blockTypeCode" value={block.blockTypeCode} onValueChange={onTypeChange}>
        <SelectTrigger size="sm" className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {blockTypes.map((bt) => (
            <SelectItem key={bt.code} value={bt.code}>
              {bt.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {blockType ? (
        <BlockDataForm blockType={blockType} blockTypes={blockTypes} initialData={block.data} erroredField={erroredField} />
      ) : (
        <Textarea name="data" defaultValue={JSON.stringify(block.data)} rows={3} className="font-mono text-xs" />
      )}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={labels.remove}
        onClick={onRemove}
        className="mt-1 text-destructive"
      >
        <Trash2 />
      </Button>
    </div>
  );
}
