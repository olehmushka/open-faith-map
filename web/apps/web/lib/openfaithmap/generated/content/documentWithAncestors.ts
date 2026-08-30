import { IDocument } from "./document";
import { IDocumentTranslation } from "./documentTranslation";

/**
 * M14.10's page-route resolver: the leaf document plus its real ancestor chain (root first, leaf excluded), in one round trip, so the tenant-subdomain catch-all page route never needs a second call to render breadcrumbs. M14.14 adds translations in the same round trip for the same reason.
 *
 */
export interface IDocumentWithAncestors {
    'document': IDocument;
    'ancestors': Array<IDocument>;
    /**
     * Every PUBLISHED document sharing the leaf's translationGroupId (leaf included), each with its own resolved href — siblings can have a different ancestor chain/slugs per locale, so this is never derived from the leaf's own href alone. Locale here is the document's own free-text content locale, independent of the site chrome's UI language.
     *
     */
    'translations': Array<IDocumentTranslation>;
}
