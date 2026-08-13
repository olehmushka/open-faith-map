import { RunStatus } from "./runStatus";

export interface IImportRun {
    /** OpenFaithMap-local RID (openfaithmap.congregationimport.run). */
    'id': string;
    'sourceCode': string;
    'status': RunStatus;
    'triggeredByPersonId': string;
    'cursorAtStart'?: string | null;
    'cursorAtEnd'?: string | null;
    'recordsFetched': number;
    'candidatesCreated': number;
    'candidatesUpdated': number;
    /** Candidates this run rejected automatically via the D-Exclusions check. */
    'candidatesAutoRejected': number;
    /** Set only when status = FAILED. */
    'error'?: string | null;
    'startedAt': string;
    'finishedAt'?: string | null;
}
