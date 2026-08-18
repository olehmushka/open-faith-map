export interface IForbidden {
    'errorCode': "PERMISSION_DENIED";
    'errorInstanceId': string;
    'errorName': "Registration:Forbidden";
    'parameters': {
    };
}

export function isForbidden(arg: any): arg is IForbidden {
    return arg && arg.errorName === "Registration:Forbidden";
}
