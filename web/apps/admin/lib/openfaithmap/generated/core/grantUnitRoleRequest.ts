/**
 * M12.2 — scope must be "unit" or "subtree"; graphId is required when scope is "subtree" (the graph a subtree grant cascades over) and must be omitted for "unit". M12.3 — expiresAt is optional and, when set, must be in the future.
 *
 */
export interface IGrantUnitRoleRequest {
    'personId': string;
    'roleId': string;
    'unitId': string;
    'scope': string;
    'graphId'?: string | null;
    'expiresAt'?: string | null;
}
