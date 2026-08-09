export interface IRequestNotPending {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Registration:RequestNotPending";
    'parameters': {
        requestId: string;
        status: string;
    };
}

export function isRequestNotPending(arg: any): arg is IRequestNotPending {
    return arg && arg.errorName === "Registration:RequestNotPending";
}
