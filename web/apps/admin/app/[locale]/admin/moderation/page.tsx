import { getTranslations } from "next-intl/server";

import { listReports, takeActionOnReport, type ActionKind } from "@/lib/moderation";
import { redirect } from "@/i18n/navigation";

import { ReportList } from "./report-list";

// Renders whatever listReports returns for the caller — openfaithmap-api itself decides
// platform-moderator standing by asking go-oikumenea's PDP live (Authorize against unit.lifecycle
// on the root unit), never a locally-cached role (D-Facade). This page adds no local "isModerator"
// gate of its own; a non-moderator's call simply comes back Forbidden, same discipline
// /admin/registrations already follows.
export default async function ModerationQueuePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations("ModerationQueuePage");
  const { reports, nextPageToken } = await listReports(undefined, "OPEN");

  async function takeAction(formData: FormData) {
    "use server";
    const reportId = String(formData.get("reportId"));
    const actionKind = String(formData.get("actionKind")) as ActionKind;
    const reason = String(formData.get("reason") ?? "").trim();
    if (!reason) return;
    await takeActionOnReport(reportId, actionKind, reason);
    redirect({ href: "/admin/moderation", locale });
  }

  async function loadMoreReports(pageToken: string) {
    "use server";
    return listReports(undefined, "OPEN", undefined, pageToken);
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <ReportList
        initialReports={reports}
        initialNextPageToken={nextPageToken}
        loadMore={loadMoreReports}
        takeAction={takeAction}
        labels={{
          noReports: t("noReports"),
          reasonPlaceholder: t("reasonPlaceholder"),
          takeAction: t("takeAction"),
          loadMore: t("loadMore"),
          loading: t("loading"),
        }}
      />
    </div>
  );
}
