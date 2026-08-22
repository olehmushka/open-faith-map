// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { CoreApiError, invitePerson } from "@/lib/core";

import { InviteForm, type InviteActionState } from "./invite-form";

// M11.6, D-InviteLinkMVP — pre-provisions a Person+Account for the given email/displayName and
// shows the admin a one-time link to copy and share manually (no email infrastructure exists in
// this repo). The createInvite Server Action below returns the token/expiry to InviteForm (a
// client component) instead of redirecting, so the one-time token never appears in a URL.
export default async function SuperAdminInvitePage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations("SuperAdminInvitePage");

  async function createInvite(_prevState: InviteActionState, formData: FormData): Promise<InviteActionState> {
    "use server";
    const email = String(formData.get("email") ?? "");
    const displayName = String(formData.get("displayName") ?? "");
    try {
      const result = await invitePerson(email, displayName);
      return {
        path: `/${locale}/accept-invite?token=${encodeURIComponent(result.token)}`,
        expiresAt: result.expiresAt,
      };
    } catch (e) {
      if (e instanceof CoreApiError) {
        return { error: e.errorName === "Core:AccountAlreadyExists" ? "errorAccountAlreadyExists" : "errorGeneric" };
      }
      throw e;
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-muted-foreground">{t("intro")}</p>
      <InviteForm action={createInvite} />
    </div>
  );
}
