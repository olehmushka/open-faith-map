export namespace AppealStatus {
    export type OPEN = "OPEN";
    export type UPHELD = "UPHELD";
    export type OVERTURNED = "OVERTURNED";

    export const OPEN = "OPEN" as "OPEN";
    export const UPHELD = "UPHELD" as "UPHELD";
    export const OVERTURNED = "OVERTURNED" as "OVERTURNED";
}

export type AppealStatus = keyof typeof AppealStatus;
