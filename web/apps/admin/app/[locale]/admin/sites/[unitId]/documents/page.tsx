import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { getSite, listDocuments } from "@/lib/content";
import { Link, redirect } from "@/i18n/navigation";

export default async function DocumentsPage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { locale, unitId } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("DocumentsPage");
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  const documents = await listDocuments(site.id);

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <Link href={`/admin/sites/${unitId}/documents/new`} className="underline">
        {t("newPage")}
      </Link>

      {documents.length === 0 && <p>{t("noPages")}</p>}
      <ul className="flex flex-col gap-3">
        {documents.map((d) => (
          <li key={d.id} className="rounded border p-4">
            <div className="flex items-baseline justify-between">
              <Link href={`/admin/sites/${unitId}/documents/${d.id}`} className="font-medium underline">
                {d.slug}
              </Link>
              <span className={`text-sm ${d.state === "DRAFT" ? "font-semibold" : ""}`}>
                {d.state === "DRAFT" ? t("draftLabel") : d.state}
              </span>
            </div>
            <p className="text-sm">
              {d.locale} · {d.kind}
            </p>
          </li>
        ))}
      </ul>
    </main>
  );
}
