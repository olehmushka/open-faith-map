/**
 * M11.6 — token is the bare, one-time raw token, not a full URL: the admin app builds the shareable link from its own known origin. Returned exactly once; only its hash is ever persisted server-side.
 *
 */
export interface IInviteResult {
    'personId': string;
    'accountId': string;
    'token': string;
    'expiresAt': string;
}
