export namespace CandidateStatus {
    export type STAGED = "STAGED";
    export type NEEDS_TAXON_REVIEW = "NEEDS_TAXON_REVIEW";
    export type NEEDS_GEOCODE = "NEEDS_GEOCODE";
    export type POSSIBLE_DUPLICATE = "POSSIBLE_DUPLICATE";
    export type APPROVED = "APPROVED";
    export type PROVISIONING = "PROVISIONING";
    export type PROVISIONED = "PROVISIONED";
    export type REJECTED = "REJECTED";
    export type REJECTED_EXCLUDED = "REJECTED_EXCLUDED";

    export const STAGED = "STAGED" as "STAGED";
    export const NEEDS_TAXON_REVIEW = "NEEDS_TAXON_REVIEW" as "NEEDS_TAXON_REVIEW";
    export const NEEDS_GEOCODE = "NEEDS_GEOCODE" as "NEEDS_GEOCODE";
    export const POSSIBLE_DUPLICATE = "POSSIBLE_DUPLICATE" as "POSSIBLE_DUPLICATE";
    export const APPROVED = "APPROVED" as "APPROVED";
    export const PROVISIONING = "PROVISIONING" as "PROVISIONING";
    export const PROVISIONED = "PROVISIONED" as "PROVISIONED";
    export const REJECTED = "REJECTED" as "REJECTED";
    export const REJECTED_EXCLUDED = "REJECTED_EXCLUDED" as "REJECTED_EXCLUDED";
}

export type CandidateStatus = keyof typeof CandidateStatus;
