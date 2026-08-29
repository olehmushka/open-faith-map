/**
 * M14.10. One entry in a site's hand-built nav menu — independent of parent_document_id (M14.0 replaced the page-tree-derived-nav assumption with a curated menu). Exactly one of targetDocumentId/targetUrl is ever set.
 *
 */
export interface INavItem {
    'id': string;
    'siteId': string;
    'label': string;
    'targetDocumentId'?: string | null;
    'targetUrl'?: string | null;
    'sortOrder': number;
}
