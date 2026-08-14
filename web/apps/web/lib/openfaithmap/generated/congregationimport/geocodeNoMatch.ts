export interface IGeocodeNoMatch {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "CongregationImport:GeocodeNoMatch";
    'parameters': {
        candidateId: string;
    };
}

export function isGeocodeNoMatch(arg: any): arg is IGeocodeNoMatch {
    return arg && arg.errorName === "CongregationImport:GeocodeNoMatch";
}
