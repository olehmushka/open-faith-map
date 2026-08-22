/**
 * M11.6 — a POST body, not a path/query token, so the one-time token never lands in a server access log or browser history the way a GET with the token in the URL would.
 *
 */
export interface IResolveInviteRequest {
    'token': string;
}
