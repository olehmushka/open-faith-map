export interface IDocumentTranslation {
    'locale': string;
    /** Root-relative, content-locale-prefixed (matches Document's own href convention) — never includes the UI chrome locale segment. */
    'href': string;
}
