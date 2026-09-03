// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useActionState, useState } from "react";
import { useTranslations } from "next-intl";

// M14.16, D-InAppInbox. A client component (not a plain <form action> + redirect like
// site-page.tsx's own report form) so the two anti-spam signals — a hidden honeypot field and a
// render timestamp — can be captured client-side, and so a spam-discarded submission and a real
// one show the identical "thanks" response with no way to tell them apart from the outside.
export type ContactFormActionState = { ok: true } | { ok: false; error: string } | null;

export function ContactFormBlock({
  heading,
  description,
  action,
}: {
  heading?: string;
  description?: string;
  action: (prevState: ContactFormActionState, formData: FormData) => Promise<ContactFormActionState>;
}) {
  const t = useTranslations("ContactFormBlock");
  const [state, formAction, pending] = useActionState(action, null);
  // Captured once, at mount — the server compares this to its own now() to decide whether the
  // submission arrived too fast to be a real visitor (application.Service's
  // minContactFormSubmitDuration). Never re-derived on re-render.
  const [renderedAt] = useState(() => new Date().toISOString());

  if (state?.ok) {
    return (
      <section className="flex flex-col gap-2 border-t pt-8">
        {heading ? <h2 className="text-xl font-semibold">{heading}</h2> : null}
        <p className="text-sm">{t("thanks")}</p>
      </section>
    );
  }

  return (
    <section className="flex flex-col gap-3 border-t pt-8">
      {heading ? <h2 className="text-xl font-semibold">{heading}</h2> : null}
      {description ? <p className="text-sm text-gray-500">{description}</p> : null}
      <form action={formAction} className="flex flex-col gap-2">
        <input type="hidden" name="formRenderedAt" value={renderedAt} />
        {/* Honeypot: visually hidden and unreachable by keyboard/AT, a field name no real visitor
            fills — a simple bot filling every field it finds triggers this. Server-side, a
            non-empty value is silently discarded, never surfaced as an error (D-InAppInbox). */}
        <input type="text" name="website" tabIndex={-1} autoComplete="off" aria-hidden="true" className="sr-only" />
        <input name="name" placeholder={t("namePlaceholder")} className="rounded border px-2 py-1 text-sm" />
        <input name="email" type="email" placeholder={t("emailPlaceholder")} className="rounded border px-2 py-1 text-sm" />
        <textarea name="message" required placeholder={t("messagePlaceholder")} className="rounded border px-2 py-1 text-sm" />
        <button type="submit" disabled={pending} className="self-start rounded border px-3 py-1 text-sm">
          {t("submit")}
        </button>
        {state && !state.ok ? <p className="text-sm text-red-600">{t("error")}</p> : null}
      </form>
    </section>
  );
}
