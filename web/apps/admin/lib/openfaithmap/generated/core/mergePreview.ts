/**
 * M11.8 — what mergePersons(personId, duplicatePersonId) will move or end, computed read-only so the admin UI can show it before the caller confirms. Does not consider registration/moderation/vouching/congregationimport rows, which reference person ids as opaque text with no FK and are out of scope for this milestone.
 *
 */
export interface IMergePreview {
    'survivorId': string;
    'duplicatePersonId': string;
    'roleAssignmentsToMove': number;
    'roleAssignmentsToRevokeAsRedundant': number;
    'membershipsToMove': number;
    'membershipsToEndAsRedundant': number;
    'instanceAdminWillMove': boolean;
    'instanceAdminWillBeRevokedAsRedundant': boolean;
    'duplicateHasActiveAccount': boolean;
    /**
     * True when the survivor already has their own active account — in that case the duplicate's account is disabled (soft-merge) rather than moved, and its login stops working. False means the duplicate's account (if any) simply moves onto the survivor.
     *
     */
    'accountConflict': boolean;
}
