/** status is "active", "disabled", or "none" (the person has never had a login attached). */
export interface IAccountStatus {
    'personId': string;
    'status': string;
}
