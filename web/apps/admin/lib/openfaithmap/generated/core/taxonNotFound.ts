export interface ITaxonNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:TaxonNotFound";
    'parameters': {
        taxonId: string;
    };
}

export function isTaxonNotFound(arg: any): arg is ITaxonNotFound {
    return arg && arg.errorName === "Core:TaxonNotFound";
}
