import { getTranslations } from "next-intl/server";

import {
  deactivateAccount,
  getAccountStatus,
  getPerson,
  grantInstanceAdmin,
  grantUnitRole,
  listInstanceAdmins,
  listRoles,
  listSessions,
  reactivateAccount,
  revokeInstanceAdmin,
  revokeSession,
} from "@/lib/core";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Super-admin person detail (M10.8): instance-admin grant/revoke plus a unit-role grant form.
// Unit id is a plain text field, matching this app's own established convention for unit selection
// elsewhere (RegistrationsPage's jurisdictionUnitIdPlaceholder, ReparentPage's
// newParentUnitIdPlaceholder) rather than introducing a new picker widget for one form.
export default async function SuperAdminPersonPage({
  params,
}: {
  params: Promise<{ locale: string; personId: string }>;
}) {
  const { locale, personId } = await params;
  const t = await getTranslations("SuperAdminPersonPage");

  const [person, instanceAdmins, roles, accountStatus, sessions] = await Promise.all([
    getPerson(personId),
    listInstanceAdmins(),
    listRoles(),
    getAccountStatus(personId),
    listSessions(personId),
  ]);
  const instanceAdminGrant = instanceAdmins.find((a) => a.personId === personId);

  async function toggleInstanceAdmin(formData: FormData) {
    "use server";
    if (String(formData.get("action")) === "grant") {
      await grantInstanceAdmin(personId);
    } else {
      await revokeInstanceAdmin(personId);
    }
    redirect({ href: `/admin/people/${personId}`, locale });
  }

  async function toggleAccountStatus(formData: FormData) {
    "use server";
    if (String(formData.get("action")) === "deactivate") {
      await deactivateAccount(personId);
    } else {
      await reactivateAccount(personId);
    }
    redirect({ href: `/admin/people/${personId}`, locale });
  }

  async function revokeSessionAction(formData: FormData) {
    "use server";
    await revokeSession(personId, String(formData.get("sessionId")));
    redirect({ href: `/admin/people/${personId}`, locale });
  }

  async function grantRole(formData: FormData) {
    "use server";
    const roleId = String(formData.get("roleId") ?? "");
    const unitId = String(formData.get("unitId") ?? "");
    await grantUnitRole(personId, roleId, unitId);
    redirect({ href: `/admin/people/${personId}`, locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading", { name: person.displayName })}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("accountStatusHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {accountStatus.status === "none" ? (
            <p className="text-sm text-muted-foreground">{t("accountStatusNone")}</p>
          ) : accountStatus.status === "active" ? (
            <>
              <p className="text-sm text-muted-foreground">{t("accountStatusActive")}</p>
              <p className="text-sm text-muted-foreground">
                {accountStatus.lastActiveAt
                  ? t("lastActive", { date: new Date(accountStatus.lastActiveAt).toLocaleString(locale) })
                  : t("neverActive")}
              </p>
              <form action={toggleAccountStatus}>
                <input type="hidden" name="action" value="deactivate" />
                <Button type="submit" variant="destructive" size="sm">
                  {t("deactivateAccount")}
                </Button>
              </form>
            </>
          ) : (
            <>
              <p className="text-sm text-muted-foreground">{t("accountStatusDisabled")}</p>
              <p className="text-sm text-muted-foreground">
                {accountStatus.lastActiveAt
                  ? t("lastActive", { date: new Date(accountStatus.lastActiveAt).toLocaleString(locale) })
                  : t("neverActive")}
              </p>
              <form action={toggleAccountStatus}>
                <input type="hidden" name="action" value="reactivate" />
                <Button type="submit" size="sm">
                  {t("reactivateAccount")}
                </Button>
              </form>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("sessionsHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {sessions.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("noSessions")}</p>
          ) : (
            sessions.map((s) => (
              <div key={s.id} className="flex items-center justify-between gap-4">
                <p className="text-sm text-muted-foreground">
                  {t("sessionLastActive", {
                    device: s.deviceLabel ?? t("sessionDeviceUnknown"),
                    date: new Date(s.lastSeenAt).toLocaleString(locale),
                  })}
                </p>
                <form action={revokeSessionAction}>
                  <input type="hidden" name="sessionId" value={s.id} />
                  <Button type="submit" variant="destructive" size="sm">
                    {t("revokeSession")}
                  </Button>
                </form>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("instanceAdminHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {instanceAdminGrant ? (
            <>
              <p className="text-sm text-muted-foreground">
                {t("isInstanceAdminYes", { date: new Date(instanceAdminGrant.grantedAt).toLocaleString(locale) })}
              </p>
              <form action={toggleInstanceAdmin}>
                <input type="hidden" name="action" value="revoke" />
                <Button type="submit" variant="destructive" size="sm">
                  {t("revokeInstanceAdmin")}
                </Button>
              </form>
            </>
          ) : (
            <>
              <p className="text-sm text-muted-foreground">{t("isInstanceAdminNo")}</p>
              <form action={toggleInstanceAdmin}>
                <input type="hidden" name="action" value="grant" />
                <Button type="submit" size="sm">
                  {t("grantInstanceAdmin")}
                </Button>
              </form>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("grantRoleHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={grantRole} className="flex flex-col gap-4">
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
            <Label className="flex flex-col items-start gap-1">
              {t("unitIdLabel")}
              <Input name="unitId" required placeholder={t("unitIdPlaceholder")} />
            </Label>
            <Button type="submit" className="self-start">
              {t("grantRole")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
