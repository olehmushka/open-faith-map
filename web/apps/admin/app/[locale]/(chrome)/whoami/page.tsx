import { getTranslations } from "next-intl/server";

import { auth, signOut } from "@/auth";
import { getPerson, listMyRoleAssignments, listMySessions, revokeMySession, updateMyProfile, whoami } from "@/lib/core";
import { Link, redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// M11.5 — the self-service profile page, replacing the raw JSON dump this route used to render.
// M1's own exit-criterion proof (a real openfaithmap-api call with the logged-in user's forwarded
// Google ID token) still lives in the whoami() call below, just rendered as real UI now instead of
// json.Stringify'd. M10.7: repointed from go-oikumenea's identityFederation.whoami() to this app's
// own core.conjure.yml (lib/core.ts).
export default async function WhoamiPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const session = await auth();
  if (!session) {
    redirect({ href: "/login", locale });
  }

  const t = await getTranslations("WhoamiPage");
  const who = await whoami();
  const [person, roleAssignments, sessions] = await Promise.all([
    getPerson(who.personId),
    listMyRoleAssignments(),
    listMySessions(),
  ]);

  async function updateProfileAction(formData: FormData) {
    "use server";
    await updateMyProfile(String(formData.get("displayName") ?? ""));
    redirect({ href: "/whoami", locale });
  }

  async function revokeMySessionAction(formData: FormData) {
    "use server";
    await revokeMySession(String(formData.get("sessionId")));
    redirect({ href: "/whoami", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p>{t.rich("description", { code: (chunks) => <code>{chunks}</code> })}</p>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("profileHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form action={updateProfileAction} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("displayNameLabel")}
              <Input name="displayName" required defaultValue={person.displayName} placeholder={t("displayNamePlaceholder")} />
            </Label>
            <Button type="submit" className="self-start">
              {t("saveProfile")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("rolesHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <p className="text-sm text-muted-foreground">
            {who.isInstanceAdmin ? t("isInstanceAdminYes") : t("isInstanceAdminNo")}
          </p>
          {roleAssignments.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("noRoles")}</p>
          ) : (
            roleAssignments.map((a) => (
              <p key={a.id} className="text-sm text-muted-foreground">
                {t("roleAssignment", { roleName: a.roleCode, unitId: a.targetUnitId })}
              </p>
            ))
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
                  {s.isCurrent ? ` ${t("sessionCurrent")}` : ""}
                </p>
                <form action={revokeMySessionAction}>
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

      <nav className="flex flex-col gap-2 text-sm">
        <Link href="/register" className="underline">
          {t("registerLink")}
        </Link>
        <Link href="/admin/registrations" className="underline">
          {t("reviewLink")}
        </Link>
        <Link href="/my-congregation" className="underline">
          {t("myCongregationLink")}
        </Link>
        <Link href="/api-keys" className="underline">
          {t("apiKeysLink")}
        </Link>
      </nav>

      <form
        action={async () => {
          "use server";
          await signOut({ redirectTo: `/${locale}` });
        }}
      >
        <button type="submit" className="rounded border px-4 py-2">
          {t("signOut")}
        </button>
      </form>
    </main>
  );
}
