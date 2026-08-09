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
    'createdAt': string;
    'updatedAt': string;
}
