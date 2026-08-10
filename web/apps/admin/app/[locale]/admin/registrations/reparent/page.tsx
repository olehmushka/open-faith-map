import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { getReparentStatus, listRegistrations, reparentRegistration } from "@/lib/registration";
import { redirect } from "@/i18n/navigation";

// Re-parenting an already-APPROVED congregation's unit onto a different jurisdiction (M4.1,
// D-JurisdictionUnits) — separate from the approval flow (page.tsx) since it acts on a congregation
// that's already live, one at a time, live-verified before batching per the milestone's own
// "highest-risk item" framing. Same operator-only gate as approve/list — openfaithmap-api's own PDP
// check decides for real, this page renders whatever listRegistrations already returns.
export default async function ReparentPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ jurisdictionQuery?: string }>;
}) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("ReparentPage");
  const { requests } = await listRegistrations("APPROVED");
  const { jurisdictionQuery } = await searchParams;
  const jurisdictionResults = jurisdictionQuery ? await searchJurisdictionUnits(jurisdictionQuery) : [];

  const jobs = await Promise.all(
    requests.map(async (r) => ({ requestId: r.id, job: await getReparentStatus(r.id) })),
  );
  const jobByRequestId = new Map(jobs.map((j) => [j.requestId, j.job]));

  async function createJurisdiction(formData: FormData) {
    "use server";
    const parentUnitId = String(formData.get("parentUnitId") ?? "").trim() || undefined;
    const code = String(formData.get("code") ?? "").trim();
    const name = String(formData.get("name") ?? "").trim();
    if (!code || !name) return;
    const rootUnitId = process.env.REGISTRATION_ROOT_UNIT_ID;
    if (!rootUnitId) throw new Error("REGISTRATION_ROOT_UNIT_ID is not set.");
    const unit = await createJurisdictionUnit(parentUnitId ?? rootUnitId, code, name);
    redirect({
      href: `/admin/registrations/reparent?jurisdictionQuery=${encodeURIComponent(unit.name)}`,
      locale,
    });
  }

  async function reparent(formData: FormData) {
    "use server";
    const id = String(formData.get("id"));
    const newParentUnitId = String(formData.get("newParentUnitId") ?? "").trim();
    if (!newParentUnitId) return;
    await reparentRegistration(id, newParentUnitId);
    redirect({ href: "/admin/registrations/reparent", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm">{t("intro")}</p>

      <section className="rounded border p-4">
        <h2 className="font-medium">{t("jurisdictionUnitsHeading")}</h2>
        <form action={`/${locale}/admin/registrations/reparent`} className="mt-2 flex gap-2">
          <input
            name="jurisdictionQuery"
            defaultValue={jurisdictionQuery}
            placeholder={t("jurisdictionSearchPlaceholder")}
            className="rounded border px-2 py-1 text-sm"
          />
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("search")}
          </button>
        </form>
        {jurisdictionQuery && (
          <ul className="mt-3 flex flex-col gap-1 text-sm">
            {jurisdictionResults.length === 0 && <li>{t("noMatches")}</li>}
            {jurisdictionResults.map((u) => (
              <li key={u.id} className="flex gap-2">
                <code className="rounded bg-gray-100 px-1">{u.id}</code>
                <span>{u.name}</span>
                {u.code && <span className="text-gray-500">({u.code})</span>}
              </li>
            ))}
          </ul>
        )}
        <form action={createJurisdiction} className="mt-3 flex flex-wrap gap-2">
          <input name="code" placeholder={t("newUnitCodePlaceholder")} required className="rounded border px-2 py-1 text-sm" />
          <input name="name" placeholder={t("newUnitNamePlaceholder")} required className="rounded border px-2 py-1 text-sm" />
          <input
            name="parentUnitId"
            placeholder={t("parentUnitIdPlaceholder")}
            className="rounded border px-2 py-1 text-sm"
          />
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("createJurisdictionUnit")}
          </button>
        </form>
      </section>

      {requests.length === 0 && <p>{t("noApprovedCongregations")}</p>}
      <ul className="flex flex-col gap-4">
        {requests.map((r) => {
          const job = jobByRequestId.get(r.id) ?? null;
          return (
            <li key={r.id} className="rounded border p-4">
              <div className="flex items-baseline justify-between">
                <span className="font-medium">{r.congregationName}</span>
                <span className="text-sm text-gray-500">{t("unitLabel", { unitId: r.createdUnitId ?? "" })}</span>
              </div>
              <p className="text-sm">
                {t("currentJurisdiction", {
                  value: r.jurisdictionUnitId ?? t("currentJurisdictionNone"),
                })}
              </p>
              {job && (
                <p className="text-sm">
                  {t("lastMove", {
                    oldParent: job.oldParentUnitId,
                    newParent: job.newParentUnitId,
                    status: job.status,
                  })}
                  {job.error && <span className="text-red-600"> ({job.error})</span>}
                </p>
              )}
              <form action={reparent} className="mt-3 flex gap-2">
                <input type="hidden" name="id" value={r.id} />
                <input
                  name="newParentUnitId"
                  placeholder={t("newParentUnitIdPlaceholder")}
                  required
                  className="rounded border px-2 py-1 text-sm"
                />
                <button type="submit" className="rounded border px-3 py-1 text-sm">
                  {job && job.status !== "VERIFIED" && job.status !== "FAILED" ? t("resumeMove") : t("reparentButton")}
                </button>
              </form>
            </li>
          );
        })}
      </ul>
    </main>
  );
}
