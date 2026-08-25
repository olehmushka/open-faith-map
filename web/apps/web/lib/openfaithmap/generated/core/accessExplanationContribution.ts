/**
 * M12.4 — one reason an ALLOW was reached; mirrors internal/authz/domain.Contribution exactly. For an instance-plane allow only instanceAdmin is true and every other field is the empty string.
 *
 */
export interface IAccessExplanationContribution {
    'instanceAdmin': boolean;
    'assignmentId': string;
    'roleCode': string;
    'targetUnitId': string;
    'scope': string;
    'graphCode': string;
}
