export interface IUnitHasActiveRoleAssignments {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:UnitHasActiveRoleAssignments";
    'parameters': {
        unitId: string;
    };
}

export function isUnitHasActiveRoleAssignments(arg: any): arg is IUnitHasActiveRoleAssignments {
    return arg && arg.errorName === "Core:UnitHasActiveRoleAssignments";
}
