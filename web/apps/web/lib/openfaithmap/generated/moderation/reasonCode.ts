/**
 * Catalog-typed report reason. OTHER accepts free text via Report.detail. Deliberately has no DOCTRINAL_CONCERN value (docs/modules/moderation.md's invariant: doctrinal disputes are not adjudicated — they are filed under OTHER and moderators decline to act on doctrinal grounds alone).
 *
 */
export namespace ReasonCode {
    export type SPAM = "SPAM";
    export type INCORRECT_INFORMATION = "INCORRECT_INFORMATION";
    export type INAPPROPRIATE_CONTENT = "INAPPROPRIATE_CONTENT";
    export type DUPLICATE = "DUPLICATE";
    export type OTHER = "OTHER";

    export const SPAM = "SPAM" as "SPAM";
    export const INCORRECT_INFORMATION = "INCORRECT_INFORMATION" as "INCORRECT_INFORMATION";
    export const INAPPROPRIATE_CONTENT = "INAPPROPRIATE_CONTENT" as "INAPPROPRIATE_CONTENT";
    export const DUPLICATE = "DUPLICATE" as "DUPLICATE";
    export const OTHER = "OTHER" as "OTHER";
}

export type ReasonCode = keyof typeof ReasonCode;
