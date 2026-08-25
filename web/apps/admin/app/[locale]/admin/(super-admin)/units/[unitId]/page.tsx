import { getTranslations } from "next-intl/server";

import {
  CoreApiError,
  createUnit,
  deleteUnit,
  getOrgProfile,
  getUnit,
  setUnitState,
  unitAncestors,
  unitDeleteEligibility,
  updateUnit,
} from "@/lib/core";
import { Link, redirect } from "@/i18n/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { UnitStatusBadge } from "@/components/status-badge";

// Super-admin unit detail (M10.8, full CRUD since M12.5). getOrgProfile is .catch(() => null)'d —
// not every unit is a religion org (jurisdiction units aren't), so no profile is a normal state,
// matching admin/sites/[unitId]/page.tsx's own .catch(() => null) convention for "not found is
// normal" reads. State-transition and delete affordances are hidden for the root unit
// (ancestors.length === 0, the same signal the ancestors card already renders as "This is a root
// unit.") since setUnitState/deleteUnit both hard-refuse it regardless of grant (RootUnitProtected).
export default async function SuperAdminUnitPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; unitId: string }>;
  searchParams: Promise<{ deleteError?: string }>;
}) {
  const { locale, unitId } = await params;
  const { deleteError } = await searchParams;
  const t = await getTranslations("SuperAdminUnitPage");

  const [unit, ancestors, orgProfile, eligibility] = await Promise.all([
    getUnit(unitId),
    unitAncestors(unitId),
    getOrgProfile(unitId).catch(() => null),
    unitDeleteEligibility(unitId).catch(() => null),
  ]);
  const isRoot = ancestors.length === 0;
  const parent = ancestors.at(-1);

  async function editUnit(formData: FormData) {
    "use server";
    const name = String(formData.get("name") ?? "");
    const code = String(formData.get("code") ?? "").trim();
    const levelRaw = String(formData.get("level") ?? "").trim();
    await updateUnit(unitId, {
      name,
      code: code === "" ? undefined : code,
      level: levelRaw === "" ? undefined : Number(levelRaw),
    });
    redirect({ href: `/admin/units/${unitId}`, locale });
  }

  async function addChildUnit(formData: FormData) {
    "use server";
    const code = String(formData.get("code") ?? "");
    const name = String(formData.get("name") ?? "");
    const levelRaw = String(formData.get("level") ?? "").trim();
    const created = await createUnit({
      parentUnitId: unitId,
      code,
      name,
      level: levelRaw === "" ? undefined : Number(levelRaw),
    });
    redirect({ href: `/admin/units/${created.id}`, locale });
  }

  async function changeState(formData: FormData) {
    "use server";
    const state = String(formData.get("state") ?? "");
    await setUnitState(unitId, state);
    redirect({ href: `/admin/units/${unitId}`, locale });
  }

  async function deleteUnitAction() {
    "use server";
    try {
      await deleteUnit(unitId);
    } catch (e) {
      if (e instanceof CoreApiError) {
        redirect({ href: `/admin/units/${unitId}?deleteError=${encodeURIComponent(e.errorName)}`, locale });
      }
      throw e;
    }
    redirect({ href: parent ? `/admin/units/${parent.id}` : "/admin/units", locale });
  }

  const deleteErrorKey =
    deleteError === "Core:UnitHasChildren"
      ? "deleteBlockedChildren"
      : deleteError === "Core:UnitHasOrgProfile"
        ? "deleteBlockedOrgProfile"
        : deleteError === "Core:UnitHasActiveRoleAssignments"
          ? "deleteBlockedRoleAssignments"
          : deleteError === "Core:RootUnitProtected"
            ? "deleteBlockedRoot"
            : deleteError
              ? "deleteBlockedGeneric"
              : null;

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold">{unit.name}</h1>
        <UnitStatusBadge status={unit.state} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("detailsHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 text-sm">
          <p>
            <span className="text-muted-foreground">{t("codeLabel")}: </span>
            {unit.code ?? "—"}
          </p>
          {orgProfile && (
            <p>
              <span className="text-muted-foreground">{t("orgProfileHeading")}: </span>
              {orgProfile.shortCode ?? "—"}
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("ancestorsHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          {ancestors.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("noAncestors")}</p>
          ) : (
            <ol className="flex flex-col gap-1 text-sm">
              {ancestors.map((a) => (
                <li key={a.id}>
                  <Link href={`/admin/units/${a.id}`} className="hover:underline">
                    {"—".repeat(a.depth)} {a.name}
                  </Link>
                </li>
              ))}
            </ol>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("editHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={editUnit} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("nameLabel")}
              <Input name="name" defaultValue={unit.name} required />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("codeLabel")}
              <Input name="code" defaultValue={unit.code ?? ""} />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("levelLabel")}
              <Input name="level" type="number" defaultValue={unit.level ?? ""} />
            </Label>
            <Button type="submit" className="self-start">
              {t("saveChanges")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("addChildHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={addChildUnit} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("nameLabel")}
              <Input name="name" required />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("codeLabel")}
              <Input name="code" required />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("levelLabel")}
              <Input name="level" type="number" />
            </Label>
            <Button type="submit" className="self-start">
              {t("addChild")}
            </Button>
          </form>
        </CardContent>
      </Card>

      {!isRoot && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("stateHeading")}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {unit.state !== "active" && (
              <form action={changeState}>
                <input type="hidden" name="state" value="active" />
                <Button type="submit" size="sm">
                  {t("activate")}
                </Button>
              </form>
            )}
            {unit.state !== "suspended" && (
              <form action={changeState}>
                <input type="hidden" name="state" value="suspended" />
                <Button type="submit" variant="outline" size="sm">
                  {t("suspend")}
                </Button>
              </form>
            )}
            {unit.state !== "archived" && (
              <form action={changeState}>
                <input type="hidden" name="state" value="archived" />
                <Button type="submit" variant="outline" size="sm">
                  {t("archive")}
                </Button>
              </form>
            )}
          </CardContent>
        </Card>
      )}

      {!isRoot && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("deleteHeading")}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {deleteErrorKey && (
              <p className="rounded-md border border-destructive p-3 text-sm">{t(deleteErrorKey)}</p>
            )}
            {eligibility && !eligibility.canDelete ? (
              <ul className="flex flex-col gap-1 text-sm text-muted-foreground">
                {eligibility.hasChildren && <li>{t("deleteBlockedChildren")}</li>}
                {eligibility.hasOrgProfile && <li>{t("deleteBlockedOrgProfile")}</li>}
                {eligibility.hasActiveRoleAssignments && <li>{t("deleteBlockedRoleAssignments")}</li>}
              </ul>
            ) : (
              <form action={deleteUnitAction}>
                <Button type="submit" variant="destructive" size="sm">
                  {t("deleteUnit")}
                </Button>
              </form>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
