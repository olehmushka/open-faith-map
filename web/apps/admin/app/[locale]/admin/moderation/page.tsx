import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { listReports, takeActionOnReport, type ActionKind } from "@/lib/moderation";
import { redirect } from "@/i18n/navigation";

// Renders whatever listReports returns for the caller — openfaithmap-api itself decides
// platform-moderator standing by asking go-oikumenea's PDP live (Authorize against unit.lifecycle
// on the root unit), never a locally-cached role (D-Facade). This page adds no local "isModerator"
// gate of its own; a non-moderator's call simply comes back Forbidden, same discipline
// /admin/registrations already follows.
const ACTION_KINDS: ActionKind[] = ["HIDE", "SUSPEND", "ARCHIVE", "WARN_ADMIN", "REVOKE_VOUCH"];

export default async function ModerationQueuePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("ModerationQueuePage");
  const { reports } = await listReports(undefined, "OPEN");

  async function takeAction(formData: FormData) {
    "use server";
    const reportId = String(formData.get("reportId"));
    const actionKind = String(formData.get("actionKind")) as ActionKind;
    const reason = String(formData.get("reason") ?? "").trim();
    if (!reason) return;
    await takeActionOnReport(reportId, actionKind, reason);
    redirect({ href: "/admin/moderation", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      {reports.length === 0 && <p>{t("noReports")}</p>}
      <ul className="flex flex-col gap-4">
        {reports.map((r) => (
          <li key={r.id} className="rounded border p-4">
            <div className="flex items-baseline justify-between">
              <span className="font-medium">
                {r.targetKind}: {r.targetRef}
              </span>
              <span className="text-sm">{r.reasonCode}</span>
            </div>
            {r.detail && <p className="text-sm">{r.detail}</p>}
            <p className="text-sm text-gray-500">{t("filedAt", { date: r.createdAt })}</p>

            <form action={takeAction} className="mt-3 flex flex-wrap gap-2">
              <input type="hidden" name="reportId" value={r.id} />
              <select name="actionKind" className="rounded border px-2 py-1 text-sm" defaultValue={ACTION_KINDS[0]}>
                {ACTION_KINDS.map((kind) => (
                  <option key={kind} value={kind}>
                    {kind}
                  </option>
                ))}
              </select>
              <input
                name="reason"
                placeholder={t("reasonPlaceholder")}
                required
                className="rounded border px-2 py-1 text-sm"
              />
              <button type="submit" className="rounded border px-3 py-1 text-sm">
                {t("takeAction")}
              </button>
            </form>
          </li>
        ))}
      </ul>
    </main>
  );
}
