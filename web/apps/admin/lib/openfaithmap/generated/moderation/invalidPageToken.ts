export interface IInvalidPageToken {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Moderation:InvalidPageToken";
    'parameters': {
    };
}

export function isInvalidPageToken(arg: any): arg is IInvalidPageToken {
    return arg && arg.errorName === "Moderation:InvalidPageToken";
}
