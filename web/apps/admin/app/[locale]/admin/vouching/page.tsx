import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { getGuarantorStatus, listVouches, revokeGuarantor } from "@/lib/vouching";
import { redirect } from "@/i18n/navigation";

// Moderator console for guarantor standing (M6). Same no-local-gate discipline as
// /admin/moderation: openfaithmap-api's own PDP check (unit.lifecycle on the root unit) is the
// real access-control decision — this page adds no local "isModerator" check of its own, a
// non-moderator's lookup simply comes back Forbidden.
export default async function VouchingConsolePage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ guarantorPersonId?: string; claimant?: string; congregation?: string }>;
}) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("VouchingConsolePage");
  const { guarantorPersonId, claimant, congregation } = await searchParams;

  const guarantorStatus = guarantorPersonId ? await getGuarantorStatus(guarantorPersonId) : undefined;
  const { vouches } = claimant || congregation ? await listVouches(claimant, congregation) : { vouches: [] };

  async function revoke(formData: FormData) {
    "use server";
    const personId = String(formData.get("guarantorPersonId") ?? "").trim();
    const reason = String(formData.get("reason") ?? "").trim();
    if (!personId || !reason) return;
    await revokeGuarantor(personId, reason);
    redirect({ href: `/admin/vouching?guarantorPersonId=${encodeURIComponent(personId)}`, locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-8 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-medium">{t("guarantorHeading")}</h2>
        <form className="flex gap-2">
          <input
            name="guarantorPersonId"
            defaultValue={guarantorPersonId}
            placeholder={t("guarantorPersonIdPlaceholder")}
            className="flex-1 rounded border px-2 py-1 text-sm"
          />
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("lookup")}
          </button>
        </form>

        {guarantorStatus && (
          <div className="rounded border p-4">
            <p className="text-sm">
              {t("status")}: <span className="font-medium">{guarantorStatus.status}</span>
            </p>
            {guarantorStatus.revokedAt && (
              <p className="text-sm text-gray-500">
                {t("revokedAt", { date: guarantorStatus.revokedAt })} — {guarantorStatus.revokedReason}
              </p>
            )}
            {guarantorStatus.status === "TRUSTED" && (
              <form action={revoke} className="mt-3 flex flex-wrap gap-2">
                <input type="hidden" name="guarantorPersonId" value={guarantorStatus.guarantorPersonId} />
                <input
                  name="reason"
                  placeholder={t("reasonPlaceholder")}
                  required
                  className="rounded border px-2 py-1 text-sm"
                />
                <button type="submit" className="rounded border px-3 py-1 text-sm">
                  {t("revoke")}
                </button>
              </form>
            )}
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-medium">{t("vouchesHeading")}</h2>
        <form className="flex flex-wrap gap-2">
          <input
            name="claimant"
            defaultValue={claimant}
            placeholder={t("claimantPlaceholder")}
            className="rounded border px-2 py-1 text-sm"
          />
          <input
            name="congregation"
            defaultValue={congregation}
            placeholder={t("congregationPlaceholder")}
            className="rounded border px-2 py-1 text-sm"
          />
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("lookup")}
          </button>
        </form>

        {vouches.length === 0 && <p>{t("noVouches")}</p>}
        <ul className="flex flex-col gap-4">
          {vouches.map((v) => (
            <li key={v.id} className="rounded border p-4">
              <p className="text-sm">
                {t("vouchLine", { guarantor: v.guarantorPersonId, claimant: v.claimantPersonId })}
              </p>
              <p className="text-sm text-gray-500">{t("congregationLine", { unit: v.congregationUnitId })}</p>
              {v.statement && <p className="text-sm">{v.statement}</p>}
              <p className="text-sm text-gray-500">{t("filedAt", { date: v.createdAt })}</p>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
