export interface IRunNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "CongregationImport:RunNotFound";
    'parameters': {
        runId: string;
    };
}

export function isRunNotFound(arg: any): arg is IRunNotFound {
    return arg && arg.errorName === "CongregationImport:RunNotFound";
}
