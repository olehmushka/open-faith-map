export interface ISessionNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:SessionNotFound";
    'parameters': {
        sessionId: string;
    };
}

export function isSessionNotFound(arg: any): arg is ISessionNotFound {
    return arg && arg.errorName === "Core:SessionNotFound";
}
