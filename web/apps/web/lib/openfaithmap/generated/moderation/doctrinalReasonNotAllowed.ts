export interface IDoctrinalReasonNotAllowed {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Moderation:DoctrinalReasonNotAllowed";
    'parameters': {
    };
}

export function isDoctrinalReasonNotAllowed(arg: any): arg is IDoctrinalReasonNotAllowed {
    return arg && arg.errorName === "Moderation:DoctrinalReasonNotAllowed";
}
