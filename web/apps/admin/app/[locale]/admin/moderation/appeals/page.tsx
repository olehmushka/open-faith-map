import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { decideAppeal, listAppeals } from "@/lib/moderation";
import { redirect } from "@/i18n/navigation";

// Same no-local-gate discipline as the moderation queue page: openfaithmap-api's PDP check is the
// real access-control decision, this page just renders whatever it returns.
export default async function AppealsPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("AppealsPage");
  const { appeals } = await listAppeals("OPEN");

  async function decide(formData: FormData) {
    "use server";
    const appealId = String(formData.get("appealId"));
    const decision = String(formData.get("decision")) as "UPHELD" | "OVERTURNED";
    const note = String(formData.get("note") ?? "").trim() || undefined;
    await decideAppeal(appealId, decision, note);
    redirect({ href: "/admin/moderation/appeals", locale });
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      {appeals.length === 0 && <p>{t("noAppeals")}</p>}
      <ul className="flex flex-col gap-4">
        {appeals.map((a) => (
          <li key={a.id} className="rounded border p-4">
            <p className="text-sm">{a.statement}</p>
            <p className="text-sm text-gray-500">{t("filedAt", { date: a.createdAt })}</p>

            <form action={decide} className="mt-3 flex flex-wrap gap-2">
              <input type="hidden" name="appealId" value={a.id} />
              <input
                name="note"
                placeholder={t("notePlaceholder")}
                className="rounded border px-2 py-1 text-sm"
              />
              <button type="submit" name="decision" value="UPHELD" className="rounded border px-3 py-1 text-sm">
                {t("uphold")}
              </button>
              <button
                type="submit"
                name="decision"
                value="OVERTURNED"
                className="rounded border px-3 py-1 text-sm"
              >
                {t("overturn")}
              </button>
            </form>
          </li>
        ))}
      </ul>
    </main>
  );
}
