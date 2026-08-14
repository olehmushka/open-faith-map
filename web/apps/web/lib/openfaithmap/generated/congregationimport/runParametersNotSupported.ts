export interface IRunParametersNotSupported {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "CongregationImport:RunParametersNotSupported";
    'parameters': {
        sourceCode: string;
    };
}

export function isRunParametersNotSupported(arg: any): arg is IRunParametersNotSupported {
    return arg && arg.errorName === "CongregationImport:RunParametersNotSupported";
}
