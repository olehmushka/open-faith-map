export interface ITaxonNotFound {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Moderation:TaxonNotFound";
    'parameters': {
        taxonId: string;
    };
}

export function isTaxonNotFound(arg: any): arg is ITaxonNotFound {
    return arg && arg.errorName === "Moderation:TaxonNotFound";
}
