import { getTranslations } from "next-intl/server";

import { getTaxon } from "@/lib/core";
import { Link } from "@/i18n/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Super-admin taxon detail (M10.8), read-only.
export default async function SuperAdminTaxonPage({
  params,
}: {
  params: Promise<{ locale: string; taxonId: string }>;
}) {
  const { taxonId } = await params;
  const t = await getTranslations("SuperAdminTaxonPage");

  const taxon = await getTaxon(taxonId);
  const parent = taxon.parentId ? await getTaxon(taxon.parentId).catch(() => null) : null;

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{taxon.name}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("detailsHeading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 text-sm">
          <p>
            <span className="text-muted-foreground">{t("codeLabel")}: </span>
            {taxon.code}
          </p>
          <p>
            <span className="text-muted-foreground">{t("rankLabel")}: </span>
            {taxon.rankCode}
          </p>
          <p>
            <span className="text-muted-foreground">{t("parentLabel")}: </span>
            {parent ? (
              <Link href={`/admin/taxa/${parent.id}`} className="hover:underline">
                {parent.name}
              </Link>
            ) : (
              t("noParent")
            )}
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
