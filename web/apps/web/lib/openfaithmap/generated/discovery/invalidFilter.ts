export interface IInvalidFilter {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Discovery:InvalidFilter";
    'parameters': {
    };
}

export function isInvalidFilter(arg: any): arg is IInvalidFilter {
    return arg && arg.errorName === "Discovery:InvalidFilter";
}
