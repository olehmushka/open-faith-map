export interface IAssignmentNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:AssignmentNotFound";
    'parameters': {
        assignmentId: string;
    };
}

export function isAssignmentNotFound(arg: any): arg is IAssignmentNotFound {
    return arg && arg.errorName === "Core:AssignmentNotFound";
}
