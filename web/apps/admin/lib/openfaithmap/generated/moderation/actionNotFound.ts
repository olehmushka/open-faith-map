export interface IActionNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Moderation:ActionNotFound";
    'parameters': {
        actionId: string;
    };
}

export function isActionNotFound(arg: any): arg is IActionNotFound {
    return arg && arg.errorName === "Moderation:ActionNotFound";
}
