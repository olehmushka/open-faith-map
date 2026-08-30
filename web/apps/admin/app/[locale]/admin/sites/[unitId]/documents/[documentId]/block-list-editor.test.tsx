// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { NextIntlClientProvider } from "next-intl";
import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import messages from "@/messages/en.json";
import type { Block, BlockType, Pattern } from "@/lib/content";

import { BlockListEditor } from "./block-list-editor";

function makeBlockType(code: string, name: string, sortOrder: number): BlockType {
  return { id: code, code, name, jsonSchema: {}, uiSchema: { fields: [] }, status: "ACTIVE", sortOrder };
}

function makeBlock(id: string, blockTypeCode: string, position: number): Block {
  return {
    id,
    documentId: "doc-1",
    blockTypeCode,
    position,
    data: {},
    createdAt: "2026-08-27T00:00:00Z",
    updatedAt: "2026-08-27T00:00:00Z",
  };
}

function makePattern(id: string, name: string, description: string, blockTypeCodes: string[]): Pattern {
  return {
    id,
    name,
    description,
    blocks: blockTypeCodes.map((blockTypeCode, position) => ({ blockTypeCode, position, data: {} })),
    sortOrder: 0,
    createdAt: "2026-08-27T00:00:00Z",
    updatedAt: "2026-08-27T00:00:00Z",
  };
}

const blockTypes: BlockType[] = [
  makeBlockType("heading", "Heading", 10),
  makeBlockType("paragraph", "Paragraph", 20),
  makeBlockType("quote", "Quote", 120),
];

const blocks: Block[] = [makeBlock("b1", "heading", 0), makeBlock("b2", "paragraph", 1), makeBlock("b3", "quote", 2)];

function renderEditor(onAutosave = vi.fn().mockResolvedValue({ ok: true }), patterns: Pattern[] = []) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <BlockListEditor blocks={blocks} blockTypes={blockTypes} patterns={patterns} onAutosave={onAutosave} />
    </NextIntlClientProvider>,
  );
}

function dragHandleNames(): (string | null)[] {
  return screen.getAllByRole("button", { name: /Drag to reorder/ }).map((el) => el.getAttribute("aria-label"));
}

