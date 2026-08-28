// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { NextIntlClientProvider } from "next-intl";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import messages from "@/messages/en.json";
import type { Document } from "@/lib/content";

import { NewDocumentForm, type CreateActionState } from "./new-document-form";

function renderForm(action: (prevState: CreateActionState, formData: FormData) => Promise<CreateActionState>) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <NewDocumentForm action={action} existingPages={[] as Document[]} />
    </NextIntlClientProvider>,
  );
}

// Fills the two required fields (locale, slug) so the browser's own required-field constraint
// validation never blocks the click-to-submit below — these tests are about the returned error
// state, not native form validation. Returns the slug input reference, captured before any error
// text can render inside its <Label> and change the label's matchable text content.
async function fillRequired(user: ReturnType<typeof userEvent.setup>, slug: string) {
  await user.type(screen.getByLabelText("Locale"), "eng");
  const slugInput = screen.getByLabelText("Slug");
  await user.type(slugInput, slug);
  return slugInput;
}

describe("NewDocumentForm", () => {
  it("renders a slug-taken failure inline on the slug field, with no query string involved", async () => {
    const user = userEvent.setup();
    const action = vi.fn().mockResolvedValue({ error: "errorSlugTaken", field: "slug" } satisfies CreateActionState);
    renderForm(action);

    const slugInput = await fillRequired(user, "home");
    await user.click(screen.getByRole("button", { name: "Create document" }));

    expect(await screen.findByText("That slug is already taken for this locale.")).toBeInTheDocument();
    expect(slugInput).toHaveAttribute("aria-invalid", "true");
  });

  it("renders a missing-start-date failure inline on the event start field", async () => {
    const user = userEvent.setup();
    const action = vi
      .fn()
      .mockResolvedValue({ error: "errorEventMissingStart", field: "eventStartsAt" } satisfies CreateActionState);
    renderForm(action);

    const slugInput = await fillRequired(user, "sunday-service");
    const eventStartsAtInput = screen.getByLabelText("Starts at (required for Event)");
    await user.click(screen.getByRole("button", { name: "Create document" }));

    expect(await screen.findByText("Events require a start date/time.")).toBeInTheDocument();
    expect(eventStartsAtInput).toHaveAttribute("aria-invalid", "true");
    // Not the slug field — the two errors don't cross-contaminate.
    expect(slugInput).not.toHaveAttribute("aria-invalid", "true");
  });

  it("shows the generic error banner for any other failure", async () => {
    const user = userEvent.setup();
    const action = vi.fn().mockResolvedValue({ error: "errorGeneric", raw: "Content:Unknown" } satisfies CreateActionState);
    renderForm(action);

    await fillRequired(user, "about");
    await user.click(screen.getByRole("button", { name: "Create document" }));

    expect(await screen.findByText("Something went wrong: Content:Unknown")).toBeInTheDocument();
  });

  it("calls the action on submit (the success path itself redirects from inside the server action)", async () => {
    const user = userEvent.setup();
    const action = vi.fn().mockResolvedValue(null satisfies CreateActionState);
    renderForm(action);

    await fillRequired(user, "about");
    await user.click(screen.getByRole("button", { name: "Create document" }));

    expect(action).toHaveBeenCalledTimes(1);
  });
});
