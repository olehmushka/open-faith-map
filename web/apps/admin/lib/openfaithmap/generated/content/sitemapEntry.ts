/**
 * M14.17. web/apps/web's app/sitemap.ts entry for one PAGE document — href already resolved server-side (same convention as PublicNavItem), never re-derived client-side. POST/EVENT documents have no entry of their own: they have no route besides the tenant root, which the sitemap lists separately.
 *
 */
export interface ISitemapEntry {
    'href': string;
    'updatedAt': string;
}
