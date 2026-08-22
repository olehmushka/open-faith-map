export interface IUnknownPermissionCode {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:UnknownPermissionCode";
    'parameters': {
    };
}

export function isUnknownPermissionCode(arg: any): arg is IUnknownPermissionCode {
    return arg && arg.errorName === "Core:UnknownPermissionCode";
}
