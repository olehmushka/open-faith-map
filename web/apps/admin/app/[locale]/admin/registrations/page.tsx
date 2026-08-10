import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { approveRegistration, listRegistrations, rejectRegistration } from "@/lib/registration";
import { redirect } from "@/i18n/navigation";

// Renders whatever listRegistrations returns for the caller — openfaithmap-api itself decides
// operator (all requests) vs. submitter (their own only) by asking go-oikumenea's PDP live
// (MyCapabilities), never a locally-cached role (D-Facade). A non-operator lands here and simply
// sees their own submissions; approve/reject actions still go through the real PDP check
// regardless of what this page renders (web-facade.md's "no client-side authorization").
export default async function RegistrationsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ jurisdictionQuery?: string }>;
}) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("RegistrationsPage");
  const { requests } = await listRegistrations();
  const { jurisdictionQuery } = await searchParams;
  const jurisdictionResults = jurisdictionQuery ? await searchJurisdictionUnits(jurisdictionQuery) : [];

  async function approve(formData: FormData) {
    "use server";
    const jurisdictionUnitId = String(formData.get("jurisdictionUnitId") ?? "").trim() || undefined;
    await approveRegistration(String(formData.get("id")), undefined, jurisdictionUnitId);
    redirect({ href: "/admin/registrations", locale });
  }

  async function createJurisdiction(formData: FormData) {
    "use server";
    const parentUnitId = String(formData.get("parentUnitId") ?? "").trim() || undefined;
    const code = String(formData.get("code") ?? "").trim();
    const name = String(formData.get("name") ?? "").trim();
    if (!code || !name) return;
    const rootUnitId = process.env.REGISTRATION_ROOT_UNIT_ID;
    if (!rootUnitId) throw new Error("REGISTRATION_ROOT_UNIT_ID is not set.");
    const unit = await createJurisdictionUnit(parentUnitId ?? rootUnitId, code, name);
    redirect({ href: `/admin/registrations?jurisdictionQuery=${encodeURIComponent(unit.name)}`, locale });
  }

  async function reject(formData: FormData) {
    "use server";
    const reason = String(formData.get("reason") ?? "").trim();
    if (!reason) return;
    await rejectRegistration(String(formData.get("id")), reason);
    redirect({ href: "/admin/registrations", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <section className="rounded border p-4">
        <h2 className="font-medium">{t("jurisdictionUnitsHeading")}</h2>
        <p className="text-sm">{t("jurisdictionSearchHint")}</p>
        <form action={`/${locale}/admin/registrations`} className="mt-2 flex gap-2">
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
          <input
            name="code"
            placeholder={t("newUnitCodePlaceholder")}
            required
            className="rounded border px-2 py-1 text-sm"
          />
          <input
            name="name"
            placeholder={t("newUnitNamePlaceholder")}
            required
            className="rounded border px-2 py-1 text-sm"
          />
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

      {requests.length === 0 && <p>{t("noRequests")}</p>}
      <ul className="flex flex-col gap-4">
        {requests.map((r) => (
          <li key={r.id} className="rounded border p-4">
            <div className="flex items-baseline justify-between">
              <span className="font-medium">{r.congregationName}</span>
              <span className="text-sm">{r.status}</span>
            </div>
            <p className="text-sm">
              {r.locality ?? ""} {r.street ?? ""}
            </p>
            {r.status === "REJECTED" && r.rejectionReason && (
              <p className="text-sm">{t("rejectionReason", { reason: r.rejectionReason })}</p>
            )}
            {r.status === "PENDING" && (
              <div className="mt-3 flex flex-col gap-3">
                <form action={approve} className="flex gap-2">
                  <input type="hidden" name="id" value={r.id} />
                  <input
                    name="jurisdictionUnitId"
                    placeholder={t("jurisdictionUnitIdPlaceholder")}
                    className="rounded border px-2 py-1 text-sm"
                  />
                  <button type="submit" className="rounded border px-3 py-1 text-sm">
                    {t("approve")}
                  </button>
                </form>
                <form action={reject} className="flex gap-2">
                  <input type="hidden" name="id" value={r.id} />
                  <input
                    name="reason"
                    placeholder={t("rejectionReasonPlaceholder")}
                    required
                    className="rounded border px-2 py-1 text-sm"
                  />
                  <button type="submit" className="rounded border px-3 py-1 text-sm">
                    {t("reject")}
                  </button>
                </form>
              </div>
            )}
          </li>
        ))}
      </ul>
    </main>
  );
}
