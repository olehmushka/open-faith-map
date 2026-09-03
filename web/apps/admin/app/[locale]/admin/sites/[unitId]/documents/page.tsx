import { getTranslations } from "next-intl/server";
import { Plus } from "lucide-react";

import { getSite, listDocuments } from "@/lib/content";
import { Link, redirect } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/status-badge";

import { DOCUMENT_STATE_TONE, documentStateLabel } from "./document-state";

export default async function DocumentsPage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { locale, unitId } = await params;
  const t = await getTranslations("DocumentsPage");
  const tState = await getTranslations("DocumentState");
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  const documents = await listDocuments(site.id);

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("heading")}</h1>
        <Button variant="outline" size="sm" asChild>
          <Link href={`/admin/sites/${unitId}/documents/new`}>
            <Plus className="size-3.5" />
            {t("newPage")}
          </Link>
        </Button>
      </div>

      {documents.length === 0 && <p className="text-muted-foreground">{t("noPages")}</p>}
      <ul className="flex flex-col gap-3">
        {documents.map((d) => (
          <li key={d.id} className="rounded-md border p-4">
            <div className="flex items-baseline justify-between">
              <Link href={`/admin/sites/${unitId}/documents/${d.id}`} className="font-medium hover:underline">
                {d.slug}
              </Link>
              <StatusBadge status={documentStateLabel(tState, d.effectiveState)} tone={DOCUMENT_STATE_TONE[d.effectiveState]} />
            </div>
            <p className="text-sm text-muted-foreground">
              {d.locale} · {d.kind}
            </p>
          </li>
        ))}
      </ul>
    </div>
  );
}
