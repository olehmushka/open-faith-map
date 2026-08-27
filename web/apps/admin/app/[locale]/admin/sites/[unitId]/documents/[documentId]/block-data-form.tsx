// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { BlockType } from "@/lib/content";

// M14.4: the generic form renderer that replaces the raw-JSON <Textarea> block editor. A block's
// form is derived from its own BlockType.uiSchema (widget/label/help/order) — never hand-written
// per block type, so a block type added later renders a working form with no code change here.
//
// Fields NOT named in ui_schema.fields are preserved untouched: every widget only ever overwrites
// its own key(s) of the data object it was seeded from. This is what lets server-derived fields
// like image.originalUrl (never user-edited, set by medianormalize.go) round-trip through a save
// with no dedicated "read-only" widget.
//
// richText fields (heading.text, paragraph.text, quote.text, staff_card.bio, list.content) are
// still a JSON textarea here, per ui_schema's own "textarea" widget with explanatory help text — a
// visual rich-text editor is a future milestone, not this one; no such editor exists anywhere in
// this codebase yet. "block-list" is the one recursive widget, used by columns.columns[].blocks: it
// resolves each nested item's own blockTypeCode against the full catalog and re-renders with THAT
// type's own ui_schema, which is the only way nested block data (never itself schema-constrained,
// see migrations/0002_content.sql) can render as a form instead of raw JSON.

type FieldWidget = "text" | "url" | "number" | "select" | "textarea" | "array" | "block-list";

interface UiSchemaField {
  name: string;
  widget: FieldWidget;
  label: string;
  help?: string;
  options?: { value: string; label: string }[];
  min?: number;
  max?: number;
  step?: number;
  minItems?: number;
  maxItems?: number;
  itemLabel?: string;
  itemFields?: UiSchemaField[];
}

interface UiSchema {
  fields: UiSchemaField[];
}

type JsonObject = Record<string, unknown>;

function parseUiSchema(raw: unknown): UiSchema {
  if (raw && typeof raw === "object" && Array.isArray((raw as UiSchema).fields)) {
    return raw as UiSchema;
  }
  return { fields: [] };
}

function asObject(value: unknown): JsonObject {
  return value && typeof value === "object" && !Array.isArray(value) ? { ...(value as JsonObject) } : {};
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

// A Content:BlockDataInvalid/BlockUrlNotAllowed error's `field` names the offending top-level
// property directly for a schema-shape failure, but the separate URL-allowlist pass names an
// array item's own path (e.g. "images[0].url") — this always resolves back to the top-level field
// that owns it, which is as precise as this form highlights (matching blockvalidation.go's own
// "as precise as it's cheap to be" precedent).
function topLevelSegment(field: string): string {
  return field.split(/[.[]/)[0];
}

/** The recursive core: renders one block's data-editing fields from its own BlockType.uiSchema. */
function BlockDataFields({
  blockType,
  blockTypes,
  value,
  onChange,
  erroredField,
}: {
  blockType: BlockType;
  blockTypes: BlockType[];
  value: unknown;
  onChange: (next: JsonObject) => void;
  erroredField?: string;
}) {
  const uiSchema = parseUiSchema(blockType.uiSchema);
  const data = asObject(value);
  const erroredTopLevelField = erroredField ? topLevelSegment(erroredField) : undefined;

  return (
    <div className="flex flex-col gap-3">
      {uiSchema.fields.map((field) => (
        <FieldControl
          key={field.name}
          field={field}
          value={data[field.name]}
          onChange={(next) => onChange({ ...data, [field.name]: next })}
          blockTypes={blockTypes}
          isErrored={field.name === erroredTopLevelField}
        />
      ))}
    </div>
  );
}

function FieldControl({
  field,
  value,
  onChange,
  blockTypes,
  isErrored,
}: {
  field: UiSchemaField;
  value: unknown;
  onChange: (value: unknown) => void;
  blockTypes: BlockType[];
  isErrored: boolean;
}) {
  switch (field.widget) {
    case "text":
    case "url":
      return (
        <ScalarField field={field} isErrored={isErrored}>
          <Input
            type={field.widget === "url" ? "url" : "text"}
            aria-invalid={isErrored}
            value={typeof value === "string" ? value : ""}
            onChange={(e) => onChange(e.target.value)}
          />
        </ScalarField>
      );
    case "number":
      return (
        <ScalarField field={field} isErrored={isErrored}>
          <Input
            type="number"
            aria-invalid={isErrored}
            min={field.min}
            max={field.max}
            step={field.step}
            value={typeof value === "number" ? value : ""}
            onChange={(e) => onChange(e.target.value === "" ? undefined : Number(e.target.value))}
          />
        </ScalarField>
      );
    case "select":
      return (
        <ScalarField field={field} isErrored={isErrored}>
          <Select value={typeof value === "string" ? value : ""} onValueChange={onChange}>
            <SelectTrigger className="w-full" aria-invalid={isErrored}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(field.options ?? []).map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ScalarField>
      );
    case "textarea":
      return <JsonTextareaField field={field} value={value} onChange={onChange} isErrored={isErrored} />;
    case "array":
      return <ArrayField field={field} value={value} onChange={onChange} blockTypes={blockTypes} />;
    case "block-list":
      return <BlockListField field={field} value={value} onChange={onChange} blockTypes={blockTypes} />;
    default:
      return null;
  }
}

function ScalarField({
  field,
  isErrored,
  children,
}: {
  field: UiSchemaField;
  isErrored: boolean;
  children: React.ReactNode;
}) {
  return (
    <Label className="flex flex-col items-start gap-1">
      {field.label}
      {children}
      {field.help && <span className="text-xs text-muted-foreground">{field.help}</span>}
      {isErrored && <span className="text-xs text-destructive">This field failed validation.</span>}
    </Label>
  );
}

// richText fields (and any other free-form JSON field) stay a JSON textarea in M14.4 — draft text
// is kept locally so a transiently-invalid mid-edit JSON string never loses keystrokes, and only
// committed to the block's data on blur once it re-parses.
function JsonTextareaField({
  field,
  value,
  onChange,
  isErrored,
}: {
  field: UiSchemaField;
  value: unknown;
  onChange: (value: unknown) => void;
  isErrored: boolean;
}) {
  const [draft, setDraft] = useState(() => JSON.stringify(value ?? null, null, 2));

  return (
    <Label className="flex flex-col items-start gap-1">
      {field.label}
      <Textarea
        rows={4}
        className="font-mono text-xs"
        aria-invalid={isErrored}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => {
          try {
            onChange(JSON.parse(draft));
          } catch {
            // Left uncommitted: the last successfully-parsed value stays the field's real data
            // until the draft is valid JSON again.
          }
        }}
      />
      {field.help && <span className="text-xs text-muted-foreground">{field.help}</span>}
      {isErrored && <span className="text-xs text-destructive">This field failed validation.</span>}
    </Label>
  );
}

// A field's own repeatable array (e.g. gallery.images, columns.columns) — explicitly in scope for
// M14.4; the milestone boundary with M14.5's inserter is about the page's outer block list, not a
// field's internal array.
function ArrayField({
  field,
  value,
  onChange,
  blockTypes,
}: {
  field: UiSchemaField;
  value: unknown;
  onChange: (value: unknown) => void;
  blockTypes: BlockType[];
}) {
  const items = asArray(value);
  const itemFields = field.itemFields ?? [];

  function updateItem(index: number, next: JsonObject) {
    const nextItems = items.slice();
    nextItems[index] = next;
    onChange(nextItems);
  }

  return (
    <div className="flex flex-col gap-2">
      <span className="text-sm font-medium">{field.label}</span>
      {field.help && <span className="text-xs text-muted-foreground">{field.help}</span>}
      {items.map((item, index) => {
        const itemData = asObject(item);
        return (
          <div key={index} className="flex flex-col gap-2 rounded-md border p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-muted-foreground">
                {field.itemLabel ?? "Item"} {index + 1}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => onChange(items.filter((_, i) => i !== index))}
              >
                Remove
              </Button>
            </div>
            <div className="flex flex-col gap-3">
              {itemFields.map((itemField) => (
                <FieldControl
                  key={itemField.name}
                  field={itemField}
                  value={itemData[itemField.name]}
                  onChange={(next) => updateItem(index, { ...itemData, [itemField.name]: next })}
                  blockTypes={blockTypes}
                  isErrored={false}
                />
              ))}
            </div>
          </div>
        );
      })}
      <Button type="button" variant="outline" size="sm" className="self-start" onClick={() => onChange([...items, {}])}>
        Add {field.itemLabel ?? "item"}
      </Button>
    </div>
  );
}

