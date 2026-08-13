export interface IForbidden {
    'errorCode': "PERMISSION_DENIED";
    'errorInstanceId': string;
    'errorName': "CongregationImport:Forbidden";
    'parameters': {
    };
}

export function isForbidden(arg: any): arg is IForbidden {
    return arg && arg.errorName === "CongregationImport:Forbidden";
}
