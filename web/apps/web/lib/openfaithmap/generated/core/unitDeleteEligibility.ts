/**
 * M12.5 — unitDeleteEligibility's response, a preview of deleteUnit's own orphan-protection outcome without deleting anything. canDelete is the AND of the four negations, computed server-side so the client never re-derives the rule.
 *
 */
export interface IUnitDeleteEligibility {
    'isRoot': boolean;
    'hasChildren': boolean;
    'hasOrgProfile': boolean;
    'hasActiveRoleAssignments': boolean;
    'canDelete': boolean;
}
