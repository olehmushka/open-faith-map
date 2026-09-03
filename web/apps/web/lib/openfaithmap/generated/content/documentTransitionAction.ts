export namespace DocumentTransitionAction {
    export type PUBLISH = "PUBLISH";
    export type UNLIST = "UNLIST";
    export type REVERT_TO_DRAFT = "REVERT_TO_DRAFT";
    export type SCHEDULE = "SCHEDULE";

    export const PUBLISH = "PUBLISH" as "PUBLISH";
    export const UNLIST = "UNLIST" as "UNLIST";
    export const REVERT_TO_DRAFT = "REVERT_TO_DRAFT" as "REVERT_TO_DRAFT";
    export const SCHEDULE = "SCHEDULE" as "SCHEDULE";
}

export type DocumentTransitionAction = keyof typeof DocumentTransitionAction;
