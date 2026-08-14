import { getTranslations } from "next-intl/server";

import { getGuarantorStatus, listVouches, revokeGuarantor } from "@/lib/vouching";
import { redirect, Link } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { GuarantorStatusBadge } from "@/components/status-badge";

// Moderator console for guarantor standing (M6). Same no-local-gate discipline as
// /admin/moderation: openfaithmap-api's own PDP check (unit.lifecycle on the root unit) is the
// real access-control decision — this page adds no local "isModerator" check of its own, a
// non-moderator's lookup simply comes back Forbidden.
export default async function VouchingConsolePage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ guarantorPersonId?: string; claimant?: string; congregation?: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations("VouchingConsolePage");
  const { guarantorPersonId, claimant, congregation } = await searchParams;

  const guarantorStatus = guarantorPersonId ? await getGuarantorStatus(guarantorPersonId) : undefined;
  const { vouches } = claimant || congregation ? await listVouches(claimant, congregation) : { vouches: [] };

  async function revoke(formData: FormData) {
    "use server";
    const personId = String(formData.get("guarantorPersonId") ?? "").trim();
    const reason = String(formData.get("reason") ?? "").trim();
    if (!personId || !reason) return;
    await revokeGuarantor(personId, reason);
    redirect({ href: `/admin/vouching?guarantorPersonId=${encodeURIComponent(personId)}`, locale });
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("heading")}</h1>
        <Button variant="outline" size="sm" asChild>
          <Link href="/admin/vouching/new">{t("newVouchLink")}</Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("guarantorHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form className="flex gap-2">
            <Input
              name="guarantorPersonId"
              defaultValue={guarantorPersonId}
              placeholder={t("guarantorPersonIdPlaceholder")}
              className="h-8 flex-1"
            />
            <Button type="submit" variant="outline" size="sm">
              {t("lookup")}
            </Button>
          </form>

          {guarantorStatus && (
            <div className="flex flex-col gap-2 rounded-md border p-4">
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">{t("status")}:</span>
                <GuarantorStatusBadge status={guarantorStatus.status} />
              </div>
              {guarantorStatus.revokedAt && (
                <p className="text-sm text-muted-foreground">
                  {t("revokedAt", { date: guarantorStatus.revokedAt })} — {guarantorStatus.revokedReason}
                </p>
              )}
              {guarantorStatus.status === "TRUSTED" && (
                <form action={revoke} className="flex flex-wrap gap-2">
                  <input type="hidden" name="guarantorPersonId" value={guarantorStatus.guarantorPersonId} />
                  <Input name="reason" placeholder={t("reasonPlaceholder")} required className="h-8 w-56" />
                  <Button type="submit" size="sm" variant="destructive">
                    {t("revoke")}
                  </Button>
                </form>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("vouchesHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form className="flex flex-wrap gap-2">
            <Input name="claimant" defaultValue={claimant} placeholder={t("claimantPlaceholder")} className="h-8" />
            <Input
              name="congregation"
              defaultValue={congregation}
              placeholder={t("congregationPlaceholder")}
              className="h-8"
            />
            <Button type="submit" variant="outline" size="sm">
              {t("lookup")}
            </Button>
          </form>

          {vouches.length === 0 && <p className="text-sm text-muted-foreground">{t("noVouches")}</p>}
          <ul className="flex flex-col gap-3">
            {vouches.map((v) => (
              <li key={v.id} className="rounded-md border p-3">
                <p className="text-sm">{t("vouchLine", { guarantor: v.guarantorPersonId, claimant: v.claimantPersonId })}</p>
                <p className="text-sm text-muted-foreground">{t("congregationLine", { unit: v.congregationUnitId })}</p>
                {v.statement && <p className="text-sm">{v.statement}</p>}
                <p className="text-xs text-muted-foreground">{t("filedAt", { date: v.createdAt })}</p>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
