import { getTranslations } from "next-intl/server";

import { listAuditLog } from "@/lib/core";
import { Link } from "@/i18n/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import { AuditLogList } from "./audit-log-list";

type Filters = {
  actorPersonId?: string;
  targetKind?: string;
  targetId?: string;
  from?: string;
  to?: string;
};

// M11.2 — the audit-log viewer under (super-admin), reusing components/data-table.tsx the same way
// the moderation queue does (report-list.tsx): DataTable owns sort/global-filter/row-expansion only,
// so "Load more" keyset pagination lives here, and the structured actor/target/date filters are a
// plain GET <form> (same pattern people/page.tsx's search box uses) rather than a Server Action —
// they need a real server-side refetch through listAuditLog, not a client-side re-render over
// whatever's already been loaded.
export default async function SuperAdminAuditLogPage({
  searchParams,
}: {
  searchParams: Promise<Filters>;
}) {
  const t = await getTranslations("SuperAdminAuditLogPage");
  const filters = await searchParams;

  // date-only inputs (no time picker in this app's minimal-UI convention, see role-grants/units'
  // plain-text-id inputs) — widen to the full day in UTC so "From 2026-08-01" means the day starts,
  // not some arbitrary instant on it.
  const apiFilters = {
    actorPersonId: filters.actorPersonId,
    targetKind: filters.targetKind,
    targetId: filters.targetId,
    from: filters.from ? `${filters.from}T00:00:00Z` : undefined,
    to: filters.to ? `${filters.to}T23:59:59Z` : undefined,
  };
  const { entries, nextPageToken } = await listAuditLog(apiFilters);

  async function loadMoreEntries(pageToken: string) {
    "use server";
    return listAuditLog({ ...apiFilters, pageToken });
  }

  const hasFilters = Boolean(
    filters.actorPersonId || filters.targetKind || filters.targetId || filters.from || filters.to,
  );

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("heading")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <form className="flex flex-wrap items-end gap-3">
            <Label className="flex flex-col items-start gap-1">
              {t("actorFilterLabel")}
              <Input name="actorPersonId" defaultValue={filters.actorPersonId ?? ""} className="w-56" />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("targetKindFilterLabel")}
              <Input
                name="targetKind"
                defaultValue={filters.targetKind ?? ""}
                placeholder={t("targetKindFilterPlaceholder")}
                className="w-48"
              />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("targetIdFilterLabel")}
              <Input name="targetId" defaultValue={filters.targetId ?? ""} className="w-56" />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("fromFilterLabel")}
              <Input type="date" name="from" defaultValue={filters.from ?? ""} className="w-40" />
            </Label>
            <Label className="flex flex-col items-start gap-1">
              {t("toFilterLabel")}
              <Input type="date" name="to" defaultValue={filters.to ?? ""} className="w-40" />
            </Label>
            <Button type="submit">{t("applyFilters")}</Button>
            {hasFilters && (
              <Button type="button" variant="outline" asChild>
                <Link href="/admin/audit-log">{t("clearFilters")}</Link>
              </Button>
            )}
          </form>

          <AuditLogList
            initialEntries={entries}
            initialNextPageToken={nextPageToken}
            loadMore={loadMoreEntries}
            labels={{
              noEntries: t("noEntries"),
              filterEntries: t("filterEntries"),
              actorColumn: t("actorColumn"),
              actionColumn: t("actionColumn"),
              targetColumn: t("targetColumn"),
              whenColumn: t("whenColumn"),
              beforeLabel: t("beforeLabel"),
              afterLabel: t("afterLabel"),
              loadMore: t("loadMore"),
              loading: t("loading"),
            }}
          />
        </CardContent>
      </Card>
    </div>
  );
}
