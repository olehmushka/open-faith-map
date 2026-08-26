export interface ISiteNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Discovery:SiteNotFound";
    'parameters': {
    };
}

export function isSiteNotFound(arg: any): arg is ISiteNotFound {
    return arg && arg.errorName === "Discovery:SiteNotFound";
}
