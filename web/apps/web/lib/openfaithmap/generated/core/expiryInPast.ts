export interface IExpiryInPast {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:ExpiryInPast";
    'parameters': {
    };
}

export function isExpiryInPast(arg: any): arg is IExpiryInPast {
    return arg && arg.errorName === "Core:ExpiryInPast";
}
