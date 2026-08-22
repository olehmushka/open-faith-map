export interface IInviteExpired {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:InviteExpired";
    'parameters': {
    };
}

export function isInviteExpired(arg: any): arg is IInviteExpired {
    return arg && arg.errorName === "Core:InviteExpired";
}
