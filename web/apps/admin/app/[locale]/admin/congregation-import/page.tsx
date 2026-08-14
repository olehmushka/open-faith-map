// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import {
  approveCandidate,
  editCandidate,
  listCandidates,
  rejectCandidate,
  runConnector,
} from "@/lib/congregation-import";
import { listCountriesForPicker, listTaxaForPicker } from "@/lib/dictionaries";
import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { redirect } from "@/i18n/navigation";

import { CandidateList } from "./candidate-list";

const STATUSES = [
  "STAGED",
  "NEEDS_TAXON_REVIEW",
  "NEEDS_GEOCODE",
  "POSSIBLE_DUPLICATE",
  "APPROVED",
  "PROVISIONING",
  "PROVISIONED",
  "REJECTED",
  "REJECTED_EXCLUDED",
];

// Static, not fetched — no "list registered connectors" endpoint exists (and isn't worth adding
// for two known source codes); add a source here when a new connector is registered in main.go.
const SOURCE_CODES = ["ua-edr", "ar-rnc"];

function requireRootUnitId(): string {
  const raw = process.env.REGISTRATION_ROOT_UNIT_ID?.trim();
  if (!raw) {
    throw new Error("REGISTRATION_ROOT_UNIT_ID is not set (see web/apps/admin/.env.example).");
  }
  return raw;
}

// Renders whatever listCandidates returns for the caller — openfaithmap-api itself decides
// operator standing by asking go-oikumenea's PDP live (Authorize against religionorg.manage on the
// root unit), never a locally-cached role (D-Facade). This page adds no local "isOperator" gate of
// its own; a non-operator's edit/approve/reject calls simply come back Forbidden, same discipline
// /admin/registrations and /admin/moderation already follow. Taxon/country/jurisdiction pickers go
// straight to go-oikumenea (lib/dictionaries.ts, lib/jurisdiction.ts) under the operator's own
// session token — same D-Facade reasoning /register and /admin/registrations/reparent already use,
// no openfaithmap-api involvement for these lookups.
export default async function CongregationImportPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ status?: string }>;
}) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("CongregationImportPage");
  const { status } = await searchParams;
  const [{ candidates, nextPageToken }, taxa, countries] = await Promise.all([
    listCandidates(status || undefined),
    listTaxaForPicker(locale),
    listCountriesForPicker(locale),
  ]);
  const rootUnitId = requireRootUnitId();

  async function triggerRun(formData: FormData) {
    "use server";
    const sourceCode = String(formData.get("sourceCode") ?? "").trim();
    if (!sourceCode) return;
    await runConnector(sourceCode);
    redirect({ href: "/admin/congregation-import", locale });
  }

  async function edit(formData: FormData) {
    "use server";
    const id = String(formData.get("id"));
    const taxonId = String(formData.get("taxonId") ?? "").trim() || undefined;
    const countryId = String(formData.get("countryId") ?? "").trim() || undefined;
    const latitudeRaw = String(formData.get("latitude") ?? "").trim();
    const longitudeRaw = String(formData.get("longitude") ?? "").trim();
    await editCandidate(id, {
      taxonId,
      countryId,
      latitude: latitudeRaw ? Number(latitudeRaw) : undefined,
      longitude: longitudeRaw ? Number(longitudeRaw) : undefined,
    });
    redirect({ href: "/admin/congregation-import", locale });
  }

  async function approve(formData: FormData) {
    "use server";
    const id = String(formData.get("id"));
    const jurisdictionUnitId = String(formData.get("jurisdictionUnitId") ?? "").trim() || undefined;
    await approveCandidate(id, jurisdictionUnitId);
    redirect({ href: "/admin/congregation-import", locale });
  }

  async function reject(formData: FormData) {
    "use server";
    const id = String(formData.get("id"));
    const reason = String(formData.get("reason") ?? "").trim();
    if (!reason) return;
    await rejectCandidate(id, reason);
    redirect({ href: "/admin/congregation-import", locale });
  }

  async function loadMoreCandidates(pageToken: string) {
    "use server";
    return listCandidates(status || undefined, undefined, pageToken);
  }

  // Thin "use server" wrappers — searchJurisdictionUnits/createJurisdictionUnit aren't themselves
  // Server Actions (lib/jurisdiction.ts is "server-only", not "use server"), same reason
  // loadMoreCandidates wraps listCandidates above. Called directly (not via <form action>) from
  // JurisdictionField, mirroring candidate-list.tsx's own handleLoadMore shape.
  async function searchJurisdictions(query: string) {
    "use server";
    return searchJurisdictionUnits(query);
  }

  async function createUnit(parentUnitId: string, code: string, name: string) {
    "use server";
    return createJurisdictionUnit(parentUnitId, code, name);
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>

      <section className="flex flex-wrap items-center justify-between gap-3 rounded border p-4">
        <form action={triggerRun} className="flex gap-2">
          <select name="sourceCode" defaultValue={SOURCE_CODES[0]} className="rounded border px-2 py-1 text-sm">
            {SOURCE_CODES.map((code) => (
              <option key={code} value={code}>
                {code}
              </option>
            ))}
          </select>
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("runConnector")}
          </button>
        </form>
        <form action={`/${locale}/admin/congregation-import`} className="flex gap-2">
          <select name="status" defaultValue={status ?? ""} className="rounded border px-2 py-1 text-sm">
            <option value="">{t("allStatuses")}</option>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            {t("filter")}
          </button>
        </form>
        <a href={`/${locale}/admin/congregation-import/aliases`} className="text-sm underline">
          {t("manageAliases")}
        </a>
      </section>

      <CandidateList
        initialCandidates={candidates}
        initialNextPageToken={nextPageToken}
        loadMore={loadMoreCandidates}
        onEdit={edit}
        onApprove={approve}
        onReject={reject}
        taxa={taxa}
        countries={countries}
        rootUnitId={rootUnitId}
        onSearchJurisdiction={searchJurisdictions}
        onCreateUnit={createUnit}
        labels={{
          noCandidates: t("noCandidates"),
          taxonId: t("taxonId"),
          taxonUnset: t("taxonUnset"),
          countryId: t("countryId"),
          countryUnset: t("countryUnset"),
          latitude: t("latitude"),
          longitude: t("longitude"),
          save: t("save"),
          jurisdictionUnitId: t("jurisdictionUnitId"),
          jurisdictionNone: t("jurisdictionNone"),
          jurisdictionSearchPlaceholder: t("jurisdictionSearchPlaceholder"),
          jurisdictionSearch: t("jurisdictionSearch"),
          jurisdictionNoMatches: t("jurisdictionNoMatches"),
          createUnit: t("createUnit"),
          createUnitHeading: t("createUnitHeading"),
          createUnitName: t("createUnitName"),
          createUnitCode: t("createUnitCode"),
          createUnitParentUnitId: t("createUnitParentUnitId"),
          createUnitSubmit: t("createUnitSubmit"),
          createUnitCancel: t("createUnitCancel"),
          approve: t("approve"),
          reasonPlaceholder: t("reasonPlaceholder"),
          reject: t("reject"),
          loadMore: t("loadMore"),
          loading: t("loading"),
        }}
      />
    </main>
  );
}
