export namespace AppealDecision {
    export type UPHELD = "UPHELD";
    export type OVERTURNED = "OVERTURNED";

    export const UPHELD = "UPHELD" as "UPHELD";
    export const OVERTURNED = "OVERTURNED" as "OVERTURNED";
}

export type AppealDecision = keyof typeof AppealDecision;
