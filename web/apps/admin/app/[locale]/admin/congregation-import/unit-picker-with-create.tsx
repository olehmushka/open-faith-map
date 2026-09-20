// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useRef, useState, useTransition } from "react";
import { useTranslations } from "next-intl";
import { Plus } from "lucide-react";

import { searchUnitsForPicker } from "@/lib/entity-search";
import {
  EntityPicker,
  type EntityPickerHandle,
  type EntityPickerOption,
} from "@/components/entity-picker";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

// slugify: no such helper exists anywhere in web/apps/admin yet (checked) — a small, local one for
// the create-unit dialog's Code field default, editable before submit either way.
function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "") // combining diacritical marks left behind by NFKD
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64);
}

/**
 * A unit EntityPicker plus an inline "create a missing unit" dialog — the one EntityPicker
 * consumer needing a create sub-flow (M17 generalizes jurisdiction-field.tsx's JurisdictionField
 * into this, for congregation-import's approve form, plus the plain EntityPicker everywhere else).
 * The dialog's own parent-unit field is a second, nested EntityPicker, per M17's own target list.
 *
 * Uses EntityPicker's imperative handle to set the selection once onCreateUnit resolves — a
 * Dialog's own submit and the picker's selection are otherwise unrelated pieces of client state.
 */
export function UnitPickerWithCreate({
  idSuffix,
  name,
  defaultValue,
  placeholder,
  createDefaultName,
  rootUnitId,
  onCreateUnit,
}: {
  idSuffix: string;
  name: string;
  defaultValue?: EntityPickerOption | null;
  placeholder: string;
  createDefaultName: string;
  rootUnitId: string;
  onCreateUnit: (
    parentUnitId: string,
    code: string,
    name: string,
  ) => Promise<{ id: string; code: string | null; name: string }>;
}) {
  const t = useTranslations("CongregationImportPage");
  const pickerRef = useRef<EntityPickerHandle>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [pending, startTransition] = useTransition();
  const nameInputId = `unit-name-${idSuffix}`;
  const formId = `create-unit-form-${idSuffix}`;

  function handleCreate(formData: FormData) {
    const parentUnitId = String(formData.get("parentUnitId") ?? "").trim() || rootUnitId;
    const code = String(formData.get("code") ?? "").trim();
    const name = String(formData.get("name") ?? "").trim();
    if (!code || !name) return;
    startTransition(async () => {
      const unit = await onCreateUnit(parentUnitId, code, name);
      pickerRef.current?.select({
        id: unit.id,
        label: unit.code ? `${unit.name} (${unit.code})` : unit.name,
      });
      setDialogOpen(false);
    });
  }

  return (
    <div className="flex flex-wrap gap-2">
      <EntityPicker
        ref={pickerRef}
        name={name}
        defaultValue={defaultValue}
        onSearch={searchUnitsForPicker}
        placeholder={placeholder}
        className="w-56"
      />
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <Button type="button" variant="outline" size="sm" onClick={() => setDialogOpen(true)}>
          <Plus className="size-3.5" />
          {t("createUnit")}
        </Button>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("createUnitHeading")}</DialogTitle>
          </DialogHeader>
          <form action={handleCreate} id={formId} className="flex flex-col gap-3">
            <Label htmlFor={nameInputId} className="flex flex-col items-start gap-1 text-xs">
              {t("createUnitName")}
              <Input id={nameInputId} name="name" required defaultValue={createDefaultName} />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("createUnitCode")}
              <Input name="code" required defaultValue={slugify(createDefaultName)} />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-xs">
              {t("createUnitParentUnitId")}
              <EntityPicker
                name="parentUnitId"
                defaultValue={{ id: rootUnitId, label: rootUnitId }}
                onSearch={searchUnitsForPicker}
                placeholder={t("createUnitParentUnitId")}
              />
            </Label>
          </form>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline">
                {t("createUnitCancel")}
              </Button>
            </DialogClose>
            <Button type="submit" form={formId} disabled={pending}>
              {t("createUnitSubmit")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
