export interface ITaxonNotFound {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Registration:TaxonNotFound";
    'parameters': {
        taxonId: string;
    };
}

export function isTaxonNotFound(arg: any): arg is ITaxonNotFound {
    return arg && arg.errorName === "Registration:TaxonNotFound";
}
