/** What a report or action targets. */
export namespace TargetKind {
    export type SITE = "SITE";
    export type DOCUMENT = "DOCUMENT";
    export type CONGREGATION = "CONGREGATION";
    export type VOUCHING_EDGE = "VOUCHING_EDGE";

    export const SITE = "SITE" as "SITE";
    export const DOCUMENT = "DOCUMENT" as "DOCUMENT";
    export const CONGREGATION = "CONGREGATION" as "CONGREGATION";
    export const VOUCHING_EDGE = "VOUCHING_EDGE" as "VOUCHING_EDGE";
}

export type TargetKind = keyof typeof TargetKind;
