// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useActionState, useEffect } from "react";
import { useTranslations } from "next-intl";

import { useRouter } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { Document } from "@/lib/content";

const NO_PARENT = "__none__";

export type DetailsActionState =
  | { ok: true }
  | { error: "errorSlugTaken"; field: "slug" }
  | { error: "errorGeneric"; raw: string }
  | null;

// M14.8: replaces the old <form action={saveDetails}> + redirect-with-?error= round trip — the last
// one left in the document editor after M14.4/M14.5/M14.6 already moved block-level errors inline —
// with useActionState, the same mechanism people/invite/invite-form.tsx already established in this
// app. Reused here for a different reason than that file's own (a one-time secret that can't survive
// a URL): eliminating the full-page navigation an error redirect forces, and highlighting the
// failing field the same way block-data-form.tsx already does for block data.
export function DocumentDetailsForm({
  action,
  doc,
  otherPages,
}: {
  action: (prevState: DetailsActionState, formData: FormData) => Promise<DetailsActionState>;
  doc: Document;
  otherPages: Document[];
}) {
  const t = useTranslations("DocumentEditorPage");
  const router = useRouter();
  const [state, formAction] = useActionState(action, null);
  const fieldError = state && "field" in state ? state.field : undefined;

  useEffect(() => {
    if (state && "ok" in state) {
      // No navigation to force a re-fetch of doc.slug/parentDocumentId (shown elsewhere on this
      // page) now that saving doesn't redirect — a plain refresh() re-runs the server component.
      router.refresh();
    }
  }, [state, router]);

  return (
    <form action={formAction} className="flex flex-col gap-4">
      <Label className="flex flex-col items-start gap-1">
        {t("slugLabel")}
        <Input name="slug" defaultValue={doc.slug} required pattern="[a-z0-9-]+" aria-invalid={fieldError === "slug"} />
        {fieldError === "slug" && <span className="text-xs text-destructive">{t("errorSlugTaken")}</span>}
      </Label>
      <Label className="flex flex-col items-start gap-1">
        {t("parentPageLabel")}
        <Select name="parentDocumentId" defaultValue={doc.parentDocumentId ?? NO_PARENT}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_PARENT}>{t("parentPageNone")}</SelectItem>
            {otherPages.map((p) => (
              <SelectItem key={p.id} value={p.id}>
                {p.slug} ({p.locale})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Label>
      <Label className="flex flex-col items-start gap-1">
        {t("metaTitleLabel")}
        <Input name="metaTitle" defaultValue={doc.metaTitle ?? ""} />
        <span className="text-xs text-muted-foreground">{t("metaTitleHint")}</span>
      </Label>
      <Label className="flex flex-col items-start gap-1">
        {t("metaDescriptionLabel")}
        <Textarea name="metaDescription" defaultValue={doc.metaDescription ?? ""} rows={2} />
        <span className="text-xs text-muted-foreground">{t("metaDescriptionHint")}</span>
      </Label>
      {state && "error" in state && state.error === "errorGeneric" && (
        <p className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {t("errorGeneric", { error: state.raw })}
        </p>
      )}
      <Button type="submit" className="self-start">
        {t("saveDetails")}
      </Button>
    </form>
  );
}
