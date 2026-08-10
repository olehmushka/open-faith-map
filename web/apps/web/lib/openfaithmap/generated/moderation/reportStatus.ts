export namespace ReportStatus {
    export type OPEN = "OPEN";
    export type ACTIONED = "ACTIONED";
    export type DISMISSED = "DISMISSED";

    export const OPEN = "OPEN" as "OPEN";
    export const ACTIONED = "ACTIONED" as "ACTIONED";
    export const DISMISSED = "DISMISSED" as "DISMISSED";
}

export type ReportStatus = keyof typeof ReportStatus;
