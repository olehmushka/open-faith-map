/**
 * M11.9 — one identity_api_keys row, metadata only: no token or token hash field exists on this type at all, so it is structurally impossible for any endpoint returning it to leak the secret, regardless of caller (self-service or admin-oversight).
 *
 */
export interface IApiKey {
    'id': string;
    'label': string;
    /**
     * The owner's chosen allowlist at creation time. The effective permission set for a request authenticated with this key is this list intersected with the owning person's LIVE authz grants at request time — never wider than either alone.
     *
     */
    'permissionCodes': Array<string>;
    'createdAt': string;
    'lastUsedAt'?: string | null;
    'revokedAt'?: string | null;
}
