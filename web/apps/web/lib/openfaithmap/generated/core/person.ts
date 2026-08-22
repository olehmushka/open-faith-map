export interface IPerson {
    'id': string;
    /** Optional stable external id, unique among active persons. */
    'code'?: string | null;
    'displayName': string;
    'createdAt': string;
    'updatedAt': string;
    /**
     * M11.4 — most recent session activity (revoked-inclusive), populated only by CoreSuperAdminService's searchPersons. Always absent from CoreService's getPerson/ getPersons, which don't compute it.
     *
     */
    'lastActiveAt'?: string | null;
}
