export interface IUpdateDocumentRequest {
    'slug'?: string | null;
    /** Set a new parent. Omit (with clearParent false) to leave the parent unchanged. */
    'parentDocumentId'?: string | null;
    /** Set true (with parentDocumentId omitted) to move the page to top level. */
    'clearParent': boolean;
}
