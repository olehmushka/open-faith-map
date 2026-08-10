export interface IAppealNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Moderation:AppealNotFound";
    'parameters': {
        appealId: string;
    };
}

export function isAppealNotFound(arg: any): arg is IAppealNotFound {
    return arg && arg.errorName === "Moderation:AppealNotFound";
}
