export interface ITaxonAlias {
    /** OpenFaithMap-local RID (openfaithmap.congregationimport.taxon-alias). */
    'id': string;
    /** Omitted means this alias applies across every source. */
    'sourceCode'?: string | null;
    'aliasText': string;
    /** The go-oikumenea religion_taxa RID this alias resolves to. */
    'taxonId': string;
    'createdByPersonId': string;
    'createdAt': string;
    'updatedAt': string;
}