describe("BlockListEditor", () => {
  it("never renders a numeric position control", () => {
    const { container } = renderEditor();
    expect(container.querySelector('input[type="number"]')).toBeNull();
    expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
    // the only "position" input is the hidden, derived one
    const positionInputs = container.querySelectorAll('input[name="position"]');
    expect(positionInputs.length).toBe(blocks.length);
    positionInputs.forEach((el) => expect(el).toHaveAttribute("type", "hidden"));
  });

  it("reorders blocks by keyboard alone via the move-down button, and announces it", async () => {
    const user = userEvent.setup();
    renderEditor();

    expect(dragHandleNames()).toEqual(["Drag to reorder Heading", "Drag to reorder Paragraph", "Drag to reorder Quote"]);

    const moveDownButtons = screen.getAllByRole("button", { name: "Move block down" });
    moveDownButtons[0].focus();
    await user.keyboard("{Enter}");

    expect(dragHandleNames()).toEqual(["Drag to reorder Paragraph", "Drag to reorder Heading", "Drag to reorder Quote"]);
    expect(screen.getByText(/Heading moved to position 2 of 3/)).toBeInTheDocument();
  });

  it("disables move-up on the first row and move-down on the last row", () => {
    renderEditor();
    const moveUpButtons = screen.getAllByRole("button", { name: "Move block up" });
    const moveDownButtons = screen.getAllByRole("button", { name: "Move block down" });

    expect(moveUpButtons[0]).toBeDisabled();
    expect(moveDownButtons[moveDownButtons.length - 1]).toBeDisabled();
    expect(moveDownButtons[0]).not.toBeDisabled();
  });

  it("removes a block via keyboard alone, and announces it", async () => {
    const user = userEvent.setup();
    renderEditor();

    const removeButtons = screen.getAllByRole("button", { name: "Remove block" });
    removeButtons[1].focus();
    await user.keyboard("{Enter}");

    expect(dragHandleNames()).toEqual(["Drag to reorder Heading", "Drag to reorder Quote"]);
    expect(screen.getByText(/Paragraph removed/)).toBeInTheDocument();
  });

  it("appends an inserted block to the end of the list", async () => {
    const user = userEvent.setup();
    renderEditor();

    await user.click(screen.getByRole("button", { name: "Add block" }));
    const input = await screen.findByPlaceholderText("Search block types…");
    await user.type(input, "Quote");
    await user.keyboard("{Enter}");

    expect(dragHandleNames()).toEqual([
      "Drag to reorder Heading",
      "Drag to reorder Paragraph",
      "Drag to reorder Quote",
      "Drag to reorder Quote",
    ]);
  });

  // M14.13, D-SitePatterns: inserting a pattern appends ALL of its blocks in one action, unsynced —
  // there is no per-block insert call and no reference back to the pattern afterward.
  it("appends every block of an inserted pattern to the end of the list", async () => {
    const user = userEvent.setup();
    const pattern = makePattern("p1", "Feast-day announcement", "A short announcement", ["heading", "paragraph"]);
    renderEditor(vi.fn().mockResolvedValue({ ok: true }), [pattern]);

    await user.click(screen.getByRole("button", { name: "Insert pattern" }));
    const input = await screen.findByPlaceholderText("Search patterns…");
    await user.type(input, "Feast");
    await user.keyboard("{Enter}");

    expect(dragHandleNames()).toEqual([
      "Drag to reorder Heading",
      "Drag to reorder Paragraph",
      "Drag to reorder Quote",
      "Drag to reorder Heading",
      "Drag to reorder Paragraph",
    ]);
    expect(screen.getByText(/Feast-day announcement pattern inserted/)).toBeInTheDocument();
  });

  it("stacks to a single column below sm: and reverts to the fixed-column grid at sm: and above", () => {
    const { container } = renderEditor();
    const row = container.querySelector(".rounded-md.border.bg-background.p-3");
    expect(row?.className).toContain("flex-col");
    expect(row?.className).toContain("sm:grid-cols-");
  });
});

