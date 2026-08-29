export interface INavTargetAmbiguous {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:NavTargetAmbiguous";
    'parameters': {
        sortOrder: number;
    };
}

export function isNavTargetAmbiguous(arg: any): arg is INavTargetAmbiguous {
    return arg && arg.errorName === "Content:NavTargetAmbiguous";
}
