export interface IPersonNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:PersonNotFound";
    'parameters': {
        personId: string;
    };
}

export function isPersonNotFound(arg: any): arg is IPersonNotFound {
    return arg && arg.errorName === "Core:PersonNotFound";
}
