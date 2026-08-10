import { DocumentKind } from "./documentKind";
import { DocumentState } from "./documentState";

export interface IDocument {
    'id': string;
    'siteId': string;
    'kind': DocumentKind;
    /** Shared across every locale variant of one conceptual document. */
    'translationGroupId': string;
    'locale': string;
    /** Pages only, up to 3 levels deep (app-enforced, not a DB constraint). */
    'parentDocumentId'?: string | null;
    'slug': string;
    'state': DocumentState;
    'publishedAt'?: string | null;
    'eventStartsAt'?: string | null;
    'eventEndsAt'?: string | null;
    'eventRecurrenceRrule'?: string | null;
    'createdAt': string;
    'updatedAt': string;
}
