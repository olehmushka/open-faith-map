export interface IForbidden {
    'errorCode': "PERMISSION_DENIED";
    'errorInstanceId': string;
    'errorName': "Moderation:Forbidden";
    'parameters': {
    };
}

export function isForbidden(arg: any): arg is IForbidden {
    return arg && arg.errorName === "Moderation:Forbidden";
}
