import { getTranslations } from "next-intl/server";

import { getSite, listFormSubmissions } from "@/lib/content";
import { redirect } from "@/i18n/navigation";

// Messages screen (M14.16, D-InAppInbox) — content.manage-gated entirely server-side, same
// discipline as /admin/block-types and /admin/patterns: no local role check here, so a caller
// without content.manage on this unit simply gets the backend's Content:Forbidden. name/email/
// message are rendered as plain text below — never dangerouslySetInnerHTML, never passed through
// the rich-text pipeline — since a submission is untrusted, anonymous input (D-InAppInbox).
export default async function MessagesPage({
  params,
}: {
  params: Promise<{ locale: string; unitId: string }>;
}) {
  const { locale, unitId } = await params;
  const t = await getTranslations("MessagesPage");
  const site = await getSite(unitId).catch(() => null);
  if (!site) return redirect({ href: `/admin/sites/${unitId}`, locale });

  const submissions = await listFormSubmissions(site.id);

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      {submissions.length === 0 && <p className="text-muted-foreground">{t("noMessages")}</p>}
      <ul className="flex flex-col gap-3">
        {submissions.map((m) => (
          <li key={m.id} className="rounded-md border p-4">
            <div className="flex items-baseline justify-between">
              <span className="font-medium">{m.name || t("anonymousLabel")}</span>
              <span className="text-sm text-muted-foreground">{new Date(m.createdAt).toLocaleString(locale)}</span>
            </div>
            {m.email && (
              <p className="text-sm text-muted-foreground">
                {t("emailLabel")}: {m.email}
              </p>
            )}
            <p className="mt-2 whitespace-pre-wrap">{m.message}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}
