/**
 * M11.7 — the batch variant of GrantUnitRoleRequest: the same role, unit, and scope, granted to every id in personIds at once, atomically. scope/graphId follow GrantUnitRoleRequest's own rules (M12.2); expiresAt follows the same rule too (M12.3).
 *
 */
export interface IBulkGrantUnitRoleRequest {
    'personIds': Array<string>;
    'roleId': string;
    'unitId': string;
    'scope': string;
    'graphId'?: string | null;
    'expiresAt'?: string | null;
}
