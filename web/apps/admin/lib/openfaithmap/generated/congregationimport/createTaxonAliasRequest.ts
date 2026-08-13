export interface ICreateTaxonAliasRequest {
    /** Omit to apply this alias across every source. */
    'sourceCode'?: string | null;
    'aliasText': string;
    'taxonId': string;
}
