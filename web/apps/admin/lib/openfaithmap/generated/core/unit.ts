export interface IUnit {
    'id': string;
    'code'?: string | null;
    'name': string;
    /** Optional sort/filter ordinal — never a PDP input. */
    'level'?: number | null;
    'state': string;
    'createdAt': string;
    'updatedAt': string;
}
