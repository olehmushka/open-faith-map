export interface IRequestNotApproved {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Registration:RequestNotApproved";
    'parameters': {
        requestId: string;
        status: string;
    };
}

export function isRequestNotApproved(arg: any): arg is IRequestNotApproved {
    return arg && arg.errorName === "Registration:RequestNotApproved";
}
