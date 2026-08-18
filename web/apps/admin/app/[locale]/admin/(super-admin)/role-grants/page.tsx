import { getTranslations } from "next-intl/server";

import {
  grantInstanceAdmin,
  grantUnitRole,
  listInstanceAdmins,
  listRoleAssignmentsByUnit,
  listRoles,
  revokeInstanceAdmin,
  revokeRoleAssignment,
} from "@/lib/core";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Super-admin role-grants console (M10.8): a unit picker over CoreSuperAdminService's
// listRoleAssignmentsByUnit/grantUnitRole/revokeRoleAssignment, plus a separate instance-admins
// section (the plane, not a unit-scoped role — D-SuperAdminFold's amendment). Unit id is a plain
// text field, same convention as people/[personId]/page.tsx and this app's existing jurisdiction
// forms elsewhere.
export default async function SuperAdminRoleGrantsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ unitId?: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations("SuperAdminRoleGrantsPage");
  const { unitId } = await searchParams;

  const [roles, instanceAdmins, assignments] = await Promise.all([
    listRoles(),
    listInstanceAdmins(),
    unitId ? listRoleAssignmentsByUnit(unitId) : Promise.resolve([]),
  ]);

  async function grantRole(formData: FormData) {
    "use server";
    const targetUnitId = String(formData.get("unitId") ?? "");
    const personId = String(formData.get("personId") ?? "");
    const roleId = String(formData.get("roleId") ?? "");
    await grantUnitRole(personId, roleId, targetUnitId);
    redirect({ href: `/admin/role-grants?unitId=${encodeURIComponent(targetUnitId)}`, locale });
  }

  async function revokeAssignment(formData: FormData) {
    "use server";
    const assignmentId = String(formData.get("assignmentId") ?? "");
    const targetUnitId = String(formData.get("unitId") ?? "");
    await revokeRoleAssignment(assignmentId);
    redirect({ href: `/admin/role-grants?unitId=${encodeURIComponent(targetUnitId)}`, locale });
  }

  async function toggleInstanceAdmin(formData: FormData) {
    "use server";
    const personId = String(formData.get("personId") ?? "");
    if (String(formData.get("action")) === "grant") {
      await grantInstanceAdmin(personId);
    } else {
      await revokeInstanceAdmin(personId);
    }
    redirect({ href: "/admin/role-grants", locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("unitHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form className="flex gap-2">
            <Input name="unitId" defaultValue={unitId ?? ""} placeholder={t("unitIdPlaceholder")} autoFocus />
            <Button type="submit">{t("load")}</Button>
          </form>

          {unitId && assignments.length === 0 && (
            <p className="text-sm text-muted-foreground">{t("noAssignments")}</p>
          )}

          {assignments.length > 0 && (
            <ul className="flex flex-col divide-y rounded-md border">
              {assignments.map((a) => (
                <li key={a.id} className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
                  <span className="flex-1">
                    {t("assignmentLine", { person: a.personName, role: a.roleCode, scope: a.scope })}
                  </span>
                  <form action={revokeAssignment}>
                    <input type="hidden" name="assignmentId" value={a.id} />
                    <input type="hidden" name="unitId" value={unitId} />
                    <Button type="submit" variant="destructive" size="sm">
                      {t("revoke")}
                    </Button>
                  </form>
                </li>
              ))}
            </ul>
          )}

          {unitId && (
            <form action={grantRole} className="flex flex-col gap-4 border-t pt-4">
              <input type="hidden" name="unitId" value={unitId} />
              <Label className="flex flex-col items-start gap-1">
                {t("personIdLabel")}
                <Input name="personId" required placeholder={t("personIdPlaceholder")} />
              </Label>
              <Label className="flex flex-col items-start gap-1">
                {t("roleLabel")}
                <select name="roleId" required className="h-8 w-full rounded-lg border border-input bg-transparent px-2.5 py-1 text-sm">
                  <option value="">{t("rolePlaceholder")}</option>
                  {roles.map((r) => (
                    <option key={r.id} value={r.id}>
                      {r.name}
                    </option>
                  ))}
                </select>
              </Label>
              <Button type="submit" className="self-start">
                {t("grant")}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("instanceAdminsHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {instanceAdmins.length === 0 && (
            <p className="text-sm text-muted-foreground">{t("noInstanceAdmins")}</p>
          )}
          {instanceAdmins.length > 0 && (
            <ul className="flex flex-col divide-y rounded-md border">
              {instanceAdmins.map((a) => (
                <li key={a.id} className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
                  <span className="flex-1">{a.personName}</span>
                  <form action={toggleInstanceAdmin}>
                    <input type="hidden" name="action" value="revoke" />
                    <input type="hidden" name="personId" value={a.personId} />
                    <Button type="submit" variant="destructive" size="sm">
                      {t("revoke")}
                    </Button>
                  </form>
                </li>
              ))}
            </ul>
          )}

          <form action={toggleInstanceAdmin} className="flex gap-2 border-t pt-4">
            <input type="hidden" name="action" value="grant" />
            <Input name="personId" required placeholder={t("personIdPlaceholder")} />
            <Button type="submit">{t("grantInstanceAdmin")}</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
