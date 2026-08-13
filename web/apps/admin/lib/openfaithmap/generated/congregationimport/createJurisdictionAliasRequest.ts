export interface ICreateJurisdictionAliasRequest {
    /** Omit to apply this alias across every source. */
    'sourceCode'?: string | null;
    'aliasText': string;
    'jurisdictionUnitId': string;
}
