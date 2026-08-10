export namespace BlockTypeStatus {
    export type ACTIVE = "ACTIVE";
    export type RETIRED = "RETIRED";

    export const ACTIVE = "ACTIVE" as "ACTIVE";
    export const RETIRED = "RETIRED" as "RETIRED";
}

export type BlockTypeStatus = keyof typeof BlockTypeStatus;
