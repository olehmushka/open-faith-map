/**
 * M12.2 — one move attempt's resumable state: PENDING -> NEW_EDGE_ADDED -> OLD_EDGE_REMOVED -> VERIFIED, or FAILED at any step (error then set). At most one non-FAILED job exists per (graphId, unitId) at a time.
 *
 */
export interface IUnitMoveJob {
    'id': string;
    'graphId': string;
    'unitId': string;
    'oldParentUnitId': string;
    'newParentUnitId': string;
    'status': string;
    'performedByPersonId': string;
    'error'?: string | null;
    'createdAt': string;
    'updatedAt': string;
}
