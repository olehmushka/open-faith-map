export interface IAccountNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:AccountNotFound";
    'parameters': {
        personId: string;
    };
}

export function isAccountNotFound(arg: any): arg is IAccountNotFound {
    return arg && arg.errorName === "Core:AccountNotFound";
}
