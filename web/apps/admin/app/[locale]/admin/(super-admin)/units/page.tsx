import { getTranslations } from "next-intl/server";
import { Building2 } from "lucide-react";

import { listUnits } from "@/lib/core";
import { Link } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Super-admin units browser (M10.8), read-only — no unit-mutation endpoint exists in
// api/core.conjure.yml, and none belongs here: the permission catalog has no unit-write code beyond
// createChildOrg, already covered by lib/jurisdiction.ts's own flow. Same list-with-search shape as
// admin/sites/page.tsx, over core.listUnits (free-text code/name search across the whole hierarchy,
// not scoped to jurisdiction units the way searchJurisdictionUnits is).
export default async function SuperAdminUnitsPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string }>;
}) {
  const t = await getTranslations("SuperAdminUnitsPage");
  const { q } = await searchParams;
  const results = q ? await listUnits(q, 50) : [];

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("searchHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form className="flex gap-2">
            <Input name="q" defaultValue={q ?? ""} placeholder={t("searchPlaceholder")} autoFocus />
            <Button type="submit">{t("search")}</Button>
          </form>

          {q && results.length === 0 && <p className="text-sm text-muted-foreground">{t("noResults")}</p>}

          {results.length > 0 && (
            <ul className="flex flex-col divide-y rounded-md border">
              {results.map((u) => (
                <li key={u.id}>
                  <Link
                    href={`/admin/units/${u.id}`}
                    className="flex items-center gap-2 px-3 py-2 text-sm hover:bg-muted"
                  >
                    <Building2 className="size-4 text-muted-foreground" />
                    <span className="flex-1">{u.name}</span>
                    {u.code && <span className="text-xs text-muted-foreground">{u.code}</span>}
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
