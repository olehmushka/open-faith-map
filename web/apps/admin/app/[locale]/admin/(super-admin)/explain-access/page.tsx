import { getTranslations } from "next-intl/server";

import { explainAccess, type AccessExplanationContribution } from "@/lib/core";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/status-badge";

type Filters = {
  subjectPersonId?: string;
  permissionCode?: string;
  unitId?: string;
};

// M12.4 — decision-tracing debug tool over CoreSuperAdminService.explainAccess: "why does this
// person have (or not have) this access", matching the role Google Cloud Policy Analyzer / AWS
// IAM Policy Simulator play in the platforms researched at this milestone's discovery pass. A
// plain GET <form> over three plain-text-id inputs, same minimal-UI convention as audit-log's own
// filter form — this is a lookup, not a mutation, so there's no Server Action here.
export default async function SuperAdminExplainAccessPage({
  searchParams,
}: {
  searchParams: Promise<Filters>;
}) {
  const t = await getTranslations("SuperAdminExplainAccessPage");
  const { subjectPersonId, permissionCode, unitId } = await searchParams;

  const hasQuery = Boolean(subjectPersonId && permissionCode && unitId);
  const explanation = hasQuery
    ? await explainAccess(subjectPersonId!, permissionCode!, unitId!)
    : null;

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("formHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form className="flex flex-wrap items-end gap-3">
            <Label className="flex flex-col items-start gap-1">
              {t("subjectPersonIdLabel")}
              <Input
                name="subjectPersonId"
                defaultValue={subjectPersonId ?? ""}
                className="w-64"
                required
              />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("permissionCodeLabel")}
              <Input
                name="permissionCode"
                defaultValue={permissionCode ?? ""}
                placeholder={t("permissionCodePlaceholder")}
                className="w-56"
                required
              />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("unitIdLabel")}
              <Input name="unitId" defaultValue={unitId ?? ""} className="w-64" required />
            </Label>
            <Button type="submit">{t("submit")}</Button>
          </form>
        </CardContent>
      </Card>

      {explanation && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">{t("resultHeading")}</CardTitle>
            <StatusBadge
              status={explanation.allow ? t("allow") : t("deny")}
              tone={explanation.allow ? "success" : "danger"}
            />
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {explanation.allow ? (
              explanation.via.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("noContributions")}</p>
              ) : (
                <ul className="flex flex-col divide-y rounded-md border">
                  {explanation.via.map((c: AccessExplanationContribution, i: number) => (
                    <li key={i} className="px-3 py-2 text-sm">
                      {c.instanceAdmin
                        ? t("instanceAdminContribution")
                        : t("assignmentContribution", {
                            role: c.roleCode,
                            unit: c.targetUnitId,
                            scope: c.scope,
                            graph: c.graphCode || t("noGraph"),
                          })}
                    </li>
                  ))}
                </ul>
              )
            ) : (
              <p className="text-sm">{t("denyReasonLabel", { reason: explanation.denyReason })}</p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
