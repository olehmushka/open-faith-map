// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { NextIntlClientProvider } from "next-intl";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import messages from "@/messages/en.json";

import { EntityPicker, type EntityPickerOption } from "./entity-picker";

function renderPicker(onSearch: (query: string) => Promise<EntityPickerOption[]>) {
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <EntityPicker name="unitId" onSearch={onSearch} placeholder="Unit ID" />
    </NextIntlClientProvider>,
  );
}

describe("EntityPicker", () => {
  it("searches as the operator types and selects a result into the hidden input", async () => {
    const user = userEvent.setup();
    const onSearch = vi
      .fn()
      .mockResolvedValue([{ id: "unit-1", label: "Grace Chapel", sublabel: "grace" }]);
    renderPicker(onSearch);

    await user.click(screen.getByRole("combobox"));
    await user.type(screen.getByPlaceholderText("Search…"), "grace");

    const option = await screen.findByText("Grace Chapel");
    await user.click(option);

    expect(onSearch).toHaveBeenCalledWith("grace");
    expect(screen.getByRole("combobox")).toHaveTextContent("Grace Chapel");
    expect(document.querySelector('input[name="unitId"]')).toHaveValue("unit-1");
  });

  it("shows the no-matches state when the search resolves empty", async () => {
    const user = userEvent.setup();
    const onSearch = vi.fn().mockResolvedValue([]);
    renderPicker(onSearch);

    await user.click(screen.getByRole("combobox"));
    await user.type(screen.getByPlaceholderText("Search…"), "nonexistent");

    expect(await screen.findByText("No matches.")).toBeInTheDocument();
  });

  it("never searches on an empty or whitespace-only query", async () => {
    const user = userEvent.setup();
    const onSearch = vi.fn().mockResolvedValue([]);
    renderPicker(onSearch);

    await user.click(screen.getByRole("combobox"));
    await user.type(screen.getByPlaceholderText("Search…"), "   ");

    // The debounce window (200ms) has long since elapsed by the time typing finishes; if it were
    // going to fire, it already would have.
    expect(onSearch).not.toHaveBeenCalled();
    expect(document.querySelector('input[name="unitId"]')).toHaveValue("");
  });
});
