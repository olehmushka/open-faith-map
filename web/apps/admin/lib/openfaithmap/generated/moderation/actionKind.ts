export namespace ActionKind {
    export type HIDE = "HIDE";
    export type SUSPEND = "SUSPEND";
    export type ARCHIVE = "ARCHIVE";
    export type WARN_ADMIN = "WARN_ADMIN";
    export type REVOKE_VOUCH = "REVOKE_VOUCH";
    export type REVERSE = "REVERSE";

    export const HIDE = "HIDE" as "HIDE";
    export const SUSPEND = "SUSPEND" as "SUSPEND";
    export const ARCHIVE = "ARCHIVE" as "ARCHIVE";
    export const WARN_ADMIN = "WARN_ADMIN" as "WARN_ADMIN";
    export const REVOKE_VOUCH = "REVOKE_VOUCH" as "REVOKE_VOUCH";
    export const REVERSE = "REVERSE" as "REVERSE";
}

export type ActionKind = keyof typeof ActionKind;
