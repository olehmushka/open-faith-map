import { getTranslations } from "next-intl/server";

import { getPerson, mergePersons, previewMergePersons } from "@/lib/core";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// M11.8 — the "merge with another person" flow off the person detail page: find the duplicate
// (via the existing /admin/people search — no new inline search built for this, matching this
// page family's own plain-text-id convention for unitId), preview what will move, confirm. The id
// field is a plain GET <form> with no action attribute (same convention audit-log/page.tsx's own
// filter form uses — a real server-side refetch through previewMergePersons, not a client
// re-render); the confirm button is a Server Action, same shape as this page's own
// deactivateAccount/revokeSession. No confirm dialog: this app has none anywhere, and the
// preview-then-destructive-button IS its existing risk-communication convention.
export default async function SuperAdminMergePersonPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; personId: string }>;
  searchParams: Promise<{ duplicatePersonId?: string }>;
}) {
  const { locale, personId } = await params;
  const { duplicatePersonId } = await searchParams;
  const t = await getTranslations("SuperAdminMergePersonPage");

  const survivor = await getPerson(personId);

  async function confirmMerge(formData: FormData) {
    "use server";
    const duplicateId = String(formData.get("duplicatePersonId") ?? "");
    await mergePersons(personId, duplicateId);
    redirect({ href: `/admin/people/${personId}`, locale });
  }

  const preview = duplicatePersonId ? await previewMergePersons(personId, duplicatePersonId) : null;
  const duplicate = duplicatePersonId ? await getPerson(duplicatePersonId) : null;

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading", { name: survivor.displayName })}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("findHeading")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1">
              {t("duplicateIdLabel")}
              <Input
                name="duplicatePersonId"
                required
                defaultValue={duplicatePersonId ?? ""}
                placeholder={t("duplicateIdPlaceholder")}
              />
            </Label>
            <Button type="submit" className="self-start">
              {t("preview")}
            </Button>
          </form>
        </CardContent>
      </Card>

      {preview && duplicate && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("previewHeading", { name: duplicate.displayName })}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <p className="text-sm text-muted-foreground">
                {t("roleAssignmentsMove", { count: preview.roleAssignmentsToMove })}
              </p>
              <p className="text-sm text-muted-foreground">
                {t("roleAssignmentsRevoke", { count: preview.roleAssignmentsToRevokeAsRedundant })}
              </p>
              <p className="text-sm text-muted-foreground">
                {t("membershipsMove", { count: preview.membershipsToMove })}
              </p>
              <p className="text-sm text-muted-foreground">
                {t("membershipsEnd", { count: preview.membershipsToEndAsRedundant })}
              </p>
              <p className="text-sm text-muted-foreground">
                {preview.instanceAdminWillMove
                  ? t("instanceAdminWillMove")
                  : preview.instanceAdminWillBeRevokedAsRedundant
                    ? t("instanceAdminWillBeRevoked")
                    : t("instanceAdminNone")}
              </p>
              <p className="text-sm text-muted-foreground">
                {!preview.duplicateHasActiveAccount
                  ? t("accountNone")
                  : preview.accountConflict
                    ? t("accountWillBeDisabled")
                    : t("accountWillMove")}
              </p>
            </div>
            <form action={confirmMerge}>
              <input type="hidden" name="duplicatePersonId" value={duplicatePersonId} />
              <Button type="submit" variant="destructive">
                {t("confirm")}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
