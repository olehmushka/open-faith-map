// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { CoreApiError, createApiKey, listMyApiKeys, listPermissionCatalog, revokeMyApiKey } from "@/lib/core";
import { redirect } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";

import { ApiKeysForm, type ApiKeyActionState } from "./api-keys-form";

// M11.9 — self-service API key management: create (secret shown once, never redirected — same
// URL-safety reasoning people/invite/invite-form.tsx documents), list, revoke. A dedicated page, not a
// Card on whoami/page.tsx, for the same reason the invite flow is its own page: the "shown once"
// secret needs a client component with useActionState, not a server-rendered Card.
export default async function ApiKeysPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const session = await auth();
  if (!session) {
    redirect({ href: "/login", locale });
  }

  const t = await getTranslations("ApiKeysPage");
  const [keys, permissionCatalog] = await Promise.all([listMyApiKeys(), listPermissionCatalog()]);

  async function createApiKeyAction(_prevState: ApiKeyActionState, formData: FormData): Promise<ApiKeyActionState> {
    "use server";
    const label = String(formData.get("label") ?? "");
    const permissionCodes = formData.getAll("permissionCodes").map(String);
    try {
      const result = await createApiKey(label, permissionCodes);
      return { id: result.id, label: result.label, permissionCodes: result.permissionCodes, token: result.token };
    } catch (e) {
      if (e instanceof CoreApiError) {
        return { error: e.errorName === "Core:UnknownPermissionCode" ? "errorUnknownPermissionCode" : "errorGeneric" };
      }
      throw e;
    }
  }

  async function revokeMyApiKeyAction(formData: FormData) {
    "use server";
    await revokeMyApiKey(String(formData.get("apiKeyId")));
    redirect({ href: "/api-keys", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-muted-foreground">{t("intro")}</p>

      <div className="flex flex-col gap-2">
        <h2 className="text-base font-medium">{t("existingHeading")}</h2>
        {keys.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("noKeys")}</p>
        ) : (
          keys.map((k) => (
            <div key={k.id} className="flex items-center justify-between gap-4 rounded-md border p-3">
              <div className="flex flex-col gap-1">
                <p className="text-sm font-medium">{k.label}</p>
                <p className="text-xs text-muted-foreground">{k.permissionCodes.join(", ")}</p>
                <p className="text-xs text-muted-foreground">
                  {t("createdAt", { date: new Date(k.createdAt).toLocaleString(locale) })}
                  {k.lastUsedAt ? ` · ${t("lastUsedAt", { date: new Date(k.lastUsedAt).toLocaleString(locale) })}` : ""}
                </p>
              </div>
              <form action={revokeMyApiKeyAction}>
                <input type="hidden" name="apiKeyId" value={k.id} />
                <Button type="submit" variant="destructive" size="sm">
                  {t("revoke")}
                </Button>
              </form>
            </div>
          ))
        )}
      </div>

      <div className="flex flex-col gap-2 border-t pt-6">
        <h2 className="text-base font-medium">{t("createHeading")}</h2>
        <ApiKeysForm action={createApiKeyAction} permissionCatalog={permissionCatalog} />
      </div>
    </main>
  );
}
