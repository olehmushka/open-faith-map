export interface IJurisdictionSyncResult {
    'sourceCode': string;
    'nodesFetched': number;
    'unitsCreated': number;
    /** Nodes already CREATED on a prior run — real work, correctly not repeated. */
    'unitsSkipped': number;
    'unitsFailed': number;
    'aliasesCreated': number;
}
