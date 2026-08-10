export interface IAppealActorConflict {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Moderation:AppealActorConflict";
    'parameters': {
        appealId: string;
    };
}

export function isAppealActorConflict(arg: any): arg is IAppealActorConflict {
    return arg && arg.errorName === "Moderation:AppealActorConflict";
}
