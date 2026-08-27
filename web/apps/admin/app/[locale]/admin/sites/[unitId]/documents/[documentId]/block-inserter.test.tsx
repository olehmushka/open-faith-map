// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.5: BlockListEditor/BlockInserter are pure, prop-driven client components with no server or
// auth dependency, so they can be exercised entirely offline — sidestepping the fact that this admin
// route has no headless Google OAuth login path for a real browser-level proof (see M14.4's own
// verification notes).
import { NextIntlClientProvider } from "next-intl";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import messages from "@/messages/en.json";
import type { BlockType } from "@/lib/content";

import { BlockInserter } from "./block-inserter";

function makeBlockType(code: string, name: string, sortOrder: number): BlockType {
  return { id: code, code, name, jsonSchema: {}, uiSchema: { fields: [] }, status: "ACTIVE", sortOrder };
}

const blockTypes: BlockType[] = [
  makeBlockType("heading", "Heading", 10),
  makeBlockType("paragraph", "Paragraph", 20),
  makeBlockType("image", "Image", 30),
  makeBlockType("gallery", "Gallery", 40),
];

function renderInserter(onInsert: (blockType: BlockType) => void) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <BlockInserter blockTypes={blockTypes} onInsert={onInsert} />
    </NextIntlClientProvider>,
  );
}

describe("BlockInserter", () => {
  it("is operable by keyboard alone: tab to open, type to filter, enter to select", async () => {
    const user = userEvent.setup();
    const onInsert = vi.fn();
    renderInserter(onInsert);

    await user.tab();
    expect(screen.getByRole("button", { name: "Add block" })).toHaveFocus();
    await user.keyboard("{Enter}");

    const input = await screen.findByPlaceholderText("Search block types…");
    await user.type(input, "Heading");
    await user.keyboard("{Enter}");

    expect(onInsert).toHaveBeenCalledTimes(1);
    expect(onInsert.mock.calls[0][0]).toMatchObject({ code: "heading" });
  });

  it("groups block types under category headings", async () => {
    const user = userEvent.setup();
    renderInserter(vi.fn());

    await user.click(screen.getByRole("button", { name: "Add block" }));

    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Text")).toBeInTheDocument();
    expect(within(dialog).getByText("Media")).toBeInTheDocument();
  });

  it("closes on Escape without inserting anything", async () => {
    const user = userEvent.setup();
    const onInsert = vi.fn();
    renderInserter(onInsert);

    await user.click(screen.getByRole("button", { name: "Add block" }));
    await screen.findByPlaceholderText("Search block types…");

    await user.keyboard("{Escape}");

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(onInsert).not.toHaveBeenCalled();
  });

  it("filters by description text, not just the block type name", async () => {
    const user = userEvent.setup();
    const onInsert = vi.fn();
    renderInserter(onInsert);

    await user.click(screen.getByRole("button", { name: "Add block" }));
    const input = await screen.findByPlaceholderText("Search block types…");
    await user.type(input, "photo");

    expect(screen.getByText("Image")).toBeInTheDocument();
    expect(screen.getByText("Gallery")).toBeInTheDocument();
    expect(screen.queryByText("Heading")).not.toBeInTheDocument();
  });
});
