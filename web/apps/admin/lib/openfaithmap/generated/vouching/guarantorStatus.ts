/**
 * Whether a guarantor is currently trusted. The absence of any vouching_guarantor_status row for a person means TRUSTED (the table's own DEFAULT) — getGuarantorStatus synthesizes that value rather than returning a not-found error.
 *
 */
export namespace GuarantorStatus {
    export type TRUSTED = "TRUSTED";
    export type REVOKED = "REVOKED";

    export const TRUSTED = "TRUSTED" as "TRUSTED";
    export const REVOKED = "REVOKED" as "REVOKED";
}

export type GuarantorStatus = keyof typeof GuarantorStatus;
