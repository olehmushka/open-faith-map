import { IDocument } from "./document";

/**
 * M14.10's page-route resolver: the leaf document plus its real ancestor chain (root first, leaf excluded), in one round trip, so the tenant-subdomain catch-all page route never needs a second call to render breadcrumbs.
 *
 */
export interface IDocumentWithAncestors {
    'document': IDocument;
    'ancestors': Array<IDocument>;
}
