import { ReparentStatus } from "./reparentStatus";

/**
 * Tracks one re-parenting attempt for an APPROVED request's congregation unit. At most one non-FAILED job may exist per congregation unit at a time (enforced by the store).
 *
 */
export interface IReparentingJob {
    /** OpenFaithMap-local RID (openfaithmap.registration.reparentingJob). */
    'id': string;
    'registrationRequestId': string;
    'congregationUnitId': string;
    'oldParentUnitId': string;
    'newParentUnitId': string;
    'status': ReparentStatus;
    'performedByPersonId': string;
    /** Set only when status = FAILED. */
    'error'?: string | null;
    'createdAt': string;
    'updatedAt': string;
}
