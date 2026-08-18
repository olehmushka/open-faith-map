import { getTranslations } from "next-intl/server";

import { getOrgProfile, getUnit, unitAncestors } from "@/lib/core";
import { Link } from "@/i18n/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Super-admin unit detail (M10.8), read-only. getOrgProfile is .catch(() => null)'d — not every
// unit is a religion org (jurisdiction units aren't), so no profile is a normal state, matching
// admin/sites/[unitId]/page.tsx's own .catch(() => null) convention for "not found is normal" reads.
export default async function SuperAdminUnitPage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { unitId } = await params;
  const t = await getTranslations("SuperAdminUnitPage");

  const [unit, ancestors, orgProfile] = await Promise.all([
    getUnit(unitId),
    unitAncestors(unitId),
    getOrgProfile(unitId).catch(() => null),
  ]);

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{unit.name}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("detailsHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 text-sm">
          <p>
            <span className="text-muted-foreground">{t("codeLabel")}: </span>
            {unit.code ?? "—"}
          </p>
          <p>
            <span className="text-muted-foreground">{t("stateLabel")}: </span>
            {unit.state}
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
    </div>
  );
}
