export interface IParentTooDeep {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:ParentTooDeep";
    'parameters': {
        parentDocumentId: string;
    };
}

export function isParentTooDeep(arg: any): arg is IParentTooDeep {
    return arg && arg.errorName === "Content:ParentTooDeep";
}
