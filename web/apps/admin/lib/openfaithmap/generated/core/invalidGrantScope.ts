export interface IInvalidGrantScope {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:InvalidGrantScope";
    'parameters': {
    };
}

export function isInvalidGrantScope(arg: any): arg is IInvalidGrantScope {
    return arg && arg.errorName === "Core:InvalidGrantScope";
}
