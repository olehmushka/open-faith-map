import { ICoordinate } from "./coordinate";
import { RegistrationStatus } from "./registrationStatus";

export interface IRegistrationRequest {
    /** OpenFaithMap-local RID (openfaithmap.registration.request). */
    'id': string;
    /** The go-oikumenea person RID of the prospective admin who submitted this request. */
    'submittedByPersonId': string;
    /** The selected go-oikumenea religion_taxa RID (the congregation's tradition). */
    'taxonId': string;
    'congregationName': string;
    /** The go-oikumenea location country RID. */
    'countryId': string;
    'adminArea1'?: string | null;
    'locality'?: string | null;
    'street'?: string | null;
    'houseNumber'?: string | null;
    'postalCode'?: string | null;
    'coordinate': ICoordinate;
    'status': RegistrationStatus;
    /** Set only when status = REJECTED. */
    'rejectionReason'?: string | null;
    /** The operator who approved/rejected this request; unset while PENDING. */
    'decidedByPersonId'?: string | null;
    'decidedAt'?: string | null;
    /**
     * The go-oikumenea unit RID createChildOrg produced. Set as soon as status = PROVISIONING (the one approval step that cannot be re-derived on a retry), and stays set through APPROVED.
     *
     */
    'createdUnitId'?: string | null;
    /**
     * The go-oikumenea unit RID the operator chose as this congregation's parent at approval time (D-JurisdictionUnits, M4.1) — a jurisdiction unit, or unset to fall back to the single shared root. A historical fact, not a live mirror of the current graph: if the congregation is later re-parented (reparentRequest), this field is NOT updated — ReparentingJob.newParentUnitId is the current source of truth for where it actually is.
     *
     */
    'jurisdictionUnitId'?: string | null;
    'createdAt': string;
    'updatedAt': string;
}
