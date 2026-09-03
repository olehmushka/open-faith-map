export namespace DocumentState {
    export type DRAFT = "DRAFT";
    export type PUBLISHED = "PUBLISHED";
    export type UNLISTED = "UNLISTED";
    export type SCHEDULED = "SCHEDULED";

    export const DRAFT = "DRAFT" as "DRAFT";
    export const PUBLISHED = "PUBLISHED" as "PUBLISHED";
    export const UNLISTED = "UNLISTED" as "UNLISTED";
    export const SCHEDULED = "SCHEDULED" as "SCHEDULED";
}

export type DocumentState = keyof typeof DocumentState;
