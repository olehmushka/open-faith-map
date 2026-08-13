export namespace RunStatus {
    export type RUNNING = "RUNNING";
    export type SUCCEEDED = "SUCCEEDED";
    export type FAILED = "FAILED";

    export const RUNNING = "RUNNING" as "RUNNING";
    export const SUCCEEDED = "SUCCEEDED" as "SUCCEEDED";
    export const FAILED = "FAILED" as "FAILED";
}

export type RunStatus = keyof typeof RunStatus;
