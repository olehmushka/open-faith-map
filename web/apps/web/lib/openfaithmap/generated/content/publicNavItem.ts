/**
 * M14.10. A nav item as read by openfaithmap-web: the target is already resolved into a ready-to-render href (an internal target's href walks its real ancestor chain into a hierarchical path — the caller never re-derives that), so this is a dumb render list. A nav item whose target document is missing or DRAFT is omitted from the list entirely, never surfaced as a broken link.
 *
 */
export interface IPublicNavItem {
    'label': string;
    'href': string;
    'external': boolean;
}
