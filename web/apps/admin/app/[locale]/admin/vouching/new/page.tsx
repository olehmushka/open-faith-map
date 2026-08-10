import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { createVouch } from "@/lib/vouching";
import { redirect } from "@/i18n/navigation";

// Guarantor-facing "vouch for someone" form (M6). Only requires being logged in, same as every
// other admin-app page — no separate client-side authorization gate. openfaithmap-api's own PDP
// check (religionorg.manage on guarantorCongregationUnitId) is the real access-control decision;
// a caller with no standing on the unit they name simply gets Vouching:Forbidden back.
//
// Deliberately the ONLY entry point built for filing a vouch — there is no claimant-facing "request
// a vouch" page, since that would need a congregation-claim flow this repo doesn't have yet
// (vouching.md names it as the eventual real caller). See docs/modules/vouching.md's open seams.
export default async function NewVouchPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("NewVouchPage");

  async function submit(formData: FormData) {
    "use server";
    const claimantPersonId = String(formData.get("claimantPersonId") ?? "").trim();
    const congregationUnitId = String(formData.get("congregationUnitId") ?? "").trim();
    const guarantorCongregationUnitId = String(formData.get("guarantorCongregationUnitId") ?? "").trim();
    const statement = String(formData.get("statement") ?? "").trim() || undefined;
    if (!claimantPersonId || !congregationUnitId || !guarantorCongregationUnitId) return;
    await createVouch(claimantPersonId, congregationUnitId, guarantorCongregationUnitId, statement);
    redirect({ href: "/admin/vouching/new?submitted=1", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-gray-500">{t("intro")}</p>

      <form action={submit} className="flex flex-col gap-3">
        <label className="flex flex-col gap-1 text-sm">
          {t("claimantLabel")}
          <input name="claimantPersonId" required className="rounded border px-2 py-1" />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          {t("congregationLabel")}
          <input name="congregationUnitId" required className="rounded border px-2 py-1" />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          {t("guarantorCongregationLabel")}
          <input name="guarantorCongregationUnitId" required className="rounded border px-2 py-1" />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          {t("statementLabel")}
          <textarea name="statement" className="rounded border px-2 py-1" rows={3} />
        </label>
        <button type="submit" className="rounded border px-3 py-1 text-sm">
          {t("submit")}
        </button>
      </form>
    </main>
  );
}