// M14.8: session-local undo/redo (hooks/use-block-history.ts) settles on a poll/settle window
// (500ms/600ms by default), so these run under fake timers — real timers would make every assertion
// wait over a second. `fireEvent` (not `userEvent`) drives the interactions here: userEvent's own
// internal pointer-event delays don't resolve deterministically under Vitest's fake timers, but
// these are plain click/change/keydown events with no timing behavior of their own to exercise.
describe("BlockListEditor undo/redo", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // Must clear at least one full poll tick (500ms) *after* the 600ms settle window closes, so the
  // earliest tick that can observe "already settled" lands at 500 + 600 = 1100ms — round up to the
  // next 500ms tick boundary (1500ms) plus margin, rather than the naive (and too-tight) 1100ms.
  async function settle() {
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
  }

  it("disables Undo and Redo when there is nothing to undo or redo", async () => {
    renderEditor();
    await settle();

    expect(screen.getByRole("button", { name: "Undo" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Redo" })).toBeDisabled();
  });

  it("undo restores the list after a block is removed, and announces it", async () => {
    renderEditor();
    await settle();

    const removeButtons = screen.getAllByRole("button", { name: "Remove block" });
    fireEvent.click(removeButtons[1]);
    await settle();
    expect(dragHandleNames()).toEqual(["Drag to reorder Heading", "Drag to reorder Quote"]);

    fireEvent.click(screen.getByRole("button", { name: "Undo" }));

    expect(dragHandleNames()).toEqual([
      "Drag to reorder Heading",
      "Drag to reorder Paragraph",
      "Drag to reorder Quote",
    ]);
    expect(screen.getByText("Change undone")).toBeInTheDocument();
  });

  it("redo re-applies an undone change, and announces it", async () => {
    renderEditor();
    await settle();

    const removeButtons = screen.getAllByRole("button", { name: "Remove block" });
    fireEvent.click(removeButtons[1]);
    await settle();
    fireEvent.click(screen.getByRole("button", { name: "Undo" }));

    fireEvent.click(screen.getByRole("button", { name: "Redo" }));

    expect(dragHandleNames()).toEqual(["Drag to reorder Heading", "Drag to reorder Quote"]);
    expect(screen.getByText("Change redone")).toBeInTheDocument();
  });

  it("undoes an edit to a block's field data, not just list-shape changes", async () => {
    const editableTypes: BlockType[] = [
      {
        id: "paragraph",
        code: "paragraph",
        name: "Paragraph",
        jsonSchema: {},
        uiSchema: { fields: [{ name: "caption", widget: "text", label: "Caption" }] },
        status: "ACTIVE",
        sortOrder: 10,
      },
    ];
    const editableBlocks: Block[] = [makeBlock("b1", "paragraph", 0)];
    render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <BlockListEditor blocks={editableBlocks} blockTypes={editableTypes} patterns={[]} onAutosave={vi.fn().mockResolvedValue({ ok: true })} />
      </NextIntlClientProvider>,
    );
    await settle();

    const captionInput = screen.getByLabelText("Caption");
    fireEvent.change(captionInput, { target: { value: "Hello" } });
    await settle();
    expect(screen.getByLabelText("Caption")).toHaveValue("Hello");

    fireEvent.click(screen.getByRole("button", { name: "Undo" }));
    expect(screen.getByLabelText("Caption")).toHaveValue("");
  });

  it("supports Ctrl+Z and Ctrl+Shift+Z", async () => {
    renderEditor();
    await settle();

    const removeButtons = screen.getAllByRole("button", { name: "Remove block" });
    fireEvent.click(removeButtons[1]);
    await settle();

    fireEvent.keyDown(document, { key: "z", ctrlKey: true });
    expect(dragHandleNames()).toEqual([
      "Drag to reorder Heading",
      "Drag to reorder Paragraph",
      "Drag to reorder Quote",
    ]);

    fireEvent.keyDown(document, { key: "z", ctrlKey: true, shiftKey: true });
    expect(dragHandleNames()).toEqual(["Drag to reorder Heading", "Drag to reorder Quote"]);
  });

  it("does not hijack Ctrl+Z when it targets a text field, leaving native undo alone", async () => {
    const { container } = renderEditor();
    await settle();

    const removeButtons = screen.getAllByRole("button", { name: "Remove block" });
    fireEvent.click(removeButtons[1]);
    await settle();
    const afterRemoval = dragHandleNames();

    // Dispatched directly on an <input> so the event's `target` is that field, regardless of focus/
    // visibility rules in jsdom — exercising the same tagName check the real handler applies.
    const input = container.querySelector('input[name="position"]');
    expect(input).not.toBeNull();
    fireEvent.keyDown(input!, { key: "z", ctrlKey: true });

    expect(dragHandleNames()).toEqual(afterRemoval);
  });
});

describe("BlockListEditor empty state", () => {
  it("shows a CTA instead of an empty form when there are no blocks", () => {
    render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <BlockListEditor blocks={[]} blockTypes={blockTypes} patterns={[]} onAutosave={vi.fn().mockResolvedValue({ ok: true })} />
      </NextIntlClientProvider>,
    );

    expect(screen.getByText("This page has no content yet.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add your first block" })).toBeInTheDocument();
  });

  it("keeps the undo/redo toolbar visible after deleting the last remaining block", async () => {
    const user = userEvent.setup();
    render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <BlockListEditor blocks={[blocks[0]]} blockTypes={blockTypes} patterns={[]} onAutosave={vi.fn().mockResolvedValue({ ok: true })} />
      </NextIntlClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Remove block" }));

    expect(screen.getByText("This page has no content yet.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Undo" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Redo" })).toBeInTheDocument();
  });

  it("does not render the empty state once a block exists", () => {
    renderEditor();
    expect(screen.queryByText("This page has no content yet.")).not.toBeInTheDocument();
  });
});