interface BlockListItem {
  blockTypeCode?: string;
  data?: unknown;
}

// The one recursive widget — each item names its own real blockTypeCode; the nested form is built
// by looking that code up in the full catalog and rendering it with its own ui_schema, which is
// how "columns" nesting can be a form at all rather than raw JSON.
function BlockListField({
  field,
  value,
  onChange,
  blockTypes,
}: {
  field: UiSchemaField;
  value: unknown;
  onChange: (value: unknown) => void;
  blockTypes: BlockType[];
}) {
  const items = asArray(value) as BlockListItem[];

  function updateItem(index: number, next: BlockListItem) {
    const nextItems = items.slice();
    nextItems[index] = next;
    onChange(nextItems);
  }

  return (
    <div className="flex flex-col gap-2">
      <span className="text-sm font-medium">{field.label}</span>
      {items.map((item, index) => {
        const nestedType = blockTypes.find((bt) => bt.code === item.blockTypeCode);
        return (
          <div key={index} className="flex flex-col gap-2 rounded-md border p-3">
            <div className="flex items-center justify-between gap-2">
              <Select
                value={item.blockTypeCode ?? ""}
                onValueChange={(code) => updateItem(index, { blockTypeCode: code, data: {} })}
              >
                <SelectTrigger size="sm" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {blockTypes.map((bt) => (
                    <SelectItem key={bt.code} value={bt.code}>
                      {bt.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => onChange(items.filter((_, i) => i !== index))}
              >
                Remove
              </Button>
            </div>
            {nestedType && (
              <BlockDataFields
                blockType={nestedType}
                blockTypes={blockTypes}
                value={item.data}
                onChange={(nextData) => updateItem(index, { blockTypeCode: item.blockTypeCode, data: nextData })}
              />
            )}
          </div>
        );
      })}
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="self-start"
        onClick={() => onChange([...items, { blockTypeCode: blockTypes[0]?.code, data: {} }])}
      >
        Add block
      </Button>
    </div>
  );
}

/**
 * Top-level entry point: owns the block's local data state and mirrors it into the one hidden
 * `data` input the surrounding <form action={saveBlocks}> already reads via
 * formData.getAll("data") — no change needed to that server action's contract.
 */
export function BlockDataForm({
  blockType,
  blockTypes,
  initialData,
  erroredField,
}: {
  blockType: BlockType;
  blockTypes: BlockType[];
  initialData: unknown;
  erroredField?: string;
}) {
  const [data, setData] = useState<JsonObject>(() => asObject(initialData));

  return (
    <div className="flex flex-col gap-3">
      <input type="hidden" name="data" value={JSON.stringify(data)} readOnly />
      <BlockDataFields
        blockType={blockType}
        blockTypes={blockTypes}
        value={data}
        onChange={setData}
        erroredField={erroredField}
      />
    </div>
  );
}
