// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useActionState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export type ScheduleActionState =
  | { error: "errorScheduleMissingPublishAt"; field: "publishAt" }
  | { error: "errorSchedulePublishAtNotFuture"; field: "publishAt" }
  | { error: "errorGeneric"; raw: string }
  | null;

// M14.15: a new one-shot action alongside the plain <form action={...}> Publish/Unlist/Back-to-draft
// buttons in page.tsx, but — unlike those — needs inline validation feedback for a bad date, so it
// follows document-details-form.tsx's useActionState convention instead of a bare server-action form.
export function ScheduleForm({
  action,
  disabled,
}: {
  action: (prevState: ScheduleActionState, formData: FormData) => Promise<ScheduleActionState>;
  disabled: boolean;
}) {
  const t = useTranslations("DocumentEditorPage");
  const [state, formAction] = useActionState(action, null);
  const fieldError = state && "field" in state ? state.field : undefined;

  return (
    <form action={formAction} className="flex flex-wrap items-end gap-2">
      <Label className="flex flex-col items-start gap-1 text-xs">
        {t("schedulePublishAtLabel")}
        <Input
          type="datetime-local"
          name="publishAt"
          disabled={disabled}
          aria-invalid={fieldError === "publishAt"}
          className="h-8 w-auto"
        />
      </Label>
      <Button type="submit" variant="outline" size="sm" disabled={disabled}>
        {t("schedule")}
      </Button>
      {state && "error" in state && (
        <span className="text-xs text-destructive">
          {state.error === "errorGeneric" ? t("errorGeneric", { error: state.raw }) : t(state.error)}
        </span>
      )}
    </form>
  );
}
