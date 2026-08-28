// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useActionState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DocumentKind } from "@/lib/openfaithmap/generated/content";
import type { Document } from "@/lib/content";

const NO_PARENT = "__none__";

export type CreateActionState =
  | { error: "errorSlugTaken"; field: "slug" }
  | { error: "errorEventMissingStart"; field: "eventStartsAt" }
  | { error: "errorGeneric"; raw: string }
  | null;

// M14.8: replaces the old <form action={create}> + redirect-with-?error= round trip with
// useActionState — see document-details-form.tsx's own comment for why this mechanism, reused from
// people/invite/invite-form.tsx, for yet another reason than that file's own. The success path still
// calls redirect() from inside the "use server" action itself, which keeps working identically under
// useActionState — only the failure path's shape changes here.
export function NewDocumentForm({
  action,
  existingPages,
}: {
  action: (prevState: CreateActionState, formData: FormData) => Promise<CreateActionState>;
  existingPages: Document[];
}) {
  const t = useTranslations("NewDocumentPage");
  const [state, formAction] = useActionState(action, null);
  const fieldError = state && "field" in state ? state.field : undefined;

  return (
    <form action={formAction} className="flex flex-col gap-4">
      <Label className="flex flex-col items-start gap-1">
        {t("kindLabel")}
        <Select name="kind" defaultValue={DocumentKind.PAGE}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={DocumentKind.PAGE}>{t("kindPage")}</SelectItem>
            <SelectItem value={DocumentKind.POST}>{t("kindPost")}</SelectItem>
            <SelectItem value={DocumentKind.EVENT}>{t("kindEvent")}</SelectItem>
          </SelectContent>
        </Select>
      </Label>

      <Label className="flex flex-col items-start gap-1">
        {t("localeLabel")}
        <Input name="locale" required placeholder="eng" />
      </Label>

      <Label className="flex flex-col items-start gap-1">
        {t("slugLabel")}
        <Input name="slug" required pattern="[a-z0-9-]+" aria-invalid={fieldError === "slug"} />
        {fieldError === "slug" && <span className="text-xs text-destructive">{t("errorSlugTaken")}</span>}
      </Label>

      <Label className="flex flex-col items-start gap-1">
        {t("parentPageLabel")}
        <Select name="parentDocumentId" defaultValue={NO_PARENT}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_PARENT}>{t("parentPageNone")}</SelectItem>
            {existingPages.map((p) => (
              <SelectItem key={p.id} value={p.id}>
                {p.slug} ({p.locale})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Label>

      <fieldset className="flex flex-col gap-4 rounded-md border p-3">
        <legend className="px-1 text-sm font-medium">{t("eventFieldsLegend")}</legend>
        <Label className="flex flex-col items-start gap-1">
          {t("eventStartsAtLabel")}
          <Input type="datetime-local" name="eventStartsAt" aria-invalid={fieldError === "eventStartsAt"} />
          {fieldError === "eventStartsAt" && (
            <span className="text-xs text-destructive">{t("errorEventMissingStart")}</span>
          )}
        </Label>
        <Label className="flex flex-col items-start gap-1">
          {t("eventEndsAtLabel")}
          <Input type="datetime-local" name="eventEndsAt" />
        </Label>
        <Label className="flex flex-col items-start gap-1">
          {t("eventRecurrenceLabel")}
          <Input name="eventRecurrenceRrule" placeholder="FREQ=WEEKLY;BYDAY=SU" />
        </Label>
      </fieldset>

      <Label className="flex flex-col items-start gap-1">
        {t("translationGroupLabel")}
        <Input name="translationGroupId" placeholder={t("translationGroupPlaceholder")} />
      </Label>

      {state && "error" in state && state.error === "errorGeneric" && (
        <p className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {t("errorGeneric", { error: state.raw })}
        </p>
      )}

      <Button type="submit" className="self-start">
        {t("submit")}
      </Button>
    </form>
  );
}
