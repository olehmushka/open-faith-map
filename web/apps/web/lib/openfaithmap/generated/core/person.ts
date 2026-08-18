export interface IPerson {
    'id': string;
    /** Optional stable external id, unique among active persons. */
    'code'?: string | null;
    'displayName': string;
    'createdAt': string;
    'updatedAt': string;
}
