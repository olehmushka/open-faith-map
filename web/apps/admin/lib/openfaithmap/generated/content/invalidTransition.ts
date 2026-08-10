export interface IInvalidTransition {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:InvalidTransition";
    'parameters': {
        documentId: string;
        fromState: string;
        action: string;
    };
}

export function isInvalidTransition(arg: any): arg is IInvalidTransition {
    return arg && arg.errorName === "Content:InvalidTransition";
}
