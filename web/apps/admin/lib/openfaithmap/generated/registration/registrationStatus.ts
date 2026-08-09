export namespace RegistrationStatus {
    export type PENDING = "PENDING";
    export type PROVISIONING = "PROVISIONING";
    export type APPROVED = "APPROVED";
    export type REJECTED = "REJECTED";

    export const PENDING = "PENDING" as "PENDING";
    export const PROVISIONING = "PROVISIONING" as "PROVISIONING";
    export const APPROVED = "APPROVED" as "APPROVED";
    export const REJECTED = "REJECTED" as "REJECTED";
}

export type RegistrationStatus = keyof typeof RegistrationStatus;
