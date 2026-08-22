// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { auth, signIn } from "@/auth";
import { CoreApiError, resolveInvite } from "@/lib/core";
import { redirect } from "@/i18n/navigation";

// M11.6, D-InviteLinkMVP — the invite link's landing page. Public: the invitee has no session at
// all yet, so this lives under (chrome), the same unauthenticated route group as /login, not under
// /admin (which requires one). resolveInvite is purely informational here — the actual account
// linking is done entirely by M10.2's existing JIT link-on-match logic on Google sign-in, this page
// only confirms to the invitee they're in the right place first.
export default async function AcceptInvitePage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ token?: string }>;
}) {
  const { locale } = await params;
  const { token } = await searchParams;
  const t = await getTranslations("AcceptInvitePage");

  const session = await auth();
  if (session) return redirect({ href: "/whoami", locale });

  let info: { displayName: string; email: string } | null = null;
  let errorKey = "errorMissingToken";
  if (token) {
    try {
      info = await resolveInvite(token);
    } catch (e) {
      if (e instanceof CoreApiError) {
        errorKey =
          e.errorName === "Core:InviteExpired"
            ? "errorExpired"
            : e.errorName === "Core:InviteAlreadyAccepted"
              ? "errorAlreadyAccepted"
              : "errorNotFound";
      } else {
        throw e;
      }
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-sm flex-col justify-center gap-4 px-6">
      <h1 className="text-xl font-semibold">{t("heading")}</h1>
      {info ? (
        <>
          <p className="text-sm">{t("welcome", { name: info.displayName, email: info.email })}</p>
          <form
            action={async () => {
              "use server";
              await signIn("google", { redirectTo: `/${locale}/whoami` });
            }}
          >
            <button type="submit" className="rounded border px-4 py-2">
              {t("googleButton")}
            </button>
          </form>
        </>
      ) : (
        <p className="rounded border border-red-500 p-3 text-sm">{t(errorKey)}</p>
      )}
    </main>
  );
}
