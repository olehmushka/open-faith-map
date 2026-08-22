/** M11.8 — what mergePersons actually did, for the audit record and the confirmation UI. */
export interface IMergeResult {
    'survivorId': string;
    'duplicatePersonId': string;
    'roleAssignmentsMoved': number;
    'roleAssignmentsRevokedRedundant': number;
    'membershipsMoved': number;
    'membershipsEnded': number;
    'instanceAdminMoved': boolean;
    'instanceAdminRevokedRedundant': boolean;
    'duplicateAccountMoved': boolean;
    'duplicateAccountDisabled': boolean;
}
