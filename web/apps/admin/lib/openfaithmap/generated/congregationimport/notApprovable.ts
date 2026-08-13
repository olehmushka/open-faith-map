export interface INotApprovable {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "CongregationImport:NotApprovable";
    'parameters': {
        candidateId: string;
    };
}

export function isNotApprovable(arg: any): arg is INotApprovable {
    return arg && arg.errorName === "CongregationImport:NotApprovable";
}
