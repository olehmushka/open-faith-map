import { getTranslations } from "next-intl/server";
import { MapPinned } from "lucide-react";

import { searchJurisdictionUnits } from "@/lib/jurisdiction";
import { Link } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// New index route for /admin/sites — previously only sites/[unitId] existed, so the sidebar and
// command palette had no landing page for "Sites" to point at. Reuses searchJurisdictionUnits
// (lib/jurisdiction.ts), the same free-text unit search already used identically by
// /admin/registrations and CongregationImportPage's JurisdictionField.
export default async function AdminSitesIndexPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string }>;
}) {
  const t = await getTranslations("AdminSitesPage");
  const { q } = await searchParams;
  const results = q ? await searchJurisdictionUnits(q) : [];

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("searchHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form className="flex gap-2">
            <Input
              name="q"
              defaultValue={q ?? ""}
              placeholder={t("searchPlaceholder")}
              autoFocus
            />
            <Button type="submit">{t("search")}</Button>
          </form>

          {q && results.length === 0 && (
            <p className="text-sm text-muted-foreground">{t("noResults")}</p>
          )}

          {results.length > 0 && (
            <ul className="flex flex-col divide-y rounded-md border">
              {results.map((unit) => (
                <li key={unit.id}>
                  <Link
                    href={`/admin/sites/${unit.id}`}
                    className="flex items-center gap-2 px-3 py-2 text-sm hover:bg-muted"
                  >
                    <MapPinned className="size-4 text-muted-foreground" />
                    <span className="flex-1">{unit.name}</span>
                    {unit.code && (
                      <span className="text-xs text-muted-foreground">{unit.code}</span>
                    )}
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
