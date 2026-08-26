import { getTranslations } from "next-intl/server";

import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { getReparentStatus, listRegistrations, reparentRegistration } from "@/lib/registration";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import { ReparentList } from "./reparent-list";

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
  const t = await getTranslations("ReparentPage");
  const { requests } = await listRegistrations("APPROVED");
  const { jurisdictionQuery } = await searchParams;
  const jurisdictionResults = jurisdictionQuery ? await searchJurisdictionUnits(jurisdictionQuery) : [];

  const jobs = await Promise.all(
    requests.map(async (r) => ({ requestId: r.id, job: await getReparentStatus(r.id) })),
  );
  const jobByRequestId = new Map(jobs.filter((j) => j.job).map((j) => [j.requestId, j.job!]));

  // Pre-formatted server-side, keyed by request id — a closure over `t` can't be passed to
  // ReparentList (a Client Component); only plain serializable values cross that boundary.
  const currentJurisdictionById: Record<string, string> = {};
  const unitLabelById: Record<string, string> = {};
  const lastMoveById: Record<string, string> = {};
  for (const r of requests) {
    currentJurisdictionById[r.id] = t("currentJurisdiction", {
      value: r.jurisdictionUnitId ?? t("currentJurisdictionNone"),
    });
    unitLabelById[r.id] = t("unitLabel", { unitId: r.createdUnitId ?? "" });
    const job = jobByRequestId.get(r.id);
    if (job) {
      lastMoveById[r.id] = t("lastMove", {
        oldParent: job.oldParentUnitId,
        newParent: job.newParentUnitId,
        status: job.status,
      });
    }
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
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold">{t("heading")}</h1>
        <p className="text-sm text-muted-foreground">{t("intro")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("jurisdictionUnitsHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form action={`/${locale}/admin/registrations/reparent`} className="flex gap-2">
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

      <ReparentList
        requests={requests}
        jobByRequestId={jobByRequestId}
        onReparent={reparent}
        labels={{
          noApprovedCongregations: t("noApprovedCongregations"),
          congregationName: t("congregationName"),
          currentJurisdictionById,
          lastMoveById,
          unitLabelById,
          newParentUnitIdPlaceholder: t("newParentUnitIdPlaceholder"),
          resumeMove: t("resumeMove"),
          reparentButton: t("reparentButton"),
          filterCongregations: t("filterCongregations"),
        }}
      />
    </div>
  );
}
