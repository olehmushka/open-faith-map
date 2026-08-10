export interface IActionNotReversible {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Moderation:ActionNotReversible";
    'parameters': {
        actionId: string;
    };
}

export function isActionNotReversible(arg: any): arg is IActionNotReversible {
    return arg && arg.errorName === "Moderation:ActionNotReversible";
}
