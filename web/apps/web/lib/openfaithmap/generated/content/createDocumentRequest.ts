import { DocumentKind } from "./documentKind";

export interface ICreateDocumentRequest {
    /** EVENT requires eventStartsAt to be set (Content:EventMissingStart otherwise). */
    'kind': DocumentKind;
    /** Omit to start a new translation group; set to join an existing one as another locale variant. */
    'translationGroupId'?: string | null;
    'locale': string;
    'parentDocumentId'?: string | null;
    'slug': string;
    /** EVENT only. Required when kind is EVENT; ignored for PAGE/POST. */
    'eventStartsAt'?: string | null;
    'eventEndsAt'?: string | null;
    'eventRecurrenceRrule'?: string | null;
}
