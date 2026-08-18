export interface IJurisdictionAlias {
    /** OpenFaithMap-local RID (openfaithmap.congregationimport.jurisdiction-alias). */
    'id': string;
    /** Omitted means this alias applies across every source. */
    'sourceCode'?: string | null;
    'aliasText': string;
    /** The jurisdiction Unit RID this alias resolves to (D-JurisdictionUnits). */
    'jurisdictionUnitId': string;
    'createdByPersonId': string;
    'createdAt': string;
    'updatedAt': string;
}
