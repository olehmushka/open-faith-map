export interface INotEditable {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "CongregationImport:NotEditable";
    'parameters': {
        candidateId: string;
        status: string;
    };
}

export function isNotEditable(arg: any): arg is INotEditable {
    return arg && arg.errorName === "CongregationImport:NotEditable";
}
