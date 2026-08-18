export interface IJurisdictionSourceNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "CongregationImport:JurisdictionSourceNotFound";
    'parameters': {
        sourceCode: string;
    };
}

export function isJurisdictionSourceNotFound(arg: any): arg is IJurisdictionSourceNotFound {
    return arg && arg.errorName === "CongregationImport:JurisdictionSourceNotFound";
}
