// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { NextIntlClientProvider } from "next-intl";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import messages from "@/messages/en.json";
import type { Block, BlockType } from "@/lib/content";

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

const blockTypes: BlockType[] = [
  makeBlockType("heading", "Heading", 10),
  makeBlockType("paragraph", "Paragraph", 20),
  makeBlockType("quote", "Quote", 120),
];

const blocks: Block[] = [makeBlock("b1", "heading", 0), makeBlock("b2", "paragraph", 1), makeBlock("b3", "quote", 2)];

function renderEditor(action = vi.fn()) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <BlockListEditor documentId="doc-1" blocks={blocks} blockTypes={blockTypes} action={action} />
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
});
