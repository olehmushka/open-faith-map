export interface IInviteAlreadyAccepted {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:InviteAlreadyAccepted";
    'parameters': {
    };
}

export function isInviteAlreadyAccepted(arg: any): arg is IInviteAlreadyAccepted {
    return arg && arg.errorName === "Core:InviteAlreadyAccepted";
}
