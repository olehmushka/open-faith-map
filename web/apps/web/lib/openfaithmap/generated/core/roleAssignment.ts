export interface IRoleAssignment {
    'id': string;
    'personId': string;
    'personName': string;
    'roleId': string;
    'roleCode': string;
    'targetUnitId': string;
    'scope': string;
    'grantedAt': string;
    /** M12.3 — set only when the grant was given an expiry; the PDP already enforces it, this just surfaces it. */
    'expiresAt'?: string | null;
}
