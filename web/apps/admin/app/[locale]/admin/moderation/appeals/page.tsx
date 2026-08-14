import { getTranslations } from "next-intl/server";

import { decideAppeal, listAppeals } from "@/lib/moderation";
import { redirect } from "@/i18n/navigation";

import { AppealList } from "./appeal-list";

// Same no-local-gate discipline as the moderation queue page: openfaithmap-api's PDP check is the
// real access-control decision, this page just renders whatever it returns.
export default async function AppealsPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations("AppealsPage");
  const { appeals, nextPageToken } = await listAppeals("OPEN");

  async function decide(formData: FormData) {
    "use server";
    const appealId = String(formData.get("appealId"));
    const decision = String(formData.get("decision")) as "UPHELD" | "OVERTURNED";
    const note = String(formData.get("note") ?? "").trim() || undefined;
    await decideAppeal(appealId, decision, note);
    redirect({ href: "/admin/moderation/appeals", locale });
  }

  async function loadMoreAppeals(pageToken: string) {
    "use server";
    return listAppeals("OPEN", undefined, pageToken);
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <AppealList
        initialAppeals={appeals}
        initialNextPageToken={nextPageToken}
        loadMore={loadMoreAppeals}
        decide={decide}
        labels={{
          noAppeals: t("noAppeals"),
          notePlaceholder: t("notePlaceholder"),
          uphold: t("uphold"),
          overturn: t("overturn"),
          loadMore: t("loadMore"),
          loading: t("loading"),
        }}
      />
    </div>
  );
}
