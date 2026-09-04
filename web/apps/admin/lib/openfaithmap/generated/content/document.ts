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
    /**
     * M14.15/D-PublishOnRead. state as the public predicate actually evaluates it right now: PUBLISHED if state is literally PUBLISHED, or if state is SCHEDULED and publishAt has passed; state unchanged otherwise. Nothing ever flips the raw state column itself — the admin UI must render this field, never state, or an editor would see "Scheduled" on a page visitors already see live.
     *
     */
    'effectiveState': DocumentState;
    'publishedAt'?: string | null;
    /**
     * M14.15. Set only while state is SCHEDULED; cleared by any other transition. When this passes, the document becomes publicly visible with no further action required — see effectiveState.
     *
     */
    'publishAt'?: string | null;
    'eventStartsAt'?: string | null;
    'eventEndsAt'?: string | null;
    'eventRecurrenceRrule'?: string | null;
    /**
     * M14.17. An explicit SEO title override; absent means the public renderer derives a fallback from this document's own blocks (first heading, else the slug) at read time.
     *
     */
    'metaTitle'?: string | null;
    /** Same "absent means derive a fallback" shape as metaTitle, from the first text-bearing block. */
    'metaDescription'?: string | null;
    'createdAt': string;
    'updatedAt': string;
}
