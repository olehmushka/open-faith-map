/**
 * M11.7 — the batch variant of GrantUnitRoleRequest: the same role and unit, granted to every id in personIds at once, atomically.
 *
 */
export interface IBulkGrantUnitRoleRequest {
    'personIds': Array<string>;
    'roleId': string;
    'unitId': string;
}
