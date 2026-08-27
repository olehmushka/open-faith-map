/**
 * One entry in a document's revision history (M14.6, D-ContentRevisions) — always a past checkpoint created at publish time, never the in-progress draft (listRevisions excludes it). Carries no blocks data: history is a timestamped list to restore from, not a diff viewer.
 *
 */
export interface IDocumentRevision {
    'revisionId': string;
    'revisionNo': number;
    'createdAt': string;
    'authorPersonId'?: string | null;
    'label'?: string | null;
}
