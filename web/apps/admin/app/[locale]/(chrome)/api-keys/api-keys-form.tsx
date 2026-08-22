// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useActionState, useState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export type ApiKeyActionState =
  | { id: string; label: string; permissionCodes: string[]; token: string }
  | { error: string }
  | null;

// M11.9's own copy of people/invite/invite-form.tsx's shape: useActionState instead of a plain
// Server Action + redirect(), because the one-time raw secret must never land in a URL/browser
// history. The one addition over the invite flow is the permission-code multi-select — a checkbox
// group with local Set<string> toggle state, the same pattern role-grants/bulk-grant-form.tsx uses
// for selecting people, since no existing permission-picker component exists to reuse.
export function ApiKeysForm({
  action,
  permissionCatalog,
}: {
  action: (prevState: ApiKeyActionState, formData: FormData) => Promise<ApiKeyActionState>;
  permissionCatalog: string[];
}) {
  const t = useTranslations("ApiKeysPage");
  const [state, formAction, pending] = useActionState(action, null);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  function toggle(code: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(code)) {
        next.delete(code);
      } else {
        next.add(code);
      }
      return next;
    });
  }

  return (
    <div className="flex flex-col gap-4">
      <form action={formAction} className="flex flex-col gap-4">
        <Label className="flex flex-col items-start gap-1">
          {t("labelField")}
          <Input name="label" required placeholder={t("labelPlaceholder")} />
        </Label>

        <div className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("permissionCodesHeading")}</span>
          <ul className="flex max-h-64 flex-col divide-y overflow-y-auto rounded-md border">
            {permissionCatalog.map((code) => (
              <li key={code} className="flex items-center gap-2 px-3 py-2 text-sm">
                <input
                  type="checkbox"
                  name="permissionCodes"
                  value={code}
                  checked={selected.has(code)}
                  onChange={() => toggle(code)}
                  className="size-4 rounded border-input"
                />
                <span className="font-mono text-xs">{code}</span>
              </li>
            ))}
          </ul>
        </div>

        <Button type="submit" disabled={pending || selected.size === 0} className="self-start">
          {t("submit")}
        </Button>
      </form>

      {state && "token" in state && (
        <div className="flex flex-col gap-2 rounded-md border p-3">
          <p className="text-sm text-muted-foreground">{t("keyGenerated")}</p>
          <div className="flex items-center gap-2">
            <Input readOnly value={state.token} className="font-mono text-xs" />
            <CopyButton value={state.token} />
          </div>
        </div>
      )}

      {state && "error" in state && (
        <p className="rounded-md border border-destructive p-3 text-sm">{t(state.error)}</p>
      )}
    </div>
  );
}

function CopyButton({ value }: { value: string }) {
  const t = useTranslations("ApiKeysPage");
  const [copied, setCopied] = useState(false);

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={async () => {
        await navigator.clipboard.writeText(value);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }}
    >
      {copied ? t("copied") : t("copyKey")}
    </Button>
  );
}
