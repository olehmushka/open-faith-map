// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.5: replaces the server-rendered, uncontrolled-form block list (manually-typed `position`
// numbers, one hardcoded "new block" row, no remove control) with a client-managed array supporting
// drag-and-drop reorder, first-class keyboard move-up/move-down, a categorized inserter
// (block-inserter.tsx), and per-block removal. The outer <form action={action}> and its
// position/blockTypeCode/data field names are unchanged, so the existing saveBlocks server action in
// page.tsx (formData.getAll("position"|"blockTypeCode"|"data")) needs no changes at all.
"use client";

import { useEffect, useState, type FormEvent } from "react";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { Block, BlockType } from "@/lib/content";

import { BlockDataForm } from "./block-data-form";
import { BlockInserter } from "./block-inserter";

interface ClientBlock {
  /** Client-only identity for React keys and drag-and-drop — never submitted; BlockInput has no id. */
  key: string;
  blockTypeCode: string;
  data: unknown;
}

function snapshotKey(documentId: string): string {
  return `ofm:blocks:${documentId}`;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;
}

/**
 * A failed save's `position` parameter indexes into the exact array that was just POSTed. A plain
 * server-action redirect re-fetches getBlocks(documentId), which — since the failed PUT changed
 * nothing — returns the last successfully saved list, not what the user had just reordered/edited in
 * the browser. Restoring from a pre-submit sessionStorage snapshot (written in handleSubmit below)
 * keeps `erroredPosition` pointing at the right row, and incidentally stops a reorder-in-progress
 * from being silently discarded on a failed save.
 */
function restoreFromSnapshot(documentId: string): ClientBlock[] | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.sessionStorage.getItem(snapshotKey(documentId));
    if (!raw) return null;
    const restored = JSON.parse(raw) as { blockTypeCode: string; data: unknown }[];
    return restored.map((b) => ({ key: newKey(), blockTypeCode: b.blockTypeCode, data: b.data }));
  } catch {
    return null;
  }
}

export function BlockListEditor({
  documentId,
  blocks,
  blockTypes,
  erroredPosition,
  erroredField,
  action,
}: {
  documentId: string;
  blocks: Block[];
  blockTypes: BlockType[];
  erroredPosition?: number;
  erroredField?: string;
  action: (formData: FormData) => void;
}) {
  const t = useTranslations("DocumentEditorPage");
  const [items, setItems] = useState<ClientBlock[]>(() => {
    const restored = erroredField !== undefined ? restoreFromSnapshot(documentId) : null;
    return restored ?? blocks.map((b) => ({ key: b.id, blockTypeCode: b.blockTypeCode, data: b.data }));
  });
  const [liveMessage, setLiveMessage] = useState("");

  // Consume the snapshot once, whether or not it was used, so a stale entry never contaminates a
  // later, unrelated error.
  useEffect(() => {
    if (erroredField === undefined) return;
    try {
      window.sessionStorage.removeItem(snapshotKey(documentId));
    } catch {
      // sessionStorage unavailable — nothing to clean up.
    }
    // Only ever meant to run once, against the load that consumed the snapshot.
  }, []);

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

  // Snapshots the exact position/blockTypeCode/data triplet about to be submitted — read from
  // FormData rather than `items` state, since each block's `data` is owned by its own BlockDataForm
  // instance, not lifted up here.
  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    try {
      const formData = new FormData(event.currentTarget);
      const blockTypeCodes = formData.getAll("blockTypeCode").map(String);
      const dataJson = formData.getAll("data").map(String);
      const snapshot = blockTypeCodes.map((blockTypeCode, i) => ({
        blockTypeCode,
        data: JSON.parse(dataJson[i] || "{}"),
      }));
      window.sessionStorage.setItem(snapshotKey(documentId), JSON.stringify(snapshot));
    } catch {
      // sessionStorage unavailable — the redirect-recovery enhancement is best-effort only.
    }
  }

  return (
    <form action={action} onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div aria-live="polite" className="sr-only">
        {liveMessage}
      </div>
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
                erroredField={index === erroredPosition ? erroredField : undefined}
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

      <BlockInserter blockTypes={blockTypes} onInsert={insertBlock} />

      <Button type="submit" className="self-start">
        {t("saveBlocks")}
      </Button>
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
      className="grid grid-cols-[auto_auto_10rem_1fr_auto] items-start gap-2 rounded-md border bg-background p-3"
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
