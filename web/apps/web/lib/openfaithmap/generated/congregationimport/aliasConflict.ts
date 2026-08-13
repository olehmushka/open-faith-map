export interface IAliasConflict {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "CongregationImport:AliasConflict";
    'parameters': {
        aliasText: string;
    };
}

export function isAliasConflict(arg: any): arg is IAliasConflict {
    return arg && arg.errorName === "CongregationImport:AliasConflict";
}
