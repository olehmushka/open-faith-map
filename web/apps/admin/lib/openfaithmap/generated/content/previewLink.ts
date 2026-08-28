/**
 * M14.7. token is a short-lived, stateless, site-scoped signed token — opaque to the caller, verified by ContentPublicService's preview endpoints, never a session or a revocable row (see D-ContentRevisions).
 *
 */
export interface IPreviewLink {
    'token': string;
}
