export interface IRequestNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Registration:RequestNotFound";
    'parameters': {
        requestId: string;
    };
}

export function isRequestNotFound(arg: any): arg is IRequestNotFound {
    return arg && arg.errorName === "Registration:RequestNotFound";
}
