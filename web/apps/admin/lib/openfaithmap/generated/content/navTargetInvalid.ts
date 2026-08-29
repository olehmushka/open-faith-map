export interface INavTargetInvalid {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:NavTargetInvalid";
    'parameters': {
        targetDocumentId: string;
    };
}

export function isNavTargetInvalid(arg: any): arg is INavTargetInvalid {
    return arg && arg.errorName === "Content:NavTargetInvalid";
}
