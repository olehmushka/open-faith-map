// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import {
  approveCandidate,
  editCandidate,
  listCandidates,
  rejectCandidate,
  runConnector,
  suggestCoordinates,
} from "@/lib/congregation-import";
import { listCountriesForPicker, listTaxaForPicker } from "@/lib/dictionaries";
import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { refreshRegionAroundPoint } from "@/lib/discovery";
import { redirect } from "@/i18n/navigation";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent } from "@/components/ui/card";

import { CandidateList, UNSET_OPTION } from "./candidate-list";
import { RunConnectorForm } from "./run-connector-form";

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
// for three known source codes); add a source here when a new connector is registered in main.go
// (and to run-connector-form.tsx's own PARAMETERIZED_SOURCES if it accepts run parameters).
const SOURCE_CODES = ["ua-edr", "ar-rnc", "osm"];

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
  searchParams: Promise<{ status?: string; source?: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations("CongregationImportPage");
  const { status: statusRaw, source: sourceRaw } = await searchParams;
  const status = statusRaw && statusRaw !== UNSET_OPTION ? statusRaw : undefined;
  const source = sourceRaw && sourceRaw !== UNSET_OPTION ? sourceRaw : undefined;
  const [{ candidates, nextPageToken }, taxa, countries] = await Promise.all([
    listCandidates(status, source),
    listTaxaForPicker(),
    listCountriesForPicker(locale),
  ]);
  const rootUnitId = requireRootUnitId();

  async function triggerRun(formData: FormData) {
    "use server";
    const sourceCode = String(formData.get("sourceCode") ?? "").trim();
    if (!sourceCode) return;
    // A blank field means "use the connector's own default," never "override with an empty
    // value" — only send a parameters map when the operator actually typed something.
    const countryCodes = String(formData.get("countryCodes") ?? "").trim();
    const parameters = countryCodes ? { countryCodes } : undefined;
    await runConnector(sourceCode, parameters);
    redirect({ href: "/admin/congregation-import", locale });
  }

  // Unlike approve/reject below, this does NOT redirect — saving a correction is something an
  // operator often does more than once while working a candidate (try coordinates, check the
  // result, adjust), and a full-page redirect would collapse the expanded row every time (DataTable's
  // expanded state, components/data-table.tsx, is plain useState local to that component instance —
  // a redirect unmounts and remounts the whole tree, wiping it). Returning the updated Candidate lets
  // CandidateList patch its own local row data in place instead, keeping the row open.
  async function edit(formData: FormData) {
    "use server";
    const id = String(formData.get("id"));
    const taxonIdRaw = String(formData.get("taxonId") ?? "").trim();
    const taxonId = taxonIdRaw && taxonIdRaw !== UNSET_OPTION ? taxonIdRaw : undefined;
    const countryIdRaw = String(formData.get("countryId") ?? "").trim();
    const countryId = countryIdRaw && countryIdRaw !== UNSET_OPTION ? countryIdRaw : undefined;
    const latitudeRaw = String(formData.get("latitude") ?? "").trim();
    const longitudeRaw = String(formData.get("longitude") ?? "").trim();
    return editCandidate(id, {
      taxonId,
      countryId,
      latitude: latitudeRaw ? Number(latitudeRaw) : undefined,
      longitude: longitudeRaw ? Number(longitudeRaw) : undefined,
    });
  }

  async function approve(formData: FormData) {
    "use server";
    const id = String(formData.get("id"));
    const jurisdictionUnitId = String(formData.get("jurisdictionUnitId") ?? "").trim() || undefined;
    const approved = await approveCandidate(id, jurisdictionUnitId);
    await refreshRegionAroundPoint(approved.latitude, approved.longitude);
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
    return listCandidates(status, source, undefined, pageToken);
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

  // ADVISORY ONLY — suggestCoordinates never writes anything; CoordinateSuggest just fills the
  // Latitude/Longitude fields with the result, the operator still has to click Save.
  async function suggestCoordinatesAction(candidateId: string) {
    "use server";
    return suggestCoordinates(candidateId);
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("heading")}</h1>
        <Button variant="outline" size="sm" asChild>
          <Link href="/admin/congregation-import/aliases">{t("manageAliases")}</Link>
        </Button>
      </div>

      <Card>
        <CardContent className="flex flex-wrap items-center gap-3 pt-6">
          <RunConnectorForm sourceCodes={SOURCE_CODES} action={triggerRun} />
          <form action={`/${locale}/admin/congregation-import`} className="flex gap-2">
            <Select name="status" defaultValue={status ?? UNSET_OPTION}>
              <SelectTrigger size="sm" className="w-56">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={UNSET_OPTION}>{t("allStatuses")}</SelectItem>
                {STATUSES.map((s) => (
                  <SelectItem key={s} value={s}>
                    {s}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select name="source" defaultValue={source ?? UNSET_OPTION}>
              <SelectTrigger size="sm" className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={UNSET_OPTION}>{t("allSources")}</SelectItem>
                {SOURCE_CODES.map((code) => (
                  <SelectItem key={code} value={code}>
                    {code}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button type="submit" variant="outline" size="sm">
              {t("filter")}
            </Button>
          </form>
        </CardContent>
      </Card>

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
        onSuggestCoordinates={suggestCoordinatesAction}
        labels={{
          noCandidates: t("noCandidates"),
          taxonId: t("taxonId"),
          taxonUnset: t("taxonUnset"),
          countryId: t("countryId"),
          countryUnset: t("countryUnset"),
          latitude: t("latitude"),
          longitude: t("longitude"),
          suggestCoordinates: t("suggestCoordinates"),
          suggesting: t("suggesting"),
          suggestedVia: t("suggestedVia"),
          geocodeNoMatch: t("geocodeNoMatch"),
          geocodeLookupFailed: t("geocodeLookupFailed"),
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
    </div>
  );
}
