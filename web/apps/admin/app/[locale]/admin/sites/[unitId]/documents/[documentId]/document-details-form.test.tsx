// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { NextIntlClientProvider } from "next-intl";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import messages from "@/messages/en.json";
import type { Document } from "@/lib/content";

import { DocumentDetailsForm, type DetailsActionState } from "./document-details-form";

const refresh = vi.fn();
vi.mock("@/i18n/navigation", () => ({
  useRouter: () => ({ refresh }),
}));

function makeDoc(overrides: Partial<Document> = {}): Document {
  return {
    id: "doc-1",
    siteId: "site-1",
    kind: "PAGE",
    state: "DRAFT",
    locale: "eng",
    slug: "home",
    parentDocumentId: null,
    translationGroupId: "tg-1",
    createdAt: "2026-08-27T00:00:00Z",
    updatedAt: "2026-08-27T00:00:00Z",
    ...overrides,
  };
}

function renderForm(action: (prevState: DetailsActionState, formData: FormData) => Promise<DetailsActionState>) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <DocumentDetailsForm action={action} doc={makeDoc()} otherPages={[]} />
    </NextIntlClientProvider>,
  );
}

describe("DocumentDetailsForm", () => {
  it("never puts an error in the URL: a slug-taken failure renders inline and doesn't navigate", async () => {
    const user = userEvent.setup();
    const action = vi.fn().mockResolvedValue({ error: "errorSlugTaken", field: "slug" } satisfies DetailsActionState);
    renderForm(action);

    // Captured before the error appears: once it does, the destructive helper text renders inside
    // the same <Label> (matching block-data-form.tsx's own ScalarField convention), which changes
    // the label's full text content — getByLabelText("Slug") would no longer match by then.
    const slugInput = screen.getByLabelText("Slug");
    await user.clear(slugInput);
    await user.type(slugInput, "taken-slug");
    await user.click(screen.getByRole("button", { name: "Save details" }));

    expect(await screen.findByText("That slug is already taken for this locale.")).toBeInTheDocument();
    expect(slugInput).toHaveAttribute("aria-invalid", "true");
    expect(refresh).not.toHaveBeenCalled();
  });

  it("refreshes (not navigates) on a successful save", async () => {
    const user = userEvent.setup();
    const action = vi.fn().mockResolvedValue({ ok: true } satisfies DetailsActionState);
    renderForm(action);

    await user.click(screen.getByRole("button", { name: "Save details" }));

    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
    expect(screen.queryByText("That slug is already taken for this locale.")).not.toBeInTheDocument();
  });

  it("shows the generic error banner for a non-slug failure", async () => {
    const user = userEvent.setup();
    const action = vi.fn().mockResolvedValue({ error: "errorGeneric", raw: "Content:Unknown" } satisfies DetailsActionState);
    renderForm(action);

    await user.click(screen.getByRole("button", { name: "Save details" }));

    expect(await screen.findByText("Something went wrong: Content:Unknown")).toBeInTheDocument();
    expect(screen.getByLabelText("Slug")).not.toHaveAttribute("aria-invalid", "true");
  });
});
