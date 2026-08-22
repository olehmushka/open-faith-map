export interface IEmptyPersonIdsList {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:EmptyPersonIdsList";
    'parameters': {
    };
}

export function isEmptyPersonIdsList(arg: any): arg is IEmptyPersonIdsList {
    return arg && arg.errorName === "Core:EmptyPersonIdsList";
}
