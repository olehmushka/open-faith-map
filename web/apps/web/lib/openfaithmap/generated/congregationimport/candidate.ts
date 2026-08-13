import { CandidateStatus } from "./candidateStatus";

export interface ICandidate {
    /** OpenFaithMap-local RID (openfaithmap.congregationimport.candidate). */
    'id': string;
    'importRunId'?: string | null;
    'sourceCode': string;
    /** The source's own natural key — the idempotency anchor for re-scrapes. */
    'sourceRecordId': string;
    'name': string;
    /** Free-text denomination/tradition hint as scraped, before alias resolution. */
    'taxonHint'?: string | null;
    /** The resolved go-oikumenea religion_taxa RID, once matched. */
    'taxonId'?: string | null;
    /**
     * Free-text hint naming the parish's superior jurisdiction (diocese/eparchy/synod), as scraped — only meaningful for denominations with a real institutional hierarchy (Catholic, Orthodox, Lutheran, Anglican/Episcopal).
     *
     */
    'jurisdictionHint'?: string | null;
    /**
     * An alias-matched suggestion for the go-oikumenea jurisdiction Unit RID — ADVISORY ONLY. D-JurisdictionUnits: jurisdiction is operator-assigned at approval time, never inferred. Never applied automatically; the operator must still pass jurisdictionUnitId on ApproveCandidateRequest explicitly, even when this matches it.
     *
     */
    'suggestedJurisdictionUnitId'?: string | null;
    'countryId'?: string | null;
    'adminArea1'?: string | null;
    'locality'?: string | null;
    'street'?: string | null;
    'houseNumber'?: string | null;
    'postalCode'?: string | null;
    'latitude'?: number | "NaN" | null;
    'longitude'?: number | "NaN" | null;
    'geocodePrecision'?: string | null;
    'status': CandidateStatus;
    'possibleDuplicateOfCandidateId'?: string | null;
    'possibleDuplicateOfUnitId'?: string | null;
    'rejectionReason'?: string | null;
    'reviewedByPersonId'?: string | null;
    'reviewedAt'?: string | null;
    /**
     * The go-oikumenea unit RID createChildOrg produced. Set as soon as status = PROVISIONING (the one approval step that cannot be re-derived on a retry), and stays set through PROVISIONED.
     *
     */
    'createdUnitId'?: string | null;
    'createdAt': string;
    'updatedAt': string;
}
