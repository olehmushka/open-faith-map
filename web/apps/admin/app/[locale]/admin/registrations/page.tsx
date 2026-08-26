import { getTranslations } from "next-intl/server";

import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { approveRegistration, listRegistrations, rejectRegistration } from "@/lib/registration";
import { refreshRegionAroundPoint } from "@/lib/discovery";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

import { RequestList } from "./request-list";

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
  const t = await getTranslations("RegistrationsPage");
  const { requests } = await listRegistrations();
  const { jurisdictionQuery } = await searchParams;

  // Pre-formatted server-side, keyed by request id — a closure over `t` can't be passed to
  // RequestList (a Client Component); only plain serializable values cross that boundary.
  const rejectionReasonById: Record<string, string> = {};
  for (const r of requests) {
    if (r.status === "REJECTED" && r.rejectionReason) {
      rejectionReasonById[r.id] = t("rejectionReason", { reason: r.rejectionReason });
    }
  }
  const jurisdictionResults = jurisdictionQuery ? await searchJurisdictionUnits(jurisdictionQuery) : [];

  async function approve(formData: FormData) {
    "use server";
    const jurisdictionUnitId = String(formData.get("jurisdictionUnitId") ?? "").trim() || undefined;
    const approved = await approveRegistration(String(formData.get("id")), undefined, jurisdictionUnitId);
    await refreshRegionAroundPoint(approved.coordinate.latitude, approved.coordinate.longitude);
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
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("jurisdictionUnitsHeading")}</CardTitle>
          <CardDescription>{t("jurisdictionSearchHint")}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form action={`/${locale}/admin/registrations`} className="flex gap-2">
            <Input
              name="jurisdictionQuery"
              defaultValue={jurisdictionQuery}
              placeholder={t("jurisdictionSearchPlaceholder")}
              className="h-8 max-w-sm"
            />
            <Button type="submit" variant="outline" size="sm">
              {t("search")}
            </Button>
          </form>
          {jurisdictionQuery && (
            <ul className="flex flex-col gap-1 text-sm">
              {jurisdictionResults.length === 0 && <li className="text-muted-foreground">{t("noMatches")}</li>}
              {jurisdictionResults.map((u) => (
                <li key={u.id} className="flex items-center gap-2">
                  <code className="rounded bg-muted px-1">{u.id}</code>
                  <span>{u.name}</span>
                  {u.code && <span className="text-muted-foreground">({u.code})</span>}
                </li>
              ))}
            </ul>
          )}
          <form action={createJurisdiction} className="flex flex-wrap gap-2">
            <Input name="code" placeholder={t("newUnitCodePlaceholder")} required className="h-8" />
            <Input name="name" placeholder={t("newUnitNamePlaceholder")} required className="h-8" />
            <Input name="parentUnitId" placeholder={t("parentUnitIdPlaceholder")} className="h-8" />
            <Button type="submit" size="sm">
              {t("createJurisdictionUnit")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <RequestList
        requests={requests}
        onApprove={approve}
        onReject={reject}
        labels={{
          noRequests: t("noRequests"),
          jurisdictionUnitIdPlaceholder: t("jurisdictionUnitIdPlaceholder"),
          approve: t("approve"),
          rejectionReasonPlaceholder: t("rejectionReasonPlaceholder"),
          reject: t("reject"),
          rejectionReasonById,
          filterRequests: t("filterRequests"),
          congregationName: t("congregationName"),
          status: t("status"),
          location: t("location"),
        }}
      />
    </div>
  );
}
