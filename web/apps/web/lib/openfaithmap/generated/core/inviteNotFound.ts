export interface IInviteNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:InviteNotFound";
    'parameters': {
    };
}

export function isInviteNotFound(arg: any): arg is IInviteNotFound {
    return arg && arg.errorName === "Core:InviteNotFound";
}
