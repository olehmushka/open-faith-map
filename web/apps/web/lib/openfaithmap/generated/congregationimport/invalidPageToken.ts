export interface IInvalidPageToken {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "CongregationImport:InvalidPageToken";
    'parameters': {
    };
}

export function isInvalidPageToken(arg: any): arg is IInvalidPageToken {
    return arg && arg.errorName === "CongregationImport:InvalidPageToken";
}
