/**
 * The state machine for re-parenting an already-APPROVED request's congregation unit onto a different jurisdiction unit (M4.1). AddEdge+RemoveEdge on the canonical directory graph is two non-transactional calls, not one atomic move — this tracks which one durably landed so a retry resumes rather than repeats. Add-before-remove by design: the congregation briefly has two canonical parents mid-migration rather than momentarily zero, so no subtree-scoped grant (registration-operator, platform-moderator) loses reach to it during the move.
 *
 */
export namespace ReparentStatus {
    export type PENDING = "PENDING";
    export type NEW_EDGE_ADDED = "NEW_EDGE_ADDED";
    export type OLD_EDGE_REMOVED = "OLD_EDGE_REMOVED";
    export type VERIFIED = "VERIFIED";
    export type FAILED = "FAILED";

    export const PENDING = "PENDING" as "PENDING";
    export const NEW_EDGE_ADDED = "NEW_EDGE_ADDED" as "NEW_EDGE_ADDED";
    export const OLD_EDGE_REMOVED = "OLD_EDGE_REMOVED" as "OLD_EDGE_REMOVED";
    export const VERIFIED = "VERIFIED" as "VERIFIED";
    export const FAILED = "FAILED" as "FAILED";
}

export type ReparentStatus = keyof typeof ReparentStatus;
