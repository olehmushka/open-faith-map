export interface IWhoami {
    'personId': string;
    /** Empty if the caller's person has no login account attached yet. */
    'accountId': string;
    /** Empty if unset. */
    'email': string;
    'isInstanceAdmin': boolean;
}
