// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use client";

import { useActionState, useState } from "react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export type InviteActionState =
  | { path: string; expiresAt: string }
  | { error: string }
  | null;

// Client component, not a plain Server Action + redirect (every other mutation on this app's
// super-admin pages, e.g. toggleAccountStatus, ends in redirect): the one-time token this action
// returns must never land in a URL/browser history/server access log, which redirecting with it as
// a query param would do. React 19's useActionState is what lets a client component call a Server
// Action and receive its return value directly, with no navigation involved.
export function InviteForm({
  action,
}: {
  action: (prevState: InviteActionState, formData: FormData) => Promise<InviteActionState>;
}) {
  const t = useTranslations("SuperAdminInvitePage");
  const [state, formAction, pending] = useActionState(action, null);

  const link = state && "path" in state && typeof window !== "undefined" ? `${window.location.origin}${state.path}` : null;

  return (
    <div className="flex flex-col gap-4">
      <form action={formAction} className="flex flex-col gap-4">
        <Label className="flex flex-col items-start gap-1">
          {t("emailLabel")}
          <Input name="email" type="email" required />
        </Label>
        <Label className="flex flex-col items-start gap-1">
          {t("displayNameLabel")}
          <Input name="displayName" required />
        </Label>
        <Button type="submit" disabled={pending} className="self-start">
          {t("submit")}
        </Button>
      </form>

      {link && (
        <div className="flex flex-col gap-2 rounded-md border p-3">
          <p className="text-sm text-muted-foreground">{t("linkGenerated")}</p>
          <div className="flex items-center gap-2">
            <Input readOnly value={link} className="font-mono text-xs" />
            <CopyButton value={link} />
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
  const t = useTranslations("SuperAdminInvitePage");
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
      {copied ? t("copied") : t("copyLink")}
    </Button>
  );
}
