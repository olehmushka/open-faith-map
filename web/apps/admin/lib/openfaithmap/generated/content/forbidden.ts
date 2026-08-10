export interface IForbidden {
    'errorCode': "PERMISSION_DENIED";
    'errorInstanceId': string;
    'errorName': "Content:Forbidden";
    'parameters': {
    };
}

export function isForbidden(arg: any): arg is IForbidden {
    return arg && arg.errorName === "Content:Forbidden";
}
