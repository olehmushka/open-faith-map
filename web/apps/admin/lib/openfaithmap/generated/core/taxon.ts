export interface ITaxon {
    'id': string;
    /** Unset for a root religion. */
    'parentId'?: string | null;
    'rankId': string;
    'rankCode': string;
    'code': string;
    'name': string;
    'sortOrder'?: number | null;
}
