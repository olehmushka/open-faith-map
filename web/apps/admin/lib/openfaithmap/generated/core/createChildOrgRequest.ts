export interface ICreateChildOrgRequest {
    'parentUnitId': string;
    'code': string;
    'name': string;
    'orgKindId'?: string | null;
    'primaryTaxonId'?: string | null;
}
