export namespace QueueScope {
    export type PLATFORM = "PLATFORM";
    export type CONGREGATION = "CONGREGATION";
    export type JURISDICTION = "JURISDICTION";

    export const PLATFORM = "PLATFORM" as "PLATFORM";
    export const CONGREGATION = "CONGREGATION" as "CONGREGATION";
    export const JURISDICTION = "JURISDICTION" as "JURISDICTION";
}

export type QueueScope = keyof typeof QueueScope;
