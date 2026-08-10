import { GuarantorStatus } from "./guarantorStatus";

export interface IGuarantorStatusRecord {
    'guarantorPersonId': string;
    'status': GuarantorStatus;
    'revokedAt'?: string | null;
    'revokedReason'?: string | null;
    'revokedByPersonId'?: string | null;
    /** Unset when this record was synthesized (no underlying row exists yet). */
    'updatedAt'?: string | null;
}
