/** status is "active", "disabled", or "none" (the person has never had a login attached). */
export interface IAccountStatus {
    'personId': string;
    'status': string;
    /**
     * M11.4 — most recent session activity (revoked-inclusive), absent for status "none" or an account that has never had a session.
     *
     */
    'lastActiveAt'?: string | null;
}
