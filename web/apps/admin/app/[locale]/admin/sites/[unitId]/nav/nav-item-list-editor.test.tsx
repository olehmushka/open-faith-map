// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { NextIntlClientProvider } from "next-intl";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import messages from "@/messages/en.json";
import type { Document, NavItem, NavItemInput } from "@/lib/content";

import { NavItemListEditor, type NavSaveResult } from "./nav-item-list-editor";

function makePage(id: string, slug: string): Document {
  return {
    id,
    siteId: "site-1",
    kind: "PAGE",
    translationGroupId: id,
    locale: "en",
    slug,
    state: "PUBLISHED",
    effectiveState: "PUBLISHED",
    createdAt: "2026-08-27T00:00:00Z",
    updatedAt: "2026-08-27T00:00:00Z",
  };
}

const pages: Document[] = [makePage("p1", "about"), makePage("p2", "contact")];

const navItems: NavItem[] = [
  { id: "n1", siteId: "site-1", label: "About", targetDocumentId: "p1", sortOrder: 0 },
  { id: "n2", siteId: "site-1", label: "Our Friends", targetUrl: "https://example.org", sortOrder: 1 },
];

function renderEditor(onSave = vi.fn().mockResolvedValue({ ok: true } satisfies NavSaveResult)) {
  return { onSave, ...render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <NavItemListEditor items={navItems} pages={pages} onSave={onSave} />
    </NextIntlClientProvider>,
  ) };
}

function dragHandleNames(): (string | null)[] {
  return screen.getAllByRole("button", { name: /Drag to reorder/ }).map((el) => el.getAttribute("aria-label"));
}

describe("NavItemListEditor", () => {
  it("reorders items by keyboard alone via the move-down button, and announces it", async () => {
    const user = userEvent.setup();
    renderEditor();

    expect(dragHandleNames()).toEqual(["Drag to reorder About", "Drag to reorder Our Friends"]);

    const moveDownButtons = screen.getAllByRole("button", { name: "Move item down" });
    moveDownButtons[0].focus();
    await user.keyboard("{Enter}");

    expect(dragHandleNames()).toEqual(["Drag to reorder Our Friends", "Drag to reorder About"]);
    expect(screen.getByText(/About moved to position 2 of 2/)).toBeInTheDocument();
  });

  it("disables move-up on the first row and move-down on the last row", () => {
    renderEditor();
    const moveUpButtons = screen.getAllByRole("button", { name: "Move item up" });
    const moveDownButtons = screen.getAllByRole("button", { name: "Move item down" });

    expect(moveUpButtons[0]).toBeDisabled();
    expect(moveDownButtons[moveDownButtons.length - 1]).toBeDisabled();
    expect(moveDownButtons[0]).not.toBeDisabled();
  });

  it("removes an item via keyboard alone, and announces it", async () => {
    const user = userEvent.setup();
    renderEditor();

    const removeButtons = screen.getAllByRole("button", { name: "Remove item" });
    removeButtons[0].focus();
    await user.keyboard("{Enter}");

    expect(dragHandleNames()).toEqual(["Drag to reorder Our Friends"]);
    expect(screen.getByText(/About removed/)).toBeInTheDocument();
  });

  it("appends a new blank row defaulting to Page mode", async () => {
    const user = userEvent.setup();
    renderEditor();

    await user.click(screen.getByRole("button", { name: "Add item" }));

    expect(dragHandleNames()).toEqual(["Drag to reorder About", "Drag to reorder Our Friends", "Drag to reorder Untitled item"]);
    // A fresh row defaults to "page" mode — its external-URL input must not be present.
    const urlInputs = screen.getAllByPlaceholderText("https://…");
    expect(urlInputs).toHaveLength(1); // only the pre-existing "Our Friends" external row
  });

  it("switching a row's mode clears the other field's visible control", async () => {
    const user = userEvent.setup();
    renderEditor();

    // The second row ("Our Friends") starts in External mode — switch it to Page.
    const pageButtons = screen.getAllByRole("button", { name: "Page" });
    await user.click(pageButtons[1]);

    expect(screen.queryByPlaceholderText("https://…")).not.toBeInTheDocument();
    // Row 2 had no targetDocumentId (it was previously External), so its Select now shows the
    // placeholder; row 1 ("About") already had one selected, so only one placeholder shows.
    expect(screen.getAllByText("Choose a page")).toHaveLength(1);
  });

  it("Save submits the current rows as NavItemInput[] with sequential sortOrder", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue({ ok: true } satisfies NavSaveResult);
    renderEditor(onSave);

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledTimes(1);
    const submitted = onSave.mock.calls[0][0] as NavItemInput[];
    expect(submitted).toEqual([
      { label: "About", targetDocumentId: "p1", targetUrl: undefined, sortOrder: 0 },
      { label: "Our Friends", targetDocumentId: undefined, targetUrl: "https://example.org", sortOrder: 1 },
    ]);
    expect(await screen.findByText("Saved")).toBeInTheDocument();
  });
});
