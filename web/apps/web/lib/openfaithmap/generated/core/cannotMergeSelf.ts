export interface ICannotMergeSelf {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:CannotMergeSelf";
    'parameters': {
    };
}

export function isCannotMergeSelf(arg: any): arg is ICannotMergeSelf {
    return arg && arg.errorName === "Core:CannotMergeSelf";
}
