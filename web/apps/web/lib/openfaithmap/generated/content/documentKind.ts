export namespace DocumentKind {
    export type PAGE = "PAGE";
    export type POST = "POST";
    export type EVENT = "EVENT";

    export const PAGE = "PAGE" as "PAGE";
    export const POST = "POST" as "POST";
    export const EVENT = "EVENT" as "EVENT";
}

export type DocumentKind = keyof typeof DocumentKind;
