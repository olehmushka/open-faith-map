import { getTranslations } from "next-intl/server";
import { UserCog } from "lucide-react";

import { searchPersons } from "@/lib/core";
import { Link } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Super-admin people search (M10.8) — same list-with-search shape as admin/sites/page.tsx, over
// CoreSuperAdminService.searchPersons instead of the jurisdiction-unit search. Empty query returns
// up to 50 people (internal/identity/adapters/store.go's SearchPersons), so this page shows a
// landing set rather than requiring a search first.
export default async function SuperAdminPeoplePage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ q?: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations("SuperAdminPeoplePage");
  const { q } = await searchParams;
  const results = await searchPersons(q, 50);

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("heading")}</h1>
        <Button asChild size="sm">
          <Link href="/admin/people/invite">{t("invite")}</Link>
        </Button>
      </div>

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
            <ul className="flex flex-col divide-y rounded-md border">
              {results.map((p) => (
                <li key={p.id}>
                  <Link
                    href={`/admin/people/${p.id}`}
                    className="flex items-center gap-2 px-3 py-2 text-sm hover:bg-muted"
                  >
                    <UserCog className="size-4 text-muted-foreground" />
                    <span className="flex-1">{p.displayName}</span>
                    <span className="text-xs text-muted-foreground">
                      {p.lastActiveAt
                        ? t("lastActive", { date: new Date(p.lastActiveAt).toLocaleString(locale) })
                        : t("neverActive")}
                    </span>
                    {p.code && <span className="text-xs text-muted-foreground">{p.code}</span>}
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
