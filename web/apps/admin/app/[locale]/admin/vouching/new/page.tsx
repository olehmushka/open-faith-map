import { getTranslations } from "next-intl/server";

import { createVouch } from "@/lib/vouching";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader } from "@/components/ui/card";

// Guarantor-facing "vouch for someone" form (M6). Only requires being logged in, same as every
// other admin-app page — no separate client-side authorization gate. openfaithmap-api's own PDP
// check (religionorg.manage on guarantorCongregationUnitId) is the real access-control decision;
// a caller with no standing on the unit they name simply gets Vouching:Forbidden back.
//
// Deliberately the ONLY entry point built for filing a vouch — there is no claimant-facing "request
// a vouch" page, since that would need a congregation-claim flow this repo doesn't have yet
// (vouching.md names it as the eventual real caller). See docs/modules/vouching.md's open seams.
export default async function NewVouchPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations("NewVouchPage");

  async function submit(formData: FormData) {
    "use server";
    const claimantPersonId = String(formData.get("claimantPersonId") ?? "").trim();
    const congregationUnitId = String(formData.get("congregationUnitId") ?? "").trim();
    const guarantorCongregationUnitId = String(formData.get("guarantorCongregationUnitId") ?? "").trim();
    const statement = String(formData.get("statement") ?? "").trim() || undefined;
    if (!claimantPersonId || !congregationUnitId || !guarantorCongregationUnitId) return;
    await createVouch(claimantPersonId, congregationUnitId, guarantorCongregationUnitId, statement);
    redirect({ href: "/admin/vouching/new?submitted=1", locale });
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <Card>
        <CardHeader>
          <CardDescription>{t("intro")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form action={submit} className="flex flex-col gap-4">
            <Label className="flex flex-col items-start gap-1 text-sm">
              {t("claimantLabel")}
              <Input name="claimantPersonId" required />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-sm">
              {t("congregationLabel")}
              <Input name="congregationUnitId" required />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-sm">
              {t("guarantorCongregationLabel")}
              <Input name="guarantorCongregationUnitId" required />
            </Label>
            <Label className="flex flex-col items-start gap-1 text-sm">
              {t("statementLabel")}
              <Textarea name="statement" rows={3} />
            </Label>
            <Button type="submit" className="self-start">
              {t("submit")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
