import { IApproveCandidateRequest } from "./approveCandidateRequest";
import { ICandidate } from "./candidate";
import { ICandidatePage } from "./candidatePage";
import { ICreateJurisdictionAliasRequest } from "./createJurisdictionAliasRequest";
import { ICreateTaxonAliasRequest } from "./createTaxonAliasRequest";
import { IEditCandidateRequest } from "./editCandidateRequest";
import { IImportRun } from "./importRun";
import { IJurisdictionAlias } from "./jurisdictionAlias";
import { IJurisdictionAliasList } from "./jurisdictionAliasList";
import { IJurisdictionSyncResult } from "./jurisdictionSyncResult";
import { IRejectCandidateRequest } from "./rejectCandidateRequest";
import { IRunConnectorRequest } from "./runConnectorRequest";
import { IRunJurisdictionSyncRequest } from "./runJurisdictionSyncRequest";
import { IRunPage } from "./runPage";
import { ISuggestCoordinatesResponse } from "./suggestCoordinatesResponse";
import { ITaxonAlias } from "./taxonAlias";
import { ITaxonAliasList } from "./taxonAliasList";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Operator-triggered connector runs and the resulting candidate review queue. See docs/modules/congregationimport.md.
 *
 */
export interface ICongregationImportService {
    /**
     * Trigger sourceCode's connector. Fetches/normalizes/D-Exclusions-checks/dedup-checks and stages results as it goes — never buffers a whole run in memory. Operator-only.
     *
     */
    runConnector(request: IRunConnectorRequest): Promise<IImportRun>;
    /** List connector runs, most recent first, optionally filtered by source. */
    listRuns(sourceCode?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IRunPage>;
    getRun(runId: string): Promise<IImportRun>;
    /** List staged candidates, most recent first, optionally filtered by status and/or source. */
    listCandidates(status?: string | null, sourceCode?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<ICandidatePage>;
    getCandidate(candidateId: string): Promise<ICandidate>;
    /**
     * Correct a staged candidate's fields before approval — scraped data is noisy by nature. Only non-omitted fields are applied. Operator-only.
     *
     */
    editCandidate(candidateId: string, request: IEditCandidateRequest): Promise<ICandidate>;
    /**
     * Approve a candidate: performs the real in-process core writes (createChildOrg, a site over a new location) under the caller's own resolved subject. Grants NO congregation-admin — there is no real submitter to grant it to (D-CongregationImport). Resumable: a retry after a partial failure continues from the persisted createdUnitId rather than re-creating the unit.
     *
     */
    approveCandidate(candidateId: string, request: IApproveCandidateRequest): Promise<ICandidate>;
    /** Reject a candidate with a reason. No core writes. Operator-only. */
    rejectCandidate(candidateId: string, request: IRejectCandidateRequest): Promise<ICandidate>;
    /**
     * List every taxon alias, source-scoped first then global — small and operator-curated, no pagination (see docs/modules/congregationimport.md). Operator-only.
     *
     */
    listTaxonAliases(sourceCode?: string | null): Promise<ITaxonAliasList>;
    /**
     * Add a free-text-hint -> religion_taxa RID alias, used by matchTaxon's substring matching on future connector runs. Operator-only.
     *
     */
    createTaxonAlias(request: ICreateTaxonAliasRequest): Promise<ITaxonAlias>;
    /**
     * List every jurisdiction alias, source-scoped first then global — small and operator-curated, no pagination. Operator-only.
     *
     */
    listJurisdictionAliases(sourceCode?: string | null): Promise<IJurisdictionAliasList>;
    /**
     * Add a free-text-hint -> jurisdiction Unit RID alias, used by matchJurisdiction's substring matching on future connector runs. Advisory only — D-JurisdictionUnits: never auto-applied at approval. Operator-only.
     *
     */
    createJurisdictionAlias(request: ICreateJurisdictionAliasRequest): Promise<IJurisdictionAlias>;
    /**
     * Trigger sourceCode's JurisdictionSource (D-CatholicJurisdictionSync, docs/architecture/decisions.md) — a narrow, deliberate exception to how every other write in this module works: creates/resolves JURISDICTION-TIER directory Units (never a congregation) under the deployment's configured anchor unit, fully automatically. The trigger itself requires an operator's resolved subject (requireOperator, gated the same as every other write in this module — a real gap found and fixed at M10.6, since an earlier version of this port left the trigger ungated); the write itself runs under authz.SystemContext() (D-InProcessAuthz amendment #5), not the caller's own grant, since it is the deployment's own anchor-unit maintenance, not a per-caller action. Idempotent by natural key (source code + the source's own external id) — a re-run only creates genuinely new/changed nodes. Suitable for an unattended scheduled trigger, the same as runConnector.
     *
     */
    runJurisdictionSync(request: IRunJurisdictionSyncRequest): Promise<IJurisdictionSyncResult>;
    /**
     * Look up approximate coordinates for a candidate's address via the configured geocoding provider (application.Geocoder, Nominatim by default) — ADVISORY ONLY, never applied automatically; the operator must still call editCandidate to persist. Operator-only, and real per-provider rate-limiting is enforced server-side — never called in bulk from runConnector.
     *
     */
    suggestCoordinates(candidateId: string): Promise<ISuggestCoordinatesResponse>;
}

export class CongregationImportService implements ICongregationImportService {
    constructor(private bridge: IHttpApiBridge) {
    }

    /**
     * Trigger sourceCode's connector. Fetches/normalizes/D-Exclusions-checks/dedup-checks and stages results as it goes — never buffers a whole run in memory. Operator-only.
     *
     */
    public runConnector(request: IRunConnectorRequest): Promise<IImportRun> {
        return this.bridge.call<IImportRun>(
            "CongregationImportService",
            "runConnector",
            "POST",
            "/congregation-import/v1/runs",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** List connector runs, most recent first, optionally filtered by source. */
    public listRuns(sourceCode?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IRunPage> {
        return this.bridge.call<IRunPage>(
            "CongregationImportService",
            "listRuns",
            "GET",
            "/congregation-import/v1/runs",
            __undefined,
            __undefined,
            {
                "sourceCode": sourceCode,
                "pageSize": pageSize,
                "pageToken": pageToken,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    public getRun(runId: string): Promise<IImportRun> {
        return this.bridge.call<IImportRun>(
            "CongregationImportService",
            "getRun",
            "GET",
            "/congregation-import/v1/runs/{runId}",
            __undefined,
            __undefined,
            __undefined,
            [
                runId,
            ],
            __undefined,
            __undefined
        );
    }

    /** List staged candidates, most recent first, optionally filtered by status and/or source. */
    public listCandidates(status?: string | null, sourceCode?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<ICandidatePage> {
        return this.bridge.call<ICandidatePage>(
            "CongregationImportService",
            "listCandidates",
            "GET",
            "/congregation-import/v1/candidates",
            __undefined,
            __undefined,
            {
                "status": status,
                "sourceCode": sourceCode,
                "pageSize": pageSize,
                "pageToken": pageToken,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    public getCandidate(candidateId: string): Promise<ICandidate> {
        return this.bridge.call<ICandidate>(
            "CongregationImportService",
            "getCandidate",
            "GET",
            "/congregation-import/v1/candidates/{candidateId}",
            __undefined,
            __undefined,
            __undefined,
            [
                candidateId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * Correct a staged candidate's fields before approval — scraped data is noisy by nature. Only non-omitted fields are applied. Operator-only.
     *
     */
    public editCandidate(candidateId: string, request: IEditCandidateRequest): Promise<ICandidate> {
        return this.bridge.call<ICandidate>(
            "CongregationImportService",
            "editCandidate",
            "POST",
            "/congregation-import/v1/candidates/{candidateId}/edit",
            request,
            __undefined,
            __undefined,
            [
                candidateId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * Approve a candidate: performs the real in-process core writes (createChildOrg, a site over a new location) under the caller's own resolved subject. Grants NO congregation-admin — there is no real submitter to grant it to (D-CongregationImport). Resumable: a retry after a partial failure continues from the persisted createdUnitId rather than re-creating the unit.
     *
     */
    public approveCandidate(candidateId: string, request: IApproveCandidateRequest): Promise<ICandidate> {
        return this.bridge.call<ICandidate>(
            "CongregationImportService",
            "approveCandidate",
            "POST",
            "/congregation-import/v1/candidates/{candidateId}/approve",
            request,
            __undefined,
            __undefined,
            [
                candidateId,
            ],
            __undefined,
            __undefined
        );
    }

    /** Reject a candidate with a reason. No core writes. Operator-only. */
    public rejectCandidate(candidateId: string, request: IRejectCandidateRequest): Promise<ICandidate> {
        return this.bridge.call<ICandidate>(
            "CongregationImportService",
            "rejectCandidate",
            "POST",
            "/congregation-import/v1/candidates/{candidateId}/reject",
            request,
            __undefined,
            __undefined,
            [
                candidateId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * List every taxon alias, source-scoped first then global — small and operator-curated, no pagination (see docs/modules/congregationimport.md). Operator-only.
     *
     */
    public listTaxonAliases(sourceCode?: string | null): Promise<ITaxonAliasList> {
        return this.bridge.call<ITaxonAliasList>(
            "CongregationImportService",
            "listTaxonAliases",
            "GET",
            "/congregation-import/v1/taxon-aliases",
            __undefined,
            __undefined,
            {
                "sourceCode": sourceCode,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Add a free-text-hint -> religion_taxa RID alias, used by matchTaxon's substring matching on future connector runs. Operator-only.
     *
     */
    public createTaxonAlias(request: ICreateTaxonAliasRequest): Promise<ITaxonAlias> {
        return this.bridge.call<ITaxonAlias>(
            "CongregationImportService",
            "createTaxonAlias",
            "POST",
            "/congregation-import/v1/taxon-aliases",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * List every jurisdiction alias, source-scoped first then global — small and operator-curated, no pagination. Operator-only.
     *
     */
    public listJurisdictionAliases(sourceCode?: string | null): Promise<IJurisdictionAliasList> {
        return this.bridge.call<IJurisdictionAliasList>(
            "CongregationImportService",
            "listJurisdictionAliases",
            "GET",
            "/congregation-import/v1/jurisdiction-aliases",
            __undefined,
            __undefined,
            {
                "sourceCode": sourceCode,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Add a free-text-hint -> jurisdiction Unit RID alias, used by matchJurisdiction's substring matching on future connector runs. Advisory only — D-JurisdictionUnits: never auto-applied at approval. Operator-only.
     *
     */
    public createJurisdictionAlias(request: ICreateJurisdictionAliasRequest): Promise<IJurisdictionAlias> {
        return this.bridge.call<IJurisdictionAlias>(
            "CongregationImportService",
            "createJurisdictionAlias",
            "POST",
            "/congregation-import/v1/jurisdiction-aliases",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Trigger sourceCode's JurisdictionSource (D-CatholicJurisdictionSync, docs/architecture/decisions.md) — a narrow, deliberate exception to how every other write in this module works: creates/resolves JURISDICTION-TIER directory Units (never a congregation) under the deployment's configured anchor unit, fully automatically. The trigger itself requires an operator's resolved subject (requireOperator, gated the same as every other write in this module — a real gap found and fixed at M10.6, since an earlier version of this port left the trigger ungated); the write itself runs under authz.SystemContext() (D-InProcessAuthz amendment #5), not the caller's own grant, since it is the deployment's own anchor-unit maintenance, not a per-caller action. Idempotent by natural key (source code + the source's own external id) — a re-run only creates genuinely new/changed nodes. Suitable for an unattended scheduled trigger, the same as runConnector.
     *
     */
    public runJurisdictionSync(request: IRunJurisdictionSyncRequest): Promise<IJurisdictionSyncResult> {
        return this.bridge.call<IJurisdictionSyncResult>(
            "CongregationImportService",
            "runJurisdictionSync",
            "POST",
            "/congregation-import/v1/jurisdiction-sync/runs",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Look up approximate coordinates for a candidate's address via the configured geocoding provider (application.Geocoder, Nominatim by default) — ADVISORY ONLY, never applied automatically; the operator must still call editCandidate to persist. Operator-only, and real per-provider rate-limiting is enforced server-side — never called in bulk from runConnector.
     *
     */
    public suggestCoordinates(candidateId: string): Promise<ISuggestCoordinatesResponse> {
        return this.bridge.call<ISuggestCoordinatesResponse>(
            "CongregationImportService",
            "suggestCoordinates",
            "POST",
            "/congregation-import/v1/candidates/{candidateId}/suggest-coordinates",
            __undefined,
            __undefined,
            __undefined,
            [
                candidateId,
            ],
            __undefined,
            __undefined
        );
    }
}
