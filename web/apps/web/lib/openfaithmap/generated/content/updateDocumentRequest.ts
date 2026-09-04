export interface IUpdateDocumentRequest {
    'slug'?: string | null;
    /** Set a new parent. Omit (with clearParent false) to leave the parent unchanged. */
    'parentDocumentId'?: string | null;
    /** Set true (with parentDocumentId omitted) to move the page to top level. */
    'clearParent': boolean;
    /**
     * M14.17. Omit to leave the stored override unchanged; an empty string clears it back to the renderer's derived fallback — never treated as "unchanged" the way omission is.
     *
     */
    'metaTitle'?: string | null;
    /** Same omit-vs-empty-string shape as metaTitle. */
    'metaDescription'?: string | null;
}
