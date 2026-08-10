export interface IKindNotSupported {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:KindNotSupported";
    'parameters': {
        kind: string;
    };
}

export function isKindNotSupported(arg: any): arg is IKindNotSupported {
    return arg && arg.errorName === "Content:KindNotSupported";
}
