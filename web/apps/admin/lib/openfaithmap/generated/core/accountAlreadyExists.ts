export interface IAccountAlreadyExists {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:AccountAlreadyExists";
    'parameters': {
    };
}

export function isAccountAlreadyExists(arg: any): arg is IAccountAlreadyExists {
    return arg && arg.errorName === "Core:AccountAlreadyExists";
}
