export interface ICandidateNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "CongregationImport:CandidateNotFound";
    'parameters': {
        candidateId: string;
    };
}

export function isCandidateNotFound(arg: any): arg is ICandidateNotFound {
    return arg && arg.errorName === "CongregationImport:CandidateNotFound";
}
