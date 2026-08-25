import { getTranslations } from "next-intl/server";

import { listUnits, setUnitState } from "@/lib/core";
import { redirect } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import { BulkArchiveForm } from "./bulk-archive-form";

// Super-admin units browser (M10.8; full CRUD entry point since M12.5 — see units/[unitId]/page.tsx
// for create/edit/state/delete). Same list-with-search shape as admin/sites/page.tsx, over
// core.listUnits (free-text code/name search across the whole hierarchy, not scoped to jurisdiction
// units the way searchJurisdictionUnits is). Bulk-archive stays deliberately cheap (a client-side
// multi-select looping setUnitState per unit, no atomicity guarantee) rather than a real bulk backend
// endpoint like M11.7's bulkGrantUnitRole — this milestone's own scope calls for "cheap" here.
export default async function SuperAdminUnitsPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ q?: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations("SuperAdminUnitsPage");
  const { q } = await searchParams;
  const results = q ? await listUnits(q, 50) : [];

  async function bulkArchive(formData: FormData) {
    "use server";
    const unitIds = formData.getAll("unitIds").map(String);
    await Promise.all(unitIds.map((id) => setUnitState(id, "archived")));
    redirect({ href: `/admin/units?q=${encodeURIComponent(q ?? "")}`, locale });
  }

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

          {results.length > 0 && <BulkArchiveForm units={results} action={bulkArchive} />}
        </CardContent>
      </Card>
    </div>
  );
}
