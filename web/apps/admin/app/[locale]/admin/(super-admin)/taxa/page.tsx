import { getTranslations } from "next-intl/server";
import { BookOpen } from "lucide-react";

import { listTaxa } from "@/lib/core";
import { Link } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Super-admin taxonomy browser (M10.8), read-only — no taxon-mutation permission exists in
// internal/authz/domain/permissions.go, confirmed at M10.7, so there is nothing to write here.
// Empty query returns the full ~500-row catalog (core.listTaxa's own default limit), so this page
// shows a landing set rather than requiring a search first, matching the people screen's convention.
export default async function SuperAdminTaxaPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string }>;
}) {
  const t = await getTranslations("SuperAdminTaxaPage");
  const { q } = await searchParams;
  const results = await listTaxa(q, 500);

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

          {results.length === 0 && <p className="text-sm text-muted-foreground">{t("noResults")}</p>}

          {results.length > 0 && (
            <ul className="flex max-h-[32rem] flex-col divide-y overflow-y-auto rounded-md border">
              {results.map((taxon) => (
                <li key={taxon.id}>
                  <Link
                    href={`/admin/taxa/${taxon.id}`}
                    className="flex items-center gap-2 px-3 py-2 text-sm hover:bg-muted"
                  >
                    <BookOpen className="size-4 text-muted-foreground" />
                    <span className="flex-1">{taxon.name}</span>
                    <span className="text-xs text-muted-foreground">{taxon.code}</span>
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
