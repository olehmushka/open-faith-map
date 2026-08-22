/**
 * M11.9 — token is the bare, one-time raw secret, prefixed "ofm_" so the authenticator can cheaply distinguish it from a JWT bearer with no DB round-trip. Returned exactly once; only its hash is ever persisted server-side (identity_api_keys.token_hash) — the same one-time- secret shape InviteResult already uses.
 *
 */
export interface ICreateApiKeyResult {
    'id': string;
    'label': string;
    'permissionCodes': Array<string>;
    'token': string;
    'createdAt': string;
}
