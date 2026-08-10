export interface ITaxonExcluded {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Registration:TaxonExcluded";
    'parameters': {
        taxonId: string;
    };
}

export function isTaxonExcluded(arg: any): arg is ITaxonExcluded {
    return arg && arg.errorName === "Registration:TaxonExcluded";
}
